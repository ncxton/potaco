package tui

import (
	"testing"
)

func TestRunUsePickerNoProvidersReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunUsePicker()
	if err == nil {
		t.Fatal("expected error when no providers connected")
	}
}
