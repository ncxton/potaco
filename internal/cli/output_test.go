package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/spf13/cobra"
)

func tinyPNGBase64(t *testing.T) string {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode tiny png: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func tinyImageResponse(t *testing.T) *adapter.GenerateResponse {
	return &adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: tinyPNGBase64(t)}}}
}

func newOutputTestCommand(stdout, view bool, output, outputFormat string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("stdout", stdout, "")
	cmd.Flags().Bool("view", view, "")
	cmd.Flags().String("output", output, "")
	cmd.Flags().String("output-format", outputFormat, "")
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("json", false, "")
	root.AddCommand(cmd)
	return cmd
}

func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()

	fnErr := fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return raw, fnErr
}

func TestProcessAndOutputStdoutOnlyDoesNotCreateAutoFile(t *testing.T) {
	// Given
	t.Chdir(t.TempDir())
	cmd := newOutputTestCommand(true, false, "", "png")
	var textOut bytes.Buffer
	cmd.SetOut(&textOut)

	// When
	raw, err := captureStdout(t, func() error {
		return processAndOutput(cmd, outputContext{
			resp: tinyImageResponse(t),
		})
	})

	// Then
	if err != nil {
		t.Fatalf("processAndOutput error: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("stdout mode should write raw image bytes")
	}
	if textOut.Len() != 0 {
		t.Fatalf("stdout mode should not write formatted output, got %q", textOut.String())
	}
	matches, err := filepath.Glob("potaco-*.png")
	if err != nil {
		t.Fatalf("glob auto output: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("stdout-only mode should not create an auto file, got %v", matches)
	}
}

func TestProcessAndOutputStdoutWithExplicitOutputWritesBothWithoutText(t *testing.T) {
	// Given
	t.Chdir(t.TempDir())
	cmd := newOutputTestCommand(true, false, "out.png", "png")
	var textOut bytes.Buffer
	cmd.SetOut(&textOut)

	// When
	raw, err := captureStdout(t, func() error {
		return processAndOutput(cmd, outputContext{
			resp: tinyImageResponse(t),
		})
	})

	// Then
	if err != nil {
		t.Fatalf("processAndOutput error: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(raw)); err != nil {
		t.Fatalf("stdout should contain raw PNG bytes: %v", err)
	}
	if _, err := os.Stat("out.png"); err != nil {
		t.Fatalf("explicit output file should exist: %v", err)
	}
	if textOut.Len() != 0 {
		t.Fatalf("stdout plus explicit output should not write formatted output, got %q", textOut.String())
	}
}

func TestProcessAndOutputJSONSuppressesViewPreview(t *testing.T) {
	// Given
	t.Chdir(t.TempDir())
	t.Setenv("TERM", "")
	t.Setenv("TERM_PROGRAM", "")
	cmd := newOutputTestCommand(false, true, "", "png")
	if err := cmd.Root().PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	resp := tinyImageResponse(t)

	// When
	err := processAndOutput(cmd, outputContext{
		resp:   resp,
		model:  "test-model",
		params: map[string]any{"n": 1},
	})

	// Then
	if err != nil {
		t.Fatalf("processAndOutput error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("json plus view should emit JSON only, got %q: %v", out.String(), err)
	}
	if strings.Contains(out.String(), "terminal does not support inline image preview") {
		t.Fatalf("json mode should suppress view preview, got %q", out.String())
	}
}

func TestProcessAndOutputAutoFilenamesAreUniqueForMultipleImages(t *testing.T) {
	// Given
	dir := t.TempDir()
	t.Chdir(dir)
	cmd := newOutputTestCommand(false, false, "", "png")
	var out bytes.Buffer
	cmd.SetOut(&out)
	resp := &adapter.GenerateResponse{
		Data: []adapter.ImageData{
			{B64JSON: tinyPNGBase64(t)},
			{B64JSON: tinyPNGBase64(t)},
		},
	}

	// When
	err := processAndOutput(cmd, outputContext{
		resp:   resp,
		model:  "test-model",
		params: map[string]any{"n": 2},
	})

	// Then
	if err != nil {
		t.Fatalf("processAndOutput error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "potaco-*.png"))
	if err != nil {
		t.Fatalf("glob auto output: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("multi-image auto output should create 2 unique files, got %d files: %v; output %q", len(matches), matches, out.String())
	}
}

func TestFormatResultDefault(t *testing.T) {
	result := OutputResult{
		Paths:   []string{"potaco-20260624-153201.png"},
		Format:  "png",
		Widths:  []int{1024},
		Heights: []int{1024},
		Model:   "gpt-image-2",
		Params:  map[string]any{"size": "1024x1024", "quality": "standard", "n": 1},
	}
	opts := OutputOptions{JSON: false, Stdout: false, View: false}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if !strings.Contains(output, "Saved to:") {
		t.Errorf("default output should contain 'Saved to:', got: %q", output)
	}
	if !strings.Contains(output, "potaco-20260624-153201.png") {
		t.Errorf("output should contain file path, got: %q", output)
	}
	if strings.Contains(output, "{") {
		t.Errorf("default output should not contain JSON, got: %q", output)
	}
}

func TestFormatResultJSON(t *testing.T) {
	result := OutputResult{
		Paths:     []string{"output.png"},
		Format:    "png",
		Widths:    []int{1024},
		Heights:   []int{1024},
		Model:     "gpt-image-2",
		Params:    map[string]any{"size": "1024x1024"},
		LatencyMs: 3420,
	}
	opts := OutputOptions{JSON: true, Stdout: false, View: false}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if !strings.Contains(output, `"path":`) {
		t.Errorf("JSON output should contain 'path' key, got: %q", output)
	}
	if !strings.Contains(output, `"model": "gpt-image-2"`) {
		t.Errorf("JSON output should contain model, got: %q", output)
	}
	if !strings.Contains(output, `"latency_ms": 3420`) {
		t.Errorf("JSON output should contain latency_ms, got: %q", output)
	}
}

func TestFormatResultMultipleImagesJSON(t *testing.T) {
	result := OutputResult{
		Paths:     []string{"img1.png", "img2.png"},
		Format:    "png",
		Widths:    []int{1024, 1024},
		Heights:   []int{1024, 1024},
		Model:     "gpt-image-2",
		Params:    map[string]any{"n": 2},
		LatencyMs: 5000,
	}
	opts := OutputOptions{JSON: true}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// Should be a JSON array
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("multiple images should produce JSON array, got: %q", output)
	}
	if !strings.Contains(output, "img1.png") {
		t.Errorf("output should contain img1.png, got: %q", output)
	}
	if !strings.Contains(output, "img2.png") {
		t.Errorf("output should contain img2.png, got: %q", output)
	}
}

func TestFormatResultStdoutSuppressed(t *testing.T) {
	result := OutputResult{
		Paths: []string{"output.png"},
	}
	opts := OutputOptions{Stdout: true}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// stdout mode: no text/JSON output, the raw bytes are handled separately
	if output != "" {
		t.Errorf("stdout mode should return empty string (no text output), got: %q", output)
	}
}
