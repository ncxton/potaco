package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestProcessAndOutputMultiImageOutputKeepsDottedDirectory(t *testing.T) {
	// Given
	dir := filepath.Join(t.TempDir(), "my.dir")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("create dotted dir: %v", err)
	}
	cmd := newOutputTestCommand(false, filepath.Join(dir, "out"), "png")
	resp := &adapter.GenerateResponse{
		Data: []adapter.ImageData{
			{B64JSON: tinyPNGBase64(t)},
			{B64JSON: tinyPNGBase64(t)},
		},
	}

	// When
	err := processAndOutput(cmd, outputContext{resp: resp})

	// Then
	if err != nil {
		t.Fatalf("processAndOutput error: %v", err)
	}
	for _, path := range []string{filepath.Join(dir, "out-0"), filepath.Join(dir, "out-1")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output file %q: %v", path, err)
		}
	}
}
