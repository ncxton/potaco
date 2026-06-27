package cli

import (
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func TestAuthAddCustomNamedProviderRequiresType(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openrouter", "--api-key", "sk-test", "--force", "--non-interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing provider type")
	}
	if !strings.Contains(err.Error(), "provider type") {
		t.Fatalf("error = %v, want provider type", err)
	}
}

func TestAuthAddCustomNamedProviderWithTypeAndBaseURL(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{
		"auth", "add", "openrouter",
		"--type", "openai-compatible",
		"--api-key", "sk-test",
		"--base-url", "https://openrouter.ai/api/v1",
		"--force",
		"--non-interactive",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add custom named provider with type error: %v", err)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc, ok := cfg.Providers["openrouter"]
	if !ok {
		t.Fatal("openrouter provider should be configured")
	}
	if pc.Type != "openai-compatible" {
		t.Fatalf("provider type = %q, want openai-compatible", pc.Type)
	}
	if pc.BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("base URL = %q, want https://openrouter.ai/api/v1", pc.BaseURL)
	}
}
