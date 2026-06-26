package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
	t.Setenv("POTACO_BASE_URL", "")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return tmpHome, &buf
}

func resetAuthAddFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"api-key", "force", "model", "base-url"} {
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

func TestAuthRemoveCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "remove" || strings.HasPrefix(cmd.Use, "remove ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'remove' subcommand")
	}
}

func TestAuthListCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "list" || strings.HasPrefix(cmd.Use, "list ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'list' subcommand")
	}
}

func TestAuthRemoveCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	// First add a provider
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Now remove it
	rootCmd.SetArgs([]string{"auth", "remove", "openai"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}

	// Verify it is actually removed from config
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	list := mgr.List()
	for _, p := range list {
		if p.Name == "openai" {
			t.Errorf("openai should have been removed from config, but found in list")
		}
	}
}

func TestAuthRemoveAliasRm(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "rm", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth rm error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") {
		t.Errorf("output should mention removal, got: %q", output)
	}
}

func TestAuthRemoveRequiresProviderArg(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove without provider argument should error")
	}
}

func TestAuthRemoveNoArgsNonInteractiveErrors(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove without args in non-interactive mode should error")
	}
	if !strings.Contains(err.Error(), "specify") && !strings.Contains(err.Error(), "Specify") {
		t.Errorf("error should ask to specify a provider, got: %v", err)
	}
}

func TestAuthRemoveUnknownProviderNonInteractive(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove with unknown provider should error")
	}
}

func TestAuthRemoveKnownProviderNonInteractiveStillWorks(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "remove", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}
}

func TestAuthListCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	// Add two providers
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// List
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("list should include openai, got: %q", output)
	}
	if !strings.Contains(output, "fal") {
		t.Errorf("list should include fal, got: %q", output)
	}
}

func TestAuthListAliasLs(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "ls"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth ls error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("ls output should include openai, got: %q", output)
	}
}

func TestAuthListEmpty(t *testing.T) {
	_, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No providers") {
		t.Errorf("empty list output should mention no providers, got: %q", output)
	}
}

func TestAuthListJSON(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list --json error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[") {
		t.Errorf("JSON output should be an array, got: %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("JSON output should include openai, got: %q", output)
	}
}

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

func TestAuthListIncludesCustom(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--base-url", "https://example.com/v1", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add custom: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom") {
		t.Errorf("list should include custom, got: %q", output)
	}
}
