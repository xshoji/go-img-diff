package core

import (
	"image/color"
	"runtime"
)

// AlignOptions configures the pyramid alignment algorithm.
type AlignOptions struct {
	MaxOffset        int // maximum pixel offset to search
	MinPyramidSize   int // minimum image dimension for pyramid (default: 32)
	RefinementRadius int // search radius at each finer level (default: 2)
}

// DiffOptions configures pixel diff detection.
type DiffOptions struct {
	Threshold      uint8 // 0-255 max channel difference
	StopAfterFirst bool  // for --exit-on-diff: stop after first diff pixel
}

// RegionOptions configures connected-component region extraction.
type RegionOptions struct {
	MinArea      int // minimum diff pixel count to keep a region
	Padding      int // pixels of padding to add around bounding boxes
	DilateRadius int // morphological dilation radius before CCL (0=none)
}

// RenderOptions configures diff visualization.
type RenderOptions struct {
	DrawOverlay      bool
	OverlayAlpha     float64 // 0.0=opaque overlay, 1.0=fully transparent overlay
	TintEnabled      bool
	TintColor        color.NRGBA
	TintStrength     float64
	TintTransparency float64
	BorderColor      color.NRGBA
	BorderWidth      int
	Layout           Layout
}

// RuntimeOptions configures execution parameters.
type RuntimeOptions struct {
	Workers int
}

// OutputOptions configures output.
type OutputOptions struct {
	Path string
}

// Options is the top-level configuration aggregating all stage options.
type Options struct {
	Input1  string
	Input2  string
	Align   AlignOptions
	Diff    DiffOptions
	Region  RegionOptions
	Render  RenderOptions
	Runtime RuntimeOptions
	Output  OutputOptions
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		Align: AlignOptions{
			MaxOffset:        10,
			MinPyramidSize:   32,
			RefinementRadius: 2,
		},
		Diff: DiffOptions{
			Threshold: 30,
		},
		Region: RegionOptions{
			MinArea:      4,
			Padding:      5,
			DilateRadius: 1,
		},
		Render: RenderOptions{
			DrawOverlay:      true,
			OverlayAlpha:     0.95,
			TintEnabled:      true,
			TintColor:        color.NRGBA{255, 0, 0, 255},
			TintStrength:     0.05,
			TintTransparency: 0.2,
			BorderColor:      color.NRGBA{255, 0, 0, 255},
			BorderWidth:      3,
			Layout:           LayoutSimple,
		},
		Runtime: RuntimeOptions{
			Workers: runtime.NumCPU(),
		},
	}
}
