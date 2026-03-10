package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

// CombineHorizontal は2つの画像を水平方向に結合する
// 左側にleft画像、右側にright画像を配置する
// 高さが異なる場合は大きい方に合わせ、小さい方は上寄せ＋背景白
func CombineHorizontal(left, right image.Image) *image.RGBA {
	leftBounds := left.Bounds()
	rightBounds := right.Bounds()

	gap := 20 // 画像間の隙間（ピクセル）
	totalWidth := leftBounds.Dx() + gap + rightBounds.Dx()
	maxHeight := leftBounds.Dy()
	if rightBounds.Dy() > maxHeight {
		maxHeight = rightBounds.Dy()
	}

	combined := image.NewRGBA(image.Rect(0, 0, totalWidth, maxHeight))

	// 背景を白で塗りつぶす
	draw.Draw(combined, combined.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// 左側に left 画像を描画
	draw.Draw(combined, image.Rect(0, 0, leftBounds.Dx(), leftBounds.Dy()), left, leftBounds.Min, draw.Over)

	// 右側に right 画像を描画（gap分オフセット）
	rightX := leftBounds.Dx() + gap
	draw.Draw(combined, image.Rect(rightX, 0, totalWidth, rightBounds.Dy()), right, rightBounds.Min, draw.Over)

	return combined
}
