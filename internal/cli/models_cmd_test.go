package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

func TestModelsListActiveProvider(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "gpt-image-2", "owned_by": "openai"},
					{"id": "text-embedding-3", "owned_by": "openai"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("models list should list gpt-image-2, got: %s", output)
	}
	if strings.Contains(output, "text-embedding-3") {
		t.Errorf("models list should not list non-image models, got: %s", output)
	}
}

func TestModelsListJSON(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })
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
	rootCmd.SetArgs([]string{"models", "list", "--json", "--base-url", srv.URL})
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
	if strings.Contains(output, "params") {
		t.Errorf("JSON models should not contain params, got: %s", output)
	}
	for _, field := range []string{"supports_gen", "supports_edit", "capabilities"} {
		if strings.Contains(output, field) {
			t.Errorf("JSON models should not expose discovered capability field %q, got: %s", field, output)
		}
	}
}

func TestModelsNonInteractiveFallback(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })
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
	rootCmd.SetArgs([]string{"models", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("models should fall back to static list, got: %s", output)
	}
	if !strings.Contains(output, "MODEL ID") {
		t.Errorf("models should print text table, got: %s", output)
	}
	if strings.Contains(output, "CAPABILITIES") || strings.Contains(output, "[edit]") {
		t.Errorf("models text should not expose discovered capabilities, got: %s", output)
	}
}

func TestModelsJSONNonInteractive(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })
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
	if strings.Contains(output, "MODEL ID") {
		t.Errorf("models --json should not print text table, got: %s", output)
	}
}

func TestModelsNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

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

func TestModelsListNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for no active provider")
	}
	if !strings.Contains(err.Error(), "no active provider") {
		t.Errorf("error should mention no active provider, got: %v", err)
	}
}

func TestModelsSpecificProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-openai", "--force", "--model", "gpt-image-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add openai: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-key", "--force", "--model", "fal-ai/flux/dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add fal: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
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
	rootCmd.SetArgs([]string{"models", "openai", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("models openai should list openai models, got: %s", output)
	}
}

func TestModelsListSpecificProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-openai", "--force", "--model", "gpt-image-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add openai: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-key", "--force", "--model", "fal-ai/flux/dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add fal: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-openai" {
			t.Errorf("Authorization = %q, want Bearer sk-openai", authHeader)
		}
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
	rootCmd.SetArgs([]string{"models", "list", "openai", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("models list openai should list openai models, got: %s", output)
	}
}

func TestModelsListSpecificProviderNotConnected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "fal"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unconnected provider")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error should mention not connected, got: %v", err)
	}
}

func TestModelsUnknownProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "unknown-provider"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error should mention unknown provider, got: %v", err)
	}
}

func TestModelsListUnknownProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "unknown-provider"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error should mention unknown provider, got: %v", err)
	}
}

func TestModelsListWithApiKeyOverride(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-stored", "gpt-image-2")
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

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
	rootCmd.SetArgs([]string{"models", "list", "--api-key", "sk-override", "--base-url", srv.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotAuth != "Bearer sk-override" {
		t.Errorf("Authorization header = %q, want Bearer sk-override", gotAuth)
	}
}

func TestModelsListUsesBaseURLFromConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.Add("openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}
	if err := mgr.SetBaseURL("openai", "https://config.example.com/v1"); err != nil {
		t.Fatalf("set base url: %v", err)
	}

	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "owned_by": "openai"},
			},
		})
	}))
	defer srv.Close()

	// Override the stored base URL with the test server URL so the request lands
	// on the mock server, but keep the resolution path.
	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pc := cfg.Providers["openai"]
	pc.BaseURL = srv.URL
	cfg.Providers["openai"] = pc
	if err := config.SaveMultiProvider(config.DefaultConfigPath(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotURL != "/v1/models" {
		t.Errorf("request path = %q, want /v1/models", gotURL)
	}
	if !strings.Contains(buf.String(), "gpt-image-2") {
		t.Errorf("output should list gpt-image-2, got: %s", buf.String())
	}
}

func TestModelsListAliasWithBuiltInTypeRequiresBaseURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	mgr, err := auth.New()
	if err != nil {
		t.Fatalf("init auth: %v", err)
	}
	if err := mgr.AddProvider("openrouter", "openai", "sk-test"); err != nil {
		t.Fatalf("add provider: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "openrouter"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing alias base URL")
	}
	if !strings.Contains(err.Error(), "base URL") {
		t.Fatalf("error = %v, want base URL", err)
	}
}

func TestModelsDiscoveryError(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "--base-url", srv.URL})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when discovery fails")
	}
	if !strings.Contains(err.Error(), "discover models") {
		t.Errorf("error should mention discover models, got: %v", err)
	}
}

func TestModelsHelp(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "models [provider]") {
		t.Errorf("help should show usage 'models [provider]', got: %s", output)
	}
	if !strings.Contains(output, "list") {
		t.Errorf("help should list subcommand 'list', got: %s", output)
	}
	if strings.Contains(output, "--params") {
		t.Errorf("help should not list --params flag, got: %s", output)
	}
}

func TestModelsListHelp(t *testing.T) {
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "list [provider]") {
		t.Errorf("help should show usage 'list [provider]', got: %s", output)
	}
}

func TestModelsMaximumNArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "openai", "extra"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args")
	}
	if !strings.Contains(err.Error(), "accepts at most 1 arg") {
		t.Errorf("error should mention arg limit, got: %v", err)
	}
}

func TestModelsListMaximumNArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)
	resetModelsCmdFlags(t)
	t.Cleanup(func() { resetModelsCmdFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "list", "openai", "extra"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args")
	}
	if !strings.Contains(err.Error(), "accepts at most 1 arg") {
		t.Errorf("error should mention arg limit, got: %v", err)
	}
}

func resetModelsCmdFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"base-url", "api-key"} {
		flag := modelsCmd.PersistentFlags().Lookup(name)
		if flag == nil {
			continue
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
		flag.Changed = false
	}
	for _, name := range []string{"base-url", "api-key"} {
		flag := modelsListCmd.Flags().Lookup(name)
		if flag == nil {
			continue
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset %s child flag: %v", name, err)
		}
		flag.Changed = false
	}
	// The help flag is added by Cobra to each command and may be left set after
	// a --help invocation, which would suppress arg validation on the next run.
	for _, cmd := range []*cobra.Command{modelsCmd, modelsListCmd} {
		for _, name := range []string{"help", "version"} {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				continue
			}
			if err := flag.Value.Set("false"); err != nil {
				t.Fatalf("reset %s flag on %s: %v", name, cmd.Name(), err)
			}
			flag.Changed = false
		}
	}
}
