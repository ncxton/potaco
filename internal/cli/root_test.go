package cli

import (
	"bytes"
	"strings"
	"testing"
)

// resetRootCmdFlags restores persistent flags on the shared global rootCmd to
// their default values. Tests that dispatch rootCmd.SetArgs must call this
// first so that flags left set by earlier tests (e.g. --json) do not leak in
// when -shuffle=on reorders test execution.
func resetRootCmdFlags(t *testing.T) {
	t.Helper()
	if err := rootCmd.PersistentFlags().Set("json", "false"); err != nil {
		t.Fatalf("reset json flag: %v", err)
	}
	if err := rootCmd.PersistentFlags().Set("verbose", "false"); err != nil {
		t.Fatalf("reset verbose flag: %v", err)
	}
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
