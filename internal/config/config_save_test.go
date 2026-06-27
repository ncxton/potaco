package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("file mode = %o, want 0600", got)
	}
}

func TestSaveMultiProviderConfigAddsCurrentSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {Type: "openai", Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	}

	if err := SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("SaveMultiProvider: %v", err)
	}

	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider after save: %v", err)
	}
	if loaded.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, CurrentSchemaVersion)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !bytes.Contains(data, []byte("schema_version:")) {
		t.Fatalf("saved config does not contain schema_version field\n%s", string(data))
	}
}

func TestSaveMultiProviderBaseURLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {
				Model:   "gpt-image-2",
				BaseURL: "https://api.example.com/v1",
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
		t.Fatalf("LoadMultiProvider after save: %v", err)
	}

	openai, ok := loaded.Providers["openai"]
	if !ok {
		t.Fatal("openai provider missing after load")
	}
	if openai.BaseURL != "https://api.example.com/v1" {
		t.Errorf("BaseURL = %q, want %q", openai.BaseURL, "https://api.example.com/v1")
	}
	if openai.Model != "gpt-image-2" {
		t.Errorf("Model = %q, want %q", openai.Model, "gpt-image-2")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !bytes.Contains(data, []byte("base_url:")) {
		t.Errorf("saved config does not contain base_url field\n%s", string(data))
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
	if err := SaveMultiProvider(path, cfg1); err != nil {
		t.Fatalf("SaveMultiProvider first: %v", err)
	}

	cfg2, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg2.Providers["fal"] = ProviderConfig{Model: "fal-ai/flux/dev", Retries: 2, Timeout: 120}
	cfg2.ActiveProvider = "fal"
	cfg2.ActiveModel = "fal-ai/flux/dev"
	if err := SaveMultiProvider(path, cfg2); err != nil {
		t.Fatalf("SaveMultiProvider second: %v", err)
	}

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
