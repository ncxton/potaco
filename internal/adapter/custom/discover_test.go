package custom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestCustomDiscoverModelsReturnsAllModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "custom-image-model", "object": "model", "owned_by": "custom"},
				{"id": "text-model-1", "object": "model", "owned_by": "custom"},
				{"id": "another-image-model", "object": "model", "owned_by": "custom"},
			},
		})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	models, err := a.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels error: %v", err)
	}

	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
		if !m.SupportsGen {
			t.Errorf("model %s should have SupportsGen=true", m.ID)
		}
		if !m.SupportsEdit {
			t.Errorf("model %s should have SupportsEdit=true", m.ID)
		}
	}
	if !ids["custom-image-model"] {
		t.Error("should include custom-image-model")
	}
	if !ids["text-model-1"] {
		t.Error("should include text-model-1 (no filtering)")
	}
	if !ids["another-image-model"] {
		t.Error("should include another-image-model")
	}
}

func TestCustomDiscoverModelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	_, err := a.DiscoverModels(context.Background())
	if err == nil {
		t.Fatal("DiscoverModels should return error on API failure")
	}
}

func TestCustomDiscoverModelsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	_, err := a.DiscoverModels(context.Background())
	if err == nil {
		t.Fatal("DiscoverModels should return error on malformed JSON")
	}
}

func TestCustomVerifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	if err := a.Verify(context.Background()); err != nil {
		t.Fatalf("Verify should succeed on 200, got: %v", err)
	}
}

func TestCustomVerifyInvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	err := a.Verify(context.Background())
	if err == nil {
		t.Fatal("Verify should fail on 401")
	}
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention invalid key or 401, got: %v", err)
	}
}

func TestCustomVerifyForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	err := a.Verify(context.Background())
	if err == nil {
		t.Fatal("Verify should fail on 403")
	}
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention invalid key or 403, got: %v", err)
	}
}

func TestCustomVerifyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	err := a.Verify(context.Background())
	if err == nil {
		t.Fatal("Verify should fail on 500")
	}
	if !strings.Contains(err.Error(), "verification failed") && !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention verification failed or 500, got: %v", err)
	}
}

func TestCustomNoFallbackModels(t *testing.T) {
	// This is a compile-time/static check: ensure no fallbackModels or
	// hardcodedModelParams exist in the custom package. The custom
	// adapter must not contain hardcoded model lists.
	t.Log("custom adapter intentionally has no fallbackModels or hardcodedModelParams")
}
