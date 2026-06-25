package provider

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestEditSuccess(t *testing.T) {
	// Create a temporary image file for the test
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	maskPath := filepath.Join(tmpDir, "mask.png")

	// Write a minimal valid PNG
	writeMinimalPNG(t, imgPath, 4, 4)
	writeMinimalPNG(t, maskPath, 4, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			t.Errorf("path = %q, want /v1/images/edits", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q, want multipart/form-data", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		if r.FormValue("prompt") != "make it blue" {
			t.Errorf("prompt = %q, want 'make it blue'", r.FormValue("prompt"))
		}
		if r.FormValue("model") != "dall-e-3" {
			t.Errorf("model = %q, want 'dall-e-3'", r.FormValue("model"))
		}

		_, _, err := r.FormFile("image")
		if err != nil {
			t.Errorf("image file missing: %v", err)
		}
		_, _, err = r.FormFile("mask")
		if err != nil {
			t.Errorf("mask file missing: %v", err)
		}

		resp := ImageResponse{
			Created: 1234567890,
			Data:    []ImageData{{B64JSON: "ZWRpdGVk"}},
		}
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

	req := EditRequest{
		Prompt:    "make it blue",
		Model:     "dall-e-3",
		ImagePath: imgPath,
		MaskPath:  maskPath,
		N:         1,
		Size:      "1024x1024",
	}

	resp, err := client.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "ZWRpdGVk" {
		t.Errorf("B64JSON = %q, want 'ZWRpdGVk'", resp.Data[0].B64JSON)
	}
}

func TestEditWithoutMask(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	writeMinimalPNG(t, imgPath, 4, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if _, _, err := r.FormFile("mask"); err == nil {
			t.Error("mask file should not be present")
		}
		if _, _, err := r.FormFile("image"); err != nil {
			t.Errorf("image file missing: %v", err)
		}
		resp := ImageResponse{Data: []ImageData{{B64JSON: "dGVzdA=="}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 0, Timeout: 5 * time.Second}
	client := NewClient(cfg)

	req := EditRequest{
		Prompt:    "test",
		ImagePath: imgPath,
		Model:     "dall-e-3",
	}

	resp, err := client.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if resp.Data[0].B64JSON != "dGVzdA==" {
		t.Errorf("B64JSON = %q, want 'dGVzdA=='", resp.Data[0].B64JSON)
	}
}

func TestEditMissingImageFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 0, Timeout: 5 * time.Second}
	client := NewClient(cfg)

	req := EditRequest{
		Prompt:    "test",
		ImagePath: "/nonexistent/file.png",
	}

	_, err := client.Edit(context.Background(), req)
	if err == nil {
		t.Fatal("Edit should error on missing image file")
	}
	if !strings.Contains(err.Error(), "image file") {
		t.Errorf("error should mention image file, got: %v", err)
	}
}

func TestProviderResponseLimit(t *testing.T) {
	oldLimit := maxProviderResponseBytes
	maxProviderResponseBytes = 16
	t.Cleanup(func() { maxProviderResponseBytes = oldLimit })

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"created":1,"data":[{"b64_json":"too-large"}]}`)),
	}

	_, err := parseResponse(resp)
	if err == nil {
		t.Fatal("parseResponse should reject responses over maxProviderResponseBytes")
	}
	if !strings.Contains(err.Error(), "response too large") {
		t.Fatalf("error should mention response too large, got: %v", err)
	}
}
