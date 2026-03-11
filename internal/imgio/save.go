package imgio

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SaveImage saves an image to the given path. Format is determined by file extension.
func SaveImage(img image.Image, path string, logger *slog.Logger) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", path, err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		err = png.Encode(file, img)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	logger.Info("saved image", "path", path, "format", ext)
	return nil
}
