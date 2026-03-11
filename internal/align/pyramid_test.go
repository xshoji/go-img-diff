package align

import (
	"image"
	"image/color"
	"log/slog"
	"os"
	"testing"

	"github.com/xshoji/go-img-diff/internal/core"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func makeFrame(w, h int, fill color.NRGBA) *core.Frame {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	return core.NewFrame(img)
}

func makeFrameWithCircle(w, h, cx, cy, radius int) *core.Frame {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 255})
		}
	}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				px, py := cx+dx, cy+dy
				if px >= 0 && px < w && py >= 0 && py < h {
					img.SetNRGBA(px, py, color.NRGBA{255, 255, 255, 255})
				}
			}
		}
	}
	return core.NewFrame(img)
}

func TestAlign_ZeroOffset(t *testing.T) {
	f := makeFrameWithCircle(100, 100, 50, 50, 15)
	opts := core.AlignOptions{MaxOffset: 10, MinPyramidSize: 16, RefinementRadius: 2}
	al := Align(f, f, opts, 1, testLogger())

	if al.DX != 0 || al.DY != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", al.DX, al.DY)
	}
}

func TestAlign_SmallOffset(t *testing.T) {
	a := makeFrameWithCircle(100, 100, 50, 50, 15)
	// Create B with circle shifted by (5, 3) — circle at (45,47) in B
	b := makeFrameWithCircle(100, 100, 50-5, 50-3, 15)
	opts := core.AlignOptions{MaxOffset: 10, MinPyramidSize: 16, RefinementRadius: 2}
	al := Align(a, b, opts, 1, testLogger())

	// The alignment finds the offset to map B→A, so DX=-5, DY=-3
	tolerance := 1
	if abs(al.DX-(-5)) > tolerance || abs(al.DY-(-3)) > tolerance {
		t.Errorf("expected ~(-5,-3), got (%d,%d)", al.DX, al.DY)
	}
}

func TestAlign_NegativeOffset(t *testing.T) {
	a := makeFrameWithCircle(100, 100, 50, 50, 15)
	b := makeFrameWithCircle(100, 100, 50+4, 50+2, 15)
	opts := core.AlignOptions{MaxOffset: 10, MinPyramidSize: 16, RefinementRadius: 2}
	al := Align(a, b, opts, 1, testLogger())

	// The alignment finds offset to map B→A, so DX=4, DY=2
	tolerance := 1
	if abs(al.DX-4) > tolerance || abs(al.DY-2) > tolerance {
		t.Errorf("expected ~(4,2), got (%d,%d)", al.DX, al.DY)
	}
}

func TestAlign_IdenticalImages(t *testing.T) {
	// Use a textured image so alignment has features to lock onto
	a := makeFrameWithCircle(100, 100, 50, 50, 20)
	opts := core.AlignOptions{MaxOffset: 5, MinPyramidSize: 8, RefinementRadius: 2}
	al := Align(a, a, opts, 2, testLogger())

	if al.DX != 0 || al.DY != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", al.DX, al.DY)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
