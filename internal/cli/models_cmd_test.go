package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelsListActiveProvider(t *testing.T) {
	// Set up auth with openai as active provider
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	// Mock the OpenAI models endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "gpt-image-2", "owned_by": "openai"},
					{"id": "dall-e-3", "owned_by": "openai"},
					{"id": "text-embedding-3", "owned_by": "openai"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// Override base-url to point at mock
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	// Should list image models (gpt-image-2, dall-e-3) but not text-embedding-3
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("models should list gpt-image-2, got: %s", output)
	}
	if strings.Contains(output, "text-embedding-3") {
		t.Errorf("models should not list non-image models, got: %s", output)
	}
}

func TestModelsListJSON(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--json", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"id\":") {
		t.Errorf("JSON models should contain id field, got: %s", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("JSON models should contain gpt-image-2, got: %s", output)
	}
}

func TestModelsListNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for no active provider")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Errorf("error should mention no active provider, got: %v", err)
	}
}

func TestModelsListSpecificProvider(t *testing.T) {
	// Set up auth with openai but request fal models
	setupAuthProviderForProvider(t, "openai", "sk-openai", "gpt-image-2")

	// Also add fal with a key
	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)
	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-key", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add fal: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)

	// Mock fal models endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") != "image" {
			t.Errorf("fal models should request category=image, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"id": "fal-ai/flux/dev", "metadata": map[string]any{"display_name": "Flux Dev"}},
			},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "fal", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "fal-ai/flux/dev") {
		t.Errorf("models fal should list fal models, got: %s", output)
	}
}

func TestModelsListSpecificProviderNotConnected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "fal"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unconnected provider")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error should mention not connected, got: %v", err)
	}
}

func TestModelsParamsKnownModel(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")
	resetModelsCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--params", "gpt-image-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "size") {
		t.Errorf("params should list size param, got: %s", output)
	}
	if !strings.Contains(output, "quality") {
		t.Errorf("params should list quality param, got: %s", output)
	}
}

func TestModelsParamsJSON(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")
	resetModelsCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	rootCmd.SetArgs([]string{"models", "--params", "gpt-image-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"name\":") {
		t.Errorf("JSON params should contain name field, got: %s", output)
	}
	if !strings.Contains(output, "\"type\":") {
		t.Errorf("JSON params should contain type field, got: %s", output)
	}
}

func TestModelsParamsUnknownModel(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")
	resetModelsCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--params", "unknown-model"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention model not found, got: %v", err)
	}
}

func TestModelsListWithApiKeyOverride(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-stored", "gpt-image-2")
	resetModelsCmdFlags(t)

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--api-key", "sk-override", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotAuth != "Bearer sk-override" {
		t.Errorf("Authorization header = %q, want Bearer sk-override", gotAuth)
	}
}

func resetModelsCmdFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"params", "base-url", "api-key"} {
		flag := modelsCmd.Flags().Lookup(name)
		if flag == nil {
			return
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
		flag.Changed = false
	}
}
