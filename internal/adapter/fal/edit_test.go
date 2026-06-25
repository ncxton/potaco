package fal

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
	"strings"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalEdit(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fal-ai/flux/dev/image-to-image" {
			t.Errorf("path = %q, want /fal-ai/flux/dev/image-to-image", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/edited.png"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	createTestPNG(t, imgPath, 4, 4)

	resp, err := ad.Edit(context.Background(), adapter.EditRequest{
		Prompt:    "make it blue",
		Model:     "fal-ai/flux/dev",
		ImagePath: imgPath,
	})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}

	imageURL, ok := gotBody["image_url"].(string)
	if !ok {
		t.Fatal("image_url not found in body")
	}
	if !strings.HasPrefix(imageURL, "data:") {
		t.Errorf("image_url does not start with data:, got: %s", imageURL)
	}
	if gotBody["prompt"] != "make it blue" {
		t.Errorf("body prompt = %v, want 'make it blue'", gotBody["prompt"])
	}
}

func TestFalEditWithMask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}

		if _, ok := gotBody["mask_url"]; !ok {
			t.Error("mask_url not found in body")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/edited.png"},
			},
		}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	maskPath := tmpDir + "/mask.png"
	createTestPNG(t, imgPath, 4, 4)
	createTestPNG(t, maskPath, 4, 4)

	_, err := ad.Edit(context.Background(), adapter.EditRequest{
		Prompt:    "make it blue",
		Model:     "fal-ai/flux/dev",
		ImagePath: imgPath,
		MaskPath:  maskPath,
	})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
}

func TestFalEditNoImagePath(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.Edit(context.Background(), adapter.EditRequest{
		Prompt: "test",
		Model:  "fal-ai/flux/dev",
	})
	if err == nil {
		t.Fatal("expected error for missing image path")
	}
}

func createTestPNG(t *testing.T, path string, w int, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
