package app

import (
	"fmt"
	"image"
	"log/slog"
	"runtime"
	"time"

	"github.com/xshoji/go-img-diff/internal/align"
	"github.com/xshoji/go-img-diff/internal/core"
	"github.com/xshoji/go-img-diff/internal/diff"
	"github.com/xshoji/go-img-diff/internal/imgio"
	"github.com/xshoji/go-img-diff/internal/region"
	"github.com/xshoji/go-img-diff/internal/render"
)

// Run executes the full image diff pipeline.
// Returns (hasDiff, error). If exitOnDiff is true and there are diffs, it returns early.
func Run(opts core.Options, exitOnDiff bool, logger *slog.Logger) (bool, error) {
	startTime := time.Now()

	runtime.GOMAXPROCS(opts.Runtime.Workers)
	logger.Info("starting pipeline", "workers", opts.Runtime.Workers)

	// 1. Load images
	frameA, err := imgio.LoadFrame(opts.Input1, logger)
	if err != nil {
		return false, fmt.Errorf("failed to load input1: %w", err)
	}

	frameB, err := imgio.LoadFrame(opts.Input2, logger)
	if err != nil {
		return false, fmt.Errorf("failed to load input2: %w", err)
	}

	if frameA.W != frameB.W || frameA.H != frameB.H {
		logger.Warn("image dimensions differ",
			"input1", [2]int{frameA.W, frameA.H},
			"input2", [2]int{frameB.W, frameB.H},
		)
	}

	// 2. Align
	alignment := align.Align(frameA, frameB, opts.Align, opts.Runtime.Workers, logger)

	// 3. Build diff mask (with StopAfterFirst if exitOnDiff)
	diffOpts := opts.Diff
	if exitOnDiff {
		diffOpts.StopAfterFirst = true
	}
	mask := diff.BuildMask(frameA, frameB, alignment, diffOpts, logger)

	hasDiff := mask.Count > 0

	if exitOnDiff {
		if hasDiff {
			logger.Info("differences detected (exit-on-diff mode)")
		} else {
			logger.Info("no differences detected")
		}
		return hasDiff, nil
	}

	// 4. Extract regions
	regions := region.Extract(mask, opts.Region, logger)

	// 5. Render
	diffImage := render.Render(frameA, frameB, mask, regions, alignment, opts.Render, logger)

	// 6. Apply layout
	var outputImage image.Image = diffImage
	if opts.Render.Layout == core.LayoutHorizontal {
		logger.Info("applying horizontal layout")
		outputImage = render.CombineHorizontal(frameA.Pix, diffImage)
	}

	// 7. Save
	if opts.Output.Path != "" {
		if err := imgio.SaveImage(outputImage, opts.Output.Path, logger); err != nil {
			return hasDiff, fmt.Errorf("failed to save output: %w", err)
		}
	}

	elapsed := time.Since(startTime)
	logger.Info("pipeline complete", "elapsed", elapsed.Round(time.Millisecond), "hasDiff", hasDiff, "regions", len(regions))

	return hasDiff, nil
}
