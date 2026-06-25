package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func newAuthTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Ensure POTACO_API_KEY does not leak from the environment so that
	// tests relying on "no api-key" behave deterministically.
	t.Setenv("POTACO_API_KEY", "")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return tmpHome, &buf
}

func resetAuthAddFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"api-key", "force", "model"} {
		flag := authAddCmd.Flags().Lookup(name)
		if flag == nil {
			return // flags not registered yet
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}

func TestAuthCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "auth" || strings.HasPrefix(cmd.Use, "auth ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'auth' subcommand")
	}
}

func TestAuthAddCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "add" || strings.HasPrefix(cmd.Use, "add ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'add' subcommand")
	}
}

func TestAuthAddNonInteractive(t *testing.T) {
	tmpHome, buf := newAuthTest(t)
	// Use --force to skip verification, which would otherwise make a real
	// HTTP call to the provider's API and is not available in tests.
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test-key", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}

	// Verify credential was stored by reading it back through the auth
	// manager, which resolves the same paths via HOME (tmpHome).
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

	// The config file should exist at ~/.potaco/config.yaml (HOME = tmpHome).
	if _, err := os.Stat(filepath.Join(tmpHome, ".potaco", "config.yaml")); err != nil {
		t.Errorf("config file should exist: %v", err)
	}
}

func TestAuthAddRequiresAPIKey(t *testing.T) {
	_, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add without --api-key should error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "api-key") && !strings.Contains(err.Error(), "API key") {
		t.Errorf("error should mention api-key, got: %v", err)
	}

	_ = buf // output may contain error message
}

func TestAuthAddUnknownProvider(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "nonexistent", "--api-key", "sk-test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add with unknown provider should error")
	}
}

func TestAuthAddUsesEnvAPIKey(t *testing.T) {
	_, buf := newAuthTest(t)
	// POTACO_API_KEY was cleared in newAuthTest; set it explicitly here.
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
