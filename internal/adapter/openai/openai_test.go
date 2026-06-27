package openai

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

func TestOpenAISupportsGenerate(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})
	if !a.SupportsGenerate() {
		t.Error("SupportsGenerate should be true")
	}
}

func TestOpenAISupportsEdit(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})
	if !a.SupportsEdit() {
		t.Error("SupportsEdit should be true")
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

func TestOpenAIEditSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	maskPath := filepath.Join(tmpDir, "mask.png")
	writeMinimalPNG(t, imgPath, 4, 4)
	writeMinimalPNG(t, maskPath, 4, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" && r.URL.Path != "/images/edits" {
			t.Errorf("path = %q, want images/edits", r.URL.Path)
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
		if r.FormValue("model") != "gpt-image-2" {
			t.Errorf("model = %q, want 'gpt-image-2'", r.FormValue("model"))
		}
		_, _, err := r.FormFile("image")
		if err != nil {
			t.Errorf("image file missing: %v", err)
		}
		_, _, err = r.FormFile("mask")
		if err != nil {
			t.Errorf("mask file missing: %v", err)
		}
		resp := adapter.GenerateResponse{
			Created: 1234567890,
			Data:    []adapter.ImageData{{B64JSON: "ZWRpdGVk"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	req := adapter.EditRequest{
		Prompt:    "make it blue",
		Model:     "gpt-image-2",
		ImagePath: imgPath,
		MaskPath:  maskPath,
		N:         1,
		Size:      "1024x1024",
	}

	resp, err := a.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if resp.Data[0].B64JSON != "ZWRpdGVk" {
		t.Errorf("B64JSON = %q, want 'ZWRpdGVk'", resp.Data[0].B64JSON)
	}
}

func TestOpenAIEditWithoutMask(t *testing.T) {
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
		resp := adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	req := adapter.EditRequest{
		Prompt:    "test",
		ImagePath: imgPath,
		Model:     "gpt-image-2",
	}

	resp, err := a.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if resp.Data[0].B64JSON != "dGVzdA==" {
		t.Errorf("B64JSON = %q, want 'dGVzdA=='", resp.Data[0].B64JSON)
	}
}

func TestOpenAIEditMissingImageFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	_, err := a.Edit(context.Background(), adapter.EditRequest{
		Prompt:    "test",
		ImagePath: "/nonexistent/file.png",
	})
	if err == nil {
		t.Fatal("Edit should error on missing image file")
	}
	if !strings.Contains(err.Error(), "image file") {
		t.Errorf("error should mention image file, got: %v", err)
	}
}

func TestOpenAIDiscoverModelsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "object": "model", "owned_by": "openai"},
				{"id": "text-davinci-003", "object": "model", "owned_by": "openai"},
			},
		})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	models, err := a.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels error: %v", err)
	}

	// Should only return image models, not text models
	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
	}
	if !ids["gpt-image-2"] {
		t.Error("should include gpt-image-2")
	}
	if ids["text-davinci-003"] {
		t.Error("should not include text model")
	}

	// Check SupportsEdit is set for gpt-image-2
	for _, m := range models {
		if m.ID == "gpt-image-2" && !m.SupportsEdit {
			t.Error("gpt-image-2 should have SupportsEdit=true")
		}
	}
}

func TestOpenAIDiscoverModelsError(t *testing.T) {
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

func TestOpenAIVerifySuccess(t *testing.T) {
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

func TestOpenAIVerifyInvalidKey(t *testing.T) {
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

func TestOpenAIDiscoverModelsMalformedJSON(t *testing.T) {
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
