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
// white rectangle at (x, y) of size (w, h) on a black background.
func RectMask(sourceWidth, sourceHeight, x, y, w, h int) (image.Image, error) {
	if x < 0 || y < 0 {
		return nil, fmt.Errorf("rect offset cannot be negative: x=%d y=%d", x, y)
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("rect dimensions must be positive: w=%d h=%d", w, h)
	}

	mask := image.NewGray(image.Rect(0, 0, sourceWidth, sourceHeight))
	for row := y; row < y+h && row < sourceHeight; row++ {
		for col := x; col < x+w && col < sourceWidth; col++ {
			mask.SetGray(col, row, color.Gray{0xff})
		}
	}
	return mask, nil
}

// CircleMask creates a mask image of the given source dimensions with a
// filled white circle centered at (cx, cy) with radius r on a black background.
func CircleMask(sourceWidth, sourceHeight, cx, cy, r int) (image.Image, error) {
	if r <= 0 {
		return nil, fmt.Errorf("radius must be positive: r=%d", r)
	}

	mask := image.NewGray(image.Rect(0, 0, sourceWidth, sourceHeight))
	r2 := r * r
	yStart := max(cy-r, 0)
	yEnd := min(cy+r, sourceHeight)
	xStart := max(cx-r, 0)
	xEnd := min(cx+r, sourceWidth)
	for y := yStart; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				mask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}
	return mask, nil
}

// LoadMaskFile loads a mask from a file path and ensures it matches the
// source image dimensions. If dimensions differ, the mask is scaled.
// Any non-black pixel becomes white; black stays black.
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
	grayMask := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rawImg.At(x, y)
			r, g, b, _ := c.RGBA()
			// If any channel is non-zero, treat as white
			if r > 0 || g > 0 || b > 0 {
				grayMask.SetGray(x, y, color.Gray{0xff})
			}
		}
	}

	srcBounds := image.Rect(0, 0, sourceWidth, sourceHeight)
	if bounds.Dx() != sourceWidth || bounds.Dy() != sourceHeight {
		scaled := image.NewGray(srcBounds)
		draw.NearestNeighbor.Scale(scaled, srcBounds, grayMask, bounds, draw.Over, nil)
		return scaled, nil
	}

	return grayMask, nil
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
