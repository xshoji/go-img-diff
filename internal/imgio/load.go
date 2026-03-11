package imgio

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"

	"github.com/xshoji/go-img-diff/internal/core"
)

// LoadFrame loads an image from the given path and normalizes it into a Frame.
func LoadFrame(path string, logger *slog.Logger) (*core.Frame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", path, err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", path, err)
	}

	frame := core.NewFrame(img)
	logger.Info("loaded image", "path", path, "format", format, "width", frame.W, "height", frame.H)
	return frame, nil
}
