package imageutil

import (
	"image"
	"image/color"
	"testing"

	"github.com/xshoji/go-img-diff/config"
)

func TestCalculateSimilarityScore(t *testing.T) {
	cfg := &config.AppConfig{
		Threshold:    10,
		SamplingRate: 1,
	}

	da := &DiffAnalyzer{cfg: cfg}

	// 完全一致のテスト画像
	imgA := image.NewRGBA(image.Rect(0, 0, 10, 10))
	imgB := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			imgA.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			imgB.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// オフセットなしで完全一致のテスト
	score := da.calculateSimilarityScore(imgA, imgB, 0, 0)
	if score != 1.0 {
		t.Errorf("Expected score to be 1.0, got %f", score)
	}

	// 少し異なる画像 (1ピクセルだけ違う)
	imgC := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			imgC.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	imgC.SetRGBA(0, 0, color.RGBA{0, 0, 0, 255}) // 1ピクセルだけ違う

	score = da.calculateSimilarityScore(imgA, imgC, 0, 0)
	expectedScore := float64(99) / float64(100) // 100ピクセル中99ピクセルが一致
	if score != expectedScore {
		t.Errorf("Expected score to be %f, got %f", expectedScore, score)
	}

	// オフセットありのテスト
	imgD := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			imgD.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	score = da.calculateSimilarityScore(imgA, imgD, 2, 3)
	// imgA(10x10)とimgD(5x5)が(2,3)のオフセットで重なる場合、重なり領域は5x5=25ピクセル
	// imgAの総ピクセル数は10x10=100, imgBの総ピクセル数は5x5=25. totalArea=100
	// overlapArea / totalArea = 25 / 100 = 0.25
	// coverageRatio < 0.5 なので、 baseScore *= coverageRatio * 2.0
	// baseScore = 25 / 25 = 1
	// expectedScore = 1 * 0.25 * 2 = 0.5
	expectedScore = 0.5
	if score != expectedScore {
		t.Errorf("Expected score to be %f, got %f", expectedScore, score)
	}

	// 重ならない画像のテスト
	imgE := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			imgE.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	score = da.calculateSimilarityScore(imgA, imgE, 20, 30)
	if score != 0 {
		t.Errorf("Expected score to be 0, got %f", score)
	}

	// 空の画像のテスト
	imgF := image.NewRGBA(image.Rect(0, 0, 0, 0))
	imgG := image.NewRGBA(image.Rect(0, 0, 0, 0))

	score = da.calculateSimilarityScore(imgF, imgG, 0, 0)
	if score != 0 {
		t.Errorf("Expected score to be 0, got %f", score)
	}
}
