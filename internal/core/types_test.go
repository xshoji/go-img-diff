package core

import (
	"image"
	"image/color"
	"testing"
)

func TestNewFrame(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetNRGBA(x, y, color.NRGBA{255, 128, 64, 255})
		}
	}

	f := NewFrame(img)
	if f.W != 10 || f.H != 10 {
		t.Errorf("expected 10x10, got %dx%d", f.W, f.H)
	}
	if len(f.Gray) != 100 {
		t.Errorf("expected 100 gray values, got %d", len(f.Gray))
	}
	// Gray should be a reasonable luminance of (255, 128, 64)
	if f.Gray[0] == 0 {
		t.Error("expected non-zero grayscale")
	}
}

func TestNewFrame_NonZeroOrigin(t *testing.T) {
	img := image.NewNRGBA(image.Rect(10, 20, 30, 40))
	for y := 20; y < 40; y++ {
		for x := 10; x < 30; x++ {
			img.SetNRGBA(x, y, color.NRGBA{100, 100, 100, 255})
		}
	}

	f := NewFrame(img)
	if f.W != 20 || f.H != 20 {
		t.Errorf("expected 20x20, got %dx%d", f.W, f.H)
	}
}

func TestDownscale2x(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.SetNRGBA(x, y, color.NRGBA{200, 100, 50, 255})
		}
	}

	f := NewFrame(img)
	d := f.Downscale2x()

	if d.W != 10 || d.H != 10 {
		t.Errorf("expected 10x10, got %dx%d", d.W, d.H)
	}
}

func TestDownscale2x_TooSmall(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 255})

	f := NewFrame(img)
	d := f.Downscale2x()

	// Should return the same frame since it can't downscale
	if d.W != 1 || d.H != 1 {
		t.Errorf("expected 1x1 (no downscale), got %dx%d", d.W, d.H)
	}
}

func TestMask(t *testing.T) {
	m := NewMask(10, 10)
	if m.Count != 0 {
		t.Errorf("expected 0 count, got %d", m.Count)
	}

	m.Set(5, 5)
	if !m.Get(5, 5) {
		t.Error("expected (5,5) to be set")
	}
	if m.Count != 1 {
		t.Errorf("expected 1 count, got %d", m.Count)
	}

	// Double set should not increment count
	m.Set(5, 5)
	if m.Count != 1 {
		t.Errorf("expected 1 count after double set, got %d", m.Count)
	}

	// Out of bounds
	if m.Get(-1, 0) {
		t.Error("out of bounds should return false")
	}
	m.Set(-1, 0) // should not panic
}

func TestBlendColors(t *testing.T) {
	tests := []struct {
		name             string
		dst              color.Color
		src              color.Color
		transparency     float64
		tint             color.NRGBA
		useTint          bool
		tintStrength     float64
		tintTransparency float64
		want             color.NRGBA
	}{
		{
			name:         "fully opaque no tint",
			dst:          color.NRGBA{0, 0, 0, 255},
			src:          color.NRGBA{255, 0, 0, 255},
			transparency: 0.0,
			tint:         color.NRGBA{},
			useTint:      false,
			want:         color.NRGBA{255, 0, 0, 255},
		},
		{
			name:         "fully transparent no tint",
			dst:          color.NRGBA{0, 0, 0, 255},
			src:          color.NRGBA{255, 0, 0, 255},
			transparency: 1.0,
			tint:         color.NRGBA{},
			useTint:      false,
			want:         color.NRGBA{0, 0, 0, 255},
		},
		{
			name:         "half transparent no tint",
			dst:          color.NRGBA{0, 0, 0, 255},
			src:          color.NRGBA{255, 0, 0, 255},
			transparency: 0.5,
			tint:         color.NRGBA{},
			useTint:      false,
			want:         color.NRGBA{127, 0, 0, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlendColors(tt.dst, tt.src, tt.transparency, tt.tint, tt.useTint, tt.tintStrength, tt.tintTransparency)
			if got != tt.want {
				t.Errorf("BlendColors() = %v, want %v", got, tt.want)
			}
		})
	}
}
