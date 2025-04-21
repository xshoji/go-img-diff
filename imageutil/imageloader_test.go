package imageutil

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImage(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// テスト用PNGファイルを作成
	pngPath := filepath.Join(tempDir, "test.png")
	createTestImageFile(t, pngPath, "png")

	// テスト用JPEGファイルを作成
	jpegPath := filepath.Join(tempDir, "test.jpg")
	createTestImageFile(t, jpegPath, "jpeg")

	// 存在しないファイルパス
	nonExistentPath := filepath.Join(tempDir, "non_existent.png")

	// サポートされていない形式のファイル
	unsupportedPath := filepath.Join(tempDir, "test.txt")
	createEmptyFile(t, unsupportedPath)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "正常系: PNG画像を読み込む",
			filePath: pngPath,
			wantErr:  false,
		},
		{
			name:     "正常系: JPEG画像を読み込む",
			filePath: jpegPath,
			wantErr:  false,
		},
		{
			name:     "異常系: 存在しないファイル",
			filePath: nonExistentPath,
			wantErr:  true,
		},
		{
			name:     "異常系: サポートされていないフォーマット",
			filePath: unsupportedPath,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.filePath
			got, err := LoadImage(&path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("LoadImage() got = nil, want image")
			}
		})
	}
}

func TestSaveDiffImage(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// テスト用の画像を作成
	img := generateTestImageData()

	tests := []struct {
		name       string
		outputPath string
		wantErr    bool
	}{
		{
			name:       "正常系: PNG画像を保存",
			outputPath: filepath.Join(tempDir, "output.png"),
			wantErr:    false,
		},
		{
			name:       "正常系: JPEG画像を保存",
			outputPath: filepath.Join(tempDir, "output.jpg"),
			wantErr:    false,
		},
		{
			name:       "異常系: サポートされていないフォーマット",
			outputPath: filepath.Join(tempDir, "output.txt"),
			wantErr:    true,
		},
		{
			name:       "異常系: 不正なパス",
			outputPath: filepath.Join(tempDir, "invalid/path/output.png"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.outputPath
			err := SaveDiffImage(img, &path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveDiffImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// ファイルが実際に作成されたか確認
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("SaveDiffImage() did not create file at %s", path)
				}
			}
		})
	}
}

// テスト用の画像ファイルを作成するヘルパー関数
func createTestImageFile(t *testing.T, path string, format string) {
	img := generateTestImageData()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("テスト用画像ファイルの作成に失敗しました: %v", err)
	}
	defer file.Close()

	switch format {
	case "png":
		encodeErr := png.Encode(file, img)
		if encodeErr != nil {
			t.Fatalf("PNG画像のエンコードに失敗しました: %v", encodeErr)
		}
	case "jpeg":
		encodeErr := jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
		if encodeErr != nil {
			t.Fatalf("JPEG画像のエンコードに失敗しました: %v", encodeErr)
		}
	default:
		t.Fatalf("サポートされていない画像フォーマット: %s", format)
	}
}

// 空のファイルを作成するヘルパー関数
func createEmptyFile(t *testing.T, path string) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("空ファイルの作成に失敗しました: %v", err)
	}
	defer file.Close()
}

// テスト用の画像データを作成するヘルパー関数
func generateTestImageData() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// 画像にいくつかのピクセルを設定
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}

	return img
}
