package core

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Align.MaxOffset != 10 {
		t.Errorf("expected MaxOffset=10, got %d", opts.Align.MaxOffset)
	}
	if opts.Diff.Threshold != 30 {
		t.Errorf("expected Threshold=30, got %d", opts.Diff.Threshold)
	}
	if opts.Region.MinArea != 4 {
		t.Errorf("expected MinArea=4, got %d", opts.Region.MinArea)
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
