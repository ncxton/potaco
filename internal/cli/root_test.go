package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

const legacyCustomProviderConfigYAML = `
active_provider: custom
providers:
  custom:
    base_url: https://api.example.com/v1
    model: gpt-image-1
    retries: 2
    timeout: 120
`

// resetRootCmdFlags restores persistent flags on the shared global rootCmd to
// their default values. Tests that dispatch rootCmd.SetArgs must call this
// first so that flags left set by earlier tests (e.g. --json) do not leak in
// when -shuffle=on reorders test execution.
func resetRootCmdFlags(t *testing.T) {
	t.Helper()
	flags := rootCmd.PersistentFlags()
	for _, name := range []string{"json", "verbose", "non-interactive"} {
		if err := flags.Set(name, "false"); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
	}
	// Reset local flags that may leak between tests sharing the global rootCmd.
	local := rootCmd.Flags()
	for _, name := range []string{"help", "version"} {
		if f := local.Lookup(name); f != nil {
			if err := local.Set(name, "false"); err != nil {
				t.Fatalf("reset %s flag: %v", name, err)
			}
		}
	}
	tui.SetNonInteractive(false)
}

func TestRootCommandPrintsHelp(t *testing.T) {
	resetRootCmdFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd --help returned error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "potaco") {
		t.Errorf("help output should contain 'potaco', got: %s", output)
	}
}

func TestRootCommandHasJsonFlag(t *testing.T) {
	jsonFlag := rootCmd.PersistentFlags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("root command should have persistent --json flag")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("json flag default should be false, got %s", jsonFlag.DefValue)
	}
}

func TestRootCommandHasVerboseFlag(t *testing.T) {
	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("root command should have persistent --verbose flag")
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("verbose flag default should be false, got %s", verboseFlag.DefValue)
	}
}

func TestRootCommandHasNonInteractiveFlag(t *testing.T) {
	resetRootCmdFlags(t)
	niFlag := rootCmd.PersistentFlags().Lookup("non-interactive")
	if niFlag == nil {
		t.Fatal("root command should have persistent --non-interactive flag")
	}
	if niFlag.DefValue != "false" {
		t.Errorf("non-interactive flag default should be false, got %s", niFlag.DefValue)
	}
}

func TestNonInteractiveFlagWiresToTUI(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() { tui.SetNonInteractive(false) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--non-interactive"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if tui.IsInteractive() {
		t.Error("IsInteractive() should return false after --non-interactive is set")
	}
}

func TestNonInteractiveFlagDefaultsToInteractive(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() { tui.SetNonInteractive(false) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// In a test environment (no TTY), IsInteractive should return false
	if tui.IsInteractive() {
		t.Error("IsInteractive() should return false in test environment (no TTY)")
	}
}

func TestRootCommandMigratesConfigOnStartup(t *testing.T) {
	resetRootCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Cleanup(func() { tui.SetNonInteractive(false) })

	configPath := filepath.Join(tmpHome, ".potaco", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(legacyCustomProviderConfigYAML), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	loaded, err := config.LoadMultiProvider(configPath)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if loaded.SchemaVersion != config.CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, config.CurrentSchemaVersion)
	}
	if got := loaded.Providers["custom"].Type; got != "openai-compatible" {
		t.Fatalf("custom Type = %q, want openai-compatible", got)
	}
}

func resetAutoUpdateTest(t *testing.T) {
	t.Helper()
	resetRootCmdFlags(t)
	resetVersionCache()
	resetConfigSetFlags(t)
	resetUpdateCmdFlags(t)

	origVersion := Version
	origCheck := autoUpdateCheckLatest
	origInstall := autoUpdateInstall
	origPrompt := autoUpdatePrompt
	origInteractive := autoUpdateIsInteractive
	origNow := autoUpdateNow
	origTTL := autoUpdateCacheTTL
	origURL := githubReleaseURL

	Version = "v1.0.0"
	autoUpdateCheckLatest = checkLatestVersion
	autoUpdateInstall = installUpdate
	autoUpdatePrompt = promptAutoUpdate
	autoUpdateIsInteractive = tui.IsInteractive
	autoUpdateNow = time.Now
	autoUpdateCacheTTL = 24 * time.Hour
	tui.SetNonInteractive(false)

	t.Cleanup(func() {
		Version = origVersion
		autoUpdateCheckLatest = origCheck
		autoUpdateInstall = origInstall
		autoUpdatePrompt = origPrompt
		autoUpdateIsInteractive = origInteractive
		autoUpdateNow = origNow
		autoUpdateCacheTTL = origTTL
		githubReleaseURL = origURL
		tui.SetNonInteractive(false)
	})
}

func seedAutoUpdateConfig(t *testing.T, home string, autoUpdate *bool) {
	t.Helper()
	cfg := &config.MultiProviderConfig{
		AutoUpdate:     autoUpdate,
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {Model: "gpt-image-2"},
		},
	}
	if err := config.SaveMultiProvider(filepath.Join(home, ".potaco", "config.yaml"), cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func runAutoUpdateRoot(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(args)
	return &buf, rootCmd.Execute()
}

func TestAutoUpdateFreshCacheAvoidsNetwork(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedAutoUpdateConfig(t, home, nil)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	autoUpdateNow = func() time.Time { return now }
	if err := config.SaveUpdateCache(config.DefaultUpdateCachePath(), &config.UpdateCache{
		LastUpdateCheck: now.Add(-time.Hour),
		LatestVersion:   "v1.0.0",
	}); err != nil {
		t.Fatalf("seed update cache: %v", err)
	}

	autoUpdateIsInteractive = func() bool { return true }
	autoUpdateCheckLatest = func() (string, error) {
		t.Fatal("fresh cache should not call latest-version network check")
		return "", nil
	}

	if _, err := runAutoUpdateRoot(t, "config", "show"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestAutoUpdateStaleCacheChecksLatestAndWritesCache(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedAutoUpdateConfig(t, home, nil)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	autoUpdateNow = func() time.Time { return now }
	if err := config.SaveUpdateCache(config.DefaultUpdateCachePath(), &config.UpdateCache{
		LastUpdateCheck: now.Add(-48 * time.Hour),
		LatestVersion:   "v1.0.0",
	}); err != nil {
		t.Fatalf("seed update cache: %v", err)
	}

	calls := 0
	autoUpdateIsInteractive = func() bool { return true }
	autoUpdateCheckLatest = func() (string, error) {
		calls++
		return "v1.0.0", nil
	}

	if _, err := runAutoUpdateRoot(t, "config", "show"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if calls != 1 {
		t.Fatalf("latest check calls = %d, want 1", calls)
	}
	cache, err := config.LoadUpdateCache(config.DefaultUpdateCachePath())
	if err != nil {
		t.Fatalf("LoadUpdateCache: %v", err)
	}
	if !cache.LastUpdateCheck.Equal(now) || cache.LatestVersion != "v1.0.0" {
		t.Fatalf("cache = %+v, want refreshed at %v with v1.0.0", cache, now)
	}
}

func TestAutoUpdateDismissedVersionSuppressesPrompt(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedAutoUpdateConfig(t, home, nil)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	autoUpdateNow = func() time.Time { return now }
	if err := config.SaveUpdateCache(config.DefaultUpdateCachePath(), &config.UpdateCache{
		LastUpdateCheck:  now.Add(-time.Hour),
		LatestVersion:    "v1.1.0",
		DismissedVersion: "v1.1.0",
	}); err != nil {
		t.Fatalf("seed update cache: %v", err)
	}

	autoUpdateIsInteractive = func() bool { return true }
	autoUpdatePrompt = func(string) (bool, error) {
		t.Fatal("dismissed version should not prompt")
		return false, nil
	}

	if _, err := runAutoUpdateRoot(t, "config", "show"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestAutoUpdateNoStoresDismissedVersion(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedAutoUpdateConfig(t, home, nil)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	autoUpdateNow = func() time.Time { return now }

	autoUpdateIsInteractive = func() bool { return true }
	autoUpdateCheckLatest = func() (string, error) { return "v1.1.0", nil }
	autoUpdatePrompt = func(string) (bool, error) { return false, nil }
	autoUpdateInstall = func(*cobra.Command, string) error {
		t.Fatal("declined update should not run installer")
		return nil
	}

	if _, err := runAutoUpdateRoot(t, "config", "show"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cache, err := config.LoadUpdateCache(config.DefaultUpdateCachePath())
	if err != nil {
		t.Fatalf("LoadUpdateCache: %v", err)
	}
	if cache.DismissedVersion != "v1.1.0" {
		t.Fatalf("DismissedVersion = %q, want v1.1.0", cache.DismissedVersion)
	}
}

func TestAutoUpdateInstallerFailureReturnsError(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedAutoUpdateConfig(t, home, nil)

	autoUpdateIsInteractive = func() bool { return true }
	autoUpdateCheckLatest = func() (string, error) { return "v1.1.0", nil }
	autoUpdatePrompt = func(string) (bool, error) { return true, nil }
	autoUpdateInstall = func(*cobra.Command, string) error { return errors.New("installer failed") }

	_, err := runAutoUpdateRoot(t, "config", "show")
	if err == nil {
		t.Fatal("accepted update should return installer failure")
	}
}

func TestAutoUpdateSkips(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		prep func()
	}{
		{name: "unknown version", args: []string{"config", "show"}, prep: func() { Version = "unknown" }},
		{name: "non interactive flag", args: []string{"--non-interactive", "config", "show"}, prep: func() { autoUpdateIsInteractive = tui.IsInteractive }},
		{name: "json", args: []string{"--json", "config", "show"}},
		{name: "json after subcommand", args: []string{"config", "show", "--json"}},
		{name: "disable auto update", args: []string{"config", "set", "auto_update", "false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetAutoUpdateTest(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			seedAutoUpdateConfig(t, home, nil)
			autoUpdateIsInteractive = func() bool { return true }
			autoUpdateCheckLatest = func() (string, error) {
				t.Fatal("skip path should not check latest version")
				return "", nil
			}
			if tc.prep != nil {
				tc.prep()
			}

			_, _ = runAutoUpdateRoot(t, tc.args...)
		})
	}
}

func TestAutoUpdateSkipsCommandPaths(t *testing.T) {
	for _, cmd := range []*cobra.Command{
		updateCmd,
		versionCmd,
		uninstallCmd,
		{Use: "help"},
		findRootSubcommand(t, "completion"),
	} {
		t.Run(cmd.Name(), func(t *testing.T) {
			resetAutoUpdateTest(t)
			autoUpdateIsInteractive = func() bool { return true }
			if !shouldSkipAutoUpdate(cmd) {
				t.Fatalf("%s path should skip auto-update", cmd.Name())
			}
		})
	}
}

func findRootSubcommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("root command %q not found", name)
	return nil
}

func TestAutoUpdateDisabledInConfigSkips(t *testing.T) {
	resetAutoUpdateTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	disabled := false
	seedAutoUpdateConfig(t, home, &disabled)
	autoUpdateIsInteractive = func() bool { return true }
	autoUpdateCheckLatest = func() (string, error) {
		t.Fatal("auto_update false should skip latest version check")
		return "", nil
	}

	if _, err := runAutoUpdateRoot(t, "config", "show"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

// TestErrorNotDuplicated verifies that when a command returns an error,
// Cobra does not print the error to stderr (SilenceErrors is set). The
// Execute() wrapper function handles all error printing. This is a
// regression test for a bug where the error appeared twice in the output
// because both Cobra and Execute() printed it.
func TestErrorNotDuplicated(t *testing.T) {
	resetRootCmdFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var errBuf bytes.Buffer
	rootCmd.SetOut(&errBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing provider config")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Errorf("error should mention 'no active provider', got: %v", err)
	}
	// With SilenceErrors=true, Cobra should NOT print the error to stderr.
	// The Execute() wrapper handles printing. So stderr should be empty.
	if errBuf.String() != "" {
		t.Errorf("Cobra should not print error to stderr with SilenceErrors=true, got: %q", errBuf.String())
	}
}
