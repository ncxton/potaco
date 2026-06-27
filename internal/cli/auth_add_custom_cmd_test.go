package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/config"
)

func TestAuthAddCustomRequiresBaseURL(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--force"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add custom without base-url should error")
	}
	if !strings.Contains(err.Error(), "base URL") && !strings.Contains(err.Error(), "base-url") {
		t.Errorf("error should mention base URL, got: %v", err)
	}
}

func TestAuthAddCustomWithBaseURLForce(t *testing.T) {
	tmpHome, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--base-url", "https://example.com/v1", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add custom with base-url --force error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom") {
		t.Errorf("output should mention custom, got: %q", output)
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".potaco", "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "base_url: https://example.com/v1") {
		t.Errorf("config should contain base_url, got: %s", string(data))
	}

	cfg, err := config.LoadMultiProvider(filepath.Join(tmpHome, ".potaco", "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Providers["custom"].Type; got != "openai-compatible" {
		t.Fatalf("custom type = %q, want openai-compatible", got)
	}
}

func TestAuthAddBuiltInTypeWithBaseURLPersistsBaseURL(t *testing.T) {
	tmpHome, _ := newAuthTest(t)
	rootCmd.SetArgs([]string{
		"auth", "add", "local-openai",
		"--type", "openai",
		"--api-key", "sk-test",
		"--base-url", "https://local.example.com/v1",
		"--force",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add named openai with base-url --force error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(filepath.Join(tmpHome, ".potaco", "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["local-openai"]
	if pc.Type != "openai" {
		t.Fatalf("type = %q, want openai", pc.Type)
	}
	if pc.BaseURL != "https://local.example.com/v1" {
		t.Fatalf("base URL = %q, want https://local.example.com/v1", pc.BaseURL)
	}
}

func TestAuthAddCustomWithEnvBaseURL(t *testing.T) {
	tmpHome, _ := newAuthTest(t)
	t.Setenv("POTACO_BASE_URL", "https://env.example.com/v1")
	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add custom with env base-url error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".potaco", "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "base_url: https://env.example.com/v1") {
		t.Errorf("config should contain env base_url, got: %s", string(data))
	}
}

func TestAuthAddCustomVerifyUsesBaseURL(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v1/models" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": [{"id": "test-model"}]}`))
	}))
	defer srv.Close()

	tmpHome, _ := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--base-url", srv.URL})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add custom with verify error: %v", err)
	}
	if calls == 0 {
		t.Error("verification should have called the mock server")
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".potaco", "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "base_url:") {
		t.Errorf("config should contain base_url, got: %s", string(data))
	}
}
