package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetUninstallCmdFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"remove-config", "yes"} {
		flag := uninstallCmd.Flags().Lookup(name)
		if flag == nil {
			return
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}

// withBinaryFinder temporarily overrides the binary locator so tests can
// point uninstall at a fake binary instead of the running test binary.
func withBinaryFinder(t *testing.T, path string) {
	t.Helper()
	orig := findPotacoBinaryFn
	findPotacoBinaryFn = func() (string, error) {
		if path == "" {
			return "", os.ErrNotExist
		}
		return path, nil
	}
	t.Cleanup(func() { findPotacoBinaryFn = orig })
}

func TestUninstallCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "uninstall" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'uninstall' subcommand")
	}
}

func TestUninstallNonInteractiveRemovesBinary(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake binary
	binPath := filepath.Join(tmpHome, ".local", "bin", "potaco")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	withBinaryFinder(t, binPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	output := strings.ToLower(buf.String())
	if !strings.Contains(output, "removed") || !strings.Contains(output, "binary") {
		t.Errorf("output should mention binary removal, got: %q", buf.String())
	}

	// Verify the binary was removed
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("binary should have been removed, but file still exists at %s", binPath)
	}
}

func TestUninstallNonInteractiveWithRemoveConfig(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake binary
	binPath := filepath.Join(tmpHome, ".local", "bin", "potaco")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	withBinaryFinder(t, binPath)

	// Create a fake config directory
	configDir := filepath.Join(tmpHome, ".potaco")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(legacyCustomProviderConfigYAML), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive", "--remove-config"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	// Verify both binary and config were removed
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("binary should have been removed")
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Errorf("config dir should have been removed")
	}
}

func TestUninstallBinaryNotFoundWarns(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Do NOT create a binary; finder returns not-found.
	withBinaryFinder(t, "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall should not error when binary not found, got: %v", err)
	}
	// Should warn but not fail
	output := buf.String()
	if !strings.Contains(output, "not found") && !strings.Contains(output, "already") {
		t.Errorf("output should mention binary not found, got: %q", output)
	}
}
