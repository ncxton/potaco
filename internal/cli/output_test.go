package cli

import (
	"strings"
	"testing"
)

func TestFormatResultDefault(t *testing.T) {
	result := OutputResult{
		Paths:   []string{"potaco-20260624-153201.png"},
		Format:  "png",
		Widths:  []int{1024},
		Heights: []int{1024},
		Model:   "dall-e-3",
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
		Model:     "dall-e-3",
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
	if !strings.Contains(output, `"model": "dall-e-3"`) {
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
		Model:     "dall-e-3",
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
