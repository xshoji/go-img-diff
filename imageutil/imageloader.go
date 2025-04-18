package imageutil

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadImage 指定されたパスから画像を読み込む
func LoadImage(filePath *string) (image.Image, error) {
	file, err := os.Open(*filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var img image.Image
	ext := strings.ToLower(filepath.Ext(*filePath))

	switch ext {
	case ".png":
		img, err = png.Decode(file)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// SaveDiffImage 差分画像をファイルに保存する
func SaveDiffImage(img image.Image, outputPath *string) error {
	fmt.Printf("[INFO] Saving diff image to %s...\n", *outputPath)
	startTime := time.Now()

	file, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(*outputPath))

	var saveErr error
	switch ext {
	case ".png":
		fmt.Printf("[INFO] Encoding as PNG...\n")
		saveErr = png.Encode(file, img)
	case ".jpg", ".jpeg":
		fmt.Printf("[INFO] Encoding as JPEG (quality: 90)...\n")
		saveErr = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}

	if saveErr != nil {
		return fmt.Errorf("failed to save image: %w", saveErr)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[INFO] Image saved successfully in %.2f seconds\n", elapsed.Seconds())
	return nil
}
