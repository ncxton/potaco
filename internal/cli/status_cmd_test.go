package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusShowsActiveProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Active provider: openai") {
		t.Errorf("status should show active provider, got: %s", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("status should show active model, got: %s", output)
	}
}

func TestStatusShowsConfigPaths(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("status should mention config.yaml, got: %s", output)
	}
	if !strings.Contains(output, "credentials") {
		t.Errorf("status should mention credentials, got: %s", output)
	}
}

func TestStatusShowsConnectedProviders(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("status should list connected providers, got: %s", output)
	}
	if !strings.Contains(output, "configured") {
		t.Errorf("status should show key status, got: %s", output)
	}
	if !strings.Contains(output, "(active)") {
		t.Errorf("status should mark active provider, got: %s", output)
	}
}

func TestStatusNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute should not error with no providers: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No active provider") {
		t.Errorf("status should show no active provider message, got: %s", output)
	}
}

func TestStatusJSON(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"active_provider\":") {
		t.Errorf("JSON status should contain active_provider, got: %s", output)
	}
	if !strings.Contains(output, "\"providers\":") {
		t.Errorf("JSON status should contain providers array, got: %s", output)
	}
}
