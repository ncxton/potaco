package cli

import (
	"reflect"
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

func TestProviderPresetHasNoDefaultModel(t *testing.T) {
	typ := reflect.TypeOf(providerPreset{})
	if _, ok := typ.FieldByName("DefaultModel"); ok {
		t.Error("providerPreset should not have a DefaultModel field")
	}
	if _, ok := typ.FieldByName("BaseURL"); !ok {
		t.Error("providerPreset should have a BaseURL field")
	}
}

func TestProviderPresetsNoCustomEntry(t *testing.T) {
	if _, ok := getProviderPreset("custom"); ok {
		t.Error("custom should not have a preset entry")
	}
}

func TestProviderPresetsOnlyKnownProviders(t *testing.T) {
	want := map[string]bool{
		"openai": true,
		"fal":    true,
		"vercel": true,
	}
	for name := range providerPresets {
		if !want[name] {
			t.Errorf("unexpected preset entry for %q", name)
		}
	}
	if len(providerPresets) != len(want) {
		t.Errorf("preset count = %d, want %d", len(providerPresets), len(want))
	}
}
