package render

import (
	"image"
	"image/color"
	"image/draw"
	"log/slog"

	"github.com/xshoji/go-img-diff/internal/core"
)

// Render creates the diff visualization image.
// Base: frame B. Overlay: aligned pixels from A with tint on diff pixels. Borders: around regions.
func Render(a, b *core.Frame, mask *core.Mask, regions []core.Region, rowAlign core.RowAlignment, opts core.RenderOptions, logger *slog.Logger) *image.NRGBA {
	w := max(a.W, b.W)
	h := max(a.H, b.H)
	result := image.NewNRGBA(image.Rect(0, 0, w, h))

	// Draw frame B as base
	draw.Draw(result, image.Rect(0, 0, b.W, b.H), b.Pix, image.Point{}, draw.Src)

	// Apply overlay on diff pixels only
	if opts.DrawOverlay {
		for _, region := range regions {
			r := region.Bounds
			bw := opts.BorderWidth
			// Only overlay inside the border area
			innerMinX := r.Min.X + bw
			innerMinY := r.Min.Y + bw
			innerMaxX := r.Max.X - bw
			innerMaxY := r.Max.Y - bw

			for y := innerMinY; y < innerMaxY; y++ {
				for x := innerMinX; x < innerMaxX; x++ {
					if x < 0 || x >= w || y < 0 || y >= h {
						continue
					}

					// Only overlay on actual diff pixels from the mask
					if !mask.Get(x, y) {
						continue
					}

					// Source pixel from A (aligned)
					srcY := rowAlign.SrcYAt(x, y)
					if srcY == -1 {
						continue
					}
					dx := rowAlign.DXAt(x, y)
					srcX := x - dx
					if srcX < 0 || srcX >= a.W || srcY < 0 || srcY >= a.H {
						continue
					}

					dstColor := result.NRGBAAt(x, y)
					srcColor := a.Pix.NRGBAAt(srcX, srcY)

					blended := core.BlendColors(
						dstColor, srcColor,
						opts.OverlayAlpha,
						opts.TintColor,
						opts.TintEnabled,
						opts.TintStrength,
						opts.TintTransparency,
					)
					result.SetNRGBA(x, y, blended)
				}
			}
		}
	}

	// Draw borders around regions
	for _, region := range regions {
		drawBorder(result, region.Bounds, opts.BorderColor, opts.BorderWidth)
	}

	logger.Info("render complete", "regions", len(regions), "size", [2]int{w, h})
	return result
}

// drawBorder draws a rectangular border of the given width and color.
func drawBorder(img *image.NRGBA, rect image.Rectangle, c color.NRGBA, width int) {
	bounds := img.Bounds()

	// Clamp rect to image bounds
	r := rect.Intersect(bounds)
	if r.Empty() {
		return
	}

	// Top and bottom edges
	for x := r.Min.X; x < r.Max.X; x++ {
		for i := 0; i < width; i++ {
			if y := r.Min.Y + i; y < r.Max.Y && y < bounds.Max.Y {
				img.SetNRGBA(x, y, c)
			}
			if y := r.Max.Y - 1 - i; y >= r.Min.Y && y >= bounds.Min.Y {
				img.SetNRGBA(x, y, c)
			}
		}
	}

	// Left and right edges
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for i := 0; i < width; i++ {
			if x := r.Min.X + i; x < r.Max.X && x < bounds.Max.X {
				img.SetNRGBA(x, y, c)
			}
			if x := r.Max.X - 1 - i; x >= r.Min.X && x >= bounds.Min.X {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}
