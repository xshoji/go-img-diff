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
	rowAlign := core.NewRowAlignmentFromAlignment(50, 50, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels, got %d", mask.Count)
	}
}

func TestBuildMask_AllDifferent(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(10, 10, color.NRGBA{0, 0, 0, 255})
	rowAlign := core.NewRowAlignmentFromAlignment(10, 10, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 100 {
		t.Errorf("expected 100 diff pixels, got %d", mask.Count)
	}
}

func TestBuildMask_StopAfterFirst(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(10, 10, color.NRGBA{0, 0, 0, 255})
	rowAlign := core.NewRowAlignmentFromAlignment(10, 10, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30, StopAfterFirst: true}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 1 {
		t.Errorf("expected 1 diff pixel (stop after first), got %d", mask.Count)
	}
}

func TestBuildMask_WithOffset(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	b := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	// With offset, some pixels fall outside A → skipped (not comparable)
	rowAlign := core.NewRowAlignmentFromAlignment(10, 10, core.Alignment{DX: 5, DY: 0})
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	// Out-of-bounds pixels are skipped, matching pixels have no diff
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels (out-of-bounds skipped, rest identical), got %d", mask.Count)
	}
}

func TestBuildMask_BelowThreshold(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{100, 100, 100, 255})
	b := makeFrame(10, 10, color.NRGBA{110, 105, 108, 255}) // diff < 30
	rowAlign := core.NewRowAlignmentFromAlignment(10, 10, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 0 {
		t.Errorf("expected 0 diff pixels (below threshold), got %d", mask.Count)
	}
}

func TestBuildMask_UnmappedRowMarksDiff(t *testing.T) {
	a := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(10, 10, color.NRGBA{255, 255, 255, 255})
	rowAlign := core.NewRowAlignmentFromAlignment(10, 10, core.Alignment{DX: 0, DY: 0})
	rowAlign.SrcYByY[4] = -1
	rowAlign.SrcYByY[5] = -1
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 20 {
		t.Fatalf("expected 20 diff pixels for 2 unmapped rows, got %d", mask.Count)
	}
}

func TestBuildMask_RowSpecificMapping(t *testing.T) {
	a := makeFrame(8, 8, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(8, 8, color.NRGBA{255, 255, 255, 255})
	for x := 0; x < 8; x++ {
		a.Pix.SetNRGBA(x, 4, color.NRGBA{0, 0, 0, 255})
		b.Pix.SetNRGBA(x, 5, color.NRGBA{0, 0, 0, 255})
	}
	rowAlign := core.NewRowAlignmentFromAlignment(8, 8, core.Alignment{DX: 0, DY: 0})
	rowAlign.SrcYByY[5] = 4
	rowAlign.SrcYByY[4] = -1
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 8 {
		t.Fatalf("expected only one inserted row to remain diff, got %d pixels", mask.Count)
	}
}

func TestBuildMask_ColumnSpecificMapping(t *testing.T) {
	a := makeFrame(8, 8, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(8, 8, color.NRGBA{255, 255, 255, 255})
	for x := 4; x < 8; x++ {
		a.Pix.SetNRGBA(x, 4, color.NRGBA{0, 0, 0, 255})
		b.Pix.SetNRGBA(x, 5, color.NRGBA{0, 0, 0, 255})
	}
	a = core.NewFrame(a.Pix)
	b = core.NewFrame(b.Pix)

	rowAlign := core.NewRowAlignmentFromAlignment(8, 8, core.Alignment{DX: 0, DY: 0})
	override := core.NewRowAlignmentFromAlignment(8, 8, core.Alignment{DX: 0, DY: 0})
	override.SrcYByY[5] = 4
	override.SrcYByY[4] = -1
	rowAlign.ApplyRange(4, 8, override)
	opts := core.DiffOptions{Threshold: 30}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if count := countPixels(mask, 0, 0, 4, 8); count != 0 {
		t.Fatalf("expected unchanged left strip to stay clean, got %d diff pixels", count)
	}
	if count := countPixels(mask, 4, 0, 8, 8); count != 4 {
		t.Fatalf("expected only shifted right strip to remain diff, got %d diff pixels", count)
	}
}

func TestBuildMask_NoiseFilterRemovesSparsePixels(t *testing.T) {
	a := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	b.Pix.SetNRGBA(2, 2, color.NRGBA{0, 0, 0, 255})
	b.Pix.SetNRGBA(10, 10, color.NRGBA{0, 0, 0, 255})
	b.Pix.SetNRGBA(17, 17, color.NRGBA{0, 0, 0, 255})
	b = core.NewFrame(b.Pix)
	rowAlign := core.NewRowAlignmentFromAlignment(20, 20, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30, NoiseWindowSize: 5, NoiseMinDiffRatio: 0.20}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 0 {
		t.Fatalf("expected sparse noise to be removed, got %d diff pixels", mask.Count)
	}
}

func TestBuildMask_NoiseFilterKeepsDenseBlock(t *testing.T) {
	a := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	for y := 8; y < 11; y++ {
		for x := 8; x < 11; x++ {
			b.Pix.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 255})
		}
	}
	b = core.NewFrame(b.Pix)
	rowAlign := core.NewRowAlignmentFromAlignment(20, 20, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30, NoiseWindowSize: 5, NoiseMinDiffRatio: 0.20}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 9 {
		t.Fatalf("expected dense block to remain, got %d diff pixels", mask.Count)
	}
}

func TestBuildMask_StopAfterFirstWithNoiseFilterBuildsFullMask(t *testing.T) {
	a := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(20, 20, color.NRGBA{255, 255, 255, 255})
	b.Pix.SetNRGBA(5, 5, color.NRGBA{0, 0, 0, 255})
	b = core.NewFrame(b.Pix)
	rowAlign := core.NewRowAlignmentFromAlignment(20, 20, core.Alignment{DX: 0, DY: 0})
	opts := core.DiffOptions{Threshold: 30, StopAfterFirst: true, NoiseWindowSize: 5, NoiseMinDiffRatio: 0.20}

	mask := BuildMask(a, b, rowAlign, opts, testLogger())
	if mask.Count != 0 {
		t.Fatalf("expected sparse diff to be filtered out even with StopAfterFirst, got %d", mask.Count)
	}
}

func countPixels(mask *core.Mask, minX, minY, maxX, maxY int) int {
	count := 0
	for y := max(0, minY); y < min(mask.H, maxY); y++ {
		for x := max(0, minX); x < min(mask.W, maxX); x++ {
			if mask.Get(x, y) {
				count++
			}
		}
	}
	return count
}
