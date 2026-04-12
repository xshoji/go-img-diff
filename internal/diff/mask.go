package diff

import (
	"log/slog"
	"math"

	"github.com/xshoji/go-img-diff/internal/core"
)

// BuildMask compares two aligned frames and produces a binary diff mask.
// The mask is in frame B's coordinate space.
// Metric: max(|dR|, |dG|, |dB|) > threshold.
func BuildMask(a, b *core.Frame, rowAlign core.RowAlignment, opts core.DiffOptions, logger *slog.Logger) *core.Mask {
	mask := core.NewMask(b.W, b.H)

	threshold := opts.Threshold
	earlyExit := opts.StopAfterFirst && !shouldApplyNoiseFilter(opts)

	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			srcY := rowAlign.SrcYAt(x, y)
			dx := rowAlign.DXAt(x, y)
			if srcY == -1 {
				mask.Set(x, y)
				if earlyExit {
					return mask
				}
				continue
			}

			// Corresponding position in frame A
			ax := x - dx
			ay := srcY

			// Out of bounds in A → skip (not comparable)
			if ax < 0 || ax >= a.W || ay < 0 || ay >= a.H {
				continue
			}

			// Read pixel values directly from NRGBA pixel slices
			aOff := ay*a.Pix.Stride + ax*4
			bOff := y*b.Pix.Stride + x*4

			ar := a.Pix.Pix[aOff]
			ag := a.Pix.Pix[aOff+1]
			ab := a.Pix.Pix[aOff+2]
			br := b.Pix.Pix[bOff]
			bg := b.Pix.Pix[bOff+1]
			bb := b.Pix.Pix[bOff+2]

			dr := absDiffU8(ar, br)
			dg := absDiffU8(ag, bg)
			db := absDiffU8(ab, bb)

			maxDiff := dr
			if dg > maxDiff {
				maxDiff = dg
			}
			if db > maxDiff {
				maxDiff = db
			}

			if maxDiff > threshold {
				mask.Set(x, y)
				if earlyExit {
					return mask
				}
			}
		}
	}

	if shouldApplyNoiseFilter(opts) {
		rawCount := mask.Count
		filterSparseNoise(mask, opts.NoiseWindowSize, opts.NoiseMinDiffRatio)
		logger.Info("diff noise filter applied",
			"windowSize", normalizeNoiseWindowSize(opts.NoiseWindowSize),
			"minDiffRatio", opts.NoiseMinDiffRatio,
			"rawDiffPixels", rawCount,
			"filteredDiffPixels", mask.Count,
		)
	}

	logger.Info("diff mask built", "width", b.W, "height", b.H, "diffPixels", mask.Count)
	return mask
}

func shouldApplyNoiseFilter(opts core.DiffOptions) bool {
	return opts.NoiseWindowSize > 1 && opts.NoiseMinDiffRatio > 0
}

func normalizeNoiseWindowSize(windowSize int) int {
	if windowSize <= 1 {
		return 0
	}
	if windowSize%2 == 0 {
		windowSize++
	}
	return windowSize
}

func filterSparseNoise(mask *core.Mask, windowSize int, minDiffRatio float64) {
	windowSize = normalizeNoiseWindowSize(windowSize)
	if windowSize == 0 || minDiffRatio <= 0 || mask.Count == 0 {
		return
	}

	radius := windowSize / 2
	prefix := make([]int, (mask.W+1)*(mask.H+1))
	for y := 0; y < mask.H; y++ {
		rowSum := 0
		for x := 0; x < mask.W; x++ {
			rowSum += int(mask.Data[y*mask.W+x])
			idx := (y+1)*(mask.W+1) + (x + 1)
			prefix[idx] = prefix[y*(mask.W+1)+(x+1)] + rowSum
		}
	}

	filtered := make([]uint8, len(mask.Data))
	count := 0
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			if mask.Data[y*mask.W+x] == 0 {
				continue
			}

			x0 := max(0, x-radius)
			x1 := min(mask.W, x+radius+1)
			y0 := max(0, y-radius)
			y1 := min(mask.H, y+radius+1)
			windowArea := (x1 - x0) * (y1 - y0)
			if windowArea == 0 {
				continue
			}

			diffCount := sumRect(prefix, mask.W+1, x0, y0, x1, y1)
			if float64(diffCount)/float64(windowArea) >= minDiffRatio-math.SmallestNonzeroFloat64 {
				filtered[y*mask.W+x] = 1
				count++
			}
		}
	}

	mask.Data = filtered
	mask.Count = count
}

func sumRect(prefix []int, stride, minX, minY, maxX, maxY int) int {
	return prefix[maxY*stride+maxX] - prefix[minY*stride+maxX] - prefix[maxY*stride+minX] + prefix[minY*stride+minX]
}

func absDiffU8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
