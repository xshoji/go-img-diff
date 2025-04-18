package config

import (
	"image/color"
)

// AppConfig は画像差分検出のための設定を保持する構造体
type AppConfig struct {
	// 位置ずれ検出のための設定
	MaxOffset     int  // 探索する最大オフセット（ピクセル単位）
	Threshold     int  // 色の差の閾値 (0-255)
	HighlightDiff bool // 差分を赤枠で強調表示するか

	// 並列処理のための設定
	NumCPU int // 使用するCPUコア数

	// サンプリング関連の設定
	SamplingRate int  // ピクセルサンプリングレート (1=全ピクセル, 2=1/2のピクセル)
	FastMode     bool // 段階的サンプリングを使用する高速モード

	// 進捗表示の設定
	ProgressStep int // 進捗表示の間隔（パーセント）

	// 透過表示の設定
	ShowTransparentOverlay bool       // 差分部分に元画像を透過表示するか
	OverlayTransparency    float64    // オーバーレイの透明度 (0.0=不透明、1.0=完全透明)
	OverlayTint            color.RGBA // 透過表示時の色調 (デフォルトは赤)
	UseTint                bool       // 色調を適用するかどうか
	TintStrength           float64    // 色調の強さ (0.0～1.0)
	TintTransparency       float64    // 色調の透明度 (0.0=不透明、1.0=完全透明)
}

// NewDefaultConfig はデフォルト設定を持つ新しいAppConfigを返す
func NewDefaultConfig() *AppConfig {
	return &AppConfig{
		MaxOffset:              10,
		Threshold:              30,
		HighlightDiff:          true,
		NumCPU:                 4,
		SamplingRate:           1,
		FastMode:               false,
		ProgressStep:           5,
		ShowTransparentOverlay: true,                       // デフォルトで透過表示を有効に
		OverlayTransparency:    0.3,                        // 30%の透明度
		OverlayTint:            color.RGBA{255, 0, 0, 255}, // 赤色のティント
		UseTint:                true,                       // デフォルトで色調を適用
		TintStrength:           0.7,                        // 70%の色調強度
		TintTransparency:       0.2,                        // 20%の色調透明度（より鮮明な色調）
	}
}
