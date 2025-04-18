package imageutil

import (
	"image"
	"image/color"
)

// drawRedBorders は指定された領域に赤枠を描画し、差分部分を透過表示する
func (da *DiffAnalyzer) drawRedBorders(img *image.RGBA, regions []image.Rectangle, srcImgA image.Image, offsetX, offsetY int) {
	red := color.RGBA{255, 0, 0, 255} // 赤枠の色

	// 枠の太さを定義
	borderThickness := 3

	for _, rect := range regions {
		// 1. 差分領域内に元画像（imgA）を透過表示
		if srcImgA != nil && da.cfg.ShowTransparentOverlay {
			// 透過率を設定
			transparency := da.cfg.OverlayTransparency
			tintStrength := da.cfg.TintStrength
			tintTransparency := da.cfg.TintTransparency

			// 差分領域内の各ピクセルについて処理
			for y := rect.Min.Y + borderThickness; y < rect.Max.Y-borderThickness; y++ {
				for x := rect.Min.X + borderThickness; x < rect.Max.X-borderThickness; x++ {
					// 画像の範囲内かチェック
					if x >= img.Bounds().Min.X && x < img.Bounds().Max.X &&
						y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
						// 元画像の座標を計算（オフセットを考慮）
						srcX := x - offsetX
						srcY := y - offsetY

						// 元画像の範囲内かチェック
						srcBounds := srcImgA.Bounds()
						if srcX >= srcBounds.Min.X && srcX < srcBounds.Max.X &&
							srcY >= srcBounds.Min.Y && srcY < srcBounds.Max.Y {
							// 現在の色と元画像の色を取得
							dstColor := img.At(x, y)
							srcColor := srcImgA.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY)

							// 色を混合（色調を付加）
							blendedColor := blendColors(
								dstColor,
								srcColor,
								transparency,
								da.cfg.OverlayTint,
								da.cfg.UseTint,
								tintStrength,
								tintTransparency,
							)
							img.Set(x, y, blendedColor)
						}
					}
				}
			}
		}

		// 2. 赤枠を描画
		// 上辺と下辺を描画
		for x := rect.Min.X; x < rect.Max.X; x++ {
			// 上辺
			for i := 0; i < borderThickness; i++ {
				if y := rect.Min.Y + i; y < rect.Max.Y {
					img.Set(x, y, red)
				}
			}
			// 下辺
			for i := 0; i < borderThickness; i++ {
				if y := rect.Max.Y - 1 - i; y >= rect.Min.Y {
					img.Set(x, y, red)
				}
			}
		}

		// 左辺と右辺を描画
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			// 左辺
			for i := 0; i < borderThickness; i++ {
				if x := rect.Min.X + i; x < rect.Max.X {
					img.Set(x, y, red)
				}
			}
			// 右辺
			for i := 0; i < borderThickness; i++ {
				if x := rect.Max.X - 1 - i; x >= rect.Min.X {
					img.Set(x, y, red)
				}
			}
		}
	}
}
