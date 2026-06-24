package cli

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInfoCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "info" || strings.HasPrefix(cmd.Use, "info ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'info' subcommand")
	}
}

func TestInfoCommandOutput(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	createInfoTestPNG(t, imgPath, 100, 200)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"info", imgPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("info command error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "png") {
		t.Errorf("output should mention format 'png', got: %q", output)
	}
	if !strings.Contains(output, "100x200") {
		t.Errorf("output should contain dimensions 100x200, got: %q", output)
	}
	if !strings.Contains(output, imgPath) {
		t.Errorf("output should contain file path, got: %q", output)
	}
}

func TestInfoCommandJSONOutput(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	createInfoTestPNG(t, imgPath, 64, 32)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"info", "--json", imgPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("info --json command error: %v", err)
	}

	output := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("info --json output should be valid JSON, got parse error: %v, output: %q", err, output)
	}
	if parsed["format"] != "png" {
		t.Errorf("JSON output format should be 'png', got: %v", parsed["format"])
	}
	if parsed["width"] != float64(64) {
		t.Errorf("JSON output width should be 64, got: %v", parsed["width"])
	}
	if parsed["height"] != float64(32) {
		t.Errorf("JSON output height should be 32, got: %v", parsed["height"])
	}
	if parsed["path"] != imgPath {
		t.Errorf("JSON output path should be %q, got: %v", imgPath, parsed["path"])
	}
}

func TestInfoCommandMissingFile(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"info", "/nonexistent/file.png"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("info should error on missing file")
	}
}

func createInfoTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
