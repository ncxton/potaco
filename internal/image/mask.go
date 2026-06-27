package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// RectMask creates a mask image of the given source dimensions with a
// transparent rectangle at (x, y) of size (w, h) on an opaque background.
// Transparent areas (alpha=0) indicate where the API should regenerate
// content; opaque areas (alpha=255) are kept as-is.
func RectMask(sourceWidth, sourceHeight, x, y, w, h int) (image.Image, error) {
	if x < 0 || y < 0 {
		return nil, fmt.Errorf("rect offset cannot be negative: x=%d y=%d", x, y)
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("rect dimensions must be positive: w=%d h=%d", w, h)
	}

	mask := image.NewRGBA(image.Rect(0, 0, sourceWidth, sourceHeight))
	opaque := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	transparent := color.RGBA{R: 0, G: 0, B: 0, A: 0}

	for row := 0; row < sourceHeight; row++ {
		for col := 0; col < sourceWidth; col++ {
			if col >= x && col < x+w && row >= y && row < y+h {
				mask.SetRGBA(col, row, transparent)
			} else {
				mask.SetRGBA(col, row, opaque)
			}
		}
	}
	return mask, nil
}

// CircleMask creates a mask image of the given source dimensions with a
// filled transparent circle centered at (cx, cy) with radius r on an
// opaque background. Transparent areas (alpha=0) indicate where the API
// should regenerate content; opaque areas (alpha=255) are kept as-is.
func CircleMask(sourceWidth, sourceHeight, cx, cy, r int) (image.Image, error) {
	if r <= 0 {
		return nil, fmt.Errorf("radius must be positive: r=%d", r)
	}

	mask := image.NewRGBA(image.Rect(0, 0, sourceWidth, sourceHeight))
	opaque := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	transparent := color.RGBA{R: 0, G: 0, B: 0, A: 0}

	r2 := r * r
	yStart := max(cy-r, 0)
	yEnd := min(cy+r, sourceHeight)
	xStart := max(cx-r, 0)
	xEnd := min(cx+r, sourceWidth)
	for y := 0; y < sourceHeight; y++ {
		for x := 0; x < sourceWidth; x++ {
			if x >= xStart && x < xEnd && y >= yStart && y < yEnd {
				dx := x - cx
				dy := y - cy
				if dx*dx+dy*dy <= r2 {
					mask.SetRGBA(x, y, transparent)
					continue
				}
			}
			mask.SetRGBA(x, y, opaque)
		}
	}
	return mask, nil
}

// LoadMaskFile loads a mask from a file path and ensures it matches the
// source image dimensions. If dimensions differ, the mask is scaled.
// The result is an RGBA image where transparent pixels (alpha=0) indicate
// areas to regenerate and opaque pixels (alpha=255) indicate areas to keep.
func LoadMaskFile(path string, sourceWidth, sourceHeight int) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mask file: %w", err)
	}

	if int64(len(data)) > maxImageFileBytes {
		return nil, fmt.Errorf("mask file too large: %d bytes exceeds limit %d", len(data), maxImageFileBytes)
	}
	if err := validateImageDimensionsFromBytes(data); err != nil {
		return nil, err
	}

	rawImg, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode mask: %w", err)
	}

	bounds := rawImg.Bounds()
	rgbaMask := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := rawImg.At(x, y).RGBA()
			if a == 0 {
				rgbaMask.SetRGBA(x, y, color.RGBA{A: 0})
			} else {
				r, g, b, _ := rawImg.At(x, y).RGBA()
				if r > 0 || g > 0 || b > 0 {
					rgbaMask.SetRGBA(x, y, color.RGBA{A: 0})
				} else {
					rgbaMask.SetRGBA(x, y, color.RGBA{A: 255})
				}
			}
		}
	}

	srcBounds := image.Rect(0, 0, sourceWidth, sourceHeight)
	if bounds.Dx() != sourceWidth || bounds.Dy() != sourceHeight {
		scaled := image.NewRGBA(srcBounds)
		draw.NearestNeighbor.Scale(scaled, srcBounds, rgbaMask, bounds, draw.Over, nil)
		return scaled, nil
	}

	return rgbaMask, nil
}

// WriteMask writes a mask image as a PNG file to the given path.
func WriteMask(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create mask file: %w", err)
	}
	defer f.Close()
	return png.Encode(f, img)
}
