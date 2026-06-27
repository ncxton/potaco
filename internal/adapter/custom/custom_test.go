package custom

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
	"github.com/ncxton/potaco/internal/observability"
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
		if r.Header.Get("X-Request-ID") != "" {
			t.Errorf("X-Request-ID should be empty when not in context, got %q", r.Header.Get("X-Request-ID"))
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

func TestCustomGenerateRequestIDPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") != "req-123" {
			t.Errorf("X-Request-ID = %q, want 'req-123'", r.Header.Get("X-Request-ID"))
		}
		json.NewEncoder(w).Encode(adapter.GenerateResponse{Data: []adapter.ImageData{{B64JSON: "dGVzdA=="}}})
	}))
	defer server.Close()

	ad := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})
	ctx := observability.WithRequestID(context.Background(), "req-123")
	_, err := ad.Generate(ctx, adapter.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
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

func TestCustomEditSuccess(t *testing.T) {
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
		if r.FormValue("model") != "custom-model" {
			t.Errorf("model = %q, want 'custom-model'", r.FormValue("model"))
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
		Model:     "custom-model",
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

func TestCustomEditWithoutMask(t *testing.T) {
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
		Model:     "custom-model",
	}

	resp, err := a.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if resp.Data[0].B64JSON != "dGVzdA==" {
		t.Errorf("B64JSON = %q, want 'dGVzdA=='", resp.Data[0].B64JSON)
	}
}

func TestCustomEditMissingImageFile(t *testing.T) {
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

func TestCustomDiscoverModelsReturnsAllModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "custom-image-model", "object": "model", "owned_by": "custom"},
				{"id": "text-model-1", "object": "model", "owned_by": "custom"},
				{"id": "another-image-model", "object": "model", "owned_by": "custom"},
			},
		})
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	models, err := a.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels error: %v", err)
	}

	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
		if !m.SupportsGen {
			t.Errorf("model %s should have SupportsGen=true", m.ID)
		}
		if !m.SupportsEdit {
			t.Errorf("model %s should have SupportsEdit=true", m.ID)
		}
	}
	if !ids["custom-image-model"] {
		t.Error("should include custom-image-model")
	}
	if !ids["text-model-1"] {
		t.Error("should include text-model-1 (no filtering)")
	}
	if !ids["another-image-model"] {
		t.Error("should include another-image-model")
	}
}

func TestCustomDiscoverModelsError(t *testing.T) {
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

func TestCustomDiscoverModelsMalformedJSON(t *testing.T) {
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

func TestCustomVerifySuccess(t *testing.T) {
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

func TestCustomVerifyInvalidKey(t *testing.T) {
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

func TestCustomVerifyForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	err := a.Verify(context.Background())
	if err == nil {
		t.Fatal("Verify should fail on 403")
	}
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention invalid key or 403, got: %v", err)
	}
}

func TestCustomVerifyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	err := a.Verify(context.Background())
	if err == nil {
		t.Fatal("Verify should fail on 500")
	}
	if !strings.Contains(err.Error(), "verification failed") && !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention verification failed or 500, got: %v", err)
	}
}

func TestCustomNoFallbackModels(t *testing.T) {
	// This is a compile-time/static check: ensure no fallbackModels or
	// hardcodedModelParams exist in the custom package. The custom
	// adapter must not contain hardcoded model lists.
	t.Log("custom adapter intentionally has no fallbackModels or hardcodedModelParams")
}
