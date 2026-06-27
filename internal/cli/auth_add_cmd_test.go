package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func TestAuthAddNonInteractive(t *testing.T) {
	tmpHome, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test-key", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	key, err := mgr.GetActiveAPIKey()
	if err != nil {
		t.Fatalf("get active API key: %v", err)
	}
	if key != "sk-test-key" {
		t.Errorf("stored key = %q, want 'sk-test-key'", key)
	}

	if _, err := os.Stat(filepath.Join(tmpHome, ".potaco", "config.yaml")); err != nil {
		t.Errorf("config file should exist: %v", err)
	}
}

func TestAuthAddRequiresAPIKey(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add without --api-key should error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "api-key") && !strings.Contains(err.Error(), "API key") {
		t.Errorf("error should mention api-key, got: %v", err)
	}

}

func TestAuthAddUnknownProvider(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "nonexistent", "--api-key", "sk-test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add with unknown provider should error")
	}
	if !strings.Contains(err.Error(), "provider type") {
		t.Fatalf("error = %v, want provider type", err)
	}
}

func TestAuthAddUsesEnvAPIKey(t *testing.T) {
	_, buf := newAuthTest(t)
	t.Setenv("POTACO_API_KEY", "sk-env-key")
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add with env API key error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	key, err := mgr.GetActiveAPIKey()
	if err != nil {
		t.Fatalf("get active API key: %v", err)
	}
	if key != "sk-env-key" {
		t.Errorf("stored key = %q, want 'sk-env-key'", key)
	}
}

func TestAuthAddRequiresProviderArg(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add without provider argument should error")
	}
}

func TestAuthAddWithModelOverride(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test-key", "--force", "--model", "custom-model"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add with model override error: %v", err)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	_, model, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}
	if model != "custom-model" {
		t.Errorf("active model = %q, want 'custom-model'", model)
	}
}
