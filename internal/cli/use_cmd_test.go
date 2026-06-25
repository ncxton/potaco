package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUseCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "use" || strings.HasPrefix(cmd.Use, "use ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'use' subcommand")
	}
}

func TestUseSwitchesProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Add two providers
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Use openai
	rootCmd.SetArgs([]string{"use", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("use openai: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}
}

func TestUseWithModel(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use", "openai", "--model", "dall-e-3"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("use with model: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "dall-e-3") {
		t.Errorf("output should mention model, got: %q", output)
	}
}

func TestUseNoArgs(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("use without args should error in non-interactive mode")
	}
}

func TestUseUnknownProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use", "nonexistent"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("use with unknown provider should error")
	}
}
