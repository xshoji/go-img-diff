package imgio

import (
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLoadFrame(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	createTestPNG(t, path, 100, 100)

	frame, err := LoadFrame(path, testLogger())
	if err != nil {
		t.Fatalf("LoadFrame failed: %v", err)
	}
	if frame.W != 100 || frame.H != 100 {
		t.Errorf("expected 100x100, got %dx%d", frame.W, frame.H)
	}
}

func TestLoadFrame_NotFound(t *testing.T) {
	_, err := LoadFrame("/nonexistent/file.png", testLogger())
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestSaveImage(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 50, 50))

	t.Run("png", func(t *testing.T) {
		path := filepath.Join(dir, "out.png")
		if err := SaveImage(img, path, testLogger()); err != nil {
			t.Fatalf("SaveImage failed: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file not created: %v", err)
		}
	})

	t.Run("jpg", func(t *testing.T) {
		path := filepath.Join(dir, "out.jpg")
		if err := SaveImage(img, path, testLogger()); err != nil {
			t.Fatalf("SaveImage failed: %v", err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		path := filepath.Join(dir, "out.bmp")
		if err := SaveImage(img, path, testLogger()); err == nil {
			t.Error("expected error for unsupported format")
		}
	})
}

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
