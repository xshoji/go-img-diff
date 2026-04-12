package app

import (
	"fmt"
	"image"
	"io"
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
	baseRowAlignment := core.NewRowAlignmentFromAlignment(frameB.W, frameB.H, alignment)
	rowAlignment := baseRowAlignment

	// 3. Build diff mask and refine dirty vertical strips with local DP.
	mask := diff.BuildMask(frameA, frameB, baseRowAlignment, opts.Diff, logger)
	baseDiffPixels := mask.Count
	if opts.VerticalAlign.Enabled && baseDiffPixels > 0 {
		stripWidth := verticalAlignStripWidth(opts.VerticalAlign, frameB.W)
		rowAlignment, correctedStrips := mergeRowAlignmentByStrip(frameA, frameB, alignment, baseRowAlignment, mask, opts, stripWidth)
		if correctedStrips > 0 {
			mask = diff.BuildMask(frameA, frameB, rowAlignment, opts.Diff, logger)
		}
		logger.Info("vertical dp alignment applied per strip",
			"baseDiffPixels", baseDiffPixels,
			"finalDiffPixels", mask.Count,
			"stripWidth", stripWidth,
			"correctedStrips", correctedStrips,
		)
	}

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
	diffImage := render.Render(frameA, frameB, mask, regions, rowAlignment, opts.Render, logger)

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

func verticalAlignStripWidth(opts core.VerticalAlignOptions, frameWidth int) int {
	if opts.StripWidth > 0 {
		return min(frameWidth, opts.StripWidth)
	}
	return min(frameWidth, 320)
}

func mergeRowAlignmentByStrip(a, b *core.Frame, global core.Alignment, base core.RowAlignment, baseMask *core.Mask, opts core.Options, stripWidth int) (core.RowAlignment, int) {
	if stripWidth <= 0 {
		stripWidth = b.W
	}
	merged := base.Clone()
	quietLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	correctedStrips := 0

	for minX := 0; minX < b.W; minX += stripWidth {
		maxX := min(b.W, minX+stripWidth)
		baseStripDiffPixels := countMaskPixelsInColumns(baseMask, minX, maxX)
		if baseStripDiffPixels == 0 {
			continue
		}

		candidate := align.VerticalDPAlignInRange(a, b, global, opts.VerticalAlign, minX, maxX, quietLogger)
		candidateMask := diff.BuildMask(a, b, candidate, opts.Diff, quietLogger)
		candidateStripDiffPixels := countMaskPixelsInColumns(candidateMask, minX, maxX)
		if candidateStripDiffPixels >= baseStripDiffPixels {
			continue
		}

		merged.ApplyRange(minX, maxX, candidate)
		correctedStrips++
	}

	return merged, correctedStrips
}

func countMaskPixelsInColumns(mask *core.Mask, minX, maxX int) int {
	if mask == nil {
		return 0
	}
	minX = max(0, min(mask.W, minX))
	maxX = max(0, min(mask.W, maxX))
	if minX >= maxX {
		return 0
	}

	count := 0
	for y := 0; y < mask.H; y++ {
		rowOffset := y * mask.W
		for x := minX; x < maxX; x++ {
			if mask.Data[rowOffset+x] != 0 {
				count++
			}
		}
	}
	return count
}
