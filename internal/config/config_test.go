package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := `default:
  base_url: "https://api.openai.com"
  api_key: "sk-test123"
  model: "dall-e-3"
  retries: 3
  timeout: "90s"
`
	path := writeTestConfig(t, dir, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.openai.com")
	}
	if cfg.APIKey != "sk-test123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test123")
	}
	if cfg.Model != "dall-e-3" {
		t.Errorf("Model = %q, want %q", cfg.Model, "dall-e-3")
	}
	if cfg.Retries != 3 {
		t.Errorf("Retries = %d, want 3", cfg.Retries)
	}
	if cfg.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 90s", cfg.Timeout)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load should return error for missing file")
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := "default: [invalid"
	path := writeTestConfig(t, dir, content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load should return error for malformed YAML")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("DefaultConfigPath should not return empty string")
	}
	// Should contain .potaco somewhere in the path
	if !strings.Contains(path, ".potaco") {
		t.Errorf("DefaultConfigPath should contain '.potaco', got %q", path)
	}
}
