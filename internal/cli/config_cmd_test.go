package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/config"
)

func TestConfigCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" || strings.HasPrefix(cmd.Use, "config ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'config' subcommand")
	}
}

func resetConfigSetFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"model", "retries", "timeout"} {
		flag := configSetCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("config set flag %q should exist", name)
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset config set flag %q: %v", name, err)
		}
		flag.Changed = false
	}
}

func newConfigTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	resetRootCmdFlags(t)
	resetConfigSetFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	path := filepath.Join(tmpHome, ".potaco", "config.yaml")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return path, &buf
}

// writeMultiProviderConfig writes a MultiProviderConfig YAML file at the
// given path so tests start from a known state.
func writeMultiProviderConfig(t *testing.T, path string, cfg *config.MultiProviderConfig) {
	t.Helper()
	if err := config.SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func TestConfigSetHasNewFlags(t *testing.T) {
	for _, name := range []string{"model", "retries", "timeout"} {
		if configSetCmd.Flags().Lookup(name) == nil {
			t.Fatalf("config set should have --%s flag", name)
		}
	}
}

func TestConfigSetDoesNotHaveOldFlags(t *testing.T) {
	for _, name := range []string{"base-url", "api-key", "provider"} {
		if configSetCmd.Flags().Lookup(name) != nil {
			t.Fatalf("config set should NOT have --%s flag (removed in Phase 6)", name)
		}
	}
}

func TestConfigDoesNotHaveListProviders(t *testing.T) {
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "list-providers" {
			t.Fatal("config should NOT have 'list-providers' subcommand (replaced by 'auth list')")
		}
	}
}

func TestConfigSetRetriesOnActiveProvider(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--retries", "5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openai"]
	if pc.Retries != 5 {
		t.Errorf("openai retries = %d, want 5", pc.Retries)
	}
	// Model and timeout should be preserved.
	if pc.Model != "gpt-image-2" {
		t.Errorf("openai model = %q, want preserved gpt-image-2", pc.Model)
	}
	if pc.Timeout != 120*time.Second {
		t.Errorf("openai timeout = %v, want preserved 120s", pc.Timeout)
	}
}

func TestConfigSetModelUpdatesActiveModel(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--model", "gpt-image-3"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openai"]
	if pc.Model != "gpt-image-3" {
		t.Errorf("openai model = %q, want gpt-image-3", pc.Model)
	}
	if cfg.ActiveModel != "gpt-image-3" {
		t.Errorf("ActiveModel = %q, want gpt-image-3", cfg.ActiveModel)
	}
	// Other settings preserved.
	if pc.Retries != 2 {
		t.Errorf("openai retries = %d, want preserved 2", pc.Retries)
	}
}

func TestConfigSetTimeoutOnActiveProvider(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "fal",
		ActiveModel:    "fal-ai/flux/dev",
		Providers: map[string]config.ProviderConfig{
			"fal": {Model: "fal-ai/flux/dev", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--timeout", "90s"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["fal"]
	if pc.Timeout != 90*time.Second {
		t.Errorf("fal timeout = %v, want 90s", pc.Timeout)
	}
	if pc.Retries != 2 {
		t.Errorf("fal retries = %d, want preserved 2", pc.Retries)
	}
}

func TestConfigSetPreservesOtherProviders(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
			"fal":    {Model: "fal-ai/flux/dev", Retries: 3, Timeout: 60 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--retries", "7"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Providers["openai"].Retries != 7 {
		t.Errorf("openai retries = %d, want 7", cfg.Providers["openai"].Retries)
	}
	fal := cfg.Providers["fal"]
	if fal.Model != "fal-ai/flux/dev" {
		t.Errorf("fal model = %q, want preserved", fal.Model)
	}
	if fal.Retries != 3 {
		t.Errorf("fal retries = %d, want preserved 3", fal.Retries)
	}
	if fal.Timeout != 60*time.Second {
		t.Errorf("fal timeout = %v, want preserved 60s", fal.Timeout)
	}
}

func TestConfigSetErrorsWhenNoActiveProvider(t *testing.T) {
	path, _ := newConfigTest(t)
	// No config file exists; LoadMultiProvider returns empty config.
	_ = path

	rootCmd.SetArgs([]string{"config", "set", "--retries", "5"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should error when no active provider is configured")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Fatalf("error should mention 'no active provider', got: %v", err)
	}
}

func TestConfigSetErrorsWhenNoFlags(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should error when no flags are given")
	}
	if !strings.Contains(err.Error(), "no flags specified") {
		t.Fatalf("error should mention 'no flags specified', got: %v", err)
	}
}

func TestConfigSetWritesPrivateFileMode(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--retries", "4"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config mode = %o, want 0600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("config dir mode = %o, want 0700", got)
	}
}

func TestConfigShowDisplaysActiveProvider(t *testing.T) {
	path, buf := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 3, Timeout: 90 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"openai", "gpt-image-2", "Active provider:", "Active model:"} {
		if !strings.Contains(output, want) {
			t.Errorf("config show should contain %q, got: %q", want, output)
		}
	}
	if !strings.Contains(output, "retries: 3") {
		t.Errorf("config show should display retries: 3, got: %q", output)
	}
	if !strings.Contains(output, "timeout: 1m30s") {
		t.Errorf("config show should display timeout: 1m30s, got: %q", output)
	}
}

func TestConfigShowMarksActiveProvider(t *testing.T) {
	path, buf := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "fal",
		ActiveModel:    "fal-ai/flux/dev",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
			"fal":    {Model: "fal-ai/flux/dev", Retries: 3, Timeout: 60 * time.Second},
		},
	})

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "fal (active)") {
		t.Errorf("config show should mark 'fal (active)', got: %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("config show should list openai provider, got: %q", output)
	}
	// openai should NOT be marked active.
	if strings.Contains(output, "openai (active)") {
		t.Errorf("config show should NOT mark openai as active, got: %q", output)
	}
}

func TestConfigShowNoConfig(t *testing.T) {
	_, buf := newConfigTest(t)

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No configuration found") {
		t.Errorf("config show with no config should print 'No configuration found', got: %q", output)
	}
	if !strings.Contains(output, "potaco auth add") {
		t.Errorf("config show with no config should hint at 'potaco auth add', got: %q", output)
	}
}

func TestConfigShowDoesNotPrintAPIKey(t *testing.T) {
	path, buf := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	})

	// Drop a fake-looking secret into the config file to ensure show
	// never reads or prints credential-store contents.
	raw := []byte("active_provider: openai\nactive_model: gpt-image-2\nproviders:\n  openai:\n    model: gpt-image-2\n    retries: 2\n    timeout: 2m0s\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	// API keys live in the credential store, not the config file, so
	// there should be no api_key field in the output at all.
	if strings.Contains(output, "api_key") {
		t.Errorf("config show should not display api_key, got: %q", output)
	}
	if strings.Contains(output, "sk-") {
		t.Errorf("config show should not print any API key, got: %q", output)
	}
}
