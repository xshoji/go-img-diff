package imageutil

import (
	"image"

	"github.com/xshoji/go-img-diff/utils"
)

// HasDifferences は2つの画像の間に差分があるかどうかを検出する
func (da *DiffAnalyzer) HasDifferences(img1, img2 image.Image, offsetX, offsetY int) bool {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	// 比較対象の領域を計算
	minX := utils.Max(bounds1.Min.X, bounds2.Min.X-offsetX)
	minY := utils.Max(bounds1.Min.Y, bounds2.Min.Y-offsetY)
	maxX := utils.Min(bounds1.Max.X, bounds2.Max.X-offsetX)
	maxY := utils.Min(bounds1.Max.Y, bounds2.Max.Y-offsetY)

	// サンプリングレート
	sampling := da.cfg.SamplingRate

	// 一定のサンプリングで差分をチェック
	for y := minY; y < maxY; y += sampling {
		for x := minX; x < maxX; x += sampling {
			// img1の色を取得
			r1, g1, b1, _ := img1.At(x, y).RGBA()

			// img2の対応するピクセルの座標を計算
			x2, y2 := x+offsetX, y+offsetY

			// 範囲外チェック
			if x2 < bounds2.Min.X || x2 >= bounds2.Max.X || y2 < bounds2.Min.Y || y2 >= bounds2.Max.Y {
				continue
			}

			// img2の色を取得
			r2, g2, b2, _ := img2.At(x2, y2).RGBA()

			// 各色チャンネルの差を計算
			diff := colorDifference(r1, g1, b1, r2, g2, b2)

			// 閾値を超える差があれば差分ありと判断
			if diff > uint32(da.cfg.Threshold) {
				return true
			}
		}
	}

	return false
}

// colorDifference は2つの色の差を計算する
func colorDifference(r1, g1, b1, r2, g2, b2 uint32) uint32 {
	// 16ビットから8ビットに変換
	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8

	// 絶対差を計算
	rDiff := utils.AbsDiff(r1, r2)
	gDiff := utils.AbsDiff(g1, g2)
	bDiff := utils.AbsDiff(b1, b2)

	// 最大差を返す
	return utils.MaxUint32(utils.MaxUint32(rDiff, gDiff), bDiff)
}
