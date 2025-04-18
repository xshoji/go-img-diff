package imageutil

import (
	"image"

	"github.com/user/go-img-diff/utils"
)

// calculateSimilarityScore は2つの画像間の類似度を計算する
// スコアは0.0～1.0の範囲（1.0が完全一致）
func (da *DiffAnalyzer) calculateSimilarityScore(imgA, imgB image.Image, offsetX, offsetY int) float64 {
	boundsA := imgA.Bounds()
	boundsB := imgB.Bounds()

	// 重なり合う領域を計算
	overlapWidth := utils.Min(boundsA.Max.X, boundsB.Max.X+offsetX) - utils.Max(boundsA.Min.X, boundsB.Min.X+offsetX)
	overlapHeight := utils.Min(boundsA.Max.Y, boundsB.Max.Y+offsetY) - utils.Max(boundsA.Min.Y, boundsB.Min.Y+offsetY)

	// 重なり合う領域がない場合は類似度0
	if overlapWidth <= 0 || overlapHeight <= 0 {
		return 0
	}

	// サンプリングレートを適用してピクセル比較を効率化
	samplingRate := da.cfg.SamplingRate

	// サンプリングした点数をカウント
	sampledPoints := 0
	matchingPoints := 0

	// 重なり領域内でのピクセル比較
	for y := 0; y < overlapHeight; y += samplingRate {
		for x := 0; x < overlapWidth; x += samplingRate {
			// A画像の座標
			xA := x + utils.Max(boundsA.Min.X, boundsB.Min.X+offsetX) - boundsA.Min.X
			yA := y + utils.Max(boundsA.Min.Y, boundsB.Min.Y+offsetY) - boundsA.Min.Y

			// B画像の座標
			xB := x + utils.Max(boundsA.Min.X, boundsB.Min.X+offsetX) - (boundsB.Min.X + offsetX)
			yB := y + utils.Max(boundsA.Min.Y, boundsB.Min.Y+offsetY) - (boundsB.Min.Y + offsetY)

			// 範囲チェック
			if xA < 0 || xA >= boundsA.Dx() || yA < 0 || yA >= boundsA.Dy() ||
				xB < 0 || xB >= boundsB.Dx() || yB < 0 || yB >= boundsB.Dy() {
				continue
			}

			// 色の差が閾値以下なら一致とみなす
			colorDiff := da.colorDifference(
				imgA.At(boundsA.Min.X+xA, boundsA.Min.Y+yA),
				imgB.At(boundsB.Min.X+xB, boundsB.Min.Y+yB),
			)

			sampledPoints++

			if colorDiff < float64(da.cfg.Threshold) {
				matchingPoints++
			}
		}
	}

	// サンプリングしたピクセルがない場合は0を返す
	if sampledPoints == 0 {
		return 0
	}

	// 重なり領域の割合の考慮（より大きな重なりを評価）
	overlapArea := overlapWidth * overlapHeight
	totalArea := utils.Max(boundsA.Dx()*boundsA.Dy(), boundsB.Dx()*boundsB.Dy())
	coverageRatio := float64(overlapArea) / float64(totalArea)

	// 重なりの割合と一致ピクセルの比率を考慮したスコア
	baseScore := float64(matchingPoints) / float64(sampledPoints)

	// 重なりが小さすぎる場合はスコアを下げる
	if coverageRatio < 0.5 {
		baseScore *= coverageRatio * 2.0 // 小さな重なりにペナルティ
	}

	return baseScore
}
