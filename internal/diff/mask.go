package diff

import (
	"log/slog"

	"github.com/xshoji/go-img-diff/internal/core"
)

// BuildMask compares two aligned frames and produces a binary diff mask.
// The mask is in frame B's coordinate space.
// Metric: max(|dR|, |dG|, |dB|) > threshold.
func BuildMask(a, b *core.Frame, al core.Alignment, opts core.DiffOptions, logger *slog.Logger) *core.Mask {
	mask := core.NewMask(b.W, b.H)

	threshold := opts.Threshold

	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			// Corresponding position in frame A
			ax := x - al.DX
			ay := y - al.DY

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
				if opts.StopAfterFirst {
					return mask
				}
			}
		}
	}

	logger.Info("diff mask built", "width", b.W, "height", b.H, "diffPixels", mask.Count)
	return mask
}

func absDiffU8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
