package imageutil

import (
	"image"
	"math"
	"sort"

	"github.com/user/go-img-diff/utils"
)

// mergeOverlappingRectangles は重なり合う矩形を連結して大きな矩形にする
// 入れ子になった赤枠や重なりを全て統合する
func mergeOverlappingRectangles(rects []image.Rectangle) []image.Rectangle {
	if len(rects) <= 1 {
		return rects
	}

	// 連結処理を繰り返し適用
	result := make([]image.Rectangle, len(rects))
	copy(result, rects)

	// 繰り返し統合が行われる限り処理を続ける
	changed := true
	maxIterations := 20 // 無限ループ防止のための最大反復回数を増やす
	iteration := 0

	for changed && iteration < maxIterations {
		iteration++
		changed = false

		// 結果をサイズ順にソート（小さい領域から処理するため）
		sort.Slice(result, func(i, j int) bool {
			if !isValidRect(result[i]) || !isValidRect(result[j]) {
				return false
			}
			area1 := rectArea(result[i])
			area2 := rectArea(result[j])
			return area1 < area2
		})

		// 無効な矩形を除去
		result = filterValidRects(result)

		// 各矩形ペアの統合をチェック
		for i := 0; i < len(result); i++ {
			// マージするかどうかの決定
			for j := i + 1; j < len(result); j++ {
				// 両方の矩形が有効か確認
				if !isValidRect(result[i]) || !isValidRect(result[j]) {
					continue
				}

				// 片方が他方を完全に含む場合(入れ子関係)は、大きい方だけを保持
				if containsRect(result[i], result[j]) {
					result[j] = image.Rectangle{} // 小さい方を無効化
					changed = true
					continue
				}

				if containsRect(result[j], result[i]) {
					result[i] = result[j]         // 大きい方を採用
					result[j] = image.Rectangle{} // 重複を避けるため無効化
					changed = true
					continue
				}

				// 重なりや近接判定
				if shouldMergeRects(result[i], result[j]) {
					// 矩形を連結
					mergedRect := unionRectangles(result[i], result[j])

					// マージ後の面積が極端に大きくなる場合は避ける
					if isReasonableMerge(result[i], result[j], mergedRect) {
						result[i] = mergedRect
						result[j] = image.Rectangle{} // 処理済みの矩形を無効化
						changed = true
					}
				}
			}
		}

		// 無効な矩形を除去
		if changed {
			result = filterValidRects(result)
		}
	}

	// 結果が多すぎる場合は、小さな矩形をフィルタリング
	if len(result) > 50 {
		// サイズでソート（大きい順）
		sort.Slice(result, func(i, j int) bool {
			area1 := rectArea(result[i])
			area2 := rectArea(result[j])
			return area1 > area2 // 大きい順
		})

		// 上位50個だけを保持
		if len(result) > 50 {
			result = result[:50]
		}
	}

	// 最終的に重なりがないことを確認するため、もう一度全ての組み合わせをチェック
	finalResult := finalizeRectangles(result)

	return finalResult
}

// rectArea は矩形の面積を計算
func rectArea(rect image.Rectangle) int {
	return (rect.Max.X - rect.Min.X) * (rect.Max.Y - rect.Min.Y)
}

// filterValidRects は有効な矩形だけを残す
func filterValidRects(rects []image.Rectangle) []image.Rectangle {
	var validRects []image.Rectangle
	for _, r := range rects {
		if isValidRect(r) {
			validRects = append(validRects, r)
		}
	}
	return validRects
}

// containsRect は矩形r1が矩形r2を完全に含むかどうかをチェック（入れ子検出）
func containsRect(r1, r2 image.Rectangle) bool {
	// 余裕を持たせるための係数(少しのはみ出しは許容)
	const margin = 5

	return r1.Min.X-margin <= r2.Min.X &&
		r1.Min.Y-margin <= r2.Min.Y &&
		r1.Max.X+margin >= r2.Max.X &&
		r1.Max.Y+margin >= r2.Max.Y
}

// shouldMergeRects は2つの矩形が統合されるべきかを判断
func shouldMergeRects(r1, r2 image.Rectangle) bool {
	// 重なりがあるか極めて近接している場合に統合
	if !doRectanglesOverlapOrTouch(r1, r2) {
		return false
	}

	// 面積が大きく異なる場合は統合を避ける
	area1 := rectArea(r1)
	area2 := rectArea(r2)

	// 面積比が10倍以上異なる場合は連結を慎重に
	const maxAreaRatio = 10.0
	if float64(area1) > float64(area2)*maxAreaRatio ||
		float64(area2) > float64(area1)*maxAreaRatio {

		// 重なり具合が大きい場合は例外的に統合する
		overlapRatio := calcOverlapRatio(r1, r2)
		if overlapRatio > 0.5 { // 50%以上重なる場合
			return true
		}
		return false
	}

	// 上記以外は統合OK
	return true
}

// calcOverlapRatio は2つの矩形の重なり具合を計算（0.0～1.0）
func calcOverlapRatio(r1, r2 image.Rectangle) float64 {
	// 交差領域を計算
	intersection := image.Rect(
		utils.Max(r1.Min.X, r2.Min.X),
		utils.Max(r1.Min.Y, r2.Min.Y),
		utils.Min(r1.Max.X, r2.Max.X),
		utils.Min(r1.Max.Y, r2.Max.Y),
	)

	// 交差領域の面積
	intersectionArea := rectArea(intersection)
	if intersectionArea <= 0 {
		return 0.0
	}

	// 小さい方の矩形に対する重なり比率
	smallerArea := utils.Min(rectArea(r1), rectArea(r2))
	if smallerArea <= 0 {
		return 0.0
	}

	return float64(intersectionArea) / float64(smallerArea)
}

// isReasonableMerge はマージが合理的かどうかを判断
func isReasonableMerge(r1, r2, mergedRect image.Rectangle) bool {
	beforeArea := rectArea(r1) + rectArea(r2)
	mergedArea := rectArea(mergedRect)

	// マージ後の面積が元の合計の1.8倍以上になる場合はマージしない
	// (値を大きくすることで、より多くの矩形を統合可能に)
	const maxAreaIncrease = 1.8
	return float64(mergedArea) <= float64(beforeArea)*maxAreaIncrease
}

// finalizeRectangles は最終的な重なりチェックを行い、必要なら追加統合する
func finalizeRectangles(rects []image.Rectangle) []image.Rectangle {
	// 重複が無くなるまで処理を繰り返す
	result := make([]image.Rectangle, len(rects))
	copy(result, rects)

	changed := true
	maxPasses := 3 // 最大パス回数

	for pass := 0; changed && pass < maxPasses; pass++ {
		changed = false

		// 完全な内包関係をチェック(入れ子になっている赤枠を排除)
		for i := 0; i < len(result); i++ {
			if !isValidRect(result[i]) {
				continue
			}

			for j := 0; j < len(result); j++ {
				if i == j || !isValidRect(result[j]) {
					continue
				}

				// 同じ矩形や非常に近い矩形を検出
				if areRectsSimilar(result[i], result[j]) {
					// 面積が大きい方を採用
					if rectArea(result[i]) >= rectArea(result[j]) {
						result[j] = image.Rectangle{} // 小さい方を無効化
					} else {
						result[i] = result[j]
						result[j] = image.Rectangle{}
					}
					changed = true
				}
			}
		}

		// 無効な矩形を除去
		if changed {
			result = filterValidRects(result)
		}
	}

	return result
}

// areRectsSimilar は2つの矩形が非常に似ているかを判定
func areRectsSimilar(r1, r2 image.Rectangle) bool {
	// 中心点間の距離を計算
	center1X := (r1.Min.X + r1.Max.X) / 2
	center1Y := (r1.Min.Y + r1.Max.Y) / 2
	center2X := (r2.Min.X + r2.Max.X) / 2
	center2Y := (r2.Min.Y + r2.Max.Y) / 2

	// 中心点間の距離
	distance := math.Sqrt(float64(
		(center1X-center2X)*(center1X-center2X) +
			(center1Y-center2Y)*(center1Y-center2Y)))

	// サイズの平均
	avgWidth := (r1.Max.X - r1.Min.X + r2.Max.X - r2.Min.X) / 2
	avgHeight := (r1.Max.Y - r1.Min.Y + r2.Max.Y - r2.Min.Y) / 2

	// 矩形の大きさを考慮した類似度判定
	// 中心点間の距離が平均幅・高さの30%未満なら類似と判断
	return distance < float64(avgWidth+avgHeight)*0.15
}

// isValidRect は矩形が有効かどうかをチェック
func isValidRect(rect image.Rectangle) bool {
	return rect.Min.X < rect.Max.X && rect.Min.Y < rect.Max.Y
}

// doRectanglesOverlapOrTouch は2つの矩形が重なっているか、または隣接しているかをチェック
func doRectanglesOverlapOrTouch(r1, r2 image.Rectangle) bool {
	// 重なりチェックの余裕を持たせる距離（より局所的な連結のために値を小さくする）
	const proximityThreshold = 10 // 20から10に縮小

	// 距離に基づく判定（実際の重なりか近接している場合のみ連結する）
	overlapX := !(r1.Max.X+proximityThreshold <= r2.Min.X || r2.Max.X+proximityThreshold <= r1.Min.X)
	overlapY := !(r1.Max.Y+proximityThreshold <= r2.Min.Y || r2.Max.Y+proximityThreshold <= r1.Min.Y)

	// 両方の軸で近接していることを確認
	if !overlapX || !overlapY {
		return false
	}

	// 重なりの程度を評価
	intersection := image.Rect(
		utils.Max(r1.Min.X, r2.Min.X),
		utils.Max(r1.Min.Y, r2.Min.Y),
		utils.Min(r1.Max.X, r2.Max.X),
		utils.Min(r1.Max.Y, r2.Max.Y),
	)

	// 交差領域の面積
	intersectionArea := (intersection.Max.X - intersection.Min.X) * (intersection.Max.Y - intersection.Min.Y)

	// 仮想的な拡張領域を含む場合は負の値になる可能性があるため、0以下なら0とする
	if intersectionArea <= 0 {
		// 実際に重なっていない場合は、中心点間の距離を計算
		center1X := (r1.Min.X + r1.Max.X) / 2
		center1Y := (r1.Min.Y + r1.Max.Y) / 2
		center2X := (r2.Min.X + r2.Max.X) / 2
		center2Y := (r2.Min.Y + r2.Max.Y) / 2

		// 対角線の長さを計算
		diagonal1 := math.Sqrt(float64((r1.Max.X-r1.Min.X)*(r1.Max.X-r1.Min.X) + (r1.Max.Y-r1.Min.Y)*(r1.Max.Y-r1.Min.Y)))
		diagonal2 := math.Sqrt(float64((r2.Max.X-r2.Min.X)*(r2.Max.X-r2.Min.X) + (r2.Max.Y-r2.Min.Y)*(r2.Max.Y-r2.Min.Y)))
		avgDiagonal := (diagonal1 + diagonal2) / 2

		// 中心点間の距離
		distance := math.Sqrt(float64((center1X-center2X)*(center1X-center2X) + (center1Y-center2Y)*(center1Y-center2Y)))

		// 対角線の平均の半分以下の距離なら連結
		return distance < avgDiagonal/2
	}

	// 交差領域が小さすぎる場合は連結しない
	smallerArea := utils.Min(rectArea(r1), rectArea(r2))
	return intersectionArea >= smallerArea/5 // 少なくとも小さい方の矩形の20%以上重なっていること
}

// unionRectangles は2つの矩形を包含する最小の矩形を返す
func unionRectangles(r1, r2 image.Rectangle) image.Rectangle {
	return image.Rect(
		utils.Min(r1.Min.X, r2.Min.X),
		utils.Min(r1.Min.Y, r2.Min.Y),
		utils.Max(r1.Max.X, r2.Max.X),
		utils.Max(r1.Max.Y, r2.Max.Y),
	)
}
