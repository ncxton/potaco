package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth = %q, want Bearer sk-test", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var genReq GenerateRequest
		if err := json.Unmarshal(body, &genReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}
		if genReq.Prompt != "a cat" {
			t.Errorf("prompt = %q, want 'a cat'", genReq.Prompt)
		}

		resp := ImageResponse{
			Created: 1234567890,
			Data: []ImageData{
				{B64JSON: "aGVsbG8="},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ClientConfig{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Retries: 1,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	req := GenerateRequest{
		Prompt: "a cat",
		Model:  "dall-e-3",
		N:      1,
		Size:   "1024x1024",
	}

	resp, err := client.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q, want 'aGVsbG8='", resp.Data[0].B64JSON)
	}
}

func TestGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: APIError{
				Type:    "invalid_request_error",
				Message: "Invalid model",
			},
		})
	}))
	defer server.Close()

	cfg := ClientConfig{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Retries: 0,
		Timeout: 5 * time.Second,
	}
	client := NewClient(cfg)

	req := GenerateRequest{Prompt: "test"}

	_, err := client.Generate(context.Background(), req)
	if err == nil {
		t.Fatal("Generate should return error on 400")
	}
	if !strings.Contains(err.Error(), "Invalid model") {
		t.Errorf("error should contain API message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should contain status code 400, got: %v", err)
	}
}

func TestGenerateEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ImageResponse{Data: []ImageData{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ClientConfig{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Retries: 0,
		Timeout: 5 * time.Second,
	}
	client := NewClient(cfg)

	resp, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("Data len = %d, want 0", len(resp.Data))
	}
}
