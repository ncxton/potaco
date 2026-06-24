package image

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

// ExtendConfig holds the pixel extension values for each direction.
type ExtendConfig struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// ParseExtend parses a string like "top=256,bottom=128" or "all=100"
// into an ExtendConfig. Returns an error on invalid format.
func ParseExtend(s string) (ExtendConfig, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ExtendConfig{}, fmt.Errorf("empty extend value")
	}

	cfg := ExtendConfig{}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return ExtendConfig{}, fmt.Errorf("invalid extend part: %q (expected key=value)", part)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		n, err := strconv.Atoi(val)
		if err != nil {
			return ExtendConfig{}, fmt.Errorf("invalid extend value %q: %w", val, err)
		}

		switch key {
		case "top":
			cfg.Top = n
		case "bottom":
			cfg.Bottom = n
		case "left":
			cfg.Left = n
		case "right":
			cfg.Right = n
		case "all":
			cfg.Top = n
			cfg.Bottom = n
			cfg.Left = n
			cfg.Right = n
		default:
			return ExtendConfig{}, fmt.Errorf("invalid extend direction: %q (use top, bottom, left, right, or all)", key)
		}
	}

	return cfg, nil
}

// ExpandCanvas creates a new canvas of size (srcW+left+right, srcH+top+bottom),
// pastes the source image at offset (left, top), and fills the new areas
// with a neutral gray (128).
func ExpandCanvas(src image.Image, cfg ExtendConfig) image.Image {
	bounds := src.Bounds()
	newW := bounds.Dx() + cfg.Left + cfg.Right
	newH := bounds.Dy() + cfg.Top + cfg.Bottom

	canvas := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// Fill with neutral gray
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			canvas.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	// Paste source at offset (left, top)
	draw.Draw(canvas, image.Rect(cfg.Left, cfg.Top, cfg.Left+bounds.Dx(), cfg.Top+bounds.Dy()), src, bounds.Min, draw.Src)

	return canvas
}

// ExpandMask creates a mask matching the expanded canvas: white where
// new pixels are (the extended areas), black where the original image is.
func ExpandMask(src image.Image, cfg ExtendConfig) image.Image {
	bounds := src.Bounds()
	newW := bounds.Dx() + cfg.Left + cfg.Right
	newH := bounds.Dy() + cfg.Top + cfg.Bottom

	mask := image.NewGray(image.Rect(0, 0, newW, newH))
	// Default is all black (zero value)

	// Mark new areas as white
	// Top strip
	if cfg.Top > 0 {
		for y := 0; y < cfg.Top; y++ {
			for x := 0; x < newW; x++ {
				mask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}
	// Bottom strip
	if cfg.Bottom > 0 {
		for y := cfg.Top + bounds.Dy(); y < newH; y++ {
			for x := 0; x < newW; x++ {
				mask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}
	// Left strip
	if cfg.Left > 0 {
		for y := cfg.Top; y < cfg.Top+bounds.Dy(); y++ {
			for x := 0; x < cfg.Left; x++ {
				mask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}
	// Right strip
	if cfg.Right > 0 {
		for y := cfg.Top; y < cfg.Top+bounds.Dy(); y++ {
			for x := cfg.Left + bounds.Dx(); x < newW; x++ {
				mask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}

	return mask
}

// PrepareOutpaint loads a source image, expands the canvas, generates the
// mask, and writes both to temporary PNG files. Returns the paths to the
// expanded image and mask files.
func PrepareOutpaint(srcPath string, cfg ExtendConfig) (string, string, error) {
	src, _, err := ReadImage(srcPath)
	if err != nil {
		return "", "", fmt.Errorf("read source: %w", err)
	}

	expanded := ExpandCanvas(src, cfg)
	mask := ExpandMask(src, cfg)

	// Write to temp files
	dir, err := os.MkdirTemp("", "potaco-outpaint-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}

	imgPath := filepath.Join(dir, "expanded.png")
	maskPath := filepath.Join(dir, "mask.png")

	if err := WriteImage(expanded, imgPath, "png"); err != nil {
		return "", "", fmt.Errorf("write expanded image: %w", err)
	}
	if err := WriteMask(mask, maskPath); err != nil {
		return "", "", fmt.Errorf("write mask: %w", err)
	}

	return imgPath, maskPath, nil
}
