package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
)

func TestStatusShowsActiveProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Active provider: openai") {
		t.Errorf("status should show active provider, got: %s", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("status should show active model, got: %s", output)
	}
}

func TestStatusShowsConfigPaths(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("status should mention config.yaml, got: %s", output)
	}
	if !strings.Contains(output, "credentials") {
		t.Errorf("status should mention credentials, got: %s", output)
	}
}

func TestStatusShowsConnectedProviders(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("status should list connected providers, got: %s", output)
	}
	if !strings.Contains(output, "configured") {
		t.Errorf("status should show key status, got: %s", output)
	}
	if !strings.Contains(output, "(active)") {
		t.Errorf("status should mark active provider, got: %s", output)
	}
}

func TestStatusNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute should not error with no providers: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No active provider") {
		t.Errorf("status should show no active provider message, got: %s", output)
	}
}

func TestStatusStyledOutputContainsHeaders(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	// Even without a TTY, the output should contain the key information
	if !strings.Contains(output, "Active provider") {
		t.Errorf("status should show active provider header, got: %s", output)
	}
}

func TestStatusJSON(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"active_provider\":") {
		t.Errorf("JSON status should contain active_provider, got: %s", output)
	}
	if !strings.Contains(output, "\"providers\":") {
		t.Errorf("JSON status should contain providers array, got: %s", output)
	}
}

func TestStatusShowsBaseURL(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	path := filepath.Join(tmpHome, ".potaco", "config.yaml")
	cfg := &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", BaseURL: "https://api.example.com/v1", Retries: 2, Timeout: 120},
		},
	}
	if err := config.SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	// Store a credential so the provider is listed as connected.
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}
	// Reset active model to empty to mimic post-refactor auth add.
	cfg.ActiveModel = ""
	cfg.Providers["openai"] = config.ProviderConfig{Model: "", BaseURL: "https://api.example.com/v1", Retries: 2, Timeout: 120}
	if err := config.SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "https://api.example.com/v1") {
		t.Errorf("status should show base URL, got: %s", output)
	}
}

func TestStatusJSONIncludesBaseURL(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	path := filepath.Join(tmpHome, ".potaco", "config.yaml")
	cfg := &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "", BaseURL: "https://api.example.com/v1", Retries: 2, Timeout: 120},
		},
	}
	if err := config.SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"base_url\": \"https://api.example.com/v1\"") {
		t.Errorf("JSON status should include base_url, got: %s", output)
	}
	if !strings.Contains(output, "\"active_model\": \"\"") {
		t.Errorf("JSON status should include empty active_model, got: %s", output)
	}
}
