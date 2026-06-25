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

// ReadImage reads and decodes an image file, auto-detecting the format
// by magic bytes. Returns the decoded image, the format name ("png" or "jpeg"),
// and an error.
func ReadImage(path string) (image.Image, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	format := FormatFromBytes(data)
	if format == "" {
		return nil, "", fmt.Errorf("unsupported image format (magic bytes not recognized)")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, format, fmt.Errorf("decode image: %w", err)
	}

	return img, format, nil
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

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	return img, nil
}
