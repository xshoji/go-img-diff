package diff

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

func TestBuildMask_Identical(t *testing.T) {
	a := makeFrame(50, 50, color.NRGBA{255, 0, 0, 255})
	b := makeFrame(50, 50, color.NRGBA{255, 0, 0, 255})
	al := core.Alignment{DX: 0, DY: 0}
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, al, opts, testLogger())
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels, got %d", mask.Count)
	}
}

func TestBuildMask_AllDifferent(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(10, 10, color.NRGBA{0, 0, 0, 255})
	al := core.Alignment{DX: 0, DY: 0}
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, al, opts, testLogger())
	if mask.Count != 100 {
		t.Errorf("expected 100 diff pixels, got %d", mask.Count)
	}
}

func TestBuildMask_StopAfterFirst(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(10, 10, color.NRGBA{0, 0, 0, 255})
	al := core.Alignment{DX: 0, DY: 0}
	opts := core.DiffOptions{Threshold: 30, StopAfterFirst: true}

	mask := BuildMask(a, b, al, opts, testLogger())
	if mask.Count != 1 {
		t.Errorf("expected 1 diff pixel (stop after first), got %d", mask.Count)
	}
}

func TestBuildMask_WithOffset(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	b := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	// With offset, some pixels fall outside A → skipped (not comparable)
	al := core.Alignment{DX: 5, DY: 0}
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, al, opts, testLogger())
	// Out-of-bounds pixels are skipped, matching pixels have no diff
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels (out-of-bounds skipped, rest identical), got %d", mask.Count)
	}
}

func TestBuildMask_BelowThreshold(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	b := makeFrame(10, 10, color.NRGBA{110, 105, 108, 255}) // diff < 30
	al := core.Alignment{DX: 0, DY: 0}
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, al, opts, testLogger())
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels (below threshold), got %d", mask.Count)
	}
}
