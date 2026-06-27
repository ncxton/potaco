package custom

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

func TestCustomGenerateSuccess(t *testing.T) {
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
		if genReq["model"] != "custom-model" {
			t.Errorf("model = %v, want 'custom-model'", genReq["model"])
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

	a, err := adapter.Get("custom", "sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("Get custom: %v", err)
	}
	customAdapter := a.(*Adapter)
	customAdapter.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })

	req := adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "custom-model",
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

func TestCustomGenerateRequestBody(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	ad := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	req := adapter.GenerateRequest{
		Prompt:         "test",
		Model:          "custom-model",
		N:              1,
		Size:           "1024x1024",
		Quality:        "hd",
		Style:          "vivid",
		ResponseFormat: "b64_json",
		Seed:           42,
		GuidanceScale:  7.5,
		NegativePrompt: "blurry",
	}

	_, err := ad.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if receivedBody["prompt"] != "test" {
		t.Errorf("prompt = %v, want 'test'", receivedBody["prompt"])
	}
	if receivedBody["model"] != "custom-model" {
		t.Errorf("model = %v, want 'custom-model'", receivedBody["model"])
	}
	if receivedBody["n"] != float64(1) {
		t.Errorf("n = %v, want 1", receivedBody["n"])
	}
	if receivedBody["size"] != "1024x1024" {
		t.Errorf("size = %v, want '1024x1024'", receivedBody["size"])
	}
	if receivedBody["quality"] != "hd" {
		t.Errorf("quality = %v, want 'hd'", receivedBody["quality"])
	}
	if receivedBody["style"] != "vivid" {
		t.Errorf("style = %v, want 'vivid'", receivedBody["style"])
	}
	if receivedBody["response_format"] != "b64_json" {
		t.Errorf("response_format = %v, want 'b64_json'", receivedBody["response_format"])
	}
	if receivedBody["seed"] != float64(42) {
		t.Errorf("seed = %v, want 42", receivedBody["seed"])
	}
	if receivedBody["guidance_scale"] != 7.5 {
		t.Errorf("guidance_scale = %v, want 7.5", receivedBody["guidance_scale"])
	}
	if receivedBody["negative_prompt"] != "blurry" {
		t.Errorf("negative_prompt = %v, want 'blurry'", receivedBody["negative_prompt"])
	}
}

func TestCustomGenerateRequestBodyOmitsZeroValues(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	ad := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if _, ok := receivedBody["model"]; ok {
		t.Errorf("model should be omitted when empty")
	}
	if _, ok := receivedBody["n"]; ok {
		t.Errorf("n should be omitted when zero")
	}
	if _, ok := receivedBody["size"]; ok {
		t.Errorf("size should be omitted when empty")
	}
}

func TestCustomGenerateExtraParamsPassthrough(t *testing.T) {
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
		Model:       "custom-model",
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

func TestCustomGenerateAPIError(t *testing.T) {
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
