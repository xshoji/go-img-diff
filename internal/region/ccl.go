package region

import (
	"image"
	"log/slog"

	"github.com/xshoji/go-img-diff/internal/core"
)

// Extract performs connected-component labeling on the diff mask and returns regions.
// Steps:
// 1. Optional dilation to bridge small gaps
// 2. 8-connected CCL via BFS
// 3. Filter by MinArea
// 4. Add padding to bounding boxes
// 5. Merge overlapping bounding boxes (single pass)
func Extract(mask *core.Mask, opts core.RegionOptions, logger *slog.Logger) []core.Region {
	w, h := mask.W, mask.H

	// Step 1: Optional dilation
	data := mask.Data
	if opts.DilateRadius > 0 {
		data = dilate(mask.Data, w, h, opts.DilateRadius)
	}

	// Step 2: 8-connected CCL via BFS
	visited := make([]bool, w*h)
	var regions []core.Region

	// 8-connected neighbor offsets
	dx := [8]int{-1, 0, 1, -1, 1, -1, 0, 1}
	dy := [8]int{-1, -1, -1, 0, 0, 1, 1, 1}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if data[idx] == 0 || visited[idx] {
				continue
			}

			// BFS flood fill
			queue := []int{idx}
			visited[idx] = true
			minX, minY, maxX, maxY := x, y, x, y
			area := 0

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]
				cx := curr % w
				cy := curr / w
				area++

				if cx < minX {
					minX = cx
				}
				if cx > maxX {
					maxX = cx
				}
				if cy < minY {
					minY = cy
				}
				if cy > maxY {
					maxY = cy
				}

				for d := 0; d < 8; d++ {
					nx, ny := cx+dx[d], cy+dy[d]
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					nidx := ny*w + nx
					if !visited[nidx] && data[nidx] != 0 {
						visited[nidx] = true
						queue = append(queue, nidx)
					}
				}
			}

			// Step 3: Filter by MinArea
			if area < opts.MinArea {
				continue
			}

			// Step 4: Add padding
			minX = max(0, minX-opts.Padding)
			minY = max(0, minY-opts.Padding)
			maxX = min(w-1, maxX+opts.Padding)
			maxY = min(h-1, maxY+opts.Padding)

			regions = append(regions, core.Region{
				Bounds: image.Rect(minX, minY, maxX+1, maxY+1),
				Area:   area,
			})
		}
	}

	// Step 5: Merge overlapping bounding boxes
	merged := mergeOverlapping(regions)

	logger.Info("region extraction complete", "raw", len(regions), "merged", len(merged))
	return merged
}

// dilate performs morphological dilation on a binary mask with the given radius.
func dilate(src []uint8, w, h, radius int) []uint8 {
	dst := make([]uint8, len(src))
	copy(dst, src)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if src[y*w+x] != 0 {
				continue
			}
			// Check if any neighbor within radius is set
			found := false
			for dy := -radius; dy <= radius && !found; dy++ {
				for dx := -radius; dx <= radius && !found; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < w && ny >= 0 && ny < h {
						if src[ny*w+nx] != 0 {
							found = true
						}
					}
				}
			}
			if found {
				dst[y*w+x] = 1
			}
		}
	}

	return dst
}

// mergeOverlapping merges regions whose bounding boxes overlap or touch.
// Uses a simple iterative approach: keep merging until no changes occur.
func mergeOverlapping(regions []core.Region) []core.Region {
	if len(regions) <= 1 {
		return regions
	}

	result := make([]core.Region, len(regions))
	copy(result, regions)

	changed := true
	for changed {
		changed = false
		for i := 0; i < len(result); i++ {
			for j := i + 1; j < len(result); j++ {
				if result[i].Bounds.Overlaps(result[j].Bounds) ||
					touches(result[i].Bounds, result[j].Bounds) {
					// Merge j into i
					result[i] = core.Region{
						Bounds: result[i].Bounds.Union(result[j].Bounds),
						Area:   result[i].Area + result[j].Area,
					}
					// Remove j
					result = append(result[:j], result[j+1:]...)
					changed = true
					j--
				}
			}
		}
	}

	return result
}

// touches returns true if two rectangles are adjacent (share an edge but don't overlap).
func touches(a, b image.Rectangle) bool {
	// Expand a by 1 pixel and check overlap
	expanded := image.Rect(a.Min.X-1, a.Min.Y-1, a.Max.X+1, a.Max.Y+1)
	return expanded.Overlaps(b)
}
