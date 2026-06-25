package vercel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestVercelAdapterName(t *testing.T) {
	ad := New("test-key", adapter.AdapterOpts{})
	if ad.Name() != "vercel" {
		t.Errorf("Name() = %q, want %q", ad.Name(), "vercel")
	}
}

func TestVercelAuthHeader(t *testing.T) {
	ad := New("my-key", adapter.AdapterOpts{})
	got := ad.AuthHeader("my-key")
	want := "Bearer my-key"
	if got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestVercelNewWithDefaults(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	va, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *vercel.Adapter, got %T", ad)
	}
	if va.baseURL != "https://ai-gateway.vercel.sh/v1" {
		t.Errorf("baseURL = %q, want %q", va.baseURL, "https://ai-gateway.vercel.sh/v1")
	}
}

func TestVercelModelsURL(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{BaseURL: "https://example.com/v1"})
	va, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *vercel.Adapter, got %T", ad)
	}

	got := va.modelsURL()
	want := "https://example.com/v1/models"
	if got != want {
		t.Errorf("modelsURL() = %q, want %q", got, want)
	}
}

func TestVercelEndpointsURL(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{BaseURL: "https://example.com/v1"})
	va, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *vercel.Adapter, got %T", ad)
	}

	got := va.endpointsURL("openai/gpt-image-2")
	want := "https://example.com/v1/models/openai/gpt-image-2/endpoints"
	if got != want {
		t.Errorf("endpointsURL() = %q, want %q", got, want)
	}
}

func TestVercelRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("vercel", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"vercel\") failed: %v", err)
	}
}

func TestVercelEditNotSupported(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.Edit(context.Background(), adapter.EditRequest{
		Prompt:    "test",
		Model:     "openai/gpt-image-2",
		ImagePath: "/tmp/test.png",
	})
	if err != adapter.ErrEditNotSupported {
		t.Errorf("Edit error = %v, want ErrEditNotSupported", err)
	}
}

func TestVercelDiscoverModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "openai/gpt-image-2", "type": "image"},
				{"id": "openai/text-embedding-3", "type": "embedding"},
				{"id": "bfl/flux-2-pro", "type": "image"},
				{"id": "meta/llama-3", "type": "chat"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d, want 2 (only image type)", len(models))
	}
	if models[0].ID != "openai/gpt-image-2" {
		t.Errorf("model[0] ID = %q, want 'openai/gpt-image-2'", models[0].ID)
	}
	if models[1].ID != "bfl/flux-2-pro" {
		t.Errorf("model[1] ID = %q, want 'bfl/flux-2-pro'", models[1].ID)
	}
	if models[0].DisplayName != "gpt-image-2" {
		t.Errorf("model[0] DisplayName = %q, want 'gpt-image-2' (prefix stripped)", models[0].DisplayName)
	}
}

func TestVercelDiscoverModelsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels should fall back: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("fallback models should not be empty")
	}
}

func TestVercelVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The verify flow makes two requests: GET /v1/models (reachability)
		// and GET /v1/models/openai/gpt-image-2/endpoints (key validation).
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/models/openai/gpt-image-2/endpoints" {
			if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
				t.Errorf("endpoints auth = %q, want 'Bearer test-key'", auth)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	if err := ad.Verify(context.Background()); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestVercelVerifyInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/models/openai/gpt-image-2/endpoints" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ad := New("bad-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	err := ad.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("error should mention invalid key, got: %v", err)
	}
}

func TestVercelModelParamsKnownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	params, err := ad.ModelParams(context.Background(), "openai/gpt-image-2")
	if err != nil {
		t.Fatalf("ModelParams: %v", err)
	}
	if len(params) == 0 {
		t.Fatal("params should not be empty")
	}
}

func TestVercelModelParamsUnknownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.ModelParams(context.Background(), "unknown/model")
	if err != adapter.ErrModelNotFound {
		t.Errorf("got %v, want ErrModelNotFound", err)
	}
}
