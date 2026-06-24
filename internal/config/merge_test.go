package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeFlagsOverrideEnv(t *testing.T) {
	t.Setenv("POTACO_BASE_URL", "https://from-env.com")
	t.Setenv("POTACO_MODEL", "env-model")

	opts := MergeOptions{
		BaseURL: ptrString("https://from-flag.com"),
		Model:   ptrString("flag-model"),
		APIKey:  ptrString("sk-flag"),
		Retries: ptrInt(5),
	}

	cfg, err := mergeInternal(opts, nil)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-flag.com" {
		t.Errorf("BaseURL = %q, want flag override", cfg.BaseURL)
	}
	if cfg.Model != "flag-model" {
		t.Errorf("Model = %q, want flag override", cfg.Model)
	}
	if cfg.APIKey != "sk-flag" {
		t.Errorf("APIKey = %q, want flag", cfg.APIKey)
	}
	if cfg.Retries != 5 {
		t.Errorf("Retries = %d, want 5", cfg.Retries)
	}
}

func TestMergeEnvOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `default:
  base_url: "https://from-file.com"
  api_key: "sk-file"
  model: "file-model"
  retries: 1
`
	os.WriteFile(path, []byte(content), 0644)

	t.Setenv("POTACO_BASE_URL", "https://from-env.com")

	opts := MergeOptions{}
	fileCfg, _ := Load(path)

	cfg, err := mergeInternal(opts, fileCfg)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-env.com" {
		t.Errorf("BaseURL = %q, want env override", cfg.BaseURL)
	}
	if cfg.Model != "file-model" {
		t.Errorf("Model = %q, want file value (env not set)", cfg.Model)
	}
	if cfg.APIKey != "sk-file" {
		t.Errorf("APIKey = %q, want file value", cfg.APIKey)
	}
}

func TestMergeFileDefaultsWhenNoFlagsOrEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `default:
  base_url: "https://from-file.com"
  api_key: "sk-file"
  model: "file-model"
  retries: 3
  timeout: "60s"
`
	os.WriteFile(path, []byte(content), 0644)

	opts := MergeOptions{}
	fileCfg, _ := Load(path)

	cfg, err := mergeInternal(opts, fileCfg)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-file.com" {
		t.Errorf("BaseURL = %q, want file value", cfg.BaseURL)
	}
	if cfg.Retries != 3 {
		t.Errorf("Retries = %d, want 3", cfg.Retries)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
}

func TestMergeMissingBaseURLError(t *testing.T) {
	opts := MergeOptions{}
	cfg, err := mergeInternal(opts, nil)
	if err == nil {
		t.Fatal("merge should error when no base_url is configured")
	}
	if cfg != nil {
		t.Fatal("should return nil config on error")
	}
}

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }
