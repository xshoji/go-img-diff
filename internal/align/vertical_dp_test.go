package align

import (
	"image/color"
	"testing"

	"github.com/xshoji/go-img-diff/internal/core"
	"github.com/xshoji/go-img-diff/internal/diff"
)

func TestVerticalDPAlign_ReducesTailDiffForInsertedSection(t *testing.T) {
	a, b := makeWebsiteLikeFrames()
	global := Align(a, b, core.AlignOptions{MaxOffset: 10, MinPyramidSize: 16, RefinementRadius: 2}, 1, testLogger())
	baseRows := core.NewRowAlignmentFromAlignment(b.W, b.H, global)
	opts := core.DefaultOptions()

	baseMask := diff.BuildMask(a, b, baseRows, opts.Diff, testLogger())
	rowAlign := VerticalDPAlign(a, b, global, opts.VerticalAlign, testLogger())
	dpMask := diff.BuildMask(a, b, rowAlign, opts.Diff, testLogger())

	if dpMask.Count >= baseMask.Count/2 {
		t.Fatalf("expected DP alignment to substantially reduce diff pixels, before=%d after=%d", baseMask.Count, dpMask.Count)
	}

	baseLower := countMaskPixels(baseMask, 0, 170, b.W, b.H)
	dpLower := countMaskPixels(dpMask, 0, 170, b.W, b.H)
	if dpLower >= baseLower/4 {
		t.Fatalf("expected lower tail diff to mostly disappear, before=%d after=%d", baseLower, dpLower)
	}

	insertBand := countMaskPixels(dpMask, 0, 96, b.W, 152)
	if insertBand == 0 {
		t.Fatal("expected inserted section to remain as diff")
	}
}

func TestVerticalDPAlignInRange_PreservesSidebarWhileResyncingContent(t *testing.T) {
	a, b := makeWebsiteLikeFramesWithSidebar()
	global := Align(a, b, core.AlignOptions{MaxOffset: 10, MinPyramidSize: 16, RefinementRadius: 2}, 1, testLogger())
	baseRows := core.NewRowAlignmentFromAlignment(b.W, b.H, global)
	opts := core.DefaultOptions()

	baseMask := diff.BuildMask(a, b, baseRows, opts.Diff, testLogger())
	contentRows := VerticalDPAlignInRange(a, b, global, opts.VerticalAlign, 96, b.W, testLogger())
	mergedRows := baseRows.Clone()
	mergedRows.ApplyRange(96, b.W, contentRows)
	mergedMask := diff.BuildMask(a, b, mergedRows, opts.Diff, testLogger())

	if sidebarDiff := countMaskPixels(baseMask, 0, 0, 96, b.H); sidebarDiff != 0 {
		t.Fatalf("expected base sidebar to be unchanged, got %d diff pixels", sidebarDiff)
	}
	if sidebarDiff := countMaskPixels(mergedMask, 0, 0, 96, b.H); sidebarDiff != 0 {
		t.Fatalf("expected sidebar to stay clean after strip DP, got %d diff pixels", sidebarDiff)
	}

	baseMain := countMaskPixels(baseMask, 96, 170, b.W, b.H)
	mergedMain := countMaskPixels(mergedMask, 96, 170, b.W, b.H)
	if mergedMain >= baseMain/2 {
		t.Fatalf("expected content strip DP to reduce main-tail diff, before=%d after=%d", baseMain, mergedMain)
	}
}

func TestVerticalDPAlign_FallbackMatchesGlobalAlignmentWhenDisabled(t *testing.T) {
	a := makeFrame(50, 50, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(50, 50, color.NRGBA{255, 255, 255, 255})
	global := core.Alignment{DX: 3, DY: -2, Score: 0.8}
	rowAlign := VerticalDPAlign(a, b, global, core.VerticalAlignOptions{Enabled: false}, testLogger())

	for y := 0; y < b.H; y++ {
		if rowAlign.DX(y) != 3 {
			t.Fatalf("expected DX=3 at row %d, got %d", y, rowAlign.DX(y))
		}
		wantSrcY := y + 2
		if wantSrcY >= a.H {
			wantSrcY = -1
		}
		if rowAlign.SrcY(y) != wantSrcY {
			t.Fatalf("expected row %d -> %d, got %d", y, wantSrcY, rowAlign.SrcY(y))
		}
	}
}

func makeWebsiteLikeFrames() (*core.Frame, *core.Frame) {
	a := makeFrame(320, 320, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(320, 320, color.NRGBA{255, 255, 255, 255})

	drawSection(a, 0, 0, 320, 36, color.NRGBA{20, 30, 55, 255})
	drawSection(b, 0, 0, 320, 36, color.NRGBA{20, 30, 55, 255})

	for y := 56; y < 120; y += 16 {
		drawRule(a, 24, 288, y, 4)
		drawRule(b, 24, 288, y, 4)
	}

	insertHeight := 40
	drawSection(b, 20, 104, 300, 144, color.NRGBA{255, 241, 214, 255})
	drawRule(b, 32, 270, 118, 4)
	drawRule(b, 32, 284, 132, 4)

	for y := 150; y < 280; y += 18 {
		drawRule(a, 24, 280, y-insertHeight, 4)
		drawRule(b, 24, 280, y, 4)
	}

	drawSection(a, 24, 220-insertHeight, 140, 270-insertHeight, color.NRGBA{210, 232, 255, 255})
	drawSection(b, 24, 220, 140, 270, color.NRGBA{210, 232, 255, 255})

	return a, b
}

func makeWebsiteLikeFramesWithSidebar() (*core.Frame, *core.Frame) {
	a := makeFrame(320, 320, color.NRGBA{255, 255, 255, 255})
	b := makeFrame(320, 320, color.NRGBA{255, 255, 255, 255})

	drawSection(a, 0, 0, 72, 320, color.NRGBA{244, 245, 247, 255})
	drawSection(b, 0, 0, 72, 320, color.NRGBA{244, 245, 247, 255})
	drawSection(a, 12, 18, 60, 34, color.NRGBA{45, 58, 92, 255})
	drawSection(b, 12, 18, 60, 34, color.NRGBA{45, 58, 92, 255})
	for y := 60; y < 260; y += 28 {
		drawSection(a, 12, y, 60, y+10, color.NRGBA{180, 186, 196, 255})
		drawSection(b, 12, y, 60, y+10, color.NRGBA{180, 186, 196, 255})
	}

	drawSection(a, 96, 0, 320, 36, color.NRGBA{20, 30, 55, 255})
	drawSection(b, 96, 0, 320, 36, color.NRGBA{20, 30, 55, 255})

	for y := 56; y < 120; y += 16 {
		drawRule(a, 112, 288, y, 4)
		drawRule(b, 112, 288, y, 4)
	}

	insertHeight := 40
	drawSection(b, 104, 104, 300, 144, color.NRGBA{255, 241, 214, 255})
	drawRule(b, 120, 270, 118, 4)
	drawRule(b, 120, 284, 132, 4)

	for y := 150; y < 280; y += 18 {
		drawRule(a, 112, 280, y-insertHeight, 4)
		drawRule(b, 112, 280, y, 4)
	}

	drawSection(a, 112, 220-insertHeight, 220, 270-insertHeight, color.NRGBA{210, 232, 255, 255})
	drawSection(b, 112, 220, 220, 270, color.NRGBA{210, 232, 255, 255})

	return a, b
}

func drawSection(f *core.Frame, minX, minY, maxX, maxY int, c color.NRGBA) {
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			f.Pix.SetNRGBA(x, y, c)
		}
	}
	refreshFrameGray(f)
}

func drawRule(f *core.Frame, minX, maxX, y, h int) {
	for yy := y; yy < y+h; yy++ {
		for x := minX; x < maxX; x++ {
			f.Pix.SetNRGBA(x, yy, color.NRGBA{35, 35, 35, 255})
		}
	}
	refreshFrameGray(f)
}

func refreshFrameGray(f *core.Frame) {
	rebuilt := core.NewFrame(f.Pix)
	f.Gray = rebuilt.Gray
	f.Pix = rebuilt.Pix
	f.W = rebuilt.W
	f.H = rebuilt.H
}

func countMaskPixels(mask *core.Mask, minX, minY, maxX, maxY int) int {
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
