package align

import (
	"log/slog"
	"math"
	"runtime"
	"sync"

	"github.com/xshoji/go-img-diff/internal/core"
)

// Align finds the best translation offset between two frames using pyramid coarse-to-fine search.
func Align(a, b *core.Frame, opts core.AlignOptions, workers int, logger *slog.Logger) core.Alignment {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Build pyramids
	pyramidA := buildPyramid(a, opts.MinPyramidSize)
	pyramidB := buildPyramid(b, opts.MinPyramidSize)

	logger.Info("pyramid built", "levels", len(pyramidA))

	bestDX, bestDY := 0, 0
	bestScore := 0.0

	for level := len(pyramidA) - 1; level >= 0; level-- {
		fA := pyramidA[level]
		fB := pyramidB[level]

		// Scale offset to current level
		if level < len(pyramidA)-1 {
			bestDX *= 2
			bestDY *= 2
		}

		// Determine search range
		var searchRadius int
		if level == len(pyramidA)-1 {
			// Coarsest level: full range scaled down
			scale := 1 << uint(level)
			searchRadius = opts.MaxOffset / scale
			if searchRadius < 1 {
				searchRadius = 1
			}
		} else {
			searchRadius = opts.RefinementRadius
		}

		// Generate candidates
		type candidate struct{ dx, dy int }
		var candidates []candidate
		for dy := bestDY - searchRadius; dy <= bestDY+searchRadius; dy++ {
			for dx := bestDX - searchRadius; dx <= bestDX+searchRadius; dx++ {
				candidates = append(candidates, candidate{dx, dy})
			}
		}

		// Evaluate candidates in parallel
		type result struct {
			dx, dy int
			mae    float64
		}

		resultCh := make(chan result, len(candidates))
		candidateCh := make(chan candidate, len(candidates))

		numWorkers := workers
		if numWorkers > len(candidates) {
			numWorkers = len(candidates)
		}

		// We need to know the current best MAE for early abandon.
		// Initialize with max value; it will be updated as results come in.
		var bestMAE float64 = math.MaxFloat64

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for c := range candidateCh {
					mae := calcMAE(fA, fB, c.dx, c.dy, bestMAE)
					resultCh <- result{c.dx, c.dy, mae}
				}
			}()
		}

		for _, c := range candidates {
			candidateCh <- c
		}
		close(candidateCh)

		go func() {
			wg.Wait()
			close(resultCh)
		}()

		for r := range resultCh {
			if r.mae < bestMAE {
				bestMAE = r.mae
				bestDX = r.dx
				bestDY = r.dy
			}
		}

		// Convert MAE to a 0..1 score (1.0 = perfect match, 0.0 = max error)
		if bestMAE < math.MaxFloat64 {
			bestScore = 1.0 - bestMAE/255.0
		}

		logger.Debug("alignment level complete",
			"level", level,
			"size", [2]int{fA.W, fA.H},
			"searchRadius", searchRadius,
			"candidates", len(candidates),
			"bestDX", bestDX,
			"bestDY", bestDY,
			"bestMAE", bestMAE,
		)
	}

	logger.Info("alignment complete", "dx", bestDX, "dy", bestDY, "score", bestScore)
	return core.Alignment{DX: bestDX, DY: bestDY, Score: bestScore}
}

// buildPyramid creates a multi-scale pyramid. Level 0 is full resolution.
func buildPyramid(f *core.Frame, minSize int) []*core.Frame {
	if minSize <= 0 {
		minSize = 32
	}
	pyramid := []*core.Frame{f}
	current := f
	for current.W > minSize && current.H > minSize {
		down := current.Downscale2x()
		if down.W == current.W && down.H == current.H {
			break // can't downscale further
		}
		pyramid = append(pyramid, down)
		current = down
	}
	return pyramid
}

// calcMAE computes mean absolute grayscale error over the overlap region.
// It uses early abandon: if cumulative error already exceeds bestMAE * overlapPixels, it returns math.MaxFloat64.
func calcMAE(a, b *core.Frame, dx, dy int, bestMAE float64) float64 {
	// Overlap region in b's coordinate space
	overlapMinX := max(0, -dx)
	overlapMinY := max(0, -dy)
	overlapMaxX := min(a.W, b.W-dx)
	overlapMaxY := min(a.H, b.H-dy)

	overlapW := overlapMaxX - overlapMinX
	overlapH := overlapMaxY - overlapMinY

	if overlapW <= 0 || overlapH <= 0 {
		return math.MaxFloat64
	}

	totalPixels := overlapW * overlapH

	// Penalize small overlaps
	totalArea := max(a.W*a.H, b.W*b.H)
	coverageRatio := float64(totalPixels) / float64(totalArea)
	if coverageRatio < 0.3 {
		return math.MaxFloat64
	}

	var cumError uint64
	earlyAbandonThreshold := uint64(bestMAE * float64(totalPixels))

	for y := overlapMinY; y < overlapMaxY; y++ {
		for x := overlapMinX; x < overlapMaxX; x++ {
			ga := a.Gray[y*a.W+x]
			bx, by := x+dx, y+dy
			gb := b.Gray[by*b.W+bx]

			var diff uint64
			if ga > gb {
				diff = uint64(ga - gb)
			} else {
				diff = uint64(gb - ga)
			}
			cumError += diff

			// Early abandon
			if cumError > earlyAbandonThreshold {
				return math.MaxFloat64
			}
		}
	}

	return float64(cumError) / float64(totalPixels)
}
