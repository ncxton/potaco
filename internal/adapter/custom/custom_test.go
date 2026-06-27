package custom

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

func writeMinimalPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.Black)
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func TestCustomAdapterRegistered(t *testing.T) {
	a, err := adapter.Get("custom", "sk-test", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("custom not registered: %v", err)
	}
	if a.Name() != "custom" {
		t.Errorf("Name = %q, want 'custom'", a.Name())
	}
}

func TestCustomAuthHeader(t *testing.T) {
	a, err := adapter.Get("custom", "sk-test", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("Get custom: %v", err)
	}
	if a.AuthHeader("sk-test") != "Bearer sk-test" {
		t.Errorf("AuthHeader = %q, want 'Bearer sk-test'", a.AuthHeader("sk-test"))
	}
}

func TestCustomSupportsGenerate(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})
	if !a.SupportsGenerate() {
		t.Error("SupportsGenerate should be true")
	}
}

func TestCustomSupportsEdit(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})
	if !a.SupportsEdit() {
		t.Error("SupportsEdit should be true")
	}
}

func TestCustomNewDefaults(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{}).(*Adapter)
	if a.timeout != 120*time.Second {
		t.Errorf("timeout = %v, want 120s", a.timeout)
	}
	if a.retries != 2 {
		t.Errorf("retries = %d, want 2", a.retries)
	}

	custom := New("sk-test", adapter.AdapterOpts{Timeout: 30 * time.Second, Retries: 5}).(*Adapter)
	if custom.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", custom.timeout)
	}
	if custom.retries != 5 {
		t.Errorf("retries = %d, want 5", custom.retries)
	}
}

func TestCustomURLConstructionWithV1Suffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	baseURL := server.URL + "/v1"
	a := New("sk-test", adapter.AdapterOpts{BaseURL: baseURL})
	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}

func TestCustomURLConstructionWithoutV1Suffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}
