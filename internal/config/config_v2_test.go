package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMultiProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
active_provider: openai
active_model: gpt-image-2
providers:
  openai:
    model: gpt-image-2
    retries: 3
    timeout: 90
  fal:
    model: fal-ai/flux/dev
    retries: 2
    timeout: 120
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
	if cfg.ActiveModel != "gpt-image-2" {
		t.Errorf("ActiveModel = %q, want 'gpt-image-2'", cfg.ActiveModel)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("Providers len = %d, want 2", len(cfg.Providers))
	}

	openai := cfg.Providers["openai"]
	if openai.Model != "gpt-image-2" {
		t.Errorf("openai model = %q", openai.Model)
	}
	if openai.Retries != 3 {
		t.Errorf("openai retries = %d, want 3", openai.Retries)
	}
	if openai.Timeout != 90 {
		t.Errorf("openai timeout = %v, want 90", openai.Timeout)
	}

	fal := cfg.Providers["fal"]
	if fal.Model != "fal-ai/flux/dev" {
		t.Errorf("fal model = %q", fal.Model)
	}
}

func TestLoadMultiProviderMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("should return empty config, not nil")
	}
	if cfg.ActiveProvider != "" {
		t.Errorf("ActiveProvider should be empty, got %q", cfg.ActiveProvider)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers should be empty, got %d", len(cfg.Providers))
	}
}

func TestSaveMultiProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	}

	if err := SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("SaveMultiProvider: %v", err)
	}

	// Read it back
	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider after save: %v", err)
	}
	if loaded.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q", loaded.ActiveProvider)
	}
	if loaded.Providers["openai"].Model != "gpt-image-2" {
		t.Errorf("model = %q", loaded.Providers["openai"].Model)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("file mode = %o, want 0600", got)
	}
}

func TestSaveMultiProviderPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg1 := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	}
	SaveMultiProvider(path, cfg1)

	// Load, add another provider, save
	cfg2, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg2.Providers["fal"] = ProviderConfig{Model: "fal-ai/flux/dev", Retries: 2, Timeout: 120}
	cfg2.ActiveProvider = "fal"
	cfg2.ActiveModel = "fal-ai/flux/dev"
	SaveMultiProvider(path, cfg2)

	// Read back and verify both providers
	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("Load after second save: %v", err)
	}
	if len(loaded.Providers) != 2 {
		t.Fatalf("Providers len = %d, want 2", len(loaded.Providers))
	}
	if loaded.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", loaded.ActiveProvider)
	}
	if _, ok := loaded.Providers["openai"]; !ok {
		t.Error("openai should still be present")
	}
}
