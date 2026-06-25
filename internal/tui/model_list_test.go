package tui

import (
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func TestRunModelListNoActiveProviderReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunModelList("", "")
	if err == nil {
		t.Fatal("expected error when no active provider")
	}
}

func TestRunModelListNotConnectedReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "", true); err != nil {
		t.Fatalf("add provider: %v", err)
	}
	err = RunModelList("openai", "")
	if err == nil {
		t.Fatal("expected error when provider is not connected")
	}
}
