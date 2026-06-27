package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makeMaskPNG(t *testing.T, w, h int, fill color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestRectMask(t *testing.T) {
	mask, err := RectMask(100, 100, 10, 20, 30, 40)
	if err != nil {
		t.Fatalf("RectMask error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("mask dimensions = %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Pixel inside the rect should be transparent (alpha=0, edit)
	_, _, _, a := mask.At(15, 25).RGBA()
	if a != 0 {
		t.Error("pixel inside rect should be transparent (alpha=0)")
	}

	// Pixel outside the rect should be opaque (alpha=255, keep)
	_, _, _, a2 := mask.At(0, 0).RGBA()
	if a2 != 0xffff {
		t.Error("pixel outside rect should be opaque (alpha=255)")
	}
}

func TestRectMaskNegativeOffset(t *testing.T) {
	_, err := RectMask(100, 100, -10, 0, 30, 40)
	if err == nil {
		t.Fatal("RectMask should error on negative x")
	}
}

func TestCircleMask(t *testing.T) {
	mask, err := CircleMask(100, 100, 50, 50, 20)
	if err != nil {
		t.Fatalf("CircleMask error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("mask dimensions = %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Pixel at center should be transparent (alpha=0, edit)
	_, _, _, a := mask.At(50, 50).RGBA()
	if a != 0 {
		t.Error("center pixel should be transparent (alpha=0)")
	}

	// Pixel far from center should be opaque (alpha=255, keep)
	_, _, _, a2 := mask.At(90, 90).RGBA()
	if a2 != 0xffff {
		t.Error("far pixel should be opaque (alpha=255)")
	}
}

func TestCircleMaskNegativeRadius(t *testing.T) {
	_, err := CircleMask(100, 100, 50, 50, -5)
	if err == nil {
		t.Fatal("CircleMask should error on negative radius")
	}
}

func TestLoadMaskFile(t *testing.T) {
	dir := t.TempDir()
	maskPath := filepath.Join(dir, "mask.png")
	// Create a mask where center is white (edit), rest is black (keep).
	// The loader maps white -> transparent, black -> opaque.
	maskImg := image.NewGray(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x > 5 && x < 15 && y > 5 && y < 15 {
				maskImg.SetGray(x, y, color.Gray{0xff})
			} else {
				maskImg.SetGray(x, y, color.Gray{0})
			}
		}
	}
	f, _ := os.Create(maskPath)
	png.Encode(f, maskImg)
	f.Close()

	mask, err := LoadMaskFile(maskPath, 20, 20)
	if err != nil {
		t.Fatalf("LoadMaskFile error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("dimensions = %dx%d, want 20x20", bounds.Dx(), bounds.Dy())
	}

	// White input pixel -> transparent (edit)
	_, _, _, a := mask.At(10, 10).RGBA()
	if a != 0 {
		t.Error("white input pixel should map to transparent (alpha=0)")
	}

	// Black input pixel -> opaque (keep)
	_, _, _, a2 := mask.At(0, 0).RGBA()
	if a2 != 0xffff {
		t.Error("black input pixel should map to opaque (alpha=255)")
	}
}

func TestLoadMaskFileMismatch(t *testing.T) {
	dir := t.TempDir()
	maskPath := filepath.Join(dir, "mask.png")
	maskData := makeMaskPNG(t, 10, 10, color.White)
	os.WriteFile(maskPath, maskData, 0644)

	// Source is 20x20, mask is 10x10 - should scale
	mask, err := LoadMaskFile(maskPath, 20, 20)
	if err != nil {
		t.Fatalf("LoadMaskFile error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("dimensions = %dx%d, want 20x20 (scaled)", bounds.Dx(), bounds.Dy())
	}
}

func TestWriteMask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output-mask.png")
	mask, _ := RectMask(50, 50, 0, 0, 25, 25)

	err := WriteMask(mask, path)
	if err != nil {
		t.Fatalf("WriteMask error: %v", err)
	}

	// Verify it's a valid PNG
	data, _ := os.ReadFile(path)
	format := FormatFromBytes(data)
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
}
