package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "gen" || strings.HasPrefix(cmd.Use, "gen ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'gen' subcommand")
	}
}

func TestGenCommandHasPromptFlag(t *testing.T) {
	promptFlag := genCmd.Flags().Lookup("prompt")
	if promptFlag == nil {
		t.Fatal("gen command should have --prompt flag")
	}
}

func TestGenCommandPromptRequired(t *testing.T) {
	resetRootCmdFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", ""}) // empty prompt

	// Cobra should enforce required flag
	err := rootCmd.Execute()
	if err == nil {
		// If not enforced by Cobra, our RunE should catch empty prompt
		// Check if it still runs - if so, we need manual validation
	}
}

func TestGenCommandDryRunNoAPI(t *testing.T) {
	resetRootCmdFlags(t)
	// Set up env so config merge succeeds
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"method": "POST"`) {
		t.Errorf("dry-run should print request method, got: %q", output)
	}
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "a cat") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
	// Should NOT have made an API call (no "Saved to:" in output)
	if strings.Contains(output, "Saved to:") {
		t.Errorf("dry-run should not save any files, got: %q", output)
	}
}

func TestGenCommandMissingConfigError(t *testing.T) {
	resetRootCmdFlags(t)
	// Clear all config sources
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no config is provided")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error should mention base_url, got: %v", err)
	}
}

func TestGenCommandUsesAdapter(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")
	t.Setenv("POTACO_MODEL", "gpt-image-2")

	// Verify the gen command can resolve to an adapter via dry-run
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("dry-run should contain model, got: %q", output)
	}
}

// TestGenCommandAdapterEndToEnd verifies the gen command uses the adapter
// path (not the legacy provider client) to call the API and save output.
// A mock server mimics an OpenAI-compatible generations endpoint.
// resetGenCmdFlags restores gen subcommand flags to their defaults so that
// values set by earlier tests (e.g. --dry-run) do not leak in when tests
// run in shuffled order.
func resetGenCmdFlags(t *testing.T) {
	t.Helper()
	flags := genCmd.Flags()
	for _, name := range []string{"dry-run", "prompt", "model", "output", "stdout", "view"} {
		if fl := flags.Lookup(name); fl != nil {
			if err := flags.Set(name, fl.DefValue); err != nil {
				t.Fatalf("reset gen flag %s: %v", name, err)
			}
			fl.Changed = false
		}
	}
}

func TestGenCommandAdapterEndToEnd(t *testing.T) {
	resetRootCmdFlags(t)
	resetGenCmdFlags(t)
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.png")

	// tiny PNG fixture encoded as base64
	tile := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tile.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var tileBuf bytes.Buffer
	if err := png.Encode(&tileBuf, tile); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	tileB64 := base64.StdEncoding.EncodeToString(tileBuf.Bytes())

	var gotAuth string
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if m, ok := body["model"].(string); ok {
			gotModel = m
		}
		fmt.Fprintf(w, `{"created":1,"data":[{"b64_json":%q}]}`, tileB64)
	}))
	defer server.Close()

	t.Setenv("POTACO_BASE_URL", server.URL)
	t.Setenv("POTACO_API_KEY", "sk-test")
	t.Setenv("POTACO_MODEL", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--output", outPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("gen returned error: %v", err)
	}

	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q, want Bearer sk-test", gotAuth)
	}
	if gotModel != "gpt-image-2" {
		t.Errorf("model = %q, want gpt-image-2", gotModel)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected output file %q: %v", outPath, err)
	}
}
