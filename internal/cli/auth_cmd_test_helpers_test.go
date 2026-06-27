package cli

import (
	"bytes"
	"testing"
)

func newAuthTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Ensure POTACO_API_KEY does not leak from the environment so that
	// tests relying on "no api-key" behave deterministically.
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return tmpHome, &buf
}

func resetAuthAddFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"api-key", "force", "model", "base-url", "type"} {
		flag := authAddCmd.Flags().Lookup(name)
		if flag == nil {
			return // flags not registered yet
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}
