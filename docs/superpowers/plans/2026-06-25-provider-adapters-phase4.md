# Phase 4: Model & Status Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `potaco models` (list models, show params, explore by provider) and `potaco status` (show active provider, connected providers, config/credential paths) commands, both non-interactive with `--json` support.

**Architecture:** Two new Cobra command files in `internal/cli/`: `models_cmd.go` for `potaco models` and `status_cmd.go` for `potaco status`. Both use the existing `auth.AuthManager` and `adapter` packages to read state. No new packages or dependencies are needed. The `models` command calls `adapter.DiscoverModels()` and `adapter.ModelParams()` on the active or specified provider. The `status` command reads config and credential paths from `config.Default*Path()` functions and uses `auth.AuthManager.List()` for provider info. Both support `--json` via the existing root persistent flag.

**Tech Stack:** Go 1.26, Cobra CLI, `github.com/ncxton/potaco/internal/adapter`, `github.com/ncxton/potaco/internal/auth`, `github.com/ncxton/potaco/internal/config`

## Global Constraints

- Go 1.26, pure Go only (no CGO)
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping
- No `_ = err` (every `(T, error)` must be checked)
- `context.Context` as first param where applicable
- Keep files under 250 pure LOC
- Table-driven tests preferred. Test files sit alongside source: `foo.go` / `foo_test.go`
- CLI tests dispatch via `rootCmd.SetArgs([]string{...})` and `rootCmd.Execute()`
- Use `t.TempDir()` for temp files and `t.Setenv()` for env vars in tests
- Exit codes: 0 success, 2 config error, 3 API error, 4 image error
- Module path: `github.com/ncxton/potaco`
- `internal/adapter/` has `Get(name, apiKey, opts)`, `List()`, `AdapterOpts{BaseURL, Timeout, Retries}`, `Adapter` interface with `DiscoverModels(ctx)`, `ModelParams(ctx, modelID)`, `Verify(ctx)`, `Name()`, `AuthHeader(key)`
- `internal/auth/` has `New()`, `LoadConfig()`, `List() []ProviderInfo`, `GetActiveProvider() (provider, model, err)`, `GetActiveAPIKey() (string, error)`, `ProviderInfo{Name, Model, HasKey, AddedAt, IsActive}`
- `internal/config/` has `DefaultConfigPath()`, `DefaultCredentialPath()`, `MultiProviderConfig{ActiveProvider, ActiveModel, Providers}`
- `internal/cli/` has `rootCmd`, `configError()`, `apiError()`, `imageError()`, `flagString()`, `flagBool()`, `--json` persistent flag on rootCmd
- The `setupAuthProviderForProvider(t, providerName, apiKey, model)` helper exists in `gen_test.go` for setting up auth in CLI tests
- `adapter.Get()` requires an API key; for `models <provider>` on an unconnected provider, we still need a key to call DiscoverModels. If the provider is connected (has credentials), use the stored key. If not connected, error with guidance.

---

## File Structure

### New Files

| File | Responsibility |
|------|----------------|
| `internal/cli/models_cmd.go` | `potaco models` command: list models for active/specified provider, show params for a model |
| `internal/cli/models_cmd_test.go` | Tests for models command |
| `internal/cli/status_cmd.go` | `potaco status` command: show active provider, model, config/credential paths, connected providers |
| `internal/cli/status_cmd_test.go` | Tests for status command |

---

## Task 1: potaco status Command

**Files:**
- Create: `internal/cli/status_cmd.go`
- Create: `internal/cli/status_cmd_test.go`

**Interfaces:**
- Consumes: `auth.New()`, `auth.AuthManager.LoadConfig()`, `auth.AuthManager.List()`, `auth.AuthManager.GetActiveProvider()`, `config.DefaultConfigPath()`, `config.DefaultCredentialPath()`, `flagBool(cmd, "json")`, `configError()`, `rootCmd`
- Produces: `statusCmd` registered on `rootCmd` via `init()`

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/status_cmd_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusShowsActiveProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Active provider: openai") {
		t.Errorf("status should show active provider, got: %s", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("status should show active model, got: %s", output)
	}
}

func TestStatusShowsConfigPaths(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("status should mention config.yaml, got: %s", output)
	}
	if !strings.Contains(output, "credentials") {
		t.Errorf("status should mention credentials, got: %s", output)
	}
}

func TestStatusShowsConnectedProviders(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("status should list connected providers, got: %s", output)
	}
	if !strings.Contains(output, "configured") {
		t.Errorf("status should show key status, got: %s", output)
	}
	if !strings.Contains(output, "(active)") {
		t.Errorf("status should mark active provider, got: %s", output)
	}
}

func TestStatusNoActiveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetRootCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute should not error with no providers: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No active provider") {
		t.Errorf("status should show no active provider message, got: %s", output)
	}
}

func TestStatusJSON(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"active_provider\":") {
		t.Errorf("JSON status should contain active_provider, got: %s", output)
	}
	if !strings.Contains(output, "\"providers\":") {
		t.Errorf("JSON status should contain providers array, got: %s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestStatus -v`
Expected: FAIL with "unknown command status" or similar

- [ ] **Step 3: Write the implementation**

Create `internal/cli/status_cmd.go`:

```go
package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current provider, model, and connection status",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	provider, model, _ := mgr.GetActiveProvider()
	providers := mgr.List()
	configPath := config.DefaultConfigPath()
	credPath := config.DefaultCredentialPath()

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		return printStatusJSON(out, provider, model, configPath, credPath, providers)
	}

	if provider == "" {
		fmt.Fprintln(out, "No active provider configured.")
		fmt.Fprintln(out, "Use 'potaco auth add <provider>' to connect one.")
	} else {
		fmt.Fprintf(out, "Active provider: %s\n", provider)
		fmt.Fprintf(out, "Active model:    %s\n", model)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Config file:     %s\n", configPath)
	fmt.Fprintf(out, "Credentials:     %s\n", credPath)
	fmt.Fprintln(out)

	if len(providers) == 0 {
		fmt.Fprintln(out, "No providers connected.")
	} else {
		fmt.Fprintln(out, "Connected providers:")
		for _, p := range providers {
			active := ""
			if p.IsActive {
				active = " (active)"
			}
			keyStatus := "missing"
			if p.HasKey {
				keyStatus = "configured"
			}
			added := ""
			if p.AddedAt != "" {
				added = "  added: " + p.AddedAt
			}
			fmt.Fprintf(out, "  %s\t%s\tkey: %s%s%s\n", p.Name, p.Model, keyStatus, active, added)
		}
	}

	return nil
}

func printStatusJSON(out io.Writer, provider, model, configPath, credPath string, providers []auth.ProviderInfo) error {
	type providerJSON struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		HasKey   bool   `json:"has_key"`
		IsActive bool   `json:"is_active"`
		AddedAt  string `json:"added_at,omitempty"`
	}

	pjs := make([]providerJSON, 0, len(providers))
	for _, p := range providers {
		pjs = append(pjs, providerJSON{
			Name:     p.Name,
			Model:    p.Model,
			HasKey:   p.HasKey,
			IsActive: p.IsActive,
			AddedAt:  p.AddedAt,
		})
	}

	status := map[string]any{
		"active_provider": provider,
		"active_model":    model,
		"config_path":     configPath,
		"credential_path": credPath,
		"providers":       pjs,
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestStatus -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (no regressions)

- [ ] **Step 6: Commit**

```bash
git add internal/cli/status_cmd.go internal/cli/status_cmd_test.go
git commit -m "cli: add potaco status command with text and JSON output"
```

---

## Task 2: potaco models Command - List Models

**Files:**
- Create: `internal/cli/models_cmd.go`
- Create: `internal/cli/models_cmd_test.go`

**Interfaces:**
- Consumes: `auth.New()`, `auth.AuthManager.GetActiveProvider()`, `auth.AuthManager.GetActiveAPIKey()`, `adapter.Get(name, key, opts)`, `adapter.Adapter.DiscoverModels(ctx)`, `configError()`, `apiError()`, `flagBool(cmd, "json")`, `rootCmd`
- Produces: `modelsCmd` registered on `rootCmd` via `init()`, `runModels` function

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/models_cmd_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestModels -v`
Expected: FAIL with "unknown command models"

- [ ] **Step 3: Write the implementation**

Create `internal/cli/models_cmd.go`:

```go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/credential"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models [provider]",
	Short: "List available image models for the active or specified provider",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runModels,
}

func init() {
	modelsCmd.Flags().String("params", "", "show supported parameters for a model")
	modelsCmd.Flags().String("base-url", "", "override API base URL")
	modelsCmd.Flags().String("api-key", "", "override API key")
	rootCmd.AddCommand(modelsCmd)
}

func runModels(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	// Determine which provider to query.
	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
	} else {
		providerName, _, err = mgr.GetActiveProvider()
		if err != nil || providerName == "" {
			return configError(fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one"))
		}
	}

	// Get API key: try flag > env > credential store.
	apiKey := flagString(cmd, "api-key")
	if apiKey == "" {
		if v := os.Getenv("POTACO_API_KEY"); v != "" {
			apiKey = v
		}
	}
	if apiKey == "" && len(args) == 0 {
		// For active provider, try the credential store.
		k, kErr := mgr.GetActiveAPIKey()
		if kErr == nil {
			apiKey = k
		}
	}
	if apiKey == "" && len(args) > 0 {
		// For a specific provider, get its credential from the store.
		cfg, cfgErr := mgr.LoadConfig()
		if cfgErr == nil && cfg != nil {
			if _, ok := cfg.Providers[providerName]; ok {
				// Provider exists in config, try to get its key.
				credPath := config.DefaultCredentialPath()
				saltPath := config.DefaultSaltPath()
				store, storeErr := credential.New(credPath, saltPath)
				if storeErr == nil {
					k, kErr := store.Get(providerName)
					if kErr == nil {
						apiKey = k
					}
				}
			}
		}
	}
	if apiKey == "" {
		return configError(fmt.Errorf("provider %q is not connected. Use 'potaco auth add %s' first", providerName, providerName))
	}

	// Build adapter.
	baseURL := flagString(cmd, "base-url")
	opts := adapter.AdapterOpts{BaseURL: baseURL}
	ad, err := adapter.Get(providerName, apiKey, opts)
	if err != nil {
		return configError(fmt.Errorf("create adapter: %w", err))
	}

	// Check if --params was specified.
	modelID := flagString(cmd, "params")
	if modelID != "" {
		return showModelParams(cmd, ad, modelID)
	}

	// Discover models.
	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		return apiError(fmt.Errorf("discover models: %w", err))
	}

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		return printModelsJSON(out, models)
	}

	return printModelsText(out, models)
}

func printModelsText(out io.Writer, models []adapter.Model) error {
	if len(models) == 0 {
		fmt.Fprintln(out, "No models found.")
		return nil
	}
	fmt.Fprintf(out, "%-40s %-20s %s\n", "MODEL ID", "DISPLAY NAME", "CAPABILITIES")
	for _, m := range models {
		editBadge := ""
		if m.SupportsEdit {
			editBadge = " [edit]"
		}
		caps := fmt.Sprintf("%v", m.Capabilities)
		fmt.Fprintf(out, "%-40s %-20s%s %s\n", m.ID, m.DisplayName, editBadge, caps)
	}
	return nil
}

func printModelsJSON(out io.Writer, models []adapter.Model) error {
	type modelJSON struct {
		ID           string   `json:"id"`
		DisplayName  string   `json:"display_name"`
		SupportsGen  bool     `json:"supports_gen"`
		SupportsEdit bool     `json:"supports_edit"`
		Capabilities []string `json:"capabilities"`
	}
	items := make([]modelJSON, 0, len(models))
	for _, m := range models {
		items = append(items, modelJSON{
			ID:           m.ID,
			DisplayName:  m.DisplayName,
			SupportsGen:  m.SupportsGen,
			SupportsEdit: m.SupportsEdit,
			Capabilities: m.Capabilities,
		})
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}

func showModelParams(cmd *cobra.Command, ad adapter.Adapter, modelID string) error {
	params, err := ad.ModelParams(context.Background(), modelID)
	if err != nil {
		return apiError(fmt.Errorf("get model params: %w", err))
	}

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		return printParamsJSON(out, params)
	}

	return printParamsText(out, params)
}

func printParamsText(out io.Writer, params []adapter.Param) error {
	if len(params) == 0 {
		fmt.Fprintln(out, "No parameters found.")
		return nil
	}
	fmt.Fprintf(out, "%-25s %-10s %-15s %s\n", "NAME", "TYPE", "DEFAULT", "DESCRIPTION")
	for _, p := range params {
		enum := ""
		if len(p.EnumValues) > 0 {
			enum = fmt.Sprintf(" (enum: %v)", p.EnumValues)
		}
		fmt.Fprintf(out, "%-25s %-10s %-15s %s%s\n", p.Name, p.Type, p.Default, p.Description, enum)
	}
	return nil
}

func printParamsJSON(out io.Writer, params []adapter.Param) error {
	type paramJSON struct {
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		Description string   `json:"description"`
		Default     string   `json:"default"`
		EnumValues  []string `json:"enum_values,omitempty"`
		Required    bool     `json:"required"`
	}
	items := make([]paramJSON, 0, len(params))
	for _, p := range params {
		items = append(items, paramJSON{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Default:     p.Default,
			EnumValues:  p.EnumValues,
			Required:    p.Required,
		})
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}
```

Add the flag reset helper in the test file:

```go
func resetModelsCmdFlags(t *testing.T) {
	t.Helper()
	if f := modelsCmd.Flags().Lookup("params"); f != nil {
		f.Value.Set("")
	}
	if f := modelsCmd.Flags().Lookup("base-url"); f != nil {
		f.Value.Set("")
	}
	if f := modelsCmd.Flags().Lookup("api-key"); f != nil {
		f.Value.Set("")
	}
}
```

Note: The `"os"` import is needed in `models_cmd.go` for `os.Getenv`. The `"config"` and `"credential"` imports are needed for looking up a specific provider's key from the credential store when `models <provider>` is used for a connected but non-active provider.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestModels -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (no regressions)

- [ ] **Step 6: Commit**

```bash
git add internal/cli/models_cmd.go internal/cli/models_cmd_test.go
git commit -m "cli: add potaco models command with list and params support"
```

---

## Task 3: potaco models --params Support

**Files:**
- Modify: `internal/cli/models_cmd_test.go` (add params tests)

**Interfaces:**
- Consumes: `adapter.Adapter.ModelParams(ctx, modelID)`, `adapter.Param`, `adapter.ErrModelNotFound`
- Produces: Tests for `--params` flag on models command

This task adds tests for the `--params <model>` flag on the `models` command. The implementation already exists in Task 2's `showModelParams` function. This task just locks it with tests.

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/models_cmd_test.go`:

```go
func TestModelsParamsKnownModel(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--params", "gpt-image-2", "--json"})
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"models", "--params", "unknown-model"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestModelsParams -v`
Expected: PASS (implementation already exists from Task 2)

If tests fail, fix the implementation in `models_cmd.go`.

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (no regressions)

- [ ] **Step 4: Commit**

```bash
git add internal/cli/models_cmd_test.go
git commit -m "cli: add tests for models --params flag"
```

---

## Task 4: Final Verification

**Files:**
- No new files. Verification only.

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Run gofmt**

Run: `gofmt -l .`
Expected: No output

- [ ] **Step 4: Verify build**

Run: `go build -o potaco .`
Expected: Success

- [ ] **Step 5: Smoke test - status command**

```bash
rm -rf /tmp/potaco-test
HOME=/tmp/potaco-test ./potaco auth add openai --api-key sk-test --force
HOME=/tmp/potaco-test ./potaco status
HOME=/tmp/potaco-test ./potaco status --json
```
Expected: Status shows active provider, config path, connected providers

- [ ] **Step 6: Smoke test - models command**

```bash
HOME=/tmp/potaco-test ./potaco models --params gpt-image-2
HOME=/tmp/potaco-test ./potaco models --params gpt-image-2 --json
```
Expected: Models params shows size, quality, n for gpt-image-2

- [ ] **Step 7: Check all file LOCs are under 250**

Run:
```bash
for f in internal/cli/status_cmd.go internal/cli/models_cmd.go; do
  loc=$(awk '!/^[[:space:]]*$/ && !/^[[:space:]]*(\/\/)/' "$f" | wc -l)
  echo "$loc  $f"
done
```
Expected: All files under 250 pure LOC

- [ ] **Step 8: Commit any final fixes if needed**

If any issues were found and fixed:
```bash
git add -A
git commit -m "fix: final verification fixes for Phase 4"
```

Otherwise, no commit needed.
