package openai

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

func TestOpenAIAdapterRegistered(t *testing.T) {
	a, err := adapter.Get("openai", "sk-test", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("openai not registered: %v", err)
	}
	if a.Name() != "openai" {
		t.Errorf("Name = %q, want 'openai'", a.Name())
	}
}

func TestOpenAIAuthHeader(t *testing.T) {
	a, err := adapter.Get("openai", "sk-test", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("Get openai: %v", err)
	}
	if a.AuthHeader("sk-test") != "Bearer sk-test" {
		t.Errorf("AuthHeader = %q, want 'Bearer sk-test'", a.AuthHeader("sk-test"))
	}
}

func TestOpenAIGenerateSuccess(t *testing.T) {
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
		var genReq map[string]any
		if err := json.Unmarshal(body, &genReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}
		if genReq["prompt"] != "a cat" {
			t.Errorf("prompt = %v, want 'a cat'", genReq["prompt"])
		}
		if genReq["model"] != "gpt-image-2" {
			t.Errorf("model = %v, want 'gpt-image-2'", genReq["model"])
		}

		resp := adapter.GenerateResponse{
			Created: 1234567890,
			Data: []adapter.ImageData{
				{B64JSON: "aGVsbG8="},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a, err := adapter.Get("openai", "sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("Get openai: %v", err)
	}
	openaiAdapter := a.(*Adapter)
	openaiAdapter.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	req := adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "gpt-image-2",
		N:      1,
		Size:   "1024x1024",
	}

	resp, err := a.Generate(context.Background(), req)
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

func TestOpenAIGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Invalid model",
			},
		})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	_, err := a.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should return error on 400")
	}
	if !strings.Contains(err.Error(), "Invalid model") {
		t.Errorf("error should contain API message, got: %v", err)
	}
}

func TestOpenAIGenerateExtraParamsPassthrough(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	ad := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	req := adapter.GenerateRequest{
		Prompt:      "test",
		Model:       "gpt-image-2",
		ExtraParams: map[string]any{"background": "transparent", "output_format": "webp"},
	}

	_, err := ad.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if receivedBody["background"] != "transparent" {
		t.Errorf("background = %v, want 'transparent'", receivedBody["background"])
	}
	if receivedBody["output_format"] != "webp" {
		t.Errorf("output_format = %v, want 'webp'", receivedBody["output_format"])
	}
}
