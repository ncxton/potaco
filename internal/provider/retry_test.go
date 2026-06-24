package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOn429ThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":{"type":"rate_limit","message":"Rate limited"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)
	}))
	defer server.Close()

	cfg := ClientConfig{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Retries: 3,
		Timeout: 30 * time.Second,
	}
	client := NewClient(cfg)
	// Override backoff for fast tests
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	resp, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q, want 'aGVsbG8='", resp.Data[0].B64JSON)
	}
	if callCount.Load() != 2 {
		t.Errorf("callCount = %d, want 2 (1 fail + 1 success)", callCount.Load())
	}
}

func TestRetryOn5xxThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"type":"server_error","message":"Internal error"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"created":1,"data":[{"b64_json":"b25rYQ=="}]}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 3, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	resp, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.Data[0].B64JSON != "b25rYQ==" {
		t.Errorf("B64JSON = %q, want 'b25rYQ=='", resp.Data[0].B64JSON)
	}
}

func TestRetryExhausted(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":{"type":"server_error","message":"Unavailable"}}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 2, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	_, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should error after retries exhausted")
	}
	// 1 initial + 2 retries = 3 total calls
	if callCount.Load() != 3 {
		t.Errorf("callCount = %d, want 3", callCount.Load())
	}
}

func TestNoRetryOn400(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"type":"invalid_request_error","message":"Bad request"}}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 3, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	_, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should error on 400")
	}
	if callCount.Load() != 1 {
		t.Errorf("callCount = %d, want 1 (no retry on 400)", callCount.Load())
	}
}
