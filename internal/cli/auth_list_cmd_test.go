package cli

import (
	"strings"
	"testing"
)

func TestAuthListCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("list should include openai, got: %q", output)
	}
	if !strings.Contains(output, "fal") {
		t.Errorf("list should include fal, got: %q", output)
	}
}

func TestAuthListAliasLs(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "ls"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth ls error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("ls output should include openai, got: %q", output)
	}
}

func TestAuthListEmpty(t *testing.T) {
	_, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No providers") {
		t.Errorf("empty list output should mention no providers, got: %q", output)
	}
}

func TestAuthListJSON(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list --json error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[") {
		t.Errorf("JSON output should be an array, got: %q", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("JSON output should include openai, got: %q", output)
	}
	if strings.Contains(output, "base_url") {
		t.Errorf("JSON output should omit base_url for providers without one, got: %q", output)
	}
}

func TestAuthListJSONIncludesBaseURL(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-1", "--base-url", "https://example.com/v1", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add custom: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list --json error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "base_url") {
		t.Errorf("JSON output should include base_url, got: %q", output)
	}
	if !strings.Contains(output, "https://example.com/v1") {
		t.Errorf("JSON output should include the configured base URL, got: %q", output)
	}
}

func TestAuthListShowsBaseURL(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-1", "--base-url", "https://example.com/v1", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add custom: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "https://example.com/v1") {
		t.Errorf("list output should include the configured base URL, got: %q", output)
	}
}

func TestAuthListIncludesCustom(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "custom", "--api-key", "sk-test", "--base-url", "https://example.com/v1", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add custom: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom") {
		t.Errorf("list should include custom, got: %q", output)
	}
}
