package fal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalGenerate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/fal-ai/flux/dev" {
			t.Errorf("path = %q, want /fal-ai/flux/dev", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/result1.png", "width": 1024, "height": 1024},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "fal-ai/flux/dev",
		N:      1,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://cdn.fal.ai/result1.png" {
		t.Errorf("URL = %q, want cdn url", resp.Data[0].URL)
	}
	if gotBody["prompt"] != "a cat" {
		t.Errorf("body prompt = %v, want 'a cat'", gotBody["prompt"])
	}
	if gotBody["num_images"] != float64(1) {
		t.Errorf("body num_images = %v, want 1", gotBody["num_images"])
	}
	if gotBody["image_size"] != "1024x1024" {
		t.Errorf("body image_size = %v, want '1024x1024'", gotBody["image_size"])
	}
}

func TestFalGenerateWithExtraParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}

		if gotBody["guidance_scale"] != float64(7.5) {
			t.Errorf("guidance_scale = %v, want 7.5", gotBody["guidance_scale"])
		}
		if gotBody["num_inference_steps"] != float64(50) {
			t.Errorf("num_inference_steps = %v, want 50", gotBody["num_inference_steps"])
		}
		if gotBody["output_format"] != "png" {
			t.Errorf("output_format = %v, want 'png'", gotBody["output_format"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/result.png"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt:        "a cat",
		Model:         "fal-ai/flux/dev",
		GuidanceScale: 7.5,
		ExtraParams: map[string]any{
			"num_inference_steps": 50,
			"output_format":       "png",
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestFalGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"detail": "invalid model",
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "fal-ai/invalid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFalGenerateRetriesOnRateLimit(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}
		if gotBody["prompt"] != "retry cat" {
			t.Errorf("body prompt = %v, want 'retry cat'", gotBody["prompt"])
		}
		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			if _, err := w.Write([]byte(`{"detail":"rate limited"}`)); err != nil {
				t.Errorf("write rate limit response: %v", err)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/retry.png"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "retry cat",
		Model:  "fal-ai/flux/dev",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://cdn.fal.ai/retry.png" {
		t.Errorf("URL = %q, want retry URL", resp.Data[0].URL)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
