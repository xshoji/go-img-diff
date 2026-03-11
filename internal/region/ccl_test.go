package region

import (
	"image"
	"log/slog"
	"os"
	"testing"

	"github.com/xshoji/go-img-diff/internal/core"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestExtract_NoRegions(t *testing.T) {
	mask := core.NewMask(50, 50) // all zeros
	opts := core.RegionOptions{MinArea: 1, Padding: 0}

	regions := Extract(mask, opts, testLogger())
	if len(regions) != 0 {
		t.Errorf("expected 0 regions, got %d", len(regions))
	}
}

func TestExtract_SingleRegion(t *testing.T) {
	mask := core.NewMask(50, 50)
	// Set a 10x10 block
	for y := 20; y < 30; y++ {
		for x := 20; x < 30; x++ {
			mask.Set(x, y)
		}
	}

	opts := core.RegionOptions{MinArea: 1, Padding: 0, DilateRadius: 0}
	regions := Extract(mask, opts, testLogger())

	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}

	r := regions[0]
	if r.Area != 100 {
		t.Errorf("expected area 100, got %d", r.Area)
	}
	// Bounds should cover [20,30) x [20,30)
	if r.Bounds.Min.X != 20 || r.Bounds.Min.Y != 20 || r.Bounds.Max.X != 30 || r.Bounds.Max.Y != 30 {
		t.Errorf("unexpected bounds: %v", r.Bounds)
	}
}

func TestExtract_TwoSeparateRegions(t *testing.T) {
	mask := core.NewMask(50, 50)
	// Region 1: top-left
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			mask.Set(x, y)
		}
	}
	// Region 2: bottom-right
	for y := 40; y < 45; y++ {
		for x := 40; x < 45; x++ {
			mask.Set(x, y)
		}
	}

	opts := core.RegionOptions{MinArea: 1, Padding: 0, DilateRadius: 0}
	regions := Extract(mask, opts, testLogger())

	if len(regions) != 2 {
		t.Errorf("expected 2 regions, got %d", len(regions))
	}
}

func TestExtract_MinAreaFilter(t *testing.T) {
	mask := core.NewMask(50, 50)
	// Set just 2 pixels
	mask.Set(10, 10)
	mask.Set(11, 10)

	opts := core.RegionOptions{MinArea: 5, Padding: 0, DilateRadius: 0}
	regions := Extract(mask, opts, testLogger())

	if len(regions) != 0 {
		t.Errorf("expected 0 regions (below MinArea), got %d", len(regions))
	}
}

func TestExtract_WithPadding(t *testing.T) {
	mask := core.NewMask(50, 50)
	for y := 20; y < 25; y++ {
		for x := 20; x < 25; x++ {
			mask.Set(x, y)
		}
	}

	opts := core.RegionOptions{MinArea: 1, Padding: 3, DilateRadius: 0}
	regions := Extract(mask, opts, testLogger())

	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}

	r := regions[0]
	if r.Bounds.Min.X != 17 || r.Bounds.Min.Y != 17 {
		t.Errorf("expected padded min (17,17), got (%d,%d)", r.Bounds.Min.X, r.Bounds.Min.Y)
	}
}

func TestExtract_Dilation(t *testing.T) {
	mask := core.NewMask(50, 50)
	// Two nearby pixels with a 1px gap
	mask.Set(20, 20)
	mask.Set(22, 20) // gap at (21,20)

	// Without dilation: might be 2 separate regions
	opts := core.RegionOptions{MinArea: 1, Padding: 0, DilateRadius: 0}
	regionsNoDilate := Extract(mask, opts, testLogger())

	// With dilation radius 1: should bridge the gap
	opts.DilateRadius = 1
	regionsWithDilate := Extract(mask, opts, testLogger())

	if len(regionsWithDilate) > len(regionsNoDilate) {
		t.Errorf("dilation should reduce or maintain region count, got %d vs %d",
			len(regionsWithDilate), len(regionsNoDilate))
	}
}

func TestMergeOverlapping(t *testing.T) {
	regions := []core.Region{
		{Bounds: image.Rect(0, 0, 20, 20), Area: 100},
		{Bounds: image.Rect(15, 15, 35, 35), Area: 100},
	}

	merged := mergeOverlapping(regions)
	if len(merged) != 1 {
		t.Errorf("expected 1 merged region, got %d", len(merged))
	}
}

func TestMergeOverlapping_NoOverlap(t *testing.T) {
	regions := []core.Region{
		{Bounds: image.Rect(0, 0, 10, 10), Area: 50},
		{Bounds: image.Rect(30, 30, 40, 40), Area: 50},
	}

	merged := mergeOverlapping(regions)
	if len(merged) != 2 {
		t.Errorf("expected 2 regions, got %d", len(merged))
	}
}
