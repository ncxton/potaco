package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
)

func TestGenDryRunUsesPresetBaseURLWhenProviderTypeIsBuiltIn(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	cfg := &config.MultiProviderConfig{
		ActiveProvider: "local-openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"local-openai": {
				Type:    "openai",
				Model:   "gpt-image-2",
				Retries: 2,
				Timeout: 120,
			},
		},
	}
	configPath := filepath.Join(tmpHome, ".potaco", "config.yaml")
	if err := config.SaveMultiProvider(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	if err := mgr.AddProvider("local-openai", "openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "api.openai.com/v1/images/generations") {
		t.Fatalf("dry-run URL = %s, want OpenAI preset generations endpoint", output)
	}
}
