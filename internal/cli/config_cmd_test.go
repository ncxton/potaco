package cli

import (
	"bytes"
	"strings"
	"testing"
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
