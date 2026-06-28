package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/custom"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
)

func TestRunModelListUsesConfiguredProviderType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.AddProvider("openrouter", "openai-compatible", "sk-openrouter"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "openrouter/image-model", "owned_by": "openrouter"},
			},
		})
	}))
	defer srv.Close()

	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openrouter"]
	pc.BaseURL = srv.URL
	cfg.Providers["openrouter"] = pc
	if err := config.SaveMultiProvider(config.DefaultConfigPath(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	mockPicker := func(providerName string, models []adapter.Model) (modelSelection, error) {
		if providerName != "openrouter" {
			t.Errorf("providerName = %q, want openrouter", providerName)
		}
		return modelSelection{ID: "openrouter/image-model"}, nil
	}

	err = runModelListWithPicker("openrouter", "", "", mockPicker)
	if err != nil {
		t.Fatalf("runModelListWithPicker: %v", err)
	}
	if gotAuth != "Bearer sk-openrouter" {
		t.Errorf("Authorization = %q, want Bearer sk-openrouter", gotAuth)
	}
}
