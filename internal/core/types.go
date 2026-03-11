package core

import (
	"image"
	"image/color"
	"image/draw"
)

// Frame is a normalized image with origin at (0,0) in NRGBA format.
// It also caches a grayscale version for alignment.
type Frame struct {
	W, H int
	Pix  *image.NRGBA
	Gray []uint8 // row-major grayscale cache (W*H)
}

// NewFrame normalizes any image.Image into a Frame.
func NewFrame(img image.Image) *Frame {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)

	gray := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := y*nrgba.Stride + x*4
			r := nrgba.Pix[off]
			g := nrgba.Pix[off+1]
			b := nrgba.Pix[off+2]
			// ITU-R BT.601 luminance
			gray[y*w+x] = uint8((19595*uint32(r) + 38470*uint32(g) + 7471*uint32(b) + 1<<15) >> 16)
		}
	}

	return &Frame{W: w, H: h, Pix: nrgba, Gray: gray}
}

// Downscale2x returns a new Frame at half resolution using box averaging.
func (f *Frame) Downscale2x() *Frame {
	nw, nh := f.W/2, f.H/2
	if nw == 0 || nh == 0 {
		return f
	}

	nrgba := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	gray := make([]uint8, nw*nh)

	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			sx, sy := x*2, y*2
			// Average 2x2 block from source
			off00 := sy*f.Pix.Stride + sx*4
			off10 := off00 + 4
			off01 := (sy+1)*f.Pix.Stride + sx*4
			off11 := off01 + 4

			r := (uint32(f.Pix.Pix[off00]) + uint32(f.Pix.Pix[off10]) + uint32(f.Pix.Pix[off01]) + uint32(f.Pix.Pix[off11])) / 4
			g := (uint32(f.Pix.Pix[off00+1]) + uint32(f.Pix.Pix[off10+1]) + uint32(f.Pix.Pix[off01+1]) + uint32(f.Pix.Pix[off11+1])) / 4
			b := (uint32(f.Pix.Pix[off00+2]) + uint32(f.Pix.Pix[off10+2]) + uint32(f.Pix.Pix[off01+2]) + uint32(f.Pix.Pix[off11+2])) / 4
			a := (uint32(f.Pix.Pix[off00+3]) + uint32(f.Pix.Pix[off10+3]) + uint32(f.Pix.Pix[off01+3]) + uint32(f.Pix.Pix[off11+3])) / 4

			doff := y*nrgba.Stride + x*4
			nrgba.Pix[doff] = uint8(r)
			nrgba.Pix[doff+1] = uint8(g)
			nrgba.Pix[doff+2] = uint8(b)
			nrgba.Pix[doff+3] = uint8(a)

			gv := (uint32(f.Gray[sy*f.W+sx]) + uint32(f.Gray[sy*f.W+sx+1]) + uint32(f.Gray[(sy+1)*f.W+sx]) + uint32(f.Gray[(sy+1)*f.W+sx+1])) / 4
			gray[y*nw+x] = uint8(gv)
		}
	}

	return &Frame{W: nw, H: nh, Pix: nrgba, Gray: gray}
}

// Alignment represents the detected positional offset between two images.
type Alignment struct {
	DX, DY int
	Score  float64 // higher is better (0..1)
}

// Mask is a full-resolution binary diff mask (row-major, 0=same, 1=diff).
type Mask struct {
	W, H  int
	Data  []uint8
	Count int // number of diff pixels
}

// NewMask creates a zero-initialized mask.
func NewMask(w, h int) *Mask {
	return &Mask{W: w, H: h, Data: make([]uint8, w*h)}
}

// Set marks pixel (x,y) as different.
func (m *Mask) Set(x, y int) {
	if x >= 0 && x < m.W && y >= 0 && y < m.H {
		idx := y*m.W + x
		if m.Data[idx] == 0 {
			m.Data[idx] = 1
			m.Count++
		}
	}
}

// Get returns true if pixel (x,y) is marked as different.
func (m *Mask) Get(x, y int) bool {
	if x >= 0 && x < m.W && y >= 0 && y < m.H {
		return m.Data[y*m.W+x] == 1
	}
	return false
}

// Region represents a detected diff region with bounding box and pixel count.
type Region struct {
	Bounds image.Rectangle
	Area   int // number of diff pixels in this region
}

// Result holds the output of the diff pipeline.
type Result struct {
	Aligned  Alignment
	HasDiff  bool
	Regions  []Region
	DiffMask *Mask
	Output   image.Image
}

// Layout defines the output image layout.
type Layout string

const (
	LayoutSimple     Layout = "simple"
	LayoutHorizontal Layout = "horizontal"
)

// BlendColors blends src color over dst with configurable overlay and tint.
func BlendColors(dst, src color.Color, transparency float64, tint color.NRGBA, useTint bool, tintStrength, tintTransparency float64) color.NRGBA {
	dr, dg, db, da := dst.RGBA()
	sr, sg, sb, sa := src.RGBA()

	dr8 := uint8(dr >> 8)
	dg8 := uint8(dg >> 8)
	db8 := uint8(db >> 8)
	da8 := uint8(da >> 8)
	sr8 := uint8(sr >> 8)
	sg8 := uint8(sg >> 8)
	sb8 := uint8(sb >> 8)
	sa8 := uint8(sa >> 8)

	var r, g, b uint8

	if useTint {
		srcWeight := 1.0 - tintStrength
		tr := uint8(float64(sr8)*srcWeight + float64(tint.R)*tintStrength)
		tg := uint8(float64(sg8)*srcWeight + float64(tint.G)*tintStrength)
		tb := uint8(float64(sb8)*srcWeight + float64(tint.B)*tintStrength)

		effectiveTransparency := (transparency + tintTransparency) / 2
		r = uint8(float64(tr)*(1-effectiveTransparency) + float64(dr8)*effectiveTransparency)
		g = uint8(float64(tg)*(1-effectiveTransparency) + float64(dg8)*effectiveTransparency)
		b = uint8(float64(tb)*(1-effectiveTransparency) + float64(db8)*effectiveTransparency)
	} else {
		r = uint8(float64(sr8)*(1-transparency) + float64(dr8)*transparency)
		g = uint8(float64(sg8)*(1-transparency) + float64(dg8)*transparency)
		b = uint8(float64(sb8)*(1-transparency) + float64(db8)*transparency)
	}

	a := da8
	if sa8 > da8 {
		a = sa8
	}

	return color.NRGBA{r, g, b, a}
}
