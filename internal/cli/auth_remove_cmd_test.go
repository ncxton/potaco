package cli

import (
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func TestAuthRemoveCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "remove", "openai"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	list := mgr.List()
	for _, p := range list {
		if p.Name == "openai" {
			t.Errorf("openai should have been removed from config, but found in list")
		}
	}
}

func TestAuthRemoveAliasRm(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "rm", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth rm error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") {
		t.Errorf("output should mention removal, got: %q", output)
	}
}

func TestAuthRemoveRequiresProviderArg(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove without provider argument should error")
	}
}

func TestAuthRemoveNoArgsNonInteractiveErrors(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove without args in non-interactive mode should error")
	}
	if !strings.Contains(err.Error(), "specify") && !strings.Contains(err.Error(), "Specify") {
		t.Errorf("error should ask to specify a provider, got: %v", err)
	}
}

func TestAuthRemoveUnknownProviderNonInteractive(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove with unknown provider should error")
	}
}

func TestAuthRemoveKnownProviderNonInteractiveStillWorks(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "remove", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}
}
