package align

import (
	"log/slog"
	"math"

	"github.com/xshoji/go-img-diff/internal/core"
)

const (
	defaultBandHeight  = 8
	defaultFeatureBins = 32
	blankStripeScale   = 0.35

	weightMeanGray = 0.45
	weightEdgeX    = 0.35
	weightInkRatio = 0.10
	weightVariance = 0.10
)

type stripeFeature struct {
	Index    int
	StartY   int
	EndY     int
	MeanGray []float64
	EdgeX    []float64
	InkRatio float64
	Variance float64
}

type dpStepKind uint8

const (
	dpStepNone dpStepKind = iota
	dpStepMatch
	dpStepDelete
	dpStepInsert
)

type dpStep struct {
	Kind   dpStepKind
	AIndex int
	BIndex int
}

type dpResult struct {
	Path    []dpStep
	Cost    float64
	Matches int
	Deletes int
	Inserts int
}

// VerticalDPAlign refines global alignment by re-synchronizing rows through
// stripe-level dynamic programming. It targets website screenshots where most
// structural changes manifest as vertical insertions and deletions.
func VerticalDPAlign(a, b *core.Frame, global core.Alignment, opts core.VerticalAlignOptions, logger *slog.Logger) core.RowAlignment {
	return VerticalDPAlignInRange(a, b, global, opts, 0, b.W, logger)
}

// VerticalDPAlignInRange runs the stripe DP against a vertical strip of the image.
func VerticalDPAlignInRange(a, b *core.Frame, global core.Alignment, opts core.VerticalAlignOptions, minX, maxX int, logger *slog.Logger) core.RowAlignment {
	fallback := core.NewRowAlignmentFromAlignment(b.W, b.H, global)
	if !opts.Enabled || a.H == 0 || b.H == 0 || a.W == 0 || b.W == 0 {
		return fallback
	}
	minX = max(0, min(minX, b.W))
	maxX = max(0, min(maxX, b.W))
	if minX >= maxX {
		return fallback
	}

	bandHeight := opts.BandHeight
	if bandHeight <= 0 {
		bandHeight = defaultBandHeight
	}
	bins := opts.FeatureBins
	if bins <= 0 {
		bins = defaultFeatureBins
	}

	stripesA := buildStripeFeaturesInRange(a, minX, maxX, bandHeight, bins)
	stripesB := buildStripeFeaturesInRange(b, minX, maxX, bandHeight, bins)
	if len(stripesA) == 0 || len(stripesB) == 0 {
		return fallback
	}

	globalBandOffset := divRound(global.DY, bandHeight)
	maxBandShift := opts.MaxBandShift
	if maxBandShift <= 0 {
		maxBandShift = max(32, max(len(stripesA), len(stripesB))/6)
	}

	result, ok := alignStripesDP(stripesA, stripesB, globalBandOffset, opts, maxBandShift)
	if !ok {
		logger.Warn("vertical dp alignment failed, falling back to global alignment",
			"bandsA", len(stripesA),
			"bandsB", len(stripesB),
			"globalDX", global.DX,
			"globalDY", global.DY,
		)
		return fallback
	}

	rowAlign := expandStripePath(result.Path, stripesA, stripesB, b.W, b.H, global)
	rowAlign.Score = scoreFromCost(result.Cost, len(result.Path))

	if minX == 0 && maxX == b.W {
		logger.Info("vertical dp complete",
			"bandsA", len(stripesA),
			"bandsB", len(stripesB),
			"bandHeight", bandHeight,
			"featureBins", bins,
			"globalDX", global.DX,
			"globalDY", global.DY,
			"maxBandShift", maxBandShift,
			"matches", result.Matches,
			"inserts", result.Inserts,
			"deletes", result.Deletes,
			"score", rowAlign.Score,
		)
	}

	return rowAlign
}

func buildStripeFeatures(f *core.Frame, bandHeight, bins int) []stripeFeature {
	return buildStripeFeaturesInRange(f, 0, f.W, bandHeight, bins)
}

func buildStripeFeaturesInRange(f *core.Frame, minX, maxX, bandHeight, bins int) []stripeFeature {
	if f.W == 0 || f.H == 0 {
		return nil
	}
	minX = max(0, min(minX, f.W))
	maxX = max(0, min(maxX, f.W))
	if minX >= maxX {
		return nil
	}
	if bandHeight <= 0 {
		bandHeight = defaultBandHeight
	}
	if bins <= 0 {
		bins = defaultFeatureBins
	}
	stripWidth := maxX - minX
	if bins > stripWidth {
		bins = stripWidth
	}

	stripes := make([]stripeFeature, 0, (f.H+bandHeight-1)/bandHeight)
	for startY, idx := 0, 0; startY < f.H; startY, idx = startY+bandHeight, idx+1 {
		endY := min(f.H, startY+bandHeight)
		meanGraySums := make([]float64, bins)
		edgeSums := make([]float64, bins)
		pixelCounts := make([]int, bins)
		edgeCounts := make([]int, bins)

		var graySum float64
		var graySumSq float64
		var inkCount float64
		pixelTotal := (endY - startY) * stripWidth

		for y := startY; y < endY; y++ {
			rowOffset := y * f.W
			var prevGray float64
			for x := minX; x < maxX; x++ {
				gray := float64(f.Gray[rowOffset+x])
				bin := (x - minX) * bins / stripWidth
				meanGraySums[bin] += gray
				pixelCounts[bin]++
				graySum += gray
				graySumSq += gray * gray
				if gray < 245 {
					inkCount++
				}
				if x > minX {
					edgeSums[bin] += math.Abs(gray - prevGray)
					edgeCounts[bin]++
				}
				prevGray = gray
			}
		}

		meanGray := make([]float64, bins)
		edgeX := make([]float64, bins)
		for i := 0; i < bins; i++ {
			if pixelCounts[i] > 0 {
				meanGray[i] = meanGraySums[i] / float64(pixelCounts[i])
			}
			if edgeCounts[i] > 0 {
				edgeX[i] = edgeSums[i] / float64(edgeCounts[i])
			}
		}

		variance := 0.0
		if pixelTotal > 0 {
			mean := graySum / float64(pixelTotal)
			variance = (graySumSq / float64(pixelTotal)) - mean*mean
			if variance < 0 {
				variance = 0
			}
			variance /= 255.0 * 255.0
		}

		stripes = append(stripes, stripeFeature{
			Index:    idx,
			StartY:   startY,
			EndY:     endY,
			MeanGray: meanGray,
			EdgeX:    edgeX,
			InkRatio: inkCount / float64(max(1, pixelTotal)),
			Variance: variance,
		})
	}

	return stripes
}

func stripeMatchCost(a, b stripeFeature, opts core.VerticalAlignOptions) float64 {
	meanGrayCost := meanAbsDiff(a.MeanGray, b.MeanGray)
	edgeCost := meanAbsDiff(a.EdgeX, b.EdgeX)
	inkCost := math.Abs(a.InkRatio-b.InkRatio) * 255.0
	varianceCost := math.Abs(a.Variance-b.Variance) * 255.0

	cost := weightMeanGray*meanGrayCost +
		weightEdgeX*edgeCost +
		weightInkRatio*inkCost +
		weightVariance*varianceCost

	if a.InkRatio <= opts.BlankInkMax && b.InkRatio <= opts.BlankInkMax {
		cost *= blankStripeScale
	}

	return cost
}

func alignStripesDP(a, b []stripeFeature, globalBandOffset int, opts core.VerticalAlignOptions, maxBandShift int) (dpResult, bool) {
	n := len(a)
	m := len(b)
	cols := m + 1
	inf := math.MaxFloat64 / 4
	gapPenalty := opts.GapPenalty
	if gapPenalty <= 0 {
		gapPenalty = 18.0
	}

	dp := make([]float64, (n+1)*cols)
	dir := make([]dpStepKind, (n+1)*cols)
	for i := range dp {
		dp[i] = inf
	}
	dp[0] = 0

	for i := 0; i <= n; i++ {
		jStart, jEnd := bandRange(i, m, globalBandOffset, maxBandShift)
		if i == 0 {
			jStart = 0
		}
		for j := jStart; j <= jEnd; j++ {
			if i == 0 && j == 0 {
				continue
			}

			idx := i*cols + j
			best := inf
			bestDir := dpStepNone

			if i > 0 && j > 0 {
				prev := (i-1)*cols + (j - 1)
				if dp[prev] < inf {
					cand := dp[prev] + stripeMatchCost(a[i-1], b[j-1], opts)
					best = cand
					bestDir = dpStepMatch
				}
			}

			if i > 0 {
				prev := (i-1)*cols + j
				if dp[prev] < inf {
					cand := dp[prev] + gapPenalty
					if cand < best {
						best = cand
						bestDir = dpStepDelete
					}
				}
			}

			if j > 0 {
				prev := i*cols + (j - 1)
				if dp[prev] < inf {
					cand := dp[prev] + gapPenalty
					if cand < best {
						best = cand
						bestDir = dpStepInsert
					}
				}
			}

			dp[idx] = best
			dir[idx] = bestDir
		}
	}

	if dp[n*cols+m] >= inf {
		return dpResult{}, false
	}

	path := make([]dpStep, 0, n+m)
	result := dpResult{Cost: dp[n*cols+m]}
	for i, j := n, m; i > 0 || j > 0; {
		kind := dir[i*cols+j]
		switch kind {
		case dpStepMatch:
			path = append(path, dpStep{Kind: kind, AIndex: i - 1, BIndex: j - 1})
			result.Matches++
			i--
			j--
		case dpStepDelete:
			path = append(path, dpStep{Kind: kind, AIndex: i - 1, BIndex: -1})
			result.Deletes++
			i--
		case dpStepInsert:
			path = append(path, dpStep{Kind: kind, AIndex: -1, BIndex: j - 1})
			result.Inserts++
			j--
		default:
			return dpResult{}, false
		}
	}

	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	result.Path = path
	return result, true
}

func expandStripePath(path []dpStep, stripesA, stripesB []stripeFeature, bWidth, bHeight int, global core.Alignment) core.RowAlignment {
	rowAlign := core.NewRowAlignmentFromAlignment(bWidth, bHeight, global)

	for _, step := range path {
		if step.Kind == dpStepDelete || step.BIndex < 0 {
			continue
		}

		stripeB := stripesB[step.BIndex]
		for y := stripeB.StartY; y < stripeB.EndY && y < bHeight; y++ {
			rowAlign.DXByY[y] = global.DX
		}

		if step.Kind == dpStepInsert || step.AIndex < 0 {
			// Keep the global row mapping for inserted bands so the changed area stays
			// localized instead of becoming a full-width diff stripe.
			continue
		}

		stripeA := stripesA[step.AIndex]
		aHeight := max(1, stripeA.EndY-stripeA.StartY)
		bHeightStripe := max(1, stripeB.EndY-stripeB.StartY)
		for y := stripeB.StartY; y < stripeB.EndY && y < bHeight; y++ {
			relY := y - stripeB.StartY
			srcY := stripeA.StartY + (relY*aHeight)/bHeightStripe
			if srcY >= stripeA.EndY {
				srcY = stripeA.EndY - 1
			}
			rowAlign.SrcYByY[y] = srcY
		}
	}

	return rowAlign
}

func scoreFromCost(totalCost float64, steps int) float64 {
	if steps <= 0 {
		return 1.0
	}
	avgCost := totalCost / float64(steps)
	return 1.0 / (1.0 + avgCost)
}

func meanAbsDiff(a, b []float64) float64 {
	n := min(len(a), len(b))
	if n == 0 {
		return 0
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += math.Abs(a[i] - b[i])
	}
	return sum / float64(n)
}

func bandRange(i, maxJ, globalBandOffset, maxBandShift int) (int, int) {
	center := i + globalBandOffset
	start := max(0, center-maxBandShift)
	end := min(maxJ, center+maxBandShift)
	if start > end {
		return 0, -1
	}
	return start, end
}

func divRound(numerator, denominator int) int {
	if denominator == 0 {
		return 0
	}
	if numerator >= 0 {
		return (numerator + denominator/2) / denominator
	}
	return -((-numerator + denominator/2) / denominator)
}
