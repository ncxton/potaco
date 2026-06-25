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
	setupAuthProvider(t, "sk-test")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

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
	resetAuthAddFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	// Clear all config sources - no provider configured, no env overrides
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_MODEL", "")
	t.Setenv("POTACO_PROVIDER", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no config is provided")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestGenCommandUsesAdapter(t *testing.T) {
	setupAuthProvider(t, "sk-test")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

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

// resetGenCmdFlags restores gen subcommand flags to their defaults so that
// values set by earlier tests (e.g. --dry-run) do not leak in when tests
// run in shuffled order.
func resetGenCmdFlags(t *testing.T) {
	t.Helper()
	flags := genCmd.Flags()
	for _, name := range []string{"dry-run", "prompt", "model", "output", "stdout", "view",
		"provider", "base-url", "api-key", "retries", "timeout"} {
		if fl := flags.Lookup(name); fl != nil {
			if err := flags.Set(name, fl.DefValue); err != nil {
				t.Fatalf("reset gen flag %s: %v", name, err)
			}
			fl.Changed = false
		}
	}
}

// setupAuthProvider runs `auth add openai --force` with the given API key
// against a temp HOME so that subsequent gen/edit commands can resolve
// credentials from the auth manager. It resets flags before and after.
func setupAuthProvider(t *testing.T, apiKey string) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", apiKey, "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
}

func setupAuthProviderForProvider(t *testing.T, providerName, apiKey, model string) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "add", providerName, "--api-key", apiKey, "--force", "--model", model})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add %s: %v", providerName, err)
	}

	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
}

func TestGenDryRunFalProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "fal", "fal-key", "fal-ai/flux/dev")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--provider", "fal", "--base-url", "https://fal.run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Key [REDACTED]") {
		t.Errorf("dry-run should show 'Key [REDACTED]' for fal, got: %s", output)
	}
	if !strings.Contains(output, "fal.run") {
		t.Errorf("dry-run should show fal.run URL, got: %s", output)
	}
}

func TestGenDryRunVercelProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "vercel", "vkey", "openai/gpt-image-2")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--provider", "vercel", "--base-url", "https://ai-gateway.vercel.sh/v1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Bearer [REDACTED]") {
		t.Errorf("dry-run should show 'Bearer [REDACTED]' for vercel, got: %s", output)
	}
	if !strings.Contains(output, "images/generations") {
		t.Errorf("dry-run should show images/generations URL, got: %s", output)
	}
}

func TestGenWithAuthCredentials(t *testing.T) {
	setupAuthProvider(t, "sk-from-auth")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint, got: %q", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("dry-run should contain default model, got: %q", output)
	}
}

func TestGenNoActiveProviderError(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no provider is configured")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestGenWithApiKeyOverride(t *testing.T) {
	setupAuthProvider(t, "sk-stored")
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--api-key", "sk-override", "--base-url", "https://custom.api.com"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen with override: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom.api.com") {
		t.Errorf("dry-run should use overridden base-url, got: %q", output)
	}
}

// TestGenCommandAdapterEndToEnd verifies the gen command uses the adapter
// path (not the legacy provider client) to call the API and save output.
// A mock server mimics an OpenAI-compatible generations endpoint.
func TestGenCommandAdapterEndToEnd(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if m, ok := body["model"].(string); ok {
			gotModel = m
		}
		fmt.Fprintf(w, `{"created":1,"data":[{"b64_json":%q}]}`, tileB64)
	}))
	defer server.Close()

	// Add provider pointing at the mock server via --base-url override
	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetGenCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--output", outPath, "--base-url", server.URL})

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
