package cli

import (
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestAllThreeProvidersRegistered(t *testing.T) {
	names := adapter.List()
	want := []string{"fal", "openai", "vercel"}
	if len(names) != len(want) {
		t.Fatalf("registered providers = %v, want %v", names, want)
	}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q not registered, got %v", w, names)
		}
	}
}

func TestVercelPresetExists(t *testing.T) {
	preset, ok := getProviderPreset("vercel")
	if !ok {
		t.Fatal("vercel preset not found")
	}
	if preset.BaseURL == "" {
		t.Error("vercel preset BaseURL should not be empty")
	}
}
