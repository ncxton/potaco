package tui

import (
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

func TestRunAuthAddUnknownProvider(t *testing.T) {
	err := RunAuthAdd("nonexistent-provider")
	if err == nil {
		t.Fatal("RunAuthAdd with unknown provider should return an error")
	}
}

func TestRunAuthAddEmptyProvider(t *testing.T) {
	err := RunAuthAdd("")
	if err == nil {
		t.Fatal("RunAuthAdd with empty provider should return an error")
	}
}

func TestRunAuthAddCustomRequiresBaseURL(t *testing.T) {
	err := RunAuthAdd("custom")
	if err == nil {
		t.Fatal("RunAuthAdd with custom provider should require a base URL")
	}
}

func TestAuthAddRequiresBaseURLForAliases(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		providerType string
		want         bool
	}{
		{name: "custom", providerName: "custom", providerType: "openai-compatible", want: true},
		{name: "openai alias", providerName: "staging-openai", providerType: "openai", want: true},
		{name: "fal alias", providerName: "staging-fal", providerType: "fal", want: true},
		{name: "vercel alias", providerName: "staging-vercel", providerType: "vercel", want: true},
		{name: "openai built-in", providerName: "openai", providerType: "openai", want: false},
		{name: "fal built-in", providerName: "fal", providerType: "fal", want: false},
		{name: "vercel built-in", providerName: "vercel", providerType: "vercel", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authAddRequiresBaseURL(tt.providerName, tt.providerType); got != tt.want {
				t.Fatalf("authAddRequiresBaseURL(%q, %q) = %v, want %v", tt.providerName, tt.providerType, got, tt.want)
			}
		})
	}
}

func TestEnsureProviderTypeReturnsOpenAICompatibleForCustomProvider(t *testing.T) {
	providerType, err := ensureProviderType("custom")
	if err != nil {
		t.Fatalf("ensureProviderType: %v", err)
	}
	if providerType != "openai-compatible" {
		t.Fatalf("provider type = %q, want openai-compatible", providerType)
	}
}

func TestPromptModelReturnsEmptyWhenNoModels(t *testing.T) {
	modelID, err := promptModel("openai", []adapter.Model{})
	if err != nil {
		t.Fatalf("promptModel: %v", err)
	}
	if modelID != "" {
		t.Fatalf("model ID = %q, want empty", modelID)
	}
}

func TestAddProviderStoresProviderTypeAndBaseURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := addProvider(
		"openrouter",
		"openai-compatible",
		"sk-test",
		"https://openrouter.ai/api/v1",
		"openrouter/image-model",
	)
	if err != nil {
		t.Fatalf("addProvider: %v", err)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc, ok := cfg.Providers["openrouter"]
	if !ok {
		t.Fatal("openrouter provider should be configured")
	}
	if pc.Type != "openai-compatible" {
		t.Fatalf("provider type = %q, want openai-compatible", pc.Type)
	}
	if pc.BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("base URL = %q, want https://openrouter.ai/api/v1", pc.BaseURL)
	}
	if cfg.ActiveModel != "openrouter/image-model" {
		t.Fatalf("active model = %q, want openrouter/image-model", cfg.ActiveModel)
	}
}
