package config

import (
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.MaxOffset != 10 {
		t.Errorf("MaxOffset should be 10, but got %d", cfg.MaxOffset)
	}

	if cfg.Threshold != 30 {
		t.Errorf("Threshold should be 30, but got %d", cfg.Threshold)
	}

	if !cfg.HighlightDiff {
		t.Errorf("HighlightDiff should be true, but got %v", cfg.HighlightDiff)
	}

	if cfg.NumCPU != 4 {
		t.Errorf("NumCPU should be 4, but got %d", cfg.NumCPU)
	}

	if cfg.SamplingRate != 1 {
		t.Errorf("SamplingRate should be 1, but got %d", cfg.SamplingRate)
	}

	if cfg.FastMode {
		t.Errorf("FastMode should be false, but got %v", cfg.FastMode)
	}

	if cfg.ProgressStep != 5 {
		t.Errorf("ProgressStep should be 5, but got %d", cfg.ProgressStep)
	}

	if !cfg.ShowTransparentOverlay {
		t.Errorf("ShowTransparentOverlay should be true, but got %v", cfg.ShowTransparentOverlay)
	}

	if cfg.OverlayTransparency != 0.3 {
		t.Errorf("OverlayTransparency should be 0.3, but got %f", cfg.OverlayTransparency)
	}

	if cfg.UseTint != true {
		t.Errorf("UseTint should be true, but got %v", cfg.UseTint)
	}

	if cfg.TintStrength != 0.7 {
		t.Errorf("TintStrength should be 0.7, but got %f", cfg.TintStrength)
	}

	if cfg.TintTransparency != 0.2 {
		t.Errorf("TintTransparency should be 0.2, but got %f", cfg.TintTransparency)
	}
}
