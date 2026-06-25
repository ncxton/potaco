package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"time"
)

// Resource budgets for image processing. These are variables, not
// constants, so package tests can temporarily lower them without
// allocating huge fixtures.
var (
	maxImageFileBytes   int64 = 128 << 20
	maxImagePixels            = 100_000_000
	maxBase64ImageBytes int64 = 128 << 20
)

// ReadImage reads and decodes an image file, auto-detecting the format
// by magic bytes. Returns the decoded image, the format name ("png" or "jpeg"),
// and an error.
func ReadImage(path string) (image.Image, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}
	if info.Size() > maxImageFileBytes {
		return nil, "", fmt.Errorf("image file too large: %d bytes exceeds limit %d", info.Size(), maxImageFileBytes)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	format := FormatFromBytes(data)
	if format == "" {
		return nil, "", fmt.Errorf("unsupported image format (magic bytes not recognized)")
	}

	if err := validateImageDimensionsFromBytes(data); err != nil {
		return nil, format, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, format, fmt.Errorf("decode image: %w", err)
	}

	return img, format, nil
}

// validateImageDimensions checks that width and height are positive and
// that their pixel count does not exceed maxImagePixels or overflow int.
func validateImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}
	pixels := width * height
	if width != 0 && pixels/width != height {
		return fmt.Errorf("image dimensions overflow: %dx%d", width, height)
	}
	if pixels > maxImagePixels {
		return fmt.Errorf("image too large: %d pixels exceeds limit %d", pixels, maxImagePixels)
	}
	return nil
}

// validateImageDimensionsFromBytes decodes only the image config (header)
// from data and validates the resulting dimensions against the pixel budget.
func validateImageDimensionsFromBytes(data []byte) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// If we cannot decode the config, let the full decode surface
		// the real error rather than a confusing config error.
		return nil
	}
	return validateImageDimensions(cfg.Width, cfg.Height)
}

// WriteImage encodes and writes an image to a file in the specified format.
// Supported formats: "png" (default), "jpeg".
func WriteImage(img image.Image, path string, format string) error {
	normalizedFormat, err := normalizeOutputFormat(format)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	switch normalizedFormat {
	case "jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(f, img)
	}
	return fmt.Errorf("unsupported output format: %s", format)
}

func normalizeOutputFormat(format string) (string, error) {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return "jpeg", nil
	case "png", "":
		return "png", nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// AutoFilename generates a timestamp-based filename: potaco-YYYYMMDD-HHMMSS.png
func AutoFilename() string {
	return "potaco-" + time.Now().Format("20060102-150405") + ".png"
}

// FormatFromBytes detects the image format from the first few bytes.
// Returns "png", "jpeg", or "" if unknown.
func FormatFromBytes(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpeg"
	}
	return ""
}

// DecodeBase64Image decodes a base64-encoded image string into an image.Image.
func DecodeBase64Image(b64 string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	if int64(len(data)) > maxBase64ImageBytes {
		return nil, fmt.Errorf("base64 image too large: %d bytes exceeds limit %d", len(data), maxBase64ImageBytes)
	}

	if err := validateImageDimensionsFromBytes(data); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	return img, nil
}
