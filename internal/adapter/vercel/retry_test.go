package vercel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

// TestVercelRetryOn5xxThenSuccess verifies that a 500 response is retried
// and the subsequent successful response is returned.
func TestVercelRetryOn5xxThenSuccess(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: server.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "test",
		Model:  "openai/gpt-image-2",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q, want 'aGVsbG8='", resp.Data[0].B64JSON)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

// TestVercelRetryExhaustion verifies that when the server always returns 429,
// the adapter gives up after maxRetries attempts and returns the last
// error response.
func TestVercelRetryExhaustion(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: server.URL + "/v1", Retries: 2})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "test",
		Model:  "openai/gpt-image-2",
	})
	if err == nil {
		t.Fatal("Generate should return error after retry exhaustion")
	}
	// With retries=2: 1 initial attempt + 2 retries = 3 total attempts
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

// TestVercelRetryAfterHeaderHonored verifies that when the server sends a
// Retry-After header, the delay from that header is used instead of the
// default backoff. We capture the actual sleep duration via SetSleep.
func TestVercelRetryAfterHeaderHonored(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: server.URL + "/v1"})
	va := ad.(*Adapter)

	var capturedDelay time.Duration
	// Backoff returns a huge value that should NOT be used when Retry-After is present.
	va.SetBackoff(func(int) time.Duration { return 100 * time.Hour })
	va.SetSleep(func(ctx context.Context, d time.Duration) {
		capturedDelay = d
	})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "test",
		Model:  "openai/gpt-image-2",
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if capturedDelay != 1*time.Second {
		t.Errorf("sleep duration = %v, want 1s (from Retry-After header)", capturedDelay)
	}
}
