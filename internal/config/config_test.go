package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestAutoUpdateDefaultsEnabledWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
active_provider: openai
providers:
  openai:
    model: gpt-image-2
`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if !cfg.AutoUpdateEnabled() {
		t.Fatal("missing auto_update should read as enabled")
	}
}

func TestLoadMultiProviderTypeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &MultiProviderConfig{
		SchemaVersion:  CurrentSchemaVersion,
		ActiveProvider: "openrouter",
		ActiveModel:    "openai/gpt-image-1",
		Providers: map[string]ProviderConfig{
			"openrouter": {
				Type:    "openai-compatible",
				Model:   "openai/gpt-image-1",
				BaseURL: "https://openrouter.ai/api/v1",
				Retries: 2,
				Timeout: 120,
			},
		},
	}

	if err := SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("SaveMultiProvider: %v", err)
	}

	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if loaded.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, CurrentSchemaVersion)
	}
	if got := loaded.Providers["openrouter"].Type; got != "openai-compatible" {
		t.Fatalf("Type = %q, want openai-compatible", got)
	}
}

func TestMigrateConfigFileAddsTypesAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
active_provider: custom
active_model: gpt-image-1
providers:
  openai:
    model: gpt-image-1
    retries: 2
    timeout: 120
  custom:
    base_url: https://api.example.com/v1
    model: gpt-image-1
    retries: 2
    timeout: 120
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 6, 27, 19, 30, 0, 0, time.UTC) }

	changed, backupPath, err := MigrateConfigFile(path, now)
	if err != nil {
		t.Fatalf("MigrateConfigFile: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if backupPath == "" {
		t.Fatal("backupPath is empty")
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
	}

	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if loaded.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, CurrentSchemaVersion)
	}
	if got := loaded.Providers["openai"].Type; got != "openai" {
		t.Fatalf("openai Type = %q, want openai", got)
	}
	if got := loaded.Providers["custom"].Type; got != "openai-compatible" {
		t.Fatalf("custom Type = %q, want openai-compatible", got)
	}
}

func TestMigrateConfigFileNoopWhenCurrentOrMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	now := func() time.Time { return time.Date(2026, 6, 27, 19, 30, 0, 0, time.UTC) }

	changed, backupPath, err := MigrateConfigFile(missing, now)
	if err != nil {
		t.Fatalf("missing config migration: %v", err)
	}
	if changed || backupPath != "" {
		t.Fatalf("missing config changed=%v backup=%q, want false empty", changed, backupPath)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &MultiProviderConfig{
		SchemaVersion:  CurrentSchemaVersion,
		ActiveProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {Type: "openai", Retries: 2, Timeout: 120},
		},
	}
	if err := SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("SaveMultiProvider: %v", err)
	}
	changed, backupPath, err = MigrateConfigFile(path, now)
	if err != nil {
		t.Fatalf("current config migration: %v", err)
	}
	if changed || backupPath != "" {
		t.Fatalf("current config changed=%v backup=%q, want false empty", changed, backupPath)
	}
}
