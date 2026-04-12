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

func TestNewRowAlignment(t *testing.T) {
	ra := NewRowAlignment(10, 5, 3, 1)

	if ra.Width != 10 || ra.Height != 5 {
		t.Fatalf("expected 10x5 row alignment, got %dx%d", ra.Width, ra.Height)
	}
	if got := ra.SrcY(0); got != -1 {
		t.Fatalf("expected row 0 to be unmapped, got %d", got)
	}
	if got := ra.SrcY(3); got != 2 {
		t.Fatalf("expected row 3 -> 2, got %d", got)
	}
	if got := ra.DX(4); got != 3 {
		t.Fatalf("expected DX 3, got %d", got)
	}
	if ra.HasMapping(0) {
		t.Fatal("expected row 0 to be unmapped")
	}
	if !ra.HasMapping(3) {
		t.Fatal("expected row 3 to be mapped")
	}
}

func TestNewRowAlignmentFromAlignment(t *testing.T) {
	ra := NewRowAlignmentFromAlignment(8, 4, Alignment{DX: 2, DY: -1, Score: 0.75})

	if ra.Score != 0.75 {
		t.Fatalf("expected score 0.75, got %f", ra.Score)
	}
	if got := ra.SrcY(0); got != 1 {
		t.Fatalf("expected row 0 -> 1, got %d", got)
	}
	if got := ra.DX(2); got != 2 {
		t.Fatalf("expected DX 2, got %d", got)
	}
}

func TestRowAlignmentApplyRange(t *testing.T) {
	base := NewRowAlignmentFromAlignment(8, 4, Alignment{DX: 0, DY: 0, Score: 0.2})
	override := NewRowAlignmentFromAlignment(8, 4, Alignment{DX: 3, DY: -1, Score: 0.9})
	base.ApplyRange(4, 8, override)

	if got := base.SrcYAt(2, 1); got != 1 {
		t.Fatalf("expected base mapping on left side, got %d", got)
	}
	if got := base.DXAt(2, 1); got != 0 {
		t.Fatalf("expected base DX on left side, got %d", got)
	}
	if got := base.SrcYAt(6, 1); got != 2 {
		t.Fatalf("expected override mapping on right side, got %d", got)
	}
	if got := base.DXAt(6, 1); got != 3 {
		t.Fatalf("expected override DX on right side, got %d", got)
	}
	if base.Score != 0.9 {
		t.Fatalf("expected score to follow stronger override, got %f", base.Score)
	}

	clone := base.Clone()
	clone.SrcYByY[1] = -1
	clone.Ranges[0].SrcYByY[1] = -1
	if got := base.SrcYAt(2, 1); got != 1 {
		t.Fatalf("expected clone mutation not to affect base row mapping, got %d", got)
	}
	if got := base.SrcYAt(6, 1); got != 2 {
		t.Fatalf("expected clone mutation not to affect base range mapping, got %d", got)
	}
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
