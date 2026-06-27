package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
)

func TestGenUsesConfiguredProviderTypeAndProviderKey(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetGenCmdFlags(t)
	t.Cleanup(func() { resetGenCmdFlags(t) })
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")
	outPath := filepath.Join(dir, "openrouter.png")

	tile := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tile.Set(0, 0, color.RGBA{R: 4, G: 5, B: 6, A: 255})
	var tileBuf bytes.Buffer
	if err := png.Encode(&tileBuf, tile); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	tileB64 := base64.StdEncoding.EncodeToString(tileBuf.Bytes())

	var gotAuth string
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if model, ok := body["model"].(string); ok {
			gotModel = model
		}
		fmt.Fprintf(w, `{"created":1,"data":[{"b64_json":%q}]}`, tileB64)
	}))
	defer server.Close()

	mgr := addOpenAICompatibleProvider(t, server.URL)

	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	openrouterConfig := cfg.Providers["openrouter"]
	openrouterConfig.Model = "openai/gpt-image-1"
	cfg.Providers["openrouter"] = openrouterConfig
	cfg.ActiveModel = "gpt-image-2"
	if err := config.SaveMultiProvider(config.DefaultConfigPath(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--provider", "openrouter", "--prompt", "a cat", "--model", "openai/gpt-image-1", "--output", outPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("gen returned error: %v", err)
	}

	if gotAuth != "Bearer sk-openrouter" {
		t.Errorf("auth header = %q, want Bearer sk-openrouter", gotAuth)
	}
	if gotModel != "openai/gpt-image-1" {
		t.Errorf("model = %q, want openai/gpt-image-1", gotModel)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected output file %q: %v", outPath, err)
	}
}

func TestModelsListConfiguredOpenAICompatibleProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "openrouter/image-model", "owned_by": "openrouter"},
			},
		})
	}))
	defer server.Close()

	addOpenAICompatibleProvider(t, server.URL)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "openrouter"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotAuth != "Bearer sk-openrouter" {
		t.Errorf("Authorization = %q, want Bearer sk-openrouter", gotAuth)
	}
	if !strings.Contains(buf.String(), "openrouter/image-model") {
		t.Errorf("models list openrouter should list configured models, got: %s", buf.String())
	}
}

func addOpenAICompatibleProvider(t *testing.T, baseURL string) *auth.AuthManager {
	t.Helper()

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	if err := mgr.Add("openai", "sk-openai"); err != nil {
		t.Fatalf("add openai: %v", err)
	}
	if err := mgr.Add("openrouter", "sk-openrouter"); err != nil {
		t.Fatalf("add openrouter: %v", err)
	}
	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	openaiConfig := cfg.Providers["openai"]
	openaiConfig.Model = "gpt-image-2"
	cfg.Providers["openai"] = openaiConfig
	openrouterConfig := cfg.Providers["openrouter"]
	openrouterConfig.Type = "openai-compatible"
	openrouterConfig.BaseURL = baseURL
	cfg.Providers["openrouter"] = openrouterConfig
	cfg.ActiveProvider = "openai"
	if err := config.SaveMultiProvider(config.DefaultConfigPath(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return mgr
}
