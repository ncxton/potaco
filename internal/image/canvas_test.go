package image

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writePNGToWriter(w io.Writer, img image.Image) {
	png.Encode(w, img)
}

func TestParseExtendSingle(t *testing.T) {
	cfg, err := ParseExtend("top=256")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 256 {
		t.Errorf("Top = %d, want 256", cfg.Top)
	}
	if cfg.Bottom != 0 || cfg.Left != 0 || cfg.Right != 0 {
		t.Errorf("others should be 0, got bottom=%d left=%d right=%d", cfg.Bottom, cfg.Left, cfg.Right)
	}
}

func TestParseExtendMultiple(t *testing.T) {
	cfg, err := ParseExtend("top=256,bottom=128,right=200")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 256 {
		t.Errorf("Top = %d, want 256", cfg.Top)
	}
	if cfg.Bottom != 128 {
		t.Errorf("Bottom = %d, want 128", cfg.Bottom)
	}
	if cfg.Right != 200 {
		t.Errorf("Right = %d, want 200", cfg.Right)
	}
	if cfg.Left != 0 {
		t.Errorf("Left = %d, want 0", cfg.Left)
	}
}

func TestParseExtendAll(t *testing.T) {
	cfg, err := ParseExtend("all=100")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 100 || cfg.Bottom != 100 || cfg.Left != 100 || cfg.Right != 100 {
		t.Errorf("all sides should be 100, got top=%d bottom=%d left=%d right=%d", cfg.Top, cfg.Bottom, cfg.Left, cfg.Right)
	}
}

func TestParseExtendInvalid(t *testing.T) {
	_, err := ParseExtend("top=abc")
	if err == nil {
		t.Fatal("ParseExtend should error on non-numeric value")
	}

	_, err = ParseExtend("invalid=100")
	if err == nil {
		t.Fatal("ParseExtend should error on invalid direction")
	}

	_, err = ParseExtend("")
	if err == nil {
		t.Fatal("ParseExtend should error on empty string")
	}
}

func TestParseExtendRejectsNegativeValues(t *testing.T) {
	cases := []string{"top=-1", "all=-5", "left=10,right=-2"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := ParseExtend(tc)
			if err == nil {
				t.Fatal("ParseExtend should reject negative values")
			}
		})
	}
}

func TestParseExtendRejectsZeroEffect(t *testing.T) {
	cases := []string{"top=0", "all=0", "top=0,bottom=0,left=0,right=0"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := ParseExtend(tc)
			if err == nil {
				t.Fatal("ParseExtend should reject all-zero extend values")
			}
		})
	}
}

func TestExpandCanvas(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	src.Set(50, 50, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	cfg := ExtendConfig{Top: 50, Bottom: 50, Left: 0, Right: 0}
	expanded := ExpandCanvas(src, cfg)

	bounds := expanded.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 200 {
		t.Errorf("dimensions = %dx%d, want 100x200", bounds.Dx(), bounds.Dy())
	}

	// Source pixel should be at offset (left=0, top=50)
	c := expanded.At(50, 100)
	r, g, b, _ := c.RGBA()
	if r == 0 || g != 0 || b != 0 {
		t.Error("source pixel should be preserved at correct offset")
	}
}

func TestExpandMask(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))

	cfg := ExtendConfig{Top: 50, Bottom: 0, Left: 0, Right: 50}
	mask := ExpandMask(src, cfg)

	bounds := mask.Bounds()
	if bounds.Dx() != 150 || bounds.Dy() != 150 {
		t.Errorf("dimensions = %dx%d, want 150x150", bounds.Dx(), bounds.Dy())
	}

	// Pixel in new area (top) should be transparent (alpha=0, edit)
	_, _, _, a := mask.At(50, 10).RGBA()
	if a != 0 {
		t.Error("pixel in new top area should be transparent (alpha=0)")
	}

	// Pixel in new area (right) should be transparent (alpha=0, edit)
	_, _, _, a2 := mask.At(130, 100).RGBA()
	if a2 != 0 {
		t.Error("pixel in new right area should be transparent (alpha=0)")
	}

	// Pixel where original image was should be opaque (alpha=255, keep)
	_, _, _, a3 := mask.At(10, 60).RGBA()
	if a3 != 0xffff {
		t.Error("pixel in original area should be opaque (alpha=255)")
	}
}

func TestPrepareOutpaint(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.png")
	// Create a source image file
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	// We need to write a valid PNG
	{
		f, _ := os.Create(srcPath)
		writePNGToWriter(f, src)
		f.Close()
	}

	cfg := ExtendConfig{Right: 25}
	imgPath, maskPath, err := PrepareOutpaint(srcPath, cfg)
	if err != nil {
		t.Fatalf("PrepareOutpaint error: %v", err)
	}

	// Verify both files exist
	if _, err := os.Stat(imgPath); err != nil {
		t.Errorf("expanded image file missing: %v", err)
	}
	if _, err := os.Stat(maskPath); err != nil {
		t.Errorf("mask file missing: %v", err)
	}

	// Verify expanded image dimensions
	expanded, format, err := ReadImage(imgPath)
	if err != nil {
		t.Fatalf("read expanded image: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	if expanded.Bounds().Dx() != 75 || expanded.Bounds().Dy() != 50 {
		t.Errorf("dimensions = %dx%d, want 75x50", expanded.Bounds().Dx(), expanded.Bounds().Dy())
	}
}

func TestPrepareOutpaintRejectsExpandedImageOverPixelLimit(t *testing.T) {
	oldLimit := maxImagePixels
	maxImagePixels = 10
	t.Cleanup(func() { maxImagePixels = oldLimit })

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.png")
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(f, src); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	f.Close()

	_, _, err = PrepareOutpaint(srcPath, ExtendConfig{Right: 2})
	if err == nil {
		t.Fatal("PrepareOutpaint should reject expanded image over pixel limit")
	}
}
