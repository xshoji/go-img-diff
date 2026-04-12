package core

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Align.MaxOffset != 10 {
		t.Errorf("expected MaxOffset=10, got %d", opts.Align.MaxOffset)
	}
	if !opts.VerticalAlign.Enabled {
		t.Error("expected VerticalAlign.Enabled=true")
	}
	if opts.VerticalAlign.BandHeight != 8 {
		t.Errorf("expected BandHeight=8, got %d", opts.VerticalAlign.BandHeight)
	}
	if opts.VerticalAlign.StripWidth != 320 {
		t.Errorf("expected StripWidth=320, got %d", opts.VerticalAlign.StripWidth)
	}
	if opts.VerticalAlign.FeatureBins != 32 {
		t.Errorf("expected FeatureBins=32, got %d", opts.VerticalAlign.FeatureBins)
	}
	if opts.Diff.Threshold != 30 {
		t.Errorf("expected Threshold=30, got %d", opts.Diff.Threshold)
	}
	if opts.Diff.NoiseWindowSize != 0 {
		t.Errorf("expected NoiseWindowSize=0, got %d", opts.Diff.NoiseWindowSize)
	}
	if opts.Diff.NoiseMinDiffRatio != 0 {
		t.Errorf("expected NoiseMinDiffRatio=0, got %f", opts.Diff.NoiseMinDiffRatio)
	}
	if opts.Region.MinArea != 4 {
		t.Errorf("expected MinArea=4, got %d", opts.Region.MinArea)
	}
	if opts.Region.Padding != 5 {
		t.Errorf("expected Padding=5, got %d", opts.Region.Padding)
	}
	if opts.Region.DilateRadius != 1 {
		t.Errorf("expected DilateRadius=1, got %d", opts.Region.DilateRadius)
	}
	if !opts.Render.DrawOverlay {
		t.Error("expected DrawOverlay=true")
	}
	if opts.Render.Layout != LayoutSimple {
		t.Errorf("expected LayoutSimple, got %s", opts.Render.Layout)
	}
	if opts.Runtime.Workers <= 0 {
		t.Errorf("expected positive Workers, got %d", opts.Runtime.Workers)
	}
}
