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
	resetRootCmdFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
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
	resetRootCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"config", "set",
		"--base-url", "https://api.test.com",
		"--api-key", "sk-test",
		"--model", "m1",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	path := filepath.Join(tmpHome, ".potaco", "config.yaml")
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
