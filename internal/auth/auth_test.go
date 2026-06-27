// internal/auth/auth_test.go
package auth

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/credential"
)

func newTestAuth(t *testing.T) *AuthManager {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// We need to override the config and credential paths for testing.
	// AuthManager uses DefaultConfigPath/DefaultCredentialPath/DefaultSaltPath
	// which read from HOME. By setting HOME to temp dir, all paths resolve there.
	return newAuthWithPaths(
		filepath.Join(dir, ".potaco", "config.yaml"),
		filepath.Join(dir, ".potaco", "credentials.enc"),
		filepath.Join(dir, ".potaco", ".salt"),
	)
}

func newAuthWithPaths(configPath, credPath, saltPath string) *AuthManager {
	store, err := credential.New(credPath, saltPath)
	if err != nil {
		panic(err)
	}
	return &AuthManager{
		store:      store,
		configPath: configPath,
	}
}

func TestAuthAdd(t *testing.T) {
	auth := newTestAuth(t)

	err := auth.Add("openai", "sk-test-key")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify key is stored
	key, err := auth.store.Get("openai")
	if err != nil {
		t.Fatalf("Get key: %v", err)
	}
	if key != "sk-test-key" {
		t.Errorf("key = %q, want 'sk-test-key'", key)
	}

	// Verify config entry is written
	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
	if _, ok := cfg.Providers["openai"]; !ok {
		t.Error("openai should be in config providers")
	}
}

func TestAuthAddDoesNotSetDefaultModel(t *testing.T) {
	auth := newTestAuth(t)

	if err := auth.Add("openai", "sk-test-key"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Providers["openai"].Model != "" {
		t.Errorf("model = %q, want empty (no auto-default)", cfg.Providers["openai"].Model)
	}
	if cfg.ActiveModel != "" {
		t.Errorf("active model = %q, want empty", cfg.ActiveModel)
	}
}

func TestAuthManagerAddProviderStoresType(t *testing.T) {
	auth := newTestAuth(t)

	if err := auth.AddProvider("openrouter", "openai-compatible", "sk-test"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	pc := cfg.Providers["openrouter"]
	if pc.Type != "openai-compatible" {
		t.Fatalf("Type = %q, want openai-compatible", pc.Type)
	}
	if cfg.ActiveProvider != "openrouter" {
		t.Fatalf("ActiveProvider = %q, want openrouter", cfg.ActiveProvider)
	}
}

func TestAuthAddSetsActiveProvider(t *testing.T) {
	auth := newTestAuth(t)

	auth.Add("openai", "sk-1")
	auth.Add("fal", "fal-1")

	cfg, _ := auth.LoadConfig()
	// Adding a second provider should switch active to the new one
	if cfg.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", cfg.ActiveProvider)
	}
}

func TestAuthRemove(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1")
	auth.Add("fal", "fal-2")

	err := auth.Remove("fal")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Key should be gone
	_, err = auth.store.Get("fal")
	if err == nil {
		t.Error("fal key should be removed")
	}

	// Config entry should be gone
	cfg, _ := auth.LoadConfig()
	if _, ok := cfg.Providers["fal"]; ok {
		t.Error("fal should be removed from config")
	}

	// Active should switch back to openai
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
}

func TestAuthRemoveActiveProvider(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1")

	err := auth.Remove("openai")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	cfg, _ := auth.LoadConfig()
	if cfg.ActiveProvider != "" {
		t.Errorf("ActiveProvider = %q, want empty", cfg.ActiveProvider)
	}
}

func TestAuthList(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1")
	auth.Add("fal", "sk-2")

	providers := auth.List()
	if len(providers) != 2 {
		t.Fatalf("List len = %d, want 2", len(providers))
	}

	found := map[string]bool{}
	for _, p := range providers {
		found[p.Name] = true
		if p.Name == "openai" {
			if !p.HasKey {
				t.Error("openai should have key")
			}
		}
	}
	if !found["openai"] || !found["fal"] {
		t.Error("missing providers in list")
	}
}

func TestAuthSetActiveProvider(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1")
	auth.Add("fal", "sk-2")

	err := auth.SetActiveProvider("fal", "fal-ai/flux/schnell")
	if err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}

	cfg, _ := auth.LoadConfig()
	if cfg.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", cfg.ActiveProvider)
	}
	if cfg.ActiveModel != "fal-ai/flux/schnell" {
		t.Errorf("ActiveModel = %q", cfg.ActiveModel)
	}
	if cfg.Providers["fal"].Model != "fal-ai/flux/schnell" {
		t.Errorf("fal model in config = %q", cfg.Providers["fal"].Model)
	}
}

func TestAuthGetActiveAPIKey(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-test")

	// This should return an API key for the active provider
	_, err := auth.GetActiveAPIKey()
	if err != nil {
		t.Fatalf("GetActiveAPIKey: %v", err)
	}
}

func TestAuthGetActiveAPIKeyNoProvider(t *testing.T) {
	auth := newTestAuth(t)

	_, err := auth.GetActiveAPIKey()
	if err == nil {
		t.Fatal("should error when no active provider")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Errorf("error should mention 'no active provider', got: %v", err)
	}
}

func TestAuthGetActiveProvider(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1")
	auth.Add("fal", "sk-2")

	provider, model, err := auth.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider: %v", err)
	}
	if provider != "fal" {
		t.Errorf("provider = %q, want 'fal'", provider)
	}
	if model != "" {
		t.Errorf("model = %q, want empty (no auto-default)", model)
	}
}
