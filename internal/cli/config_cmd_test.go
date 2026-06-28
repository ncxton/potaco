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
	for _, name := range []string{"model", "base-url", "retries", "timeout"} {
		flag := configSetCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("config set flag %q should exist", name)
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset config set flag %q: %v", name, err)
		}
		flag.Changed = false
	}
	if flag := configSetCmd.Flags().Lookup("help"); flag != nil {
		if err := flag.Value.Set("false"); err != nil {
			t.Fatalf("reset config set help flag: %v", err)
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
	for _, name := range []string{"model", "base-url", "retries", "timeout"} {
		if configSetCmd.Flags().Lookup(name) == nil {
			t.Fatalf("config set should have --%s flag", name)
		}
	}
}

func TestConfigSetHelpHidesLegacyFlags(t *testing.T) {
	_, buf := newConfigTest(t)
	rootCmd.SetArgs([]string{"config", "set", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set --help: %v", err)
	}
	out := buf.String()
	for _, s := range []string{"--model", "--base-url", "--retries", "--timeout"} {
		if strings.Contains(out, s) {
			t.Fatalf("help should not show legacy flag %s:\n%s", s, out)
		}
	}
	if !strings.Contains(out, "set <key> <value>") {
		t.Fatalf("help should show key/value usage:\n%s", out)
	}
}

func TestConfigSetDoesNotHaveOldFlags(t *testing.T) {
	for _, name := range []string{"api-key", "provider"} {
		if configSetCmd.Flags().Lookup(name) != nil {
			t.Fatalf("config set should NOT have --%s flag", name)
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
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
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
	if pc.Timeout != 120 {
		t.Errorf("openai timeout = %v, want preserved 120", pc.Timeout)
	}
}

func TestConfigSetModelUpdatesActiveModel(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
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
			"fal": {Model: "fal-ai/flux/dev", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--timeout", "90"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["fal"]
	if pc.Timeout != 90 {
		t.Errorf("fal timeout = %v, want 90", pc.Timeout)
	}
	if pc.Retries != 2 {
		t.Errorf("fal retries = %d, want preserved 2", pc.Retries)
	}
}

func TestConfigSetTimeoutRejectsSuffix(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--timeout", "90s"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should reject '90s' with suffix")
	}
}

func TestConfigSetPreservesOtherProviders(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
			"fal":    {Model: "fal-ai/flux/dev", Retries: 3, Timeout: 60},
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
	if fal.Timeout != 60 {
		t.Errorf("fal timeout = %v, want preserved 60", fal.Timeout)
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
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should error when no value is given")
	}
	if !strings.Contains(err.Error(), "potaco config set <key> <value>") {
		t.Fatalf("error should mention key/value usage, got: %v", err)
	}
}

func TestConfigSetWritesPrivateFileMode(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
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
			"openai": {Model: "gpt-image-2", Retries: 3, Timeout: 90},
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
	if !strings.Contains(output, "timeout: 90s") {
		t.Errorf("config show should display timeout: 90s, got: %q", output)
	}
}

func TestConfigShowMarksActiveProvider(t *testing.T) {
	path, buf := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "fal",
		ActiveModel:    "fal-ai/flux/dev",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
			"fal":    {Model: "fal-ai/flux/dev", Retries: 3, Timeout: 60},
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
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	// Drop a fake-looking secret into the config file to ensure show
	// never reads or prints credential-store contents.
	raw := []byte("active_provider: openai\nactive_model: gpt-image-2\nproviders:\n  openai:\n    model: gpt-image-2\n    retries: 2\n    timeout: 120\n")
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

func TestConfigSetBaseURL(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--base-url", "https://api.example.com/v1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set --base-url error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openai"]
	if pc.BaseURL != "https://api.example.com/v1" {
		t.Errorf("openai base_url = %q, want https://api.example.com/v1", pc.BaseURL)
	}
	// Other fields preserved.
	if pc.Model != "gpt-image-2" {
		t.Errorf("openai model = %q, want preserved gpt-image-2", pc.Model)
	}
	if pc.Retries != 2 {
		t.Errorf("openai retries = %d, want preserved 2", pc.Retries)
	}
	if pc.Timeout != 120 {
		t.Errorf("openai timeout = %v, want preserved 120", pc.Timeout)
	}
}

func TestConfigSetBaseURLTrimsTrailingSlash(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--base-url", "https://api.example.com/v1/"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set --base-url error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Providers["openai"].BaseURL; got != "https://api.example.com/v1" {
		t.Errorf("base_url = %q, want trailing slash trimmed", got)
	}
}

func TestConfigSetKeyValueActiveProviderFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		value string
		check func(t *testing.T, cfg *config.MultiProviderConfig)
	}{
		{
			name:  "model",
			key:   "model",
			value: "gpt-image-3",
			check: func(t *testing.T, cfg *config.MultiProviderConfig) {
				t.Helper()
				if got := cfg.Providers["openai"].Model; got != "gpt-image-3" {
					t.Fatalf("model = %q, want gpt-image-3", got)
				}
				if got := cfg.ActiveModel; got != "gpt-image-3" {
					t.Fatalf("ActiveModel = %q, want gpt-image-3", got)
				}
			},
		},
		{
			name:  "base_url",
			key:   "base_url",
			value: "https://api.example.com/v1/",
			check: func(t *testing.T, cfg *config.MultiProviderConfig) {
				t.Helper()
				if got := cfg.Providers["openai"].BaseURL; got != "https://api.example.com/v1" {
					t.Fatalf("base_url = %q, want trimmed URL", got)
				}
			},
		},
		{
			name:  "retries",
			key:   "retries",
			value: "6",
			check: func(t *testing.T, cfg *config.MultiProviderConfig) {
				t.Helper()
				if got := cfg.Providers["openai"].Retries; got != 6 {
					t.Fatalf("retries = %d, want 6", got)
				}
			},
		},
		{
			name:  "timeout",
			key:   "timeout",
			value: "45",
			check: func(t *testing.T, cfg *config.MultiProviderConfig) {
				t.Helper()
				if got := cfg.Providers["openai"].Timeout; got != 45 {
					t.Fatalf("timeout = %d, want 45", got)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, _ := newConfigTest(t)
			writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
				ActiveProvider: "openai",
				ActiveModel:    "gpt-image-2",
				Providers: map[string]config.ProviderConfig{
					"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
				},
			})

			rootCmd.SetArgs([]string{"config", "set", tc.key, tc.value})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("config set %s error: %v", tc.key, err)
			}

			cfg, err := config.LoadMultiProvider(path)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			tc.check(t, cfg)
		})
	}
}

func TestConfigSetKeyValueExplicitProviderFields(t *testing.T) {
	for _, tc := range []struct {
		key   string
		value string
		check func(t *testing.T, pc config.ProviderConfig)
	}{
		{
			key:   "providers.vercel.model",
			value: "openai/gpt-image-2",
			check: func(t *testing.T, pc config.ProviderConfig) {
				t.Helper()
				if pc.Model != "openai/gpt-image-2" {
					t.Fatalf("model = %q, want openai/gpt-image-2", pc.Model)
				}
			},
		},
		{
			key:   "providers.vercel.base_url",
			value: "https://gateway.ai/v1/",
			check: func(t *testing.T, pc config.ProviderConfig) {
				t.Helper()
				if pc.BaseURL != "https://gateway.ai/v1" {
					t.Fatalf("base_url = %q, want trimmed URL", pc.BaseURL)
				}
			},
		},
		{
			key:   "providers.vercel.retries",
			value: "5",
			check: func(t *testing.T, pc config.ProviderConfig) {
				t.Helper()
				if pc.Retries != 5 {
					t.Fatalf("retries = %d, want 5", pc.Retries)
				}
			},
		},
		{
			key:   "providers.vercel.timeout",
			value: "75",
			check: func(t *testing.T, pc config.ProviderConfig) {
				t.Helper()
				if pc.Timeout != 75 {
					t.Fatalf("timeout = %d, want 75", pc.Timeout)
				}
			},
		},
	} {
		t.Run(tc.key, func(t *testing.T) {
			path, _ := newConfigTest(t)
			writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
				ActiveProvider: "openai",
				ActiveModel:    "gpt-image-2",
				Providers: map[string]config.ProviderConfig{
					"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
					"vercel": {Model: "old", Retries: 1, Timeout: 30},
				},
			})

			rootCmd.SetArgs([]string{"config", "set", tc.key, tc.value})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("config set %s error: %v", tc.key, err)
			}

			cfg, err := config.LoadMultiProvider(path)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			tc.check(t, cfg.Providers["vercel"])
		})
	}
}

func TestConfigSetProviderModelEditCapability(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {
				Model: "gpt-image-2",
				Models: map[string]config.ModelConfig{
					"gpt-image-2": {Edit: false},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "providers.openai.models.gpt-image-2.edit", "true"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set model edit error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Providers["openai"].Models["gpt-image-2"].Edit {
		t.Fatal("model edit capability = false, want true")
	}
}

func TestConfigSetActiveModelEditCapability(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2"},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "model.edit", "true"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set active model edit error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Providers["openai"].Models["gpt-image-2"].Edit {
		t.Fatal("active model edit capability = false, want true")
	}
}

func TestConfigSetAutoUpdate(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2"},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "auto_update", "false"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set auto_update error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AutoUpdate == nil || *cfg.AutoUpdate {
		t.Fatalf("AutoUpdate = %v, want explicit false", cfg.AutoUpdate)
	}
	if cfg.AutoUpdateEnabled() {
		t.Fatal("AutoUpdateEnabled = true, want false")
	}
}

func TestConfigSetUnknownKeyErrors(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2"},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "unknown", "value"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should reject unknown keys")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Fatalf("error should mention unknown config key, got: %v", err)
	}
}

func TestConfigSetExplicitProviderRequiresConfiguredProvider(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2"},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "providers.openrouter.model", "openai/gpt-image-2"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should reject unknown provider entries")
	}
	if !strings.Contains(err.Error(), `provider "openrouter" is not configured`) {
		t.Fatalf("error should mention missing provider, got: %v", err)
	}
}

func TestConfigSetOtherFieldsPreservesBaseURL(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", BaseURL: "https://api.example.com/v1", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set", "--model", "gpt-image-3"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set --model error: %v", err)
	}

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openai"]
	if pc.Model != "gpt-image-3" {
		t.Errorf("openai model = %q, want gpt-image-3", pc.Model)
	}
	if pc.BaseURL != "https://api.example.com/v1" {
		t.Errorf("openai base_url = %q, want preserved https://api.example.com/v1", pc.BaseURL)
	}
}

func TestConfigSetNoFlagsIncludesBaseURL(t *testing.T) {
	path, _ := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120},
		},
	})

	rootCmd.SetArgs([]string{"config", "set"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should error when no value is given")
	}
	if !strings.Contains(err.Error(), "potaco config set <key> <value>") {
		t.Errorf("error should mention key/value usage, got: %v", err)
	}
	_ = path
}

func TestConfigShowDisplaysBaseURL(t *testing.T) {
	path, buf := newConfigTest(t)
	writeMultiProviderConfig(t, path, &config.MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:    "gpt-image-2",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2", BaseURL: "https://api.example.com/v1", Retries: 3, Timeout: 90},
		},
	})

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "base_url: https://api.example.com/v1") {
		t.Errorf("config show should display base_url, got: %q", output)
	}
}

func TestConfigShowOldConfigWithoutBaseURL(t *testing.T) {
	path, buf := newConfigTest(t)
	// Old-format config without base_url.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	raw := []byte("active_provider: openai\nactive_model: gpt-image-2\nproviders:\n  openai:\n    model: gpt-image-2\n    retries: 2\n    timeout: 120\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config show error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("config show should display provider, got: %q", output)
	}
	if !strings.Contains(output, "base_url: default") {
		t.Errorf("config show should display default base_url for old config, got: %q", output)
	}
}
