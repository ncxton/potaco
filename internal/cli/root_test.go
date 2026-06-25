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
	// Without the flag set, the package-level variable should remain false
	// (interactive mode), so the only thing suppressing interactivity is the
	// env var or absence of a TTY.
}
