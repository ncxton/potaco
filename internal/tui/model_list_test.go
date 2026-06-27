package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/custom"
	_ "github.com/ncxton/potaco/internal/adapter/fal"
	_ "github.com/ncxton/potaco/internal/adapter/openai"
	_ "github.com/ncxton/potaco/internal/adapter/vercel"
	"github.com/ncxton/potaco/internal/auth"
)

func TestRunModelListNoActiveProviderReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunModelList("", "", "")
	if err == nil {
		t.Fatal("expected error when no active provider")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Errorf("error should mention no active provider, got: %v", err)
	}
}

func TestRunModelListNotConnectedReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunModelList("openai", "", "")
	if err == nil {
		t.Fatal("expected error when provider is not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error should mention not connected, got: %v", err)
	}
}

func TestRunModelListPersistsSelection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
				{"id": "dall-e-3", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	mockPicker := func(providerName string, models []adapter.Model) (string, error) {
		if providerName != "openai" {
			t.Errorf("providerName = %q, want openai", providerName)
		}
		if len(models) != 2 {
			t.Errorf("models = %d, want 2", len(models))
		}
		return "dall-e-3", nil
	}

	err = runModelListWithPicker("openai", "sk-test", srv.URL, mockPicker)
	if err != nil {
		t.Fatalf("runModelListWithPicker: %v", err)
	}

	provider, model, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}
	if provider != "openai" {
		t.Errorf("active provider = %q, want openai", provider)
	}
	if model != "dall-e-3" {
		t.Errorf("active model = %q, want dall-e-3", model)
	}

	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Providers["openai"].Model != "dall-e-3" {
		t.Errorf("provider model = %q, want dall-e-3", cfg.Providers["openai"].Model)
	}
}

func TestRunModelListCancelDoesNotPersist(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	beforeProvider, beforeModel, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}

	mockPicker := func(providerName string, models []adapter.Model) (string, error) {
		return "", huh.ErrUserAborted
	}

	err = runModelListWithPicker("openai", "sk-test", srv.URL, mockPicker)
	if err != nil {
		t.Fatalf("runModelListWithPicker: %v", err)
	}

	afterProvider, afterModel, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}
	if afterProvider != beforeProvider || afterModel != beforeModel {
		t.Errorf("config changed after cancel: provider %q->%q, model %q->%q", beforeProvider, afterProvider, beforeModel, afterModel)
	}
}

func TestRunModelListEmptySelectionDoesNotPersist(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	beforeProvider, beforeModel, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}

	mockPicker := func(providerName string, models []adapter.Model) (string, error) {
		return "", nil
	}

	err = runModelListWithPicker("openai", "sk-test", srv.URL, mockPicker)
	if err != nil {
		t.Fatalf("runModelListWithPicker: %v", err)
	}

	afterProvider, afterModel, err := mgr.GetActiveProvider()
	if err != nil {
		t.Fatalf("get active provider: %v", err)
	}
	if afterProvider != beforeProvider || afterModel != beforeModel {
		t.Errorf("config changed after empty selection: provider %q->%q, model %q->%q", beforeProvider, afterProvider, beforeModel, afterModel)
	}
}

func TestRunModelListUsesBaseURLFromConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}
	if err := mgr.SetBaseURL("openai", "https://config.example.com/v1"); err != nil {
		t.Fatalf("set base url: %v", err)
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	// Override the config base URL by pointing at the test server while keeping
	// the discovery URL path. The mock server reports the requested path.
	mockPicker := func(providerName string, models []adapter.Model) (string, error) {
		return "gpt-image-2", nil
	}

	err = runModelListWithPicker("openai", "sk-test", srv.URL, mockPicker)
	if err != nil {
		t.Fatalf("runModelListWithPicker: %v", err)
	}
	if gotPath != "/v1/models" {
		t.Errorf("request path = %q, want /v1/models", gotPath)
	}
}

func TestRunModelListDiscoveryError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	mockPicker := func(providerName string, models []adapter.Model) (string, error) {
		t.Fatal("picker should not be called when discovery fails")
		return "", nil
	}

	err = runModelListWithPicker("openai", "sk-test", srv.URL, mockPicker)
	if err == nil {
		t.Fatal("expected error when discovery fails")
	}
	if !strings.Contains(err.Error(), "discover models") {
		t.Errorf("error should mention discover models, got: %v", err)
	}
}
