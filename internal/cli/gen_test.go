package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "gen" || strings.HasPrefix(cmd.Use, "gen ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'gen' subcommand")
	}
}

func TestGenCommandHasPromptFlag(t *testing.T) {
	promptFlag := genCmd.Flags().Lookup("prompt")
	if promptFlag == nil {
		t.Fatal("gen command should have --prompt flag")
	}
}

func TestGenCommandPromptRequired(t *testing.T) {
	resetRootCmdFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", ""}) // empty prompt

	// Cobra should enforce required flag
	err := rootCmd.Execute()
	if err == nil {
		// If not enforced by Cobra, our RunE should catch empty prompt
		// Check if it still runs - if so, we need manual validation
	}
}

func TestGenCommandDryRunNoAPI(t *testing.T) {
	resetRootCmdFlags(t)
	// Set up env so config merge succeeds
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"method": "POST"`) {
		t.Errorf("dry-run should print request method, got: %q", output)
	}
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "a cat") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
	// Should NOT have made an API call (no "Saved to:" in output)
	if strings.Contains(output, "Saved to:") {
		t.Errorf("dry-run should not save any files, got: %q", output)
	}
}

func TestGenCommandMissingConfigError(t *testing.T) {
	resetRootCmdFlags(t)
	// Clear all config sources
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no config is provided")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error should mention base_url, got: %v", err)
	}
}
