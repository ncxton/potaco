package fal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalAdapterName(t *testing.T) {
	ad := New("test-key", adapter.AdapterOpts{})
	if ad.Name() != "fal" {
		t.Errorf("Name() = %q, want %q", ad.Name(), "fal")
	}
}

func TestFalAuthHeader(t *testing.T) {
	ad := New("my-key", adapter.AdapterOpts{})
	got := ad.AuthHeader("my-key")
	want := "Key my-key"
	if got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestFalNewWithDefaults(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	fa, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *fal.Adapter, got %T", ad)
	}
	if fa.baseURL != "https://fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://fal.run")
	}
	if fa.retries != 2 {
		t.Errorf("retries = %d, want 2", fa.retries)
	}
}

func TestFalNewWithOverrides(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{
		BaseURL: "https://custom.fal.run",
		Retries: 5,
	})
	fa := ad.(*Adapter)
	if fa.baseURL != "https://custom.fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://custom.fal.run")
	}
	if fa.retries != 5 {
		t.Errorf("retries = %d, want 5", fa.retries)
	}
}

func TestFalRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("fal", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"fal\") failed: %v", err)
	}
}

func TestFalDiscoverModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		if qs := r.URL.RawQuery; qs != "category=image" {
			t.Errorf("query = %q, want 'category=image'", qs)
		}
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"id": "fal-ai/flux/dev",
					"metadata": map[string]any{
						"display_name": "Flux Dev",
					},
				},
				{
					"id": "fal-ai/flux/dev/image-to-image",
					"metadata": map[string]any{
						"display_name": "Flux Dev Image-to-Image",
					},
				},
				{
					"id": "fal-ai/nano-banana",
					"metadata": map[string]any{
						"display_name": "Nano Banana",
					},
				},
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("models len = %d, want 3", len(models))
	}
	if models[0].ID != "fal-ai/flux/dev" {
		t.Errorf("model[0] ID = %q, want 'fal-ai/flux/dev'", models[0].ID)
	}
	if models[0].DisplayName != "Flux Dev" {
		t.Errorf("model[0] DisplayName = %q, want 'Flux Dev'", models[0].DisplayName)
	}
	if !models[0].SupportsGen {
		t.Error("model[0] should support gen")
	}
	if models[0].SupportsEdit {
		t.Error("model[0] should not support edit")
	}
	if models[1].ID != "fal-ai/flux/dev/image-to-image" {
		t.Errorf("model[1] ID = %q", models[1].ID)
	}
	if !models[1].SupportsEdit {
		t.Error("model[1] should support edit (has 'image-to-image' in ID)")
	}
}

func TestFalDiscoverModelsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels should fall back, got error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("fallback models should not be empty")
	}
}

func TestFalVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	if err := ad.Verify(context.Background()); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestFalVerifyInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	ad := New("bad-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	err := ad.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("error should mention invalid key, got: %v", err)
	}
}

func TestFalModelParamsKnownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	params, err := ad.ModelParams(context.Background(), "fal-ai/flux/dev")
	if err != nil {
		t.Fatalf("ModelParams: %v", err)
	}
	if len(params) == 0 {
		t.Fatal("params should not be empty for known model")
	}
}

func TestFalModelParamsUnknownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.ModelParams(context.Background(), "unknown-model")
	if !errors.Is(err, adapter.ErrModelNotFound) {
		t.Errorf("ModelParams unknown model: got %v, want ErrModelNotFound", err)
	}
}
