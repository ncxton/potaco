package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

// TestRetryOn429ThenSuccess verifies that a 429 response is retried and
// the subsequent successful response is returned.
func TestRetryOn429ThenSuccess(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created":123,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	resp, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
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

// TestRetryOn5xxThenSuccess verifies that a 500 response is retried and
// the subsequent successful response is returned.
func TestRetryOn5xxThenSuccess(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created":123,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	resp, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

// TestRetryExhaustion verifies that when the server always returns 429,
// the adapter gives up after maxRetries attempts and returns the last
// error response.
func TestRetryExhaustion(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL, Retries: 2})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should return error after retry exhaustion")
	}
	// With retries=2, we expect 1 initial attempt + 2 retries = 3 total attempts
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

// TestNoRetryOn400 verifies that a 400 response is not retried.
func TestNoRetryOn400(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should return error on 400")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 400)", attempts)
	}
}

// TestRetryAfterHeaderHonored verifies that when the server sends a
// Retry-After header, the delay from that header is used instead of the
// default backoff. We capture the actual sleep duration via SetSleep.
func TestRetryAfterHeaderHonored(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created":123,"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	oa := a.(*Adapter)

	var sleptDuration time.Duration
	oa.SetBackoff(func(int) time.Duration { return 100 * time.Hour }) // should NOT be used
	oa.SetSleep(func(ctx context.Context, d time.Duration) {
		sleptDuration = d
	})

	resp, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if sleptDuration != 2*time.Second {
		t.Errorf("slept duration = %v, want 2s (from Retry-After header)", sleptDuration)
	}
}

// TestContextCancellationDuringBackoff verifies that cancelling the
// context during a backoff sleep stops the retry loop. We use a custom
// sleep function that waits on the context to detect cancellation.
func TestContextCancellationDuringBackoff(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL, Retries: 10})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 50 * time.Millisecond })

	ctx, cancel := context.WithCancel(context.Background())
	oa.SetSleep(func(c context.Context, d time.Duration) {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-c.Done():
			return
		case <-timer.C:
		}
	})

	// Cancel after the first attempt so the backoff sleep is interrupted
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := a.Generate(ctx, adapter.GenerateRequest{Prompt: "test"})
	// Context cancellation during backoff should cause the request to fail.
	// The exact error depends on whether the http.Do call or the sleep
	// detects cancellation first, but either way it should be an error.
	if err == nil {
		t.Fatal("Generate should return error after context cancellation")
	}
	// Should have made at least 1 attempt but not exhausted all retries
	if attempts < 1 {
		t.Errorf("attempts = %d, want at least 1", attempts)
	}
}

// TestRetriesFromOpts verifies that the Retries value from AdapterOpts
// is passed through to the adapter. We confirm by checking how many
// attempts are made before giving up on a server that always returns 429.
func TestRetriesFromOpts(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL, Retries: 3})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should return error after retry exhaustion")
	}
	// With retries=3: 1 initial + 3 retries = 4 total attempts
	if attempts != 4 {
		t.Errorf("attempts = %d, want 4 (retries=3 means 1+3 attempts)", attempts)
	}
}

// TestRetriesDefaultWhenZero verifies that when Retries is 0 (not set),
// the adapter defaults to 2 retries.
func TestRetriesDefaultWhenZero(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	oa := a.(*Adapter)
	oa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should return error after retry exhaustion")
	}
	// Default retries=2: 1 initial + 2 retries = 3 total attempts
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (default retries=2)", attempts)
	}
}
