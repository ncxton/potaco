package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	for _, name := range []string{"base-url", "api-key", "model", "retries", "timeout", "provider"} {
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

func TestConfigSetHasFlags(t *testing.T) {
	if configSetCmd.Flags().Lookup("base-url") == nil {
		t.Fatal("config set should have --base-url flag")
	}
	if configSetCmd.Flags().Lookup("api-key") == nil {
		t.Fatal("config set should have --api-key flag")
	}
	if configSetCmd.Flags().Lookup("model") == nil {
		t.Fatal("config set should have --model flag")
	}
}

func TestConfigListProviders(t *testing.T) {
	_, buf := newConfigTest(t)
	rootCmd.SetArgs([]string{"config", "list-providers"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config list-providers error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should list 'openai' preset, got: %q", output)
	}
	if !strings.Contains(output, "together") {
		t.Errorf("output should list 'together' preset, got: %q", output)
	}
	if !strings.Contains(output, "fal") {
		t.Errorf("output should list 'fal' preset, got: %q", output)
	}
}

func TestConfigSetWritesValidConfigFile(t *testing.T) {
	path, _ := newConfigTest(t)
	rootCmd.SetArgs([]string{"config", "set",
		"--base-url", "https://api.test.com",
		"--api-key", "sk-test",
		"--model", "m1",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config file should be parseable by config.Load: %v", err)
	}
	if cfg.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.test.com")
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test")
	}
	if cfg.Model != "m1" {
		t.Errorf("Model = %q, want %q", cfg.Model, "m1")
	}

	content := string(data)
	if !strings.Contains(content, "base_url") {
		t.Errorf("config file should contain base_url field, got: %q", content)
	}
	if !strings.Contains(content, "api_key") {
		t.Errorf("config file should contain api_key field, got: %q", content)
	}
	if !strings.Contains(content, "model") {
		t.Errorf("config file should contain model field, got: %q", content)
	}
}

func TestConfigSetPartialUpdatePreservesExistingValues(t *testing.T) {
	path, _ := newConfigTest(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	initial := []byte("default:\n  base_url: https://api.example.test\n  api_key: placeholder-key\n  model: old-model\n  retries: 1\n  timeout: 30s\n")
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "set", "--retries", "4"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set partial update error: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.BaseURL != "https://api.example.test" {
		t.Errorf("BaseURL = %q, want preserved base URL", cfg.BaseURL)
	}
	if cfg.APIKey != "placeholder-key" {
		t.Errorf("APIKey was not preserved")
	}
	if cfg.Model != "old-model" {
		t.Errorf("Model = %q, want old-model", cfg.Model)
	}
	if cfg.Retries != 4 {
		t.Errorf("Retries = %d, want 4", cfg.Retries)
	}
	if cfg.Timeout.String() != "30s" {
		t.Errorf("Timeout = %s, want 30s", cfg.Timeout)
	}
}

func TestConfigSetProviderPreservesExplicitValues(t *testing.T) {
	path, _ := newConfigTest(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	initial := []byte("default:\n  base_url: https://api.example.test\n  api_key: placeholder-key\n  model: old-model\n  retries: 1\n  timeout: 30s\n")
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "set", "--provider", "openai", "--model", "explicit-model"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set provider update error: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want openai preset base URL", cfg.BaseURL)
	}
	if cfg.APIKey != "placeholder-key" {
		t.Errorf("APIKey was not preserved")
	}
	if cfg.Model != "explicit-model" {
		t.Errorf("Model = %q, want explicit-model", cfg.Model)
	}
	if cfg.Retries != 1 {
		t.Errorf("Retries = %d, want preserved retries", cfg.Retries)
	}
}

func TestConfigShowRedactsAPIKey(t *testing.T) {
	path, buf := newConfigTest(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("default:\n  base_url: https://api.example.test\n  api_key: placeholder-key\n  model: model-a\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "show"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "placeholder-key") {
		t.Fatalf("config show printed the API key: %q", output)
	}
	if !strings.Contains(output, "api_key: REDACTED") {
		t.Fatalf("config show should include redacted api_key, got: %q", output)
	}
}

func TestConfigSetRejectsSymlinkConfigFile(t *testing.T) {
	path, _ := newConfigTest(t)
	tmpHome := filepath.Dir(filepath.Dir(path))
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	target := filepath.Join(tmpHome, "elsewhere.yaml")
	if err := os.WriteFile(target, []byte("default: {}\n"), 0600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "set", "--model", "model-a"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should reject symlinked config files")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error should mention symlink, got: %v", err)
	}
}

func TestConfigSetWritesPrivateFileMode(t *testing.T) {
	path, _ := newConfigTest(t)
	rootCmd.SetArgs([]string{"config", "set", "--base-url", "https://api.example.test", "--api-key", "placeholder-key", "--model", "model-a"})

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
