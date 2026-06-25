package image

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestReadImagePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	os.WriteFile(path, makeTestPNG(t, 8, 8), 0644)

	img, format, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Errorf("dimensions = %dx%d, want 8x8", bounds.Dx(), bounds.Dy())
	}
}

func TestReadImageMissingFile(t *testing.T) {
	_, _, err := ReadImage("/nonexistent/file.png")
	if err == nil {
		t.Fatal("ReadImage should error on missing file")
	}
}

func TestReadImageUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("not an image"), 0644)

	_, _, err := ReadImage(path)
	if err == nil {
		t.Fatal("ReadImage should error on unsupported format")
	}
}

func TestWriteImagePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.png")
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	err := WriteImage(img, path, "png")
	if err != nil {
		t.Fatalf("WriteImage error: %v", err)
	}

	// Verify by reading back
	rImg, format, err := ReadImage(path)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	if rImg.Bounds().Dx() != 4 {
		t.Errorf("width = %d, want 4", rImg.Bounds().Dx())
	}
}

func TestWriteImageUnsupportedFormatDoesNotClobberExistingFile(t *testing.T) {
	// Given
	dir := t.TempDir()
	path := filepath.Join(dir, "output.png")
	original := []byte("keep me")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("write original file: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	// When
	err := WriteImage(img, path, "webp")

	// Then
	if err == nil {
		t.Fatal("WriteImage should reject unsupported output format")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read output file: %v", readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("unsupported format should not clobber existing file, got %q", string(got))
	}
}

func TestAutoFilename(t *testing.T) {
	name := AutoFilename()
	if !strings.HasPrefix(name, "potaco-") {
		t.Errorf("filename should start with 'potaco-', got %q", name)
	}
	if !strings.HasSuffix(name, ".png") {
		t.Errorf("filename should end with '.png', got %q", name)
	}
}

func TestFormatFromBytesPNG(t *testing.T) {
	data := makeTestPNG(t, 4, 4)
	format := FormatFromBytes(data)
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
}

func TestFormatFromBytesJPEG(t *testing.T) {
	// JPEG magic bytes: FF D8 FF
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x00}
	format := FormatFromBytes(data)
	if format != "jpeg" {
		t.Errorf("format = %q, want 'jpeg'", format)
	}
}

func TestFormatFromBytesUnknown(t *testing.T) {
	data := []byte("hello world")
	format := FormatFromBytes(data)
	if format != "" {
		t.Errorf("format = %q, want ''", format)
	}
}

func TestDecodeBase64Image(t *testing.T) {
	pngData := makeTestPNG(t, 4, 4)
	b64 := base64.StdEncoding.EncodeToString(pngData)

	img, err := DecodeBase64Image(b64)
	if err != nil {
		t.Fatalf("DecodeBase64Image error: %v", err)
	}
	if img.Bounds().Dx() != 4 {
		t.Errorf("width = %d, want 4", img.Bounds().Dx())
	}
}

func TestReadImageRejectsFileOverLimit(t *testing.T) {
	oldLimit := maxImageFileBytes
	maxImageFileBytes = 4
	t.Cleanup(func() { maxImageFileBytes = oldLimit })

	path := filepath.Join(t.TempDir(), "large.png")
	if err := os.WriteFile(path, makeTestPNG(t, 2, 2), 0600); err != nil {
		t.Fatalf("write png: %v", err)
	}

	_, _, err := ReadImage(path)
	if err == nil {
		t.Fatal("ReadImage should reject files over maxImageFileBytes")
	}
	if !strings.Contains(err.Error(), "image file too large") {
		t.Fatalf("error should mention image file too large, got: %v", err)
	}
}

func TestDecodeBase64ImageRejectsEncodedDataOverLimit(t *testing.T) {
	oldLimit := maxBase64ImageBytes
	maxBase64ImageBytes = 4
	t.Cleanup(func() { maxBase64ImageBytes = oldLimit })

	b64 := base64.StdEncoding.EncodeToString(makeTestPNG(t, 2, 2))
	_, err := DecodeBase64Image(b64)
	if err == nil {
		t.Fatal("DecodeBase64Image should reject decoded data over maxBase64ImageBytes")
	}
	if !strings.Contains(err.Error(), "base64 image too large") {
		t.Fatalf("error should mention base64 image too large, got: %v", err)
	}
}
