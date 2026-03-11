package render

import (
	"image"
	"image/color"
	"image/draw"
)

// CombineHorizontal places left and right images side by side with a gap.
func CombineHorizontal(left, right image.Image) *image.NRGBA {
	lb := left.Bounds()
	rb := right.Bounds()

	gap := 20
	totalWidth := lb.Dx() + gap + rb.Dx()
	maxHeight := lb.Dy()
	if rb.Dy() > maxHeight {
		maxHeight = rb.Dy()
	}

	combined := image.NewNRGBA(image.Rect(0, 0, totalWidth, maxHeight))

	// White background
	draw.Draw(combined, combined.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Left image
	draw.Draw(combined, image.Rect(0, 0, lb.Dx(), lb.Dy()), left, lb.Min, draw.Over)

	// Right image
	rightX := lb.Dx() + gap
	draw.Draw(combined, image.Rect(rightX, 0, totalWidth, rb.Dy()), right, rb.Min, draw.Over)

	return combined
}
