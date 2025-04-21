package imageutil

import (
	"image"
	"image/color"
	"testing"

	"github.com/xshoji/go-img-diff/config" // Config をインポート
)

func TestDiffAnalyzer_drawRedBorders(t *testing.T) {
	// テスト用のDiffAnalyzerを初期化
	da := &DiffAnalyzer{
		cfg: &config.AppConfig{ // config.Config に変更
			ShowTransparentOverlay: true,
			OverlayTransparency:    0.5,
			OverlayTint:            color.RGBA{0, 0, 255, 255}, // 青色
			UseTint:                true,
			TintStrength:           0.3,
			TintTransparency:       0.7,
		},
	}

	// テスト用の画像と領域を初期化
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	regions := []image.Rectangle{
		image.Rect(10, 10, 20, 20),
		image.Rect(30, 30, 40, 40),
	}
	srcImgA := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// 元画像の特定領域を緑色で塗りつぶす
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			srcImgA.Set(x, y, color.RGBA{0, 255, 0, 255}) // 緑色
		}
	}

	// drawRedBorders関数を呼び出し
	da.drawRedBorders(img, regions, srcImgA, 0, 0)

	// 結果を検証（簡単な検証。実際にはより厳密な検証が必要）
	// 例えば、枠の色が赤色になっているか、透過処理が適用されているかなどを確認する
	red := color.RGBA{255, 0, 0, 255}
	borderThickness := 3

	// 最初の矩形領域の枠が赤色で描画されているか確認
	for x := regions[0].Min.X; x < regions[0].Max.X; x++ {
		for i := 0; i < borderThickness; i++ {
			// 上辺
			if y := regions[0].Min.Y + i; y < regions[0].Max.Y {
				r, g, b, _ := img.At(x, y).RGBA()
				if uint8(r>>8) != red.R || uint8(g>>8) != red.G || uint8(b>>8) != red.B {
					t.Errorf("Expected red color at (%d, %d), got (%d, %d, %d)", x, y, r>>8, g>>8, b>>8)
				}
			}
			// 下辺
			if y := regions[0].Max.Y - 1 - i; y >= regions[0].Min.Y {
				r, g, b, _ := img.At(x, y).RGBA()
				if uint8(r>>8) != red.R || uint8(g>>8) != red.G || uint8(b>>8) != red.B {
					t.Errorf("Expected red color at (%d, %d), got (%d, %d, %d)", x, y, r>>8, g>>8, b>>8)
				}
			}
		}
	}

	// 最初の矩形領域の内側が透過されているか確認（簡易的な確認）
	innerX := regions[0].Min.X + borderThickness
	innerY := regions[0].Min.Y + borderThickness
	_, g, _, a := img.At(innerX, innerY).RGBA()

	// 透過処理が適用されているかの確認（アルファ値が0でないことを確認）
	if a == 0 {
		t.Errorf("Expected transparency at (%d, %d), got alpha = %d", innerX, innerY, a)
	}

	// 色調が適用されているかの確認（緑色が混ざっていることを確認）
	if g>>8 == 0 {
		t.Errorf("Expected tint at (%d, %d), got green = %d", innerX, innerY, g>>8)
	}
}
