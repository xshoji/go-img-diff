//go:build !light_test_only

package imageutil

import (
	"image"
	"image/color"
	"testing"

	"github.com/xshoji/go-img-diff/config"
	"github.com/xshoji/go-img-diff/utils"
)

// createTestImage は指定されたサイズとパターンでテスト画像を作成する
func createTestImage(width, height int, fill color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	return img
}

// createTestImageWithOffset はオフセットを持つテスト画像ペアを作成する
// img1: 基準となる画像
// img2: img1をoffsetX, offsetY分だけずらした画像
func createTestImageWithOffset(width, height, offsetX, offsetY int) (*image.RGBA, *image.RGBA) {
	// 元画像（黒地に白い丸を描画）
	img1 := createTestImage(width, height, color.RGBA{0, 0, 0, 255})

	// 中心に白い丸を描画
	centerX, centerY := width/2, height/2
	radius := width / 8

	// img1に円を描画
	drawCircle(img1, centerX, centerY, radius, color.RGBA{255, 255, 255, 255})

	// 2つ目の画像（オフセット分ずらした同じパターン）
	img2 := createTestImage(width, height, color.RGBA{0, 0, 0, 255})

	// 注意: 画像処理のオフセット検出では、img2上での座標(x,y)が
	// img1上のどの座標(x+offsetX, y+offsetY)に対応するかを求める
	// したがって、img2にパターンを描画する際は逆方向に-offsetXと-offsetYを適用する
	offsetCenterX := centerX - offsetX
	offsetCenterY := centerY - offsetY
	drawCircle(img2, offsetCenterX, offsetCenterY, radius, color.RGBA{255, 255, 255, 255})

	return img1, img2
}

// 描画用ヘルパー関数
// drawCircle は指定された位置に円を描画
func drawCircle(img *image.RGBA, centerX, centerY, radius int, c color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := centerX+dx, centerY+dy
				if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

// drawRect は指定された位置に長方形を描画
func drawRect(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && ny >= 0 && nx < img.Bounds().Dx() && ny < img.Bounds().Dy() {
				img.SetRGBA(nx, ny, c)
			}
		}
	}
}

// drawTriangle は指定された位置に三角形を描画
func drawTriangle(img *image.RGBA, centerX, centerY, size int, c color.RGBA) {
	for dy := 0; dy < size; dy++ {
		width := 2 * dy
		startX := centerX - dy
		y := centerY - size + dy

		for dx := 0; dx <= width; dx++ {
			x := startX + dx
			if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// drawCross は指定された位置に十字を描く
func drawCross(img *image.RGBA, centerX, centerY, size int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	for dy := -size; dy <= size; dy++ {
		y := centerY + dy
		if y >= 0 && y < height {
			for dx := -size; dx <= size; dx++ {
				x := centerX + dx
				if x >= 0 && x < width {
					if dx == 0 || dy == 0 {
						img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}
}

// TestDiffAnalyzer_NewDiffAnalyzer は初期化のテスト
func TestDiffAnalyzer_NewDiffAnalyzer(t *testing.T) {
	cfg := config.NewDefaultConfig()
	analyzer := NewDiffAnalyzer(cfg)

	if analyzer == nil {
		t.Error("NewDiffAnalyzer should return a non-nil DiffAnalyzer")
	}

	if analyzer.cfg != cfg {
		t.Error("DiffAnalyzer should store the provided config")
	}
}

// TestDiffAnalyzer_CalculateSimilarityScore は類似スコア計算のテスト
func TestDiffAnalyzer_CalculateSimilarityScore(t *testing.T) {
	// 基本設定
	cfg := config.NewDefaultConfig()
	cfg.Threshold = 30
	cfg.SamplingRate = 1
	analyzer := NewDiffAnalyzer(cfg)

	testCases := []struct {
		name           string
		img1Color      color.RGBA
		img2Color      color.RGBA
		offsetX        int
		offsetY        int
		expectedScore  float64
		scoreThreshold float64 // 期待値との許容誤差
	}{
		{
			name:           "identical_images",
			img1Color:      color.RGBA{255, 0, 0, 255},
			img2Color:      color.RGBA{255, 0, 0, 255},
			offsetX:        0,
			offsetY:        0,
			expectedScore:  1.0,
			scoreThreshold: 0.01,
		},
		{
			name:           "completely_different",
			img1Color:      color.RGBA{255, 255, 255, 255},
			img2Color:      color.RGBA{0, 0, 0, 255},
			offsetX:        0,
			offsetY:        0,
			expectedScore:  0.0,
			scoreThreshold: 0.01,
		},
		{
			name:           "outside_bounds_offset",
			img1Color:      color.RGBA{255, 0, 0, 255},
			img2Color:      color.RGBA{255, 0, 0, 255},
			offsetX:        100, // 画像サイズより大きいオフセット
			offsetY:        100,
			expectedScore:  0.0,
			scoreThreshold: 0.01,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			img1 := createTestImage(50, 50, tc.img1Color)
			img2 := createTestImage(50, 50, tc.img2Color)

			score := analyzer.calculateSimilarityScore(img1, img2, tc.offsetX, tc.offsetY)

			// スコアを期待値と比較（許容誤差内か確認）
			if score < tc.expectedScore-tc.scoreThreshold || score > tc.expectedScore+tc.scoreThreshold {
				t.Errorf("Expected score around %.2f, got %.2f", tc.expectedScore, score)
			}
		})
	}
}

// TestDiffAnalyzer_FindBestAlignment はオフセット検出のテスト
func TestDiffAnalyzer_FindBestAlignment(t *testing.T) {
	// 基本設定
	cfg := config.NewDefaultConfig()
	cfg.MaxOffset = 10
	cfg.SamplingRate = 2
	cfg.FastMode = false
	cfg.NumCPU = 1 // テスト時は単一コアに制限

	analyzer := NewDiffAnalyzer(cfg)

	testCases := []struct {
		name            string
		width           int
		height          int
		offsetX         int
		offsetY         int
		expectedOffsetX int
		expectedOffsetY int
		threshold       int // 期待値との許容誤差（ピクセル）
	}{
		{
			name:            "small_positive_offset",
			width:           100,
			height:          100,
			offsetX:         5,
			offsetY:         3,
			expectedOffsetX: 5,
			expectedOffsetY: 3,
			threshold:       0,
		},
		{
			name:            "small_negative_offset",
			width:           100,
			height:          100,
			offsetX:         -4,
			offsetY:         -2,
			expectedOffsetX: -4,
			expectedOffsetY: -2,
			threshold:       0,
		},
		{
			name:            "zero_offset",
			width:           100,
			height:          100,
			offsetX:         0,
			offsetY:         0,
			expectedOffsetX: 0,
			expectedOffsetY: 0,
			threshold:       0,
		},
		{
			name:            "max_offset",
			width:           100,
			height:          100,
			offsetX:         cfg.MaxOffset,
			offsetY:         cfg.MaxOffset,
			expectedOffsetX: cfg.MaxOffset,
			expectedOffsetY: cfg.MaxOffset,
			threshold:       1, // 境界値なので少し許容
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 正しいオフセット方向でテスト画像を作成
			img1, img2 := createTestImageWithOffset(tc.width, tc.height, tc.offsetX, tc.offsetY)

			foundOffsetX, foundOffsetY := analyzer.FindBestAlignment(img1, img2)

			// 許容誤差内かチェック
			if utils.AbsInt(foundOffsetX-tc.expectedOffsetX) > tc.threshold || utils.AbsInt(foundOffsetY-tc.expectedOffsetY) > tc.threshold {
				t.Errorf("Expected offset around (%d, %d), got (%d, %d)",
					tc.expectedOffsetX, tc.expectedOffsetY, foundOffsetX, foundOffsetY)
			}
		})
	}

	// 高速モードのテスト
	t.Run("fast_mode", func(t *testing.T) {
		cfg.FastMode = true
		analyzer := NewDiffAnalyzer(cfg)

		// 基本的なオフセット検出のテスト
		img1, img2 := createTestImageWithOffset(100, 100, 5, 3)
		foundOffsetX, foundOffsetY := analyzer.FindBestAlignment(img1, img2)

		// 高速モードでも合理的な範囲で正確に検出できるか
		if utils.AbsInt(foundOffsetX-5) > 1 || utils.AbsInt(foundOffsetY-3) > 1 {
			t.Errorf("Fast mode failed to detect correct offset. Expected around (5, 3), got (%d, %d)",
				foundOffsetX, foundOffsetY)
		}
	})
}

// TestDiffAnalyzer_SearchBestOffsetInRange は部分検索のテスト
func TestDiffAnalyzer_SearchBestOffsetInRange(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.SamplingRate = 2
	cfg.NumCPU = 1 // テスト時は単一コアに制限
	analyzer := NewDiffAnalyzer(cfg)

	// 既知のオフセットを持つ画像を作成
	img1, img2 := createTestImageWithOffset(80, 80, 3, 2)

	// 範囲を指定して検索
	foundX, foundY, score := analyzer.searchBestOffsetInRange(img1, img2, 0, 5, 0, 5)

	// 期待されるオフセットに近いか検証
	if foundX != 3 || foundY != 2 {
		t.Errorf("searchBestOffsetInRange failed. Expected (3, 2), got (%d, %d) with score %.4f",
			foundX, foundY, score)
	}

	// オフセットが範囲外の場合の検証
	bestScore := analyzer.calculateSimilarityScore(img1, img2, foundX, foundY)
	wrongScore := analyzer.calculateSimilarityScore(img1, img2, -foundX, -foundY) // 逆方向のオフセット

	if wrongScore >= bestScore {
		t.Errorf("Expected correct offset to have better score than incorrect offset, but got wrong:%.4f >= best:%.4f",
			wrongScore, bestScore)
	}
}

// TestDiffAnalyzer_ComplexPatterns は複雑なパターンでのオフセット検出テスト
func TestDiffAnalyzer_ComplexPatterns(t *testing.T) {
	// テストの設定
	cfg := config.NewDefaultConfig()
	cfg.SamplingRate = 1 // ピクセル単位の精度
	cfg.MaxOffset = 40   // より大きなオフセットを探索可能に
	cfg.Threshold = 30   // デフォルト閾値
	cfg.NumCPU = 1
	cfg.FastMode = false // より精密な検索を行う
	analyzer := NewDiffAnalyzer(cfg)

	// サイズ定義
	width, height := 200, 200

	t.Run("simple_square_pattern", func(t *testing.T) {
		// 黒地に白い正方形パターンの画像ペア
		img1 := createTestImage(width, height, color.RGBA{0, 0, 0, 255})
		img2 := createTestImage(width, height, color.RGBA{0, 0, 0, 255})

		// img1の中央に白い正方形を描画
		squareSize := 50
		startX := width/2 - squareSize/2
		startY := height/2 - squareSize/2
		drawRect(img1, startX, startY, squareSize, squareSize, color.RGBA{255, 255, 255, 255})

		// 明確なオフセットをつけてimg2にも同じ正方形を描画
		expectedOffsetX, expectedOffsetY := 25, 25
		drawRect(img2, startX-expectedOffsetX, startY-expectedOffsetY, squareSize, squareSize, color.RGBA{255, 255, 255, 255})

		// オフセット検出のテスト
		foundX, foundY := analyzer.FindBestAlignment(img1, img2)

		// 検証（許容誤差1ピクセル）
		if utils.AbsInt(foundX-expectedOffsetX) > 1 || utils.AbsInt(foundY-expectedOffsetY) > 1 {
			t.Errorf("Simple pattern test failed: expected offset (%d, %d), got (%d, %d)",
				expectedOffsetX, expectedOffsetY, foundX, foundY)
		}

		// スコア比較による検証
		zeroScore := analyzer.calculateSimilarityScore(img1, img2, 0, 0)
		bestScore := analyzer.calculateSimilarityScore(img1, img2, foundX, foundY)
		if bestScore <= zeroScore {
			t.Errorf("Expected better score at correct offset: zero=%.4f, best=%.4f",
				zeroScore, bestScore)
		}
	})

	t.Run("complex_patterns", func(t *testing.T) {
		// 複雑なパターンを持つ画像ペア
		img1 := createTestImage(width, height, color.RGBA{255, 255, 255, 255})
		img2 := createTestImage(width, height, color.RGBA{255, 255, 255, 255})

		// img1に複数の形状を描画
		drawRect(img1, 30, 30, 40, 40, color.RGBA{0, 0, 0, 255})
		drawCircle(img1, width-50, 50, 20, color.RGBA{0, 0, 0, 255})
		drawTriangle(img1, 30, height-30, 40, color.RGBA{0, 0, 0, 255})
		drawCross(img1, width-50, height-50, 15)

		// 明確なオフセットをつけてimg2にも同じパターンを描画
		offsetX, offsetY := -15, 10
		drawRect(img2, 30+offsetX, 30+offsetY, 40, 40, color.RGBA{0, 0, 0, 255})
		drawCircle(img2, width-50+offsetX, 50+offsetY, 20, color.RGBA{0, 0, 0, 255})
		drawTriangle(img2, 30+offsetX, height-30+offsetY, 40, color.RGBA{0, 0, 0, 255})
		drawCross(img2, width-50+offsetX, height-50+offsetY, 15)

		// オフセット検出
		expectedDetectedX := -offsetX // 検出では逆符号になる
		expectedDetectedY := -offsetY // 検出では逆符号になる
		detectedX, detectedY := analyzer.FindBestAlignment(img1, img2)

		// 検証（許容誤差1ピクセル）
		if utils.AbsInt(detectedX-expectedDetectedX) > 1 || utils.AbsInt(detectedY-expectedDetectedY) > 1 {
			t.Errorf("Complex pattern test failed: expected detection (%d, %d), got (%d, %d)",
				expectedDetectedX, expectedDetectedY, detectedX, detectedY)
		}

		// スコア比較による検証
		correctScore := analyzer.calculateSimilarityScore(img1, img2, detectedX, detectedY)
		wrongScore := analyzer.calculateSimilarityScore(img1, img2, -detectedX, -detectedY)
		if wrongScore >= correctScore {
			t.Errorf("Detected offset (%d, %d) should give better score than its inverse: %.4f vs %.4f",
				detectedX, detectedY, correctScore, wrongScore)
		}
	})
}
