package custom

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

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
		if ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var editReq map[string]any
		if err := json.Unmarshal(body, &editReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if editReq["prompt"] != "make it blue" {
			t.Errorf("prompt = %v, want 'make it blue'", editReq["prompt"])
		}
		if editReq["model"] != "custom-model" {
			t.Errorf("model = %v, want 'custom-model'", editReq["model"])
		}
		imageURL, ok := editReq["image_url"]
		if ok {
			t.Errorf("image_url should not be a top-level field; got %v", imageURL)
		}
		imagesRaw, ok := editReq["images"]
		if !ok {
			t.Fatal("images array not found in body")
		}
		images, ok := imagesRaw.([]any)
		if !ok {
			t.Fatalf("images should be an array, got %T", imagesRaw)
		}
		if len(images) != 1 {
			t.Fatalf("images len = %d, want 1", len(images))
		}
		entry, ok := images[0].(map[string]any)
		if !ok {
			t.Fatalf("images[0] should be an object, got %T", images[0])
		}
		imageURLStr, ok := entry["image_url"].(string)
		if !ok {
			t.Fatal("images[0].image_url not found")
		}
		if !strings.HasPrefix(imageURLStr, "data:image/png;base64,") {
			t.Errorf("images[0].image_url should be a png data URL, got: %s", imageURLStr[:min(len(imageURLStr), 40)])
		}
		maskURL, ok := editReq["mask"].(string)
		if !ok {
			t.Fatal("mask not found in body")
		}
		if !strings.HasPrefix(maskURL, "data:image/png;base64,") {
			t.Errorf("mask should be a png data URL, got: %s", maskURL[:min(len(maskURL), 40)])
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
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var editReq map[string]any
		if err := json.Unmarshal(body, &editReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if _, ok := editReq["mask_url"]; ok {
			t.Error("mask_url should not be a top-level field")
		}
		imagesRaw, ok := editReq["images"]
		if !ok {
			t.Fatal("images array missing from body")
		}
		images, ok := imagesRaw.([]any)
		if !ok {
			t.Fatalf("images should be an array, got %T", imagesRaw)
		}
		if len(images) != 1 {
			t.Fatalf("images len = %d, want 1", len(images))
		}
		entry, ok := images[0].(map[string]any)
		if !ok {
			t.Fatalf("images[0] should be an object, got %T", images[0])
		}
		if _, ok := entry["image_url"]; !ok {
			t.Error("images[0].image_url missing")
		}
		if _, ok := entry["mask_url"]; ok {
			t.Error("images[0].mask_url should not be present")
		}
		if _, ok := editReq["mask"]; ok {
			t.Error("mask should not be present when no mask path given")
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
	if !strings.Contains(err.Error(), "image") {
		t.Errorf("error should mention image, got: %v", err)
	}
}
