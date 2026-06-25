package tui

import (
	"testing"
)

func TestIsInteractiveReturnsFalseInTestEnv(t *testing.T) {
	if IsInteractive() {
		t.Error("IsInteractive() should return false when stdin is not a TTY")
	}
}

func TestIsInteractiveReturnsFalseWhenNonInteractiveEnv(t *testing.T) {
	t.Setenv("POTACO_NON_INTERACTIVE", "1")
	if IsInteractive() {
		t.Error("IsInteractive() should return false when POTACO_NON_INTERACTIVE=1")
	}
}

func TestIsInteractiveReturnsFalseWhenSetNonInteractiveTrue(t *testing.T) {
	orig := nonInteractive
	t.Cleanup(func() { nonInteractive = orig })
	SetNonInteractive(true)
	if IsInteractive() {
		t.Error("IsInteractive() should return false after SetNonInteractive(true)")
	}
}

func TestIsTTYReturnsFalseInTestEnv(t *testing.T) {
	if IsTTY() {
		t.Error("IsTTY() should return false when stdin is not a character device")
	}
}
