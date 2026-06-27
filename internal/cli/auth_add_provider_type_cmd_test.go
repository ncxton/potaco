package cli

import (
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
)

func TestAuthAddCustomNamedProviderRequiresType(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openrouter", "--api-key", "sk-test", "--force", "--non-interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing provider type")
	}
	if !strings.Contains(err.Error(), "provider type") {
		t.Fatalf("error = %v, want provider type", err)
	}
}

func TestAuthAddCustomNamedProviderWithTypeAndBaseURL(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{
		"auth", "add", "openrouter",
		"--type", "openai-compatible",
		"--api-key", "sk-test",
		"--base-url", "https://openrouter.ai/api/v1",
		"--force",
		"--non-interactive",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add custom named provider with type error: %v", err)
	}

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
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
}

func TestShouldRunInteractiveAuthAdd(t *testing.T) {
	tests := []struct {
		name             string
		providerName     string
		providerTypeFlag string
		apiKey           string
		baseURL          string
		interactive      bool
		want             bool
	}{
		{
			name:         "unknown provider without type prompts in interactive mode",
			providerName: "openrouter",
			interactive:  true,
			want:         true,
		},
		{
			name:         "unknown provider without type errors in non-interactive mode",
			providerName: "openrouter",
			interactive:  false,
			want:         false,
		},
		{
			name:             "openai-compatible provider without base URL prompts in interactive mode",
			providerName:     "openrouter",
			providerTypeFlag: "openai-compatible",
			apiKey:           "sk-test",
			interactive:      true,
			want:             true,
		},
		{
			name:         "known provider without key prompts in interactive mode",
			providerName: "openai",
			interactive:  true,
			want:         true,
		},
		{
			name:             "complete openai-compatible provider does not prompt",
			providerName:     "openrouter",
			providerTypeFlag: "openai-compatible",
			apiKey:           "sk-test",
			baseURL:          "https://openrouter.ai/api/v1",
			interactive:      true,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunInteractiveAuthAdd(authAddInteractiveInput{
				providerName:     tt.providerName,
				providerTypeFlag: tt.providerTypeFlag,
				apiKey:           tt.apiKey,
				baseURL:          tt.baseURL,
				interactive:      tt.interactive,
			})
			if got != tt.want {
				t.Fatalf("shouldRunInteractiveAuthAdd() = %v, want %v", got, tt.want)
			}
		})
	}
}
