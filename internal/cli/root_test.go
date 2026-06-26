package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/tui"
)

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
