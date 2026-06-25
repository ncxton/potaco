package vercel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestVercelRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("vercel", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"vercel\") failed: %v", err)
	}
}

func TestVercelGenerate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want 'Bearer test-key'", auth)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data": []map[string]any{
				{"b64_json": "iVBORw0KGgo="},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "openai/gpt-image-2",
		N:      1,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "iVBORw0KGgo=" {
		t.Errorf("b64_json = %q, want 'iVBORw0KGgo='", resp.Data[0].B64JSON)
	}
	if gotBody["model"] != "openai/gpt-image-2" {
		t.Errorf("body model = %v, want 'openai/gpt-image-2'", gotBody["model"])
	}
}

func TestVercelGenerateWithProviderOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
			return
		}

		po, ok := gotBody["providerOptions"]
		if !ok {
			t.Fatal("providerOptions not found in body")
		}
		poMap, ok := po.(map[string]any)
		if !ok {
			t.Fatalf("providerOptions is %T, want map", po)
		}
		if _, ok := poMap["blackForestLabs"]; !ok {
			t.Error("providerOptions.blackForestLabs not found")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"url": "https://example.com/result.png"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "bfl/flux-2-pro",
		ExtraParams: map[string]any{
			"providerOptions": map[string]any{
				"blackForestLabs": map[string]any{
					"outputFormat": "png",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestVercelGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid model id",
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "test",
		Model:  "invalid/model",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid model id") {
		t.Errorf("error should contain 'invalid model id', got: %v", err)
	}
}
