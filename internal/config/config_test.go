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
  model: "gpt-image-2"
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
	if cfg.Model != "gpt-image-2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-image-2")
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

func TestFromEnvNoVars(t *testing.T) {
	clearPotacoEnv(t)
	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv error: %v", err)
	}
	if cfg != nil {
		t.Errorf("FromEnv = %v, want nil when no env vars set", cfg)
	}
}

func TestFromEnvValidVars(t *testing.T) {
	clearPotacoEnv(t)
	t.Setenv("POTACO_BASE_URL", "https://env.example.com")
	t.Setenv("POTACO_API_KEY", "sk-env")
	t.Setenv("POTACO_MODEL", "env-model")
	t.Setenv("POTACO_RETRIES", "5")
	t.Setenv("POTACO_TIMEOUT", "30s")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv error: %v", err)
	}
	if cfg == nil {
		t.Fatal("FromEnv = nil, want non-nil config")
	}
	if cfg.BaseURL != "https://env.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://env.example.com")
	}
	if cfg.APIKey != "sk-env" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-env")
	}
	if cfg.Model != "env-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "env-model")
	}
	if cfg.Retries != 5 {
		t.Errorf("Retries = %d, want 5", cfg.Retries)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestFromEnvInvalidRetries(t *testing.T) {
	clearPotacoEnv(t)
	t.Setenv("POTACO_BASE_URL", "https://env.example.com")
	t.Setenv("POTACO_RETRIES", "not-a-number")

	cfg, err := FromEnv()
	if err == nil {
		t.Fatal("FromEnv should return error for invalid POTACO_RETRIES")
	}
	if cfg != nil {
		t.Errorf("FromEnv = %v, want nil on parse error", cfg)
	}
	if !strings.Contains(err.Error(), "POTACO_RETRIES") {
		t.Errorf("error = %q, want it to mention POTACO_RETRIES", err.Error())
	}
}

func TestFromEnvInvalidTimeout(t *testing.T) {
	clearPotacoEnv(t)
	t.Setenv("POTACO_BASE_URL", "https://env.example.com")
	t.Setenv("POTACO_TIMEOUT", "not-a-duration")

	cfg, err := FromEnv()
	if err == nil {
		t.Fatal("FromEnv should return error for invalid POTACO_TIMEOUT")
	}
	if cfg != nil {
		t.Errorf("FromEnv = %v, want nil on parse error", cfg)
	}
	if !strings.Contains(err.Error(), "POTACO_TIMEOUT") {
		t.Errorf("error = %q, want it to mention POTACO_TIMEOUT", err.Error())
	}
}

// clearPotacoEnv unsets every POTACO_ var FromEnv reads so tests start
// from a known empty state regardless of the host environment.
func clearPotacoEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"POTACO_BASE_URL",
		"POTACO_API_KEY",
		"POTACO_MODEL",
		"POTACO_RETRIES",
		"POTACO_TIMEOUT",
	} {
		t.Setenv(name, "")
	}
}
