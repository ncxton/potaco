package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandPrintsHelp(t *testing.T) {
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
