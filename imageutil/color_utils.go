package imageutil

import (
	"image/color"
	"math"

	"github.com/user/go-img-diff/utils"
)

// colorDifference は2つの色の間の差（ユークリッド距離）を計算する
// 0.0~765.0の範囲で値を返す（0=完全一致、765=最大差異[白と黒]）
func (da *DiffAnalyzer) colorDifference(c1, c2 color.Color) float64 {
	// RGBAに変換
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	// 16ビットから8ビットに変換（上位8ビットを取り出す）
	r1, g1, b1, a1 = r1>>8, g1>>8, b1>>8, a1>>8
	r2, g2, b2, a2 = r2>>8, g2>>8, b2>>8, a2>>8

	// 完全透明ピクセルの処理
	if a1 == 0 && a2 == 0 {
		return 0.0 // 両方透明なら差なし
	}

	// アルファ値も考慮したRGBの差を計算
	// アルファ値で重み付けした差分
	alphaFactor := float64(a1+a2) / (2.0 * 255.0) // 平均アルファ値（0.0～1.0）

	// 各成分のユークリッド距離を計算
	distance := math.Sqrt(
		math.Pow(float64(int(r1)-int(r2)), 2) +
			math.Pow(float64(int(g1)-int(g2)), 2) +
			math.Pow(float64(int(b1)-int(b2)), 2))

	// アルファ値の差も考慮
	alphaDiff := math.Abs(float64(int(a1) - int(a2)))

	// 色の差とアルファの差を合成（アルファの差の影響は小さめに）
	return distance*alphaFactor + alphaDiff*0.3
}

// blendColors は色を混合する拡張版関数
// dst: 背景色（比較先画像のピクセル）
// src: 元画像のピクセル色
// transparency: 元画像の透明度 (0.0=不透明、1.0=完全透明)
// tint: 適用する色調
// useTint: 色調を適用するかどうか
// tintStrength: 色調の強さ (0.0=色調なし、1.0=完全に色調のみ)
// tintTransparency: 色調の透明度 (0.0=不透明、1.0=完全透明)
func blendColors(
	dst, src color.Color,
	transparency float64,
	tint color.RGBA,
	useTint bool,
	tintStrength, tintTransparency float64,
) color.RGBA {
	// 色をRGBAに変換
	dr, dg, db, da := dst.RGBA()
	sr, sg, sb, sa := src.RGBA()

	// 上位8ビットを取得（0-255の範囲に変換）
	dr8 := uint8(dr >> 8)
	dg8 := uint8(dg >> 8)
	db8 := uint8(db >> 8)
	da8 := uint8(da >> 8)

	sr8 := uint8(sr >> 8)
	sg8 := uint8(sg >> 8)
	sb8 := uint8(sb >> 8)
	sa8 := uint8(sa >> 8)

	var r, g, b uint8

	if useTint {
		// 1. 色調と元画像を混合
		srcWeight := 1.0 - tintStrength
		tr := uint8(float64(sr8)*srcWeight + float64(tint.R)*tintStrength)
		tg := uint8(float64(sg8)*srcWeight + float64(tint.G)*tintStrength)
		tb := uint8(float64(sb8)*srcWeight + float64(tint.B)*tintStrength)

		// 2. 色調適用済みの色を背景と混合（色調の透明度を考慮）
		effectiveTransparency := (transparency + tintTransparency) / 2
		r = uint8(float64(tr)*(1-effectiveTransparency) + float64(dr8)*effectiveTransparency)
		g = uint8(float64(tg)*(1-effectiveTransparency) + float64(dg8)*effectiveTransparency)
		b = uint8(float64(tb)*(1-effectiveTransparency) + float64(db8)*effectiveTransparency)
	} else {
		// 色調なしの通常の透過処理
		r = uint8(float64(sr8)*(1-transparency) + float64(dr8)*transparency)
		g = uint8(float64(sg8)*(1-transparency) + float64(dg8)*transparency)
		b = uint8(float64(sb8)*(1-transparency) + float64(db8)*transparency)
	}

	// アルファは大きい方を採用
	a := uint8(utils.Max(int(sa8), int(da8)))

	return color.RGBA{r, g, b, a}
}

// 下位互換性のための簡易版blendColors
func blendColorsSimple(dst, src color.Color, transparency float64, tint color.RGBA, useTint bool) color.RGBA {
	// デフォルト値でフル版の関数を呼び出す
	tintStrength := 0.7
	tintTransparency := transparency
	if !useTint {
		tintStrength = 0.0
	}

	return blendColors(dst, src, transparency, tint, useTint, tintStrength, tintTransparency)
}
