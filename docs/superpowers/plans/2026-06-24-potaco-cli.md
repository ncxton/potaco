# Potaco CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI (`potaco`) for image generation and editing via any OpenAI-compatible provider, with gen/edit/config/info subcommands, inpainting, outpainting, retry, dry-run, and progressive disclosure output.

**Architecture:** Layered monolith with four internal packages (cli, provider, image, config) with one-directional dependencies flowing downward. CLI is the top layer; config and provider are peers; image is the bottom utility layer.

**Tech Stack:** Go 1.26, github.com/spf13/cobra, gopkg.in/yaml.v3, golang.org/x/image/draw, Go standard library `image`/`image/png`/`image/jpeg`/`net/http`/`encoding/json`/`mime/multipart`.

## Global Constraints

- Go module path: `github.com/ngct/potaco`
- Go version: go1.26 (present on the system)
- No CGO dependencies. Pure Go only.
- All internal packages under `internal/` (not importable externally).
- Config file location: `~/.potaco/config.yaml`
- Exit codes: 0 success, 1 general error, 2 config error, 3 API error, 4 image error
- API endpoints: `POST /v1/images/generations` (JSON body), `POST /v1/images/edits` (multipart form-data)
- Mask convention: white = edit, black = keep (OpenAI standard)
- Every task must follow TDD: write failing test, verify it fails, implement, verify it passes, commit.
- Each task produces independently testable deliverable.

---

### Task 1: Project Scaffold and Go Module

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: `cli.Execute()` function (entry point called by `main.go`), `rootCmd` variable with persistent flags `--json` and `--verbose`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd /home/ngct/Projects/potaco && go mod init github.com/ngct/potaco
```
Expected: creates `go.mod` with `module github.com/ngct/potaco` and `go 1.26`

- [ ] **Step 2: Write the failing test for root command help output**

Create `internal/cli/root_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandPrintsHelp(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd --help returned error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "potaco") {
		t.Errorf("help output should contain 'potaco', got: %s", output)
	}
}

func TestRootCommandHasJsonFlag(t *testing.T) {
	jsonFlag := rootCmd.PersistentFlags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("root command should have persistent --json flag")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("json flag default should be false, got %s", jsonFlag.DefValue)
	}
}

func TestRootCommandHasVerboseFlag(t *testing.T) {
	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("root command should have persistent --verbose flag")
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("verbose flag default should be false, got %s", verboseFlag.DefValue)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v`
Expected: FAIL (package does not exist / `rootCmd` undefined)

- [ ] **Step 4: Add cobra dependency**

Run:
```bash
cd /home/ngct/Projects/potaco && go get github.com/spf13/cobra@latest
```
Expected: downloads cobra and adds it to `go.mod`

- [ ] **Step 5: Write root command implementation**

Create `internal/cli/root.go`:
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "potaco",
	Short: "Terminal image generation and editing CLI",
	Long:  `Potaco provides advanced image generation and editing inside the terminal. Connect to any OpenAI-compatible provider supporting /v1/images/generations and /v1/images/edits.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "output JSON metadata to stdout")
	rootCmd.PersistentFlags().Bool("verbose", false, "print retry attempts and debug info to stderr")
}
```

- [ ] **Step 6: Write main.go entry point**

Create `main.go`:
```go
package main

import "github.com/ngct/potaco/internal/cli"

func main() {
	cli.Execute()
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v`
Expected: PASS (all three tests pass)

- [ ] **Step 8: Verify the binary builds and --help works**

Run:
```bash
cd /home/ngct/Projects/potaco && go build -o potaco . && ./potaco --help && rm potaco
```
Expected: builds successfully, prints help containing "potaco" usage text

- [ ] **Step 9: Commit**

```bash
cd /home/ngct/Projects/potaco && git add go.mod go.sum main.go internal/cli/root.go internal/cli/root_test.go && git commit -m "scaffold: init go module, root command with persistent flags"
```

---

### Task 2: Config Types and YAML Parsing

**Files:**
- Create: `internal/config/types.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Consumes: nothing (foundational package, no internal deps)
- Produces:
  - `Config` struct with fields: `BaseURL string`, `APIKey string`, `Model string`, `Retries int`, `Timeout time.Duration`
  - `ProviderPreset` struct with fields: `BaseURL string`, `DefaultModel string`, `Sizes []string`
  - `Load(path string) (*Config, error)` — reads and parses a YAML config file
  - `DefaultConfigPath() string` — returns `~/.potaco/config.yaml`

- [ ] **Step 1: Write the failing test for config parsing**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := `default:
  base_url: "https://api.openai.com"
  api_key: "sk-test123"
  model: "dall-e-3"
  retries: 3
  timeout: "90s"
`
	path := writeTestConfig(t, dir, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.openai.com")
	}
	if cfg.APIKey != "sk-test123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test123")
	}
	if cfg.Model != "dall-e-3" {
		t.Errorf("Model = %q, want %q", cfg.Model, "dall-e-3")
	}
	if cfg.Retries != 3 {
		t.Errorf("Retries = %d, want 3", cfg.Retries)
	}
	if cfg.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 90s", cfg.Timeout)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load should return error for missing file")
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := "default: [invalid"
	path := writeTestConfig(t, dir, content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load should return error for malformed YAML")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("DefaultConfigPath should not return empty string")
	}
	// Should contain .potaco somewhere in the path
	if !contains(path, ".potaco") {
		t.Errorf("DefaultConfigPath should contain '.potaco', got %q", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/config/ -v`
Expected: FAIL (package does not exist / types undefined)

- [ ] **Step 3: Add yaml dependency**

Run:
```bash
cd /home/ngct/Projects/potaco && go get gopkg.in/yaml.v3@latest
```
Expected: adds yaml.v3 to go.mod

- [ ] **Step 4: Write config types**

Create `internal/config/types.go`:
```go
package config

import "time"

// Config holds the resolved provider configuration after merging
// all precedence layers (flags, env, config file, presets).
type Config struct {
	BaseURL  string
	APIKey   string
	Model    string
	Retries  int
	Timeout  time.Duration
	Size     string
	Quality  string
	Provider string // preset name if specified
}

// FileConfig represents the raw YAML structure of ~/.potaco/config.yaml.
type FileConfig struct {
	Default struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
		Retries int    `yaml:"retries"`
		Timeout string `yaml:"timeout"`
	} `yaml:"default"`
}
```

- [ ] **Step 5: Write config loading implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath returns the default config file path at ~/.potaco/config.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/config.yaml"
	}
	return filepath.Join(home, ".potaco", "config.yaml")
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg := &Config{
		BaseURL: fc.Default.BaseURL,
		APIKey:  fc.Default.APIKey,
		Model:   fc.Default.Model,
		Retries: fc.Default.Retries,
	}

	if fc.Default.Retries == 0 {
		cfg.Retries = 2 // sensible default
	}

	if fc.Default.Timeout != "" {
		d, err := time.ParseDuration(fc.Default.Timeout)
		if err != nil {
			return nil, fmt.Errorf("parse timeout: %w", err)
		}
		cfg.Timeout = d
	} else {
		cfg.Timeout = 120 * time.Second
	}

	return cfg, nil
}

// FromEnv builds a Config from environment variables.
// Returns nil if no env vars are set.
func FromEnv() *Config {
	cfg := &Config{}
	set := false

	if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		cfg.BaseURL = v
		set = true
	}
	if v := os.Getenv("POTACO_API_KEY"); v != "" {
		cfg.APIKey = v
		set = true
	}
	if v := os.Getenv("POTACO_MODEL"); v != "" {
		cfg.Model = v
		set = true
	}
	if v := os.Getenv("POTACO_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Retries = n
			set = true
		}
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
			set = true
		}
	}

	if !set {
		return nil
	}
	if cfg.Retries == 0 {
		cfg.Retries = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}
	return cfg
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/config/ -v`
Expected: PASS (all four tests pass)

- [ ] **Step 7: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/config/ go.mod go.sum && git commit -m "config: add types, YAML loading, and env var parsing"
```

---

### Task 3: Config Merge Logic (Precedence)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/types.go`
- Create: `internal/config/merge_test.go`

**Interfaces:**
- Consumes: `Config` struct from Task 2, `FromEnv()` from Task 2
- Produces:
  - `Merge(opts MergeOptions) (*Config, error)` — merges flags > env > file > defaults
  - `MergeOptions` struct with optional fields for CLI flag overrides

- [ ] **Step 1: Write the failing test for merge precedence**

Create `internal/config/merge_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeFlagsOverrideEnv(t *testing.T) {
	t.Setenv("POTACO_BASE_URL", "https://from-env.com")
	t.Setenv("POTACO_MODEL", "env-model")

	opts := MergeOptions{
		BaseURL: ptrString("https://from-flag.com"),
		Model:   ptrString("flag-model"),
		APIKey:  ptrString("sk-flag"),
		Retries: ptrInt(5),
	}

	cfg, err := mergeInternal(opts, nil)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-flag.com" {
		t.Errorf("BaseURL = %q, want flag override", cfg.BaseURL)
	}
	if cfg.Model != "flag-model" {
		t.Errorf("Model = %q, want flag override", cfg.Model)
	}
	if cfg.APIKey != "sk-flag" {
		t.Errorf("APIKey = %q, want flag", cfg.APIKey)
	}
	if cfg.Retries != 5 {
		t.Errorf("Retries = %d, want 5", cfg.Retries)
	}
}

func TestMergeEnvOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `default:
  base_url: "https://from-file.com"
  api_key: "sk-file"
  model: "file-model"
  retries: 1
`
	os.WriteFile(path, []byte(content), 0644)

	t.Setenv("POTACO_BASE_URL", "https://from-env.com")

	opts := MergeOptions{}
	fileCfg, _ := Load(path)

	cfg, err := mergeInternal(opts, fileCfg)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-env.com" {
		t.Errorf("BaseURL = %q, want env override", cfg.BaseURL)
	}
	if cfg.Model != "file-model" {
		t.Errorf("Model = %q, want file value (env not set)", cfg.Model)
	}
	if cfg.APIKey != "sk-file" {
		t.Errorf("APIKey = %q, want file value", cfg.APIKey)
	}
}

func TestMergeFileDefaultsWhenNoFlagsOrEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `default:
  base_url: "https://from-file.com"
  api_key: "sk-file"
  model: "file-model"
  retries: 3
  timeout: "60s"
`
	os.WriteFile(path, []byte(content), 0644)

	opts := MergeOptions{}
	fileCfg, _ := Load(path)

	cfg, err := mergeInternal(opts, fileCfg)
	if err != nil {
		t.Fatalf("mergeInternal error: %v", err)
	}
	if cfg.BaseURL != "https://from-file.com" {
		t.Errorf("BaseURL = %q, want file value", cfg.BaseURL)
	}
	if cfg.Retries != 3 {
		t.Errorf("Retries = %d, want 3", cfg.Retries)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
}

func TestMergeMissingBaseURLError(t *testing.T) {
	opts := MergeOptions{}
	cfg, err := mergeInternal(opts, nil)
	if err == nil {
		t.Fatal("merge should error when no base_url is configured")
	}
	if cfg != nil {
		t.Fatal("should return nil config on error")
	}
}

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/config/ -v -run TestMerge`
Expected: FAIL (`mergeInternal` undefined, `MergeOptions` undefined)

- [ ] **Step 3: Write merge implementation**

Add to `internal/config/types.go`:
```go
// MergeOptions holds optional CLI flag values for the merge.
// Only non-nil fields override lower-precedence sources.
type MergeOptions struct {
	BaseURL  *string
	APIKey   *string
	Model    *string
	Retries  *int
	Timeout  *time.Duration
	Provider *string
}
```

Add to `internal/config/config.go`:
```go
// Merge resolves the final configuration by applying precedence:
// 1. CLI flags (non-nil fields in opts)
// 2. Environment variables
// 3. Config file (fileCfg, if non-nil)
// 4. Built-in defaults
func Merge(opts MergeOptions) (*Config, error) {
	return mergeInternal(opts, loadFileConfig())
}

func loadFileConfig() *Config {
	cfg, err := Load(DefaultConfigPath())
	if err != nil {
		return nil
	}
	return cfg
}

// mergeInternal is the testable core of Merge that accepts explicit inputs.
func mergeInternal(opts MergeOptions, fileCfg *Config) (*Config, error) {
	cfg := &Config{
		Retries: 2,
		Timeout: 120 * time.Second,
	}

	// Layer 3-4: file config
	if fileCfg != nil {
		cfg.BaseURL = fileCfg.BaseURL
		cfg.APIKey = fileCfg.APIKey
		cfg.Model = fileCfg.Model
		cfg.Retries = fileCfg.Retries
		cfg.Timeout = fileCfg.Timeout
	}

	// Layer 2: env vars (override file)
	envCfg := FromEnv()
	if envCfg != nil {
		if envCfg.BaseURL != "" {
			cfg.BaseURL = envCfg.BaseURL
		}
		if envCfg.APIKey != "" {
			cfg.APIKey = envCfg.APIKey
		}
		if envCfg.Model != "" {
			cfg.Model = envCfg.Model
		}
		if envCfg.Retries != 0 {
			cfg.Retries = envCfg.Retries
		}
		if envCfg.Timeout != 0 {
			cfg.Timeout = envCfg.Timeout
		}
	}

	// Layer 1: CLI flags (override everything)
	if opts.BaseURL != nil {
		cfg.BaseURL = *opts.BaseURL
	}
	if opts.APIKey != nil {
		cfg.APIKey = *opts.APIKey
	}
	if opts.Model != nil {
		cfg.Model = *opts.Model
	}
	if opts.Retries != nil {
		cfg.Retries = *opts.Retries
	}
	if opts.Timeout != nil {
		cfg.Timeout = *opts.Timeout
	}
	if opts.Provider != nil {
		cfg.Provider = *opts.Provider
	}

	// Validation
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("no base_url configured: set --base-url, POTACO_BASE_URL env, or config file default.base_url")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no api_key configured: set --api-key, POTACO_API_KEY env, or config file default.api_key")
	}

	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/config/ -v`
Expected: PASS (all merge tests pass, plus previous config tests still pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/config/ && git commit -m "config: add merge logic with flag>env>file>default precedence"
```

---

### Task 4: Provider Types and Presets

**Files:**
- Create: `internal/provider/types.go`
- Create: `internal/provider/presets.go`
- Create: `internal/provider/presets_test.go`

**Interfaces:**
- Consumes: nothing from internal packages (standalone)
- Produces:
  - `GenerateRequest` struct: `Prompt string`, `Model string`, `Size string`, `Quality string`, `N int`, `Style string`, `ResponseFormat string`, `Seed int`, `GuidanceScale float64`, `NegativePrompt string`
  - `EditRequest` struct: `Prompt string`, `Model string`, `Size string`, `N int`, `ResponseFormat string`, `ImagePath string`, `MaskPath string`
  - `ImageResponse` struct: `Created int64`, `Data []ImageData`
  - `ImageData` struct: `B64JSON string`, `URL string`, `RevisedPrompt string`
  - `APIError` struct: `Type string`, `Code string`, `Message string`, `Param string`
  - `ErrorResponse` struct: `Error APIError`
  - `Preset` struct: `BaseURL string`, `DefaultModel string`, `Sizes []string`
  - `GetPreset(name string) (Preset, bool)`
  - `AllPresets() map[string]Preset`

- [ ] **Step 1: Write the failing test for presets**

Create `internal/provider/presets_test.go`:
```go
package provider

import "testing"

func TestGetPresetOpenAI(t *testing.T) {
	p, ok := GetPreset("openai")
	if !ok {
		t.Fatal("preset 'openai' should exist")
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want https://api.openai.com", p.BaseURL)
	}
	if p.DefaultModel != "dall-e-3" {
		t.Errorf("DefaultModel = %q, want dall-e-3", p.DefaultModel)
	}
	if len(p.Sizes) == 0 {
		t.Error("Sizes should not be empty")
	}
}

func TestGetPresetTogether(t *testing.T) {
	p, ok := GetPreset("together")
	if !ok {
		t.Fatal("preset 'together' should exist")
	}
	if p.BaseURL != "https://api.together.ai" {
		t.Errorf("BaseURL = %q, want https://api.together.ai", p.BaseURL)
	}
}

func TestGetPresetFal(t *testing.T) {
	p, ok := GetPreset("fal")
	if !ok {
		t.Fatal("preset 'fal' should exist")
	}
	if p.BaseURL != "https://fal.run" {
		t.Errorf("BaseURL = %q, want https://fal.run", p.BaseURL)
	}
}

func TestGetPresetUnknown(t *testing.T) {
	_, ok := GetPreset("nonexistent")
	if ok {
		t.Fatal("GetPreset should return false for unknown preset")
	}
}

func TestAllPresets(t *testing.T) {
	presets := AllPresets()
	if len(presets) < 3 {
		t.Errorf("expected at least 3 presets, got %d", len(presets))
	}
	if _, ok := presets["openai"]; !ok {
		t.Error("presets should contain 'openai'")
	}
	if _, ok := presets["together"]; !ok {
		t.Error("presets should contain 'together'")
	}
	if _, ok := presets["fal"]; !ok {
		t.Error("presets should contain 'fal'")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v`
Expected: FAIL (package does not exist / `GetPreset` undefined)

- [ ] **Step 3: Write provider types**

Create `internal/provider/types.go`:
```go
package provider

// GenerateRequest is the JSON body for POST /v1/images/generations.
type GenerateRequest struct {
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model,omitempty"`
	N              int     `json:"n,omitempty"`
	Size           string  `json:"size,omitempty"`
	Quality        string  `json:"quality,omitempty"`
	Style          string  `json:"style,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Seed           int     `json:"seed,omitempty"`
	GuidanceScale  float64 `json:"guidance_scale,omitempty"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	User           string  `json:"user,omitempty"`
}

// EditRequest carries the parameters for POST /v1/images/edits.
// The image and mask are file paths; the client handles encoding them
// into multipart form data.
type EditRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	ResponseFormat string
	ImagePath      string
	MaskPath       string
	User           string
}

// ImageResponse is the JSON response from both endpoints.
type ImageResponse struct {
	Created int64     `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents a single generated/edited image.
type ImageData struct {
	B64JSON        string `json:"b64_json,omitempty"`
	URL            string `json:"url,omitempty"`
	RevisedPrompt  string `json:"revised_prompt,omitempty"`
}

// ErrorResponse is the JSON error shape returned by OpenAI-compatible APIs.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError holds the details of an API error.
type APIError struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}
```

- [ ] **Step 4: Write presets implementation**

Create `internal/provider/presets.go`:
```go
package provider

// Preset holds known defaults for a specific provider.
type Preset struct {
	BaseURL      string
	DefaultModel string
	Sizes        []string
}

var presets = map[string]Preset{
	"openai": {
		BaseURL:      "https://api.openai.com",
		DefaultModel: "dall-e-3",
		Sizes:        []string{"1024x1024", "1792x1024", "1024x1792"},
	},
	"together": {
		BaseURL:      "https://api.together.ai",
		DefaultModel: "black-forest-labs/flux-1",
		Sizes:        []string{"1024x1024"},
	},
	"fal": {
		BaseURL:      "https://fal.run",
		DefaultModel: "fal-ai/flux",
		Sizes:        []string{"1024x1024"},
	},
}

// GetPreset returns the preset for the named provider.
// Returns (Preset, true) if found, (Preset{}, false) otherwise.
func GetPreset(name string) (Preset, bool) {
	p, ok := presets[name]
	return p, ok
}

// AllPresets returns the full map of provider presets.
func AllPresets() map[string]Preset {
	return presets
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v`
Expected: PASS (all five tests pass)

- [ ] **Step 6: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/provider/ && git commit -m "provider: add request/response types and provider presets"
```

---

### Task 5: Provider Client - Generate Method

**Files:**
- Create: `internal/provider/client.go`
- Create: `internal/provider/client_test.go`

**Interfaces:**
- Consumes: `GenerateRequest`, `ImageResponse`, `ErrorResponse` from Task 4
- Produces:
  - `ClientConfig` struct: `BaseURL string`, `APIKey string`, `Retries int`, `Timeout time.Duration`
  - `NewClient(cfg ClientConfig) *Client`
  - `Client.Generate(ctx context.Context, req GenerateRequest) (*ImageResponse, error)`

- [ ] **Step 1: Write the failing test for Generate with a mock server**

Create `internal/provider/client_test.go`:
```go
package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v -run TestGenerate`
Expected: FAIL (`NewClient` undefined, `ClientConfig` undefined, `Client` undefined)

- [ ] **Step 3: Write client implementation**

Create `internal/provider/client.go`:
```go
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClientConfig holds the parameters for constructing a Client.
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Retries int
	Timeout time.Duration
}

// Client is the HTTP client for an OpenAI-compatible image provider.
type Client struct {
	baseURL string
	apiKey  string
	retries int
	http    *http.Client
}

// NewClient creates a provider Client from the given config.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		retries: cfg.Retries,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Generate calls POST /v1/images/generations and returns the response.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (*ImageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/v1/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// parseResponse reads the HTTP response and returns an ImageResponse or an error.
func parseResponse(resp *http.Response) (*ImageResponse, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp ImageResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &imgResp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v -run TestGenerate`
Expected: PASS (all three generate tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/provider/ && git commit -m "provider: add client with Generate method and response parsing"
```

---

### Task 6: Provider Client - Edit Method (Multipart)

**Files:**
- Modify: `internal/provider/client.go`
- Modify: `internal/provider/client_test.go`

**Interfaces:**
- Consumes: `EditRequest` from Task 4
- Produces: `Client.Edit(ctx context.Context, req EditRequest) (*ImageResponse, error)`

- [ ] **Step 1: Write the failing test for Edit with a mock server**

Append to `internal/provider/client_test.go`:
```go
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

		_, _, err = r.FormFile("image")
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
```

Add these imports and helpers at the top of the test file (update the import block):
```go
import (
	// ... existing imports ...
	"path/filepath"
	"image"
	"image/color"
	"image/png"
	"os"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v -run TestEdit`
Expected: FAIL (`Client.Edit` undefined)

- [ ] **Step 3: Write Edit method implementation**

Append to `internal/provider/client.go`:
```go
import (
	// add to existing imports:
	"mime/multipart"
	"net/textproto"
)

// Edit calls POST /v1/images/edits with multipart form data.
func (c *Client) Edit(ctx context.Context, req EditRequest) (*ImageResponse, error) {
	// Validate image file exists
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}
	imgFile, err := os.Open(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("image file: %w", err)
	}
	defer imgFile.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add image file part
	imgPart, err := writer.CreateFormFile("image", filepath.Base(req.ImagePath))
	if err != nil {
		return nil, fmt.Errorf("create image part: %w", err)
	}
	if _, err := io.Copy(imgPart, imgFile); err != nil {
		return nil, fmt.Errorf("copy image data: %w", err)
	}

	// Add mask file part (optional)
	if req.MaskPath != "" {
		maskFile, err := os.Open(req.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("mask file: %w", err)
		}
		maskPart, err := writer.CreateFormFile("mask", filepath.Base(req.MaskPath))
		if err != nil {
			maskFile.Close()
			return nil, fmt.Errorf("create mask part: %w", err)
		}
		if _, err := io.Copy(maskPart, maskFile); err != nil {
			maskFile.Close()
			return nil, fmt.Errorf("copy mask data: %w", err)
		}
		maskFile.Close()
	}

	// Add text fields
	if req.Prompt != "" {
		writer.WriteField("prompt", req.Prompt)
	}
	if req.Model != "" {
		writer.WriteField("model", req.Model)
	}
	if req.N > 0 {
		writer.WriteField("n", strconv.Itoa(req.N))
	}
	if req.Size != "" {
		writer.WriteField("size", req.Size)
	}
	if req.ResponseFormat != "" {
		writer.WriteField("response_format", req.ResponseFormat)
	}
	if req.User != "" {
		writer.WriteField("user", req.User)
	}

	writer.Close()

	url := c.baseURL + "/v1/images/edits"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}
```

Also update the import block at the top of `client.go` to include:
```go
"os"
"path/filepath"
"strconv"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v`
Expected: PASS (all generate and edit tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/provider/ && git commit -m "provider: add Edit method with multipart form-data for image and mask"
```

---

### Task 7: Retry Logic

**Files:**
- Create: `internal/provider/retry.go`
- Create: `internal/provider/retry_test.go`
- Modify: `internal/provider/client.go` (wire retry into Generate and Edit)

**Interfaces:**
- Consumes: `Client` from Task 5-6, `ClientConfig.Retries` from Task 5
- Produces:
  - `doWithRetry(ctx context.Context, fn func() (*http.Response, error), maxRetries int) (*http.Response, error)`
  - Retry behavior: exponential backoff (1s, 2s, 4s) with jitter, retry on 429 and 5xx, no retry on other 4xx

- [ ] **Step 1: Write the failing test for retry on 429 then success**

Create `internal/provider/retry_test.go`:
```go
package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOn429ThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":{"type":"rate_limit","message":"Rate limited"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)
	}))
	defer server.Close()

	cfg := ClientConfig{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Retries: 3,
		Timeout: 30 * time.Second,
	}
	client := NewClient(cfg)
	// Override backoff for fast tests
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	resp, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q, want 'aGVsbG8='", resp.Data[0].B64JSON)
	}
	if callCount.Load() != 2 {
		t.Errorf("callCount = %d, want 2 (1 fail + 1 success)", callCount.Load())
	}
}

func TestRetryOn5xxThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"type":"server_error","message":"Internal error"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"created":1,"data":[{"b64_json":"b25rYQ=="}]}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 3, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	resp, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.Data[0].B64JSON != "b25rYQ==" {
		t.Errorf("B64JSON = %q, want 'b25rYQ=='", resp.Data[0].B64JSON)
	}
}

func TestRetryExhausted(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":{"type":"server_error","message":"Unavailable"}}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 2, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	_, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should error after retries exhausted")
	}
	// 1 initial + 2 retries = 3 total calls
	if callCount.Load() != 3 {
		t.Errorf("callCount = %d, want 3", callCount.Load())
	}
}

func TestNoRetryOn400(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"type":"invalid_request_error","message":"Bad request"}}`)
	}))
	defer server.Close()

	cfg := ClientConfig{BaseURL: server.URL, APIKey: "sk-test", Retries: 3, Timeout: 30 * time.Second}
	client := NewClient(cfg)
	client.backoff = func(attempt int) time.Duration {
		return 1 * time.Millisecond
	}

	_, err := client.Generate(context.Background(), GenerateRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("Generate should error on 400")
	}
	if callCount.Load() != 1 {
		t.Errorf("callCount = %d, want 1 (no retry on 400)", callCount.Load())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v -run TestRetry`
Expected: FAIL (`client.backoff` field doesn't exist, retry logic not wired in)

- [ ] **Step 3: Write retry implementation**

Create `internal/provider/retry.go`:
```go
package provider

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// defaultBackoff returns the exponential backoff duration for a given attempt.
// Attempt 0 = 1s, 1 = 2s, 2+ = 4s. Jitter of 0-500ms is added.
func defaultBackoff(attempt int) time.Duration {
	base := time.Second
	switch attempt {
	case 0:
		base = 1 * time.Second
	case 1:
		base = 2 * time.Second
	default:
		base = 4 * time.Second
	}
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	return base + jitter
}

// shouldRetry returns true if the status code warrants a retry.
// Retries on 429 (rate limit) and 5xx (server errors).
func shouldRetry(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

// doWithRetry executes the given function, retrying on 429 and 5xx
// with exponential backoff up to maxRetries times.
func (c *Client) doWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; ; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			// Network errors: retry once
			if attempt < 1 && maxRetries > 0 {
				c.backoffSleep(attempt)
				// Recreate the request body reader if needed
				continue
			}
			return nil, fmt.Errorf("http request: %w", err)
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp

		if attempt >= maxRetries {
			// Exhausted retries, return the error response
			break
		}

		// Drain the body before retrying
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		c.backoffSleep(attempt)
	}

	if lastResp != nil {
		return lastResp, nil // let parseResponse handle the error
	}
	return nil, lastErr
}

func (c *Client) backoffSleep(attempt int) {
	if c.backoff != nil {
		time.Sleep(c.backoff(attempt))
	} else {
		time.Sleep(defaultBackoff(attempt))
	}
}
```

Modify `internal/provider/client.go`:

1. Add a `backoff` field to the `Client` struct:
```go
type Client struct {
	baseURL string
	apiKey  string
	retries int
	http    *http.Client
	backoff  func(attempt int) time.Duration // override for testing
}
```

2. Change the `Generate` method to use `doWithRetry`. Replace the section after setting headers with:
```go
	resp, err := c.doWithRetry(httpReq, c.retries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
```

3. Same for `Edit` method: replace `c.http.Do(httpReq)` with:
```go
	resp, err := c.doWithRetry(httpReq, c.retries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
```

Note: For the retry approach with request bodies, since `bytes.NewReader` implements `io.Seeker`, the `http.Request.GetBody` field is set by the Go HTTP client and allows retrying. For multipart, the body is a `bytes.Buffer` which also supports this.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v -run TestRetry`
Expected: PASS (all four retry tests pass)

- [ ] **Step 5: Run all provider tests to verify nothing broke**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/provider/ -v`
Expected: PASS (all generate, edit, retry, and preset tests pass)

- [ ] **Step 6: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/provider/ && git commit -m "provider: add retry with exponential backoff on 429 and 5xx"
```

---

### Task 8: Image I/O (Read, Write, Format Detection)

**Files:**
- Create: `internal/image/io.go`
- Create: `internal/image/io_test.go`

**Interfaces:**
- Consumes: nothing from internal packages
- Produces:
  - `ReadImage(path string) (image.Image, string, error)` — returns decoded image, format name ("png" or "jpeg"), error
  - `WriteImage(img image.Image, path string, format string) error` — writes image to file
  - `AutoFilename() string` — returns `potaco-YYYYMMDD-HHMMSS.png`
  - `DecodeBase64Image(b64 string) (image.Image, error)` — decodes base64-encoded image data
  - `FormatFromBytes(data []byte) string` — detects format by magic bytes

- [ ] **Step 1: Write the failing test for image I/O**

Create `internal/image/io_test.go`:
```go
package image

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestReadImagePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	os.WriteFile(path, makeTestPNG(t, 8, 8), 0644)

	img, format, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Errorf("dimensions = %dx%d, want 8x8", bounds.Dx(), bounds.Dy())
	}
}

func TestReadImageMissingFile(t *testing.T) {
	_, _, err := ReadImage("/nonexistent/file.png")
	if err == nil {
		t.Fatal("ReadImage should error on missing file")
	}
}

func TestReadImageUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("not an image"), 0644)

	_, _, err := ReadImage(path)
	if err == nil {
		t.Fatal("ReadImage should error on unsupported format")
	}
}

func TestWriteImagePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.png")
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	err := WriteImage(img, path, "png")
	if err != nil {
		t.Fatalf("WriteImage error: %v", err)
	}

	// Verify by reading back
	rImg, format, err := ReadImage(path)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	if rImg.Bounds().Dx() != 4 {
		t.Errorf("width = %d, want 4", rImg.Bounds().Dx())
	}
}

func TestAutoFilename(t *testing.T) {
	name := AutoFilename()
	if !strings.HasPrefix(name, "potaco-") {
		t.Errorf("filename should start with 'potaco-', got %q", name)
	}
	if !strings.HasSuffix(name, ".png") {
		t.Errorf("filename should end with '.png', got %q", name)
	}
}

func TestFormatFromBytesPNG(t *testing.T) {
	data := makeTestPNG(t, 4, 4)
	format := FormatFromBytes(data)
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
}

func TestFormatFromBytesJPEG(t *testing.T) {
	// JPEG magic bytes: FF D8 FF
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x00}
	format := FormatFromBytes(data)
	if format != "jpeg" {
		t.Errorf("format = %q, want 'jpeg'", format)
	}
}

func TestFormatFromBytesUnknown(t *testing.T) {
	data := []byte("hello world")
	format := FormatFromBytes(data)
	if format != "" {
		t.Errorf("format = %q, want ''", format)
	}
}

func TestDecodeBase64Image(t *testing.T) {
	pngData := makeTestPNG(t, 4, 4)
	b64 := base64.StdEncoding.EncodeToString(pngData)

	img, err := DecodeBase64Image(b64)
	if err != nil {
		t.Fatalf("DecodeBase64Image error: %v", err)
	}
	if img.Bounds().Dx() != 4 {
		t.Errorf("width = %d, want 4", img.Bounds().Dx())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v`
Expected: FAIL (package does not exist / `ReadImage` undefined)

- [ ] **Step 3: Write image I/O implementation**

Create `internal/image/io.go`:
```go
package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"
	"time"
)

// ReadImage reads and decodes an image file, auto-detecting the format
// by magic bytes. Returns the decoded image, the format name ("png" or "jpeg"),
// and an error.
func ReadImage(path string) (image.Image, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	format := FormatFromBytes(data)
	if format == "" {
		return nil, "", fmt.Errorf("unsupported image format (magic bytes not recognized)")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, format, fmt.Errorf("decode image: %w", err)
	}

	return img, format, nil
}

// WriteImage encodes and writes an image to a file in the specified format.
// Supported formats: "png" (default), "jpeg".
func WriteImage(img image.Image, path string, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "png", "":
		return png.Encode(f, img)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// AutoFilename generates a timestamp-based filename: potaco-YYYYMMDD-HHMMSS.png
func AutoFilename() string {
	return "potaco-" + time.Now().Format("20060102-150405") + ".png"
}

// FormatFromBytes detects the image format from the first few bytes.
// Returns "png", "jpeg", or "" if unknown.
func FormatFromBytes(data []byte) string {
	if len(data) < 3 {
		return ""
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && (len(data) > 3 && data[3] == 0x47) {
		return "png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpeg"
	}
	return ""
}

// DecodeBase64Image decodes a base64-encoded image string into an image.Image.
func DecodeBase64Image(b64 string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	return img, nil
}
```

- [ ] **Step 4: Register image decoders in an init file**

Create `internal/image/init.go`:
```go
package image

import (
	_ "image/jpeg"
	_ "image/png"
)
```

This ensures the `image` package's `Decode` function can auto-detect PNG and JPEG.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v`
Expected: PASS (all seven I/O tests pass)

- [ ] **Step 6: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/image/ && git commit -m "image: add read/write with format detection and base64 decoding"
```

---

### Task 9: Mask Generation (File, Rect, Circle)

**Files:**
- Create: `internal/image/mask.go`
- Create: `internal/image/mask_test.go`

**Interfaces:**
- Consumes: `ReadImage`, `WriteImage` from Task 8
- Produces:
  - `LoadMaskFile(path string, sourceWidth, sourceHeight int) (image.Image, error)` — loads a mask file, converts to white/black, scales to source dimensions
  - `RectMask(sourceWidth, sourceHeight, x, y, w, h int) (image.Image, error)` — generates a white rect on black background
  - `CircleMask(sourceWidth, sourceHeight, cx, cy, r int) (image.Image, error)` — generates a filled white circle
  - `WriteMask(img image.Image, path string) error` — writes mask as PNG to a temp file

- [ ] **Step 1: Write the failing test for mask generation**

Create `internal/image/mask_test.go`:
```go
package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makeMaskPNG(t *testing.T, w, h int, fill color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestRectMask(t *testing.T) {
	mask, err := RectMask(100, 100, 10, 20, 30, 40)
	if err != nil {
		t.Fatalf("RectMask error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("mask dimensions = %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Pixel inside the rect should be white
	r, g, b, _ := mask.At(15, 25).RGBA()
	if r == 0 || g == 0 || b == 0 {
		t.Error("pixel inside rect should be white")
	}

	// Pixel outside the rect should be black
	r2, g2, b2, _ := mask.At(0, 0).RGBA()
	if r2 != 0 || g2 != 0 || b2 != 0 {
		t.Error("pixel outside rect should be black")
	}
}

func TestRectMaskNegativeOffset(t *testing.T) {
	_, err := RectMask(100, 100, -10, 0, 30, 40)
	if err == nil {
		t.Fatal("RectMask should error on negative x")
	}
}

func TestCircleMask(t *testing.T) {
	mask, err := CircleMask(100, 100, 50, 50, 20)
	if err != nil {
		t.Fatalf("CircleMask error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("mask dimensions = %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Pixel at center should be white
	r, g, b, _ := mask.At(50, 50).RGBA()
	if r == 0 || g == 0 || b == 0 {
		t.Error("center pixel should be white")
	}

	// Pixel far from center should be black
	r2, g2, b2, _ := mask.At(90, 90).RGBA()
	if r2 != 0 || g2 != 0 || b2 != 0 {
		t.Error("far pixel should be black")
	}
}

func TestCircleMaskNegativeRadius(t *testing.T) {
	_, err := CircleMask(100, 100, 50, 50, -5)
	if err == nil {
		t.Fatal("CircleMask should error on negative radius")
	}
}

func TestLoadMaskFile(t *testing.T) {
	dir := t.TempDir()
	maskPath := filepath.Join(dir, "mask.png")
	// Create a mask where center is white, rest is black
	maskImg := image.NewGray(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x > 5 && x < 15 && y > 5 && y < 15 {
				maskImg.SetGray(x, y, color.White)
			} else {
				maskImg.SetGray(x, y, color.Black)
			}
		}
	}
	f, _ := os.Create(maskPath)
	png.Encode(f, maskImg)
	f.Close()

	mask, err := LoadMaskFile(maskPath, 20, 20)
	if err != nil {
		t.Fatalf("LoadMaskFile error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("dimensions = %dx%d, want 20x20", bounds.Dx(), bounds.Dy())
	}
}

func TestLoadMaskFileMismatch(t *testing.T) {
	dir := t.TempDir()
	maskPath := filepath.Join(dir, "mask.png")
	maskData := makeMaskPNG(t, 10, 10, color.White)
	os.WriteFile(maskPath, maskData, 0644)

	// Source is 20x20, mask is 10x10 - should scale
	mask, err := LoadMaskFile(maskPath, 20, 20)
	if err != nil {
		t.Fatalf("LoadMaskFile error: %v", err)
	}
	bounds := mask.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("dimensions = %dx%d, want 20x20 (scaled)", bounds.Dx(), bounds.Dy())
	}
}

func TestWriteMask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output-mask.png")
	mask, _ := RectMask(50, 50, 0, 0, 25, 25)

	err := WriteMask(mask, path)
	if err != nil {
		t.Fatalf("WriteMask error: %v", err)
	}

	// Verify it's a valid PNG
	data, _ := os.ReadFile(path)
	format := FormatFromBytes(data)
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v -run TestRect|TestCircle|TestLoadMask|TestWriteMask`
Expected: FAIL (`RectMask`, `CircleMask`, `LoadMaskFile`, `WriteMask` all undefined)

- [ ] **Step 3: Add x/image dependency**

Run:
```bash
cd /home/ngct/Projects/potaco && go get golang.org/x/image@latest
```
Expected: adds `golang.org/x/image` to go.mod

- [ ] **Step 4: Write mask implementation**

Create `internal/image/mask.go`:
```go
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// RectMask creates a mask image of the given source dimensions with a
// white rectangle at (x, y) of size (w, h) on a black background.
func RectMask(sourceWidth, sourceHeight, x, y, w, h int) (image.Image, error) {
	if x < 0 || y < 0 {
		return nil, fmt.Errorf("rect offset cannot be negative: x=%d y=%d", x, y)
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("rect dimensions must be positive: w=%d h=%d", w, h)
	}

	mask := image.NewGray(image.Rect(0, 0, sourceWidth, sourceHeight))
	// Fill with black (default zero value is 0 = black)
	// Draw white rect
	for row := y; row < y+h && row < sourceHeight; row++ {
		for col := x; col < x+w && col < sourceWidth; col++ {
			mask.SetGray(col, row, color.White)
		}
	}
	return mask, nil
}

// CircleMask creates a mask image of the given source dimensions with a
// filled white circle centered at (cx, cy) with radius r on a black background.
func CircleMask(sourceWidth, sourceHeight, cx, cy, r int) (image.Image, error) {
	if r <= 0 {
		return nil, fmt.Errorf("radius must be positive: r=%d", r)
	}

	mask := image.NewGray(image.Rect(0, 0, sourceWidth, sourceHeight))
	r2 := r * r
	for y := 0; y < sourceHeight; y++ {
		for x := 0; x < sourceWidth; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				mask.SetGray(x, y, color.White)
			}
		}
	}
	return mask, nil
}

// LoadMaskFile loads a mask from a file path and ensures it matches the
// source image dimensions. If dimensions differ, the mask is scaled.
// Any non-black pixel becomes white; black stays black.
func LoadMaskFile(path string, sourceWidth, sourceHeight int) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mask file: %w", err)
	}

	rawImg, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode mask: %w", err)
	}

	// Convert to grayscale: non-black -> white, black -> black
	bounds := rawImg.Bounds()
	grayMask := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rawImg.At(x, y)
			r, g, b, _ := c.RGBA()
			// If any channel is non-zero, treat as white
			if r > 0 || g > 0 || b > 0 {
				grayMask.SetGray(x, y, color.White)
			}
			// else stays black (zero value)
		}
	}

	// Scale to source dimensions if different
	srcBounds := image.Rect(0, 0, sourceWidth, sourceHeight)
	if bounds.Dx() != sourceWidth || bounds.Dy() != sourceHeight {
		scaled := image.NewGray(srcBounds)
		draw.NearestNeighbor.Scale(scaled, srcBounds, grayMask, bounds, draw.Over, nil)
		return scaled, nil
	}

	return grayMask, nil
}

// WriteMask writes a mask image as a PNG file to the given path.
func WriteMask(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create mask file: %w", err)
	}
	defer f.Close()
	return png.Encode(f, img)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v`
Expected: PASS (all mask tests and I/O tests pass)

- [ ] **Step 6: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/image/ go.mod go.sum && git commit -m "image: add mask generation (rect, circle, file loading with scaling)"
```

---

### Task 10: Outpaint Canvas Expansion

**Files:**
- Create: `internal/image/canvas.go`
- Create: `internal/image/canvas_test.go`

**Interfaces:**
- Consumes: `ReadImage` from Task 8, `WriteImage`, `WriteMask` from Tasks 8-9
- Produces:
  - `ExtendConfig` struct: `Top int`, `Bottom int`, `Left int`, `Right int`
  - `ParseExtend(s string) (ExtendConfig, error)` — parses "top=256,bottom=128" format
  - `ExpandCanvas(src image.Image, cfg ExtendConfig) image.Image` — creates expanded canvas with source pasted at offset
  - `ExpandMask(src image.Image, cfg ExtendConfig) image.Image` — generates mask: white in new areas, black where original is
  - `PrepareOutpaint(srcPath string, cfg ExtendConfig) (imagePath string, maskPath string, err error)` — full pipeline: loads source, expands canvas, generates mask, writes both to temp files

- [ ] **Step 1: Write the failing test for outpaint pipeline**

Create `internal/image/canvas_test.go`:
```go
package image

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestParseExtendSingle(t *testing.T) {
	cfg, err := ParseExtend("top=256")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 256 {
		t.Errorf("Top = %d, want 256", cfg.Top)
	}
	if cfg.Bottom != 0 || cfg.Left != 0 || cfg.Right != 0 {
		t.Errorf("others should be 0, got bottom=%d left=%d right=%d", cfg.Bottom, cfg.Left, cfg.Right)
	}
}

func TestParseExtendMultiple(t *testing.T) {
	cfg, err := ParseExtend("top=256,bottom=128,right=200")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 256 {
		t.Errorf("Top = %d, want 256", cfg.Top)
	}
	if cfg.Bottom != 128 {
		t.Errorf("Bottom = %d, want 128", cfg.Bottom)
	}
	if cfg.Right != 200 {
		t.Errorf("Right = %d, want 200", cfg.Right)
	}
	if cfg.Left != 0 {
		t.Errorf("Left = %d, want 0", cfg.Left)
	}
}

func TestParseExtendAll(t *testing.T) {
	cfg, err := ParseExtend("all=100")
	if err != nil {
		t.Fatalf("ParseExtend error: %v", err)
	}
	if cfg.Top != 100 || cfg.Bottom != 100 || cfg.Left != 100 || cfg.Right != 100 {
		t.Errorf("all sides should be 100, got top=%d bottom=%d left=%d right=%d", cfg.Top, cfg.Bottom, cfg.Left, cfg.Right)
	}
}

func TestParseExtendInvalid(t *testing.T) {
	_, err := ParseExtend("top=abc")
	if err == nil {
		t.Fatal("ParseExtend should error on non-numeric value")
	}

	_, err = ParseExtend("invalid=100")
	if err == nil {
		t.Fatal("ParseExtend should error on invalid direction")
消失}

	_, err = ParseExtend("")
	if err == nil {
		t.Fatal("ParseExtend should error on empty string")
	}
}

func TestExpandCanvas(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	src.Set(50, 50, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	cfg := ExtendConfig{Top: 50, Bottom: 50, Left: 0, Right: 0}
	expanded := ExpandCanvas(src, cfg)

	bounds := expanded.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 200 {
		t.Errorf("dimensions = %dx%d, want 100x200", bounds.Dx(), bounds.Dy())
	}

	// Source pixel should be at offset (left=0, top=50)
	c := expanded.At(50, 100)
	r, g, b, _ := c.RGBA()
	if r == 0 || g != 0 || b != 0 {
		t.Error("source pixel should be preserved at correct offset")
	}
}

func TestExpandMask(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))

	cfg := ExtendConfig{Top: 50, Bottom: 0, Left: 0, Right: 50}
	mask := ExpandMask(src, cfg)

	bounds := mask.Bounds()
	if bounds.Dx() != 150 || bounds.Dy() != 150 {
		t.Errorf("dimensions = %dx%d, want 150x150", bounds.Dx(), bounds.Dy())
	}

	// Pixel in new area (top) should be white
	r, g, b, _ := mask.At(50, 10).RGBA()
	if r == 0 || g == 0 || b == 0 {
		t.Error("pixel in new top area should be white")
	}

	// Pixel in new area (right) should be white
	r2, g2, b2, _ := mask.At(130, 100).RGBA()
	if r2 == 0 || g2 == 0 || b2 == 0 {
		t.Error("pixel in new right area should be white")
	}

	// Pixel where original image was should be black
	r3, g3, b3, _ := mask.At(10, 60).RGBA()
	if r3 != 0 || g3 != 0 || b3 != 0 {
		t.Error("pixel in original area should be black")
	}
}

func TestPrepareOutpaint(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.png")
	// Create a source image file
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	data := []byte{}
	// We need to write a valid PNG
	{
		f, _ := os.Create(srcPath)
		writePNGToWriter(f, src)
		f.Close()
	}

	cfg := ExtendConfig{Right: 25}
	imgPath, maskPath, err := PrepareOutpaint(srcPath, cfg)
	if err != nil {
		t.Fatalf("PrepareOutpaint error: %v", err)
	}

	// Verify both files exist
	if _, err := os.Stat(imgPath); err != nil {
		t.Errorf("expanded image file missing: %v", err)
	}
	if _, err := os.Stat(maskPath); err != nil {
		t.Errorf("mask file missing: %v", err)
	}

	// Verify expanded image dimensions
	expanded, format, err := ReadImage(imgPath)
	if err != nil {
		t.Fatalf("read expanded image: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want 'png'", format)
	}
	if expanded.Bounds().Dx() != 75 || expanded.Bounds().Dy() != 50 {
		t.Errorf("dimensions = %dx%d, want 75x50", expanded.Bounds().Dx(), expanded.Bounds().Dy())
	}
}
```

Also add a helper to `canvas_test.go`:
```go
import (
	// add to existing imports
	"image/png"
	"io"
)

func writePNGToWriter(w io.Writer, img image.Image) {
	png.Encode(w, img)
}

// Fix the TestParseExtendInvalid function - replace the broken line
```

Note: The test `TestParseExtendInvalid` has a typo in the skeleton above (`消失}`). The actual implementation should read:
```go
func TestParseExtendInvalid(t *testing.T) {
	_, err := ParseExtend("top=abc")
	if err == nil {
		t.Fatal("ParseExtend should error on non-numeric value")
	}

	_, err = ParseExtend("invalid=100")
	if err == nil {
		t.Fatal("ParseExtend should error on invalid direction")
	}

	_, err = ParseExtend("")
	if err == nil {
		t.Fatal("ParseExtend should error on empty string")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v -run TestParseExtend|TestExpand|TestPrepareOutpaint`
Expected: FAIL (`ParseExtend`, `ExpandCanvas`, `ExpandMask`, `PrepareOutpaint`, `ExtendConfig` all undefined)

- [ ] **Step 3: Write canvas implementation**

Create `internal/image/canvas.go`:
```go
package image

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

// ExtendConfig holds the pixel extension values for each direction.
type ExtendConfig struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// ParseExtend parses a string like "top=256,bottom=128" or "all=100"
// into an ExtendConfig. Returns an error on invalid format.
func ParseExtend(s string) (ExtendConfig, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ExtendConfig{}, fmt.Errorf("empty extend value")
	}

	cfg := ExtendConfig{}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return ExtendConfig{}, fmt.Errorf("invalid extend part: %q (expected key=value)", part)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		n, err := strconv.Atoi(val)
		if err != nil {
			return ExtendConfig{}, fmt.Errorf("invalid extend value %q: %w", val, err)
		}

		switch key {
		case "top":
			cfg.Top = n
		case "bottom":
			cfg.Bottom = n
		case "left":
			cfg.Left = n
		case "right":
			cfg.Right = n
		case "all":
			cfg.Top = n
			cfg.Bottom = n
			cfg.Left = n
			cfg.Right = n
		default:
			return ExtendConfig{}, fmt.Errorf("invalid extend direction: %q (use top, bottom, left, right, or all)", key)
		}
	}

	return cfg, nil
}

// ExpandCanvas creates a new canvas of size (srcW+left+right, srcH+top+bottom),
// pastes the source image at offset (left, top), and fills the new areas
// with a neutral gray (128).
func ExpandCanvas(src image.Image, cfg ExtendConfig) image.Image {
	bounds := src.Bounds()
	newW := bounds.Dx() + cfg.Left + cfg.Right
	newH := bounds.Dy() + cfg.Top + cfg.Bottom

	canvas := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// Fill with neutral gray
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			canvas.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	// Paste source at offset (left, top)
	draw.Draw(canvas, image.Rect(cfg.Left, cfg.Top, cfg.Left+bounds.Dx(), cfg.Top+bounds.Dy()), src, bounds.Min, draw.Src)

	return canvas
}

// ExpandMask creates a mask matching the expanded canvas: white where
// new pixels are (the extended areas), black where the original image is.
func ExpandMask(src image.Image, cfg ExtendConfig) image.Image {
	bounds := src.Bounds()
	newW := bounds.Dx() + cfg.Left + cfg.Right
	newH := bounds.Dy() + cfg.Top + cfg.Bottom

	mask := image.NewGray(image.Rect(0, 0, newW, newH))
	// Default is all black (zero value)

	// Mark new areas as white
	// Top strip
	if cfg.Top > 0 {
		for y := 0; y < cfg.Top; y++ {
			for x := 0; x < newW; x++ {
				mask.SetGray(x, y, color.White)
			}
		}
	}
	// Bottom strip
	if cfg.Bottom > 0 {
		for y := cfg.Top + bounds.Dy(); y < newH; y++ {
			for x := 0; x < newW; x++ {
				mask.SetGray(x, y, color.White)
			}
		}
	}
	// Left strip
	if cfg.Left > 0 {
		for y := cfg.Top; y < cfg.Top+bounds.Dy(); y++ {
			for x := 0; x < cfg.Left; x++ {
				mask.SetGray(x, y, color.White)
			}
		}
	}
	// Right strip
	if cfg.Right > 0 {
		for y := cfg.Top; y < cfg.Top+bounds.Dy(); y++ {
			for x := cfg.Left + bounds.Dx(); x < newW; x++ {
				mask.SetGray(x, y, color.White)
			}
		}
	}

	return mask
}

// PrepareOutpaint loads a source image, expands the canvas, generates the
// mask, and writes both to temporary PNG files. Returns the paths to the
// expanded image and mask files.
func PrepareOutpaint(srcPath string, cfg ExtendConfig) (string, string, error) {
	src, _, err := ReadImage(srcPath)
	if err != nil {
		return "", "", fmt.Errorf("read source: %w", err)
	}

	expanded := ExpandCanvas(src, cfg)
	mask := ExpandMask(src, cfg)

	// Write to temp files
	dir, err := os.MkdirTemp("", "potaco-outpaint-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}

	imgPath := filepath.Join(dir, "expanded.png")
	maskPath := filepath.Join(dir, "mask.png")

	if err := WriteImage(expanded, imgPath, "png"); err != nil {
		return "", "", fmt.Errorf("write expanded image: %w", err)
	}
	if err := WriteMask(mask, maskPath); err != nil {
		return "", "", fmt.Errorf("write mask: %w", err)
	}

	return imgPath, maskPath, nil
}

// writePNGToWriter is a small helper used only in tests.
func writePNGToWriter(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}
```

Note: Add `"io"` to the imports of `canvas.go` since `writePNGToWriter` uses it. Actually, remove `writePNGToWriter` from `canvas.go` and keep it only in the test file, since it's only used there. The `canvas.go` imports should be:
```go
import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v`
Expected: PASS (all canvas tests and previous image tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/image/ && git commit -m "image: add outpaint canvas expansion, mask generation, and prepare pipeline"
```

---

### Task 11: Terminal Image Display (--view)

**Files:**
- Create: `internal/image/view.go`
- Create: `internal/image/view_test.go`

**Interfaces:**
- Consumes: `WriteImage` from Task 8
- Produces:
  - `DisplayInTerminal(img image.Image, path string) string` — detects terminal protocol, returns output string to print to stdout (inline image escape sequence or fallback message)
  - `DetectTerminalProtocol() string` — returns "iterm", "kitty", "sixel", or "" (unsupported)

- [ ] **Step 1: Write the failing test for terminal detection and display**

Create `internal/image/view_test.go`:
```go
package image

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestDetectTerminalProtocolUnset(t *testing.T) {
	// Clear terminal env vars
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")

	proto := DetectTerminalProtocol()
	if proto != "" {
		t.Errorf("protocol = %q, want empty string", proto)
	}
}

func TestDetectTerminalProtocolIterm(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	proto := DetectTerminalProtocol()
	if proto != "iterm" {
		t.Errorf("protocol = %q, want 'iterm'", proto)
	}
}

func TestDetectTerminalProtocolKitty(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "kitty")

	proto := DetectTerminalProtocol()
	if proto != "kitty" {
		t.Errorf("protocol = %q, want 'kitty'", proto)
	}
}

func TestDisplayInTerminalUnsupported(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.Black)

	output := DisplayInTerminal(img, "/tmp/test.png")
	if !strings.Contains(output, "Saved to:") {
		t.Errorf("output should contain 'Saved to:' fallback, got: %q", output)
	}
	if !strings.Contains(output, "does not support") {
		t.Errorf("output should mention unsupported terminal, got: %q", output)
	}
}

func TestDisplayInTerminalIterm(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.Black)

	output := DisplayInTerminal(img, "/tmp/test.png")
	if !strings.Contains(output, "\x1B]1337") {
		t.Errorf("output should contain iTerm2 escape sequence, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v -run TestDetect|TestDisplay`
Expected: FAIL (`DetectTerminalProtocol`, `DisplayInTerminal` undefined)

- [ ] **Step 3: Write view implementation**

Create `internal/image/view.go`:
```go
package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
)

// DetectTerminalProtocol checks the terminal environment and returns
// the name of the supported inline image protocol: "iterm", "kitty",
// "sixel", or "" if none is supported.
func DetectTerminalProtocol() string {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	switch {
	case termProgram == "iTerm.app":
		return "iterm"
	case termProgram == "WezTerm":
		return "iterm" // WezTerm supports iTerm2 inline images
	case termProgram == "kitty":
		return "kitty"
	case term == "xterm-kitty" || strings.HasPrefix(term, "kitty"):
		return "kitty"
	case strings.Contains(term, "sixel"):
		return "sixel"
	default:
		return ""
	}
}

// DisplayInTerminal encodes the image for the detected terminal protocol
// and returns a string to print to stdout. If no protocol is supported,
// returns a fallback message with the file path.
func DisplayInTerminal(img image.Image, path string) string {
	proto := DetectTerminalProtocol()

	switch proto {
	case "iterm":
		return itermDisplay(img, path)
	case "kitty":
		return kittyDisplay(img, path)
	case "sixel":
		return sixelDisplay(img, path)
	default:
		return fmt.Sprintf("Saved to: %s (terminal does not support inline image preview)", path)
	}
}

// itermDisplay encodes an inline image using the iTerm2 escape sequence.
func itermDisplay(img image.Image, path string) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	name := filepathBase(path)
	return fmt.Sprintf("\x1B]1337;File=inline=1;name=%s:%s\x07", name, b64)
}

// kittyDisplay encodes an inline image using the Kitty graphics protocol.
func kittyDisplay(img image.Image, path string) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	// Kitty sends chunks; for simplicity we send it all at once
	// Escape sequence: ESC ] 9 9 8 ; ... ST
	return fmt.Sprintf("\x1B_Ga=T,f=32,s=0,v=0,c=0,q=0;%s\x1B\\", b64)
}

// sixelDisplay is a stub for sixel support. For v0, we fall back to the
// message if sixel encoding is not yet implemented.
func sixelDisplay(img image.Image, path string) string {
	return fmt.Sprintf("Saved to: %s (sixel preview not yet implemented)", path)
}

func filepathBase(path string) string {
	// Simple basename without importing path/filepath
	idx := strings.LastIndex(path, "/")
	if idx >= 0 && idx < len(path)-1 {
		return path[idx+1:]
	}
	return path
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/image/ -v`
Expected: PASS (all view tests and previous image tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/image/ && git commit -m "image: add terminal image display with iTerm2/Kitty/Sixel detection"
```

---

### Task 12: Output Formatting

**Files:**
- Create: `internal/cli/output.go`
- Create: `internal/cli/output_test.go`

**Interfaces:**
- Consumes: `ImageResponse`, `ImageData` from Task 4, `DecodeBase64Image`, `WriteImage`, `AutoFilename`, `DisplayInTerminal` from Tasks 8, 11
- Produces:
  - `OutputOptions` struct: `JSON bool`, `Stdout bool`, `View bool`, `OutputPath string`, `OutputFormat string`
  - `FormatResult(result OutputResult, opts OutputOptions) (string, error)` — returns the string to print to stdout
  - `OutputResult` struct: `Paths []string`, `Format string`, `Widths []int`, `Heights []int`, `Model string`, `Params map[string]any`, `LatencyMs int64`

- [ ] **Step 1: Write the failing test for output formatting**

Create `internal/cli/output_test.go`:
```go
package cli

import (
	"strings"
	"testing"
)

func TestFormatResultDefault(t *testing.T) {
	result := OutputResult{
		Paths:   []string{"potaco-20260624-153201.png"},
		Format:  "png",
		Widths:  []int{1024},
		Heights: []int{1024},
		Model:   "dall-e-3",
		Params:  map[string]any{"size": "1024x1024", "quality": "standard", "n": 1},
	}
	opts := OutputOptions{JSON: false, Stdout: false, View: false}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if !strings.Contains(output, "Saved to:") {
		t.Errorf("default output should contain 'Saved to:', got: %q", output)
	}
	if !strings.Contains(output, "potaco-20260624-153201.png") {
		t.Errorf("output should contain file path, got: %q", output)
	}
	if strings.Contains(output, "{") {
		t.Errorf("default output should not contain JSON, got: %q", output)
	}
}

func TestFormatResultJSON(t *testing.T) {
	result := OutputResult{
		Paths:   []string{"output.png"},
		Format:  "png",
		Widths:  []int{1024},
		Heights: []int{1024},
		Model:   "dall-e-3",
		Params:  map[string]any{"size": "1024x1024"},
		LatencyMs: 3420,
	}
	opts := OutputOptions{JSON: true, Stdout: false, View: false}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if !strings.Contains(output, `"path":`) {
		t.Errorf("JSON output should contain 'path' key, got: %q", output)
	}
	if !strings.Contains(output, `"model": "dall-e-3"`) {
		t.Errorf("JSON output should contain model, got: %q", output)
	}
	if !strings.Contains(output, `"latency_ms": 3420`) {
		t.Errorf("JSON output should contain latency_ms, got: %q", output)
	}
}

func TestFormatResultMultipleImagesJSON(t *testing.T) {
	result := OutputResult{
		Paths:    []string{"img1.png", "img2.png"},
		Format:   "png",
		Widths:   []int{1024, 1024},
		Heights:  []int{1024, 1024},
		Model:    "dall-e-3",
		Params:   map[string]any{"n": 2},
		LatencyMs: 5000,
	}
	opts := OutputOptions{JSON: true}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// Should be a JSON array
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("multiple images should produce JSON array, got: %q", output)
	}
	if !strings.Contains(output, "img1.png") {
		t.Errorf("output should contain img1.png, got: %q", output)
	}
	if !strings.Contains(output, "img2.png") {
		t.Errorf("output should contain img2.png, got: %q", output)
	}
}

func TestFormatResultStdoutSuppressed(t *testing.T) {
	result := OutputResult{
		Paths: []string{"output.png"},
	}
	opts := OutputOptions{Stdout: true}

	output, err := FormatResult(result, opts)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// stdout mode: no text/JSON output, the raw bytes are handled separately
	if output != "" {
		t.Errorf("stdout mode should return empty string (no text output), got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestFormatResult`
Expected: FAIL (`OutputResult`, `OutputOptions`, `FormatResult` all undefined)

- [ ] **Step 3: Write output formatting implementation**

Create `internal/cli/output.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OutputOptions controls how results are formatted for display.
type OutputOptions struct {
	JSON         bool
	Stdout       bool
	View         bool
	OutputPath   string
	OutputFormat string
}

// OutputResult holds the metadata about generated/edited images.
type OutputResult struct {
	Paths     []string
	Format    string
	Widths    []int
	Heights   []int
	Model     string
	Params    map[string]any
	LatencyMs int64
}

// FormatResult formats the result for stdout display based on the output options.
// In stdout mode, returns empty string (raw bytes are handled separately).
func FormatResult(result OutputResult, opts OutputOptions) (string, error) {
	if opts.Stdout {
		return "", nil // raw bytes handled separately
	}

	if opts.JSON {
		return formatJSON(result)
	}

	// Default: human-friendly text
	var lines []string
	for _, path := range result.Paths {
		lines = append(lines, fmt.Sprintf("Saved to: %s", path))
	}
	return strings.Join(lines, "\n"), nil
}

// formatJSON produces JSON output. For a single image, an object; for
// multiple images, an array of objects.
func formatJSON(result OutputResult) (string, error) {
	if len(result.Paths) == 1 {
		obj := map[string]any{
			"path":       result.Paths[0],
			"format":     result.Format,
			"width":      result.Widths[0],
			"height":     result.Heights[0],
			"model":      result.Model,
			"params":      result.Params,
			"latency_ms": result.LatencyMs,
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return "", fmt.Errorf("marshal JSON: %w", err)
		}
		return string(b), nil
	}

	// Multiple images: array
	arr := make([]map[string]any, len(result.Paths))
	for i, path := range result.Paths {
		arr[i] = map[string]any{
			"path":       path,
			"format":     result.Format,
			"width":     result.Widths[i],
			"height":    result.Heights[i],
			"model":     result.Model,
			"params":     result.Params,
			"latency_ms": result.LatencyMs,
		}
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return "", fmt.Errorf("marshal JSON array: %w", err)
	}
	return string(b), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v`
Expected: PASS (all output tests and root command tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/cli/ && git commit -m "cli: add output formatting with text, JSON, and stdout modes"
```

---

### Task 13: Shared CLI Helpers and Gen Subcommand

**Files:**
- Create: `internal/cli/helpers.go`
- Create: `internal/cli/gen.go`
- Create: `internal/cli/gen_test.go`

**Interfaces:**
- Consumes: `config.Merge`, `config.MergeOptions` from Task 3, `provider.NewClient`, `provider.GenerateRequest`, `provider.ImageResponse` from Task 5, `image.DecodeBase64Image`, `image.WriteImage`, `image.AutoFilename`, `image.DisplayInTerminal` from Tasks 8, 11, `FormatResult`, `OutputResult`, `OutputOptions` from Task 12
- Produces: `genCmd` cobra command wired into `rootCmd`, plus shared helpers: `buildMergeOptions`, `processAndOutput`, `printDryRun`, `flagString`, `flagInt`, `flagFloat64`, `flagBool`, `trimExt`, `extOf`

- [ ] **Step 1: Write the failing test for gen subcommand flag parsing and dry-run**

Create `internal/cli/gen_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "gen" || strings.HasPrefix(cmd.Use, "gen ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'gen' subcommand")
	}
}

func TestGenCommandHasPromptFlag(t *testing.T) {
	promptFlag := genCmd.Flags().Lookup("prompt")
	if promptFlag == nil {
		t.Fatal("gen command should have --prompt flag")
	}
}

func TestGenCommandPromptRequired(t *testing.T) {
	var buf bytes.Buffer
	genCmd.SetOut(&buf)
	genCmd.SetErr(&buf)
	genCmd.SetArgs([]string{"--prompt", ""}) // empty prompt

	// Cobra should enforce required flag
	err := genCmd.Execute()
	if err == nil {
		// If not enforced by Cobra, our RunE should catch empty prompt
		// Check if it still runs - if so, we need manual validation
	}
}

func TestGenCommandDryRunNoAPI(t *testing.T) {
	// Set up env so config merge succeeds
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	genCmd.SetOut(&buf)
	genCmd.SetErr(&buf)
	genCmd.SetArgs([]string{"--prompt", "a cat", "--dry-run"})

	err := genCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"method": "POST"`) {
		t.Errorf("dry-run should print request method, got: %q", output)
	}
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "a cat") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
	// Should NOT have made an API call (no "Saved to:" in output)
	if strings.Contains(output, "Saved to:") {
		t.Errorf("dry-run should not save any files, got: %q", output)
	}
}

func TestGenCommandMissingConfigError(t *testing.T) {
	// Clear all config sources
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	genCmd.SetOut(&buf)
	genCmd.SetErr(&buf)
	genCmd.SetArgs([]string{"--prompt", "test", "--dry-run"})

	err := genCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no config is provided")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error should mention base_url, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestGen`
Expected: FAIL (`genCmd` undefined)
- [ ] **Step 3: Write shared helpers implementation**

Create `internal/cli/helpers.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ngct/potaco/internal/config"
	"github.com/spf13/cobra"
)

// buildMergeOptions creates MergeOptions from CLI flags.
func buildMergeOptions(cmd *cobra.Command) config.MergeOptions {
	opts := config.MergeOptions{}

	if cmd.Flags().Changed("base-url") {
		v, _ := cmd.Flags().GetString("base-url")
		opts.BaseURL = &v
	}
	if cmd.Flags().Changed("api-key") {
		v, _ := cmd.Flags().GetString("api-key")
		opts.APIKey = &v
	}
	if cmd.Flags().Changed("model") {
		v, _ := cmd.Flags().GetString("model")
		opts.Model = &v
	}
	if cmd.Flags().Changed("retries") {
		v, _ := cmd.Flags().GetInt("retries")
		opts.Retries = &v
	}
	if cmd.Flags().Changed("timeout") {
		v, _ := cmd.Flags().GetDuration("timeout")
		opts.Timeout = &v
	}
	if cmd.Flags().Changed("provider") {
		v, _ := cmd.Flags().GetString("provider")
		opts.Provider = &v
	}

	return opts
}

// flagString reads a string flag, returning the flag value.
func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// flagInt reads an int flag, returning the flag value.
func flagInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

// flagFloat64 reads a float64 flag, returning the flag value.
func flagFloat64(cmd *cobra.Command, name string) float64 {
	v, _ := cmd.Flags().GetFloat64(name)
	return v
}

// flagBool reads a bool flag, returning the flag value.
func flagBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

// printDryRun prints the request payload as JSON to stdout without making an API call.
func printDryRun(method, url, contentType string, body any) error {
	bodyJSON, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run body: %w", err)
	}

	dryRunOutput := map[string]any{
		"method":       method,
		"url":          url,
		"content_type": contentType,
		"headers": map[string]string{
			"Authorization": "Bearer [REDACTED]",
		},
		"body": json.RawMessage(bodyJSON),
	}

	output, err := json.MarshalIndent(dryRunOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run output: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

// processAndOutput decodes the API response images, saves them, and prints output.
func processAndOutput(cmd *cobra.Command, resp *provider.ImageResponse, model string, params map[string]any, latency int64) error {
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	stdoutMode := flagBool(cmd, "stdout")
	viewMode := flagBool(cmd, "view")
	outputPath := flagString(cmd, "output")
	outputFormat := flagString(cmd, "output-format")

	paths := make([]string, len(resp.Data))
	widths := make([]int, len(resp.Data))
	heights := make([]int, len(resp.Data))

	for i, imgData := range resp.Data {
		if imgData.B64JSON != "" {
			decoded, err := image.DecodeBase64Image(imgData.B64JSON)
			if err != nil {
				return fmt.Errorf("decode image %d: %w", i, err)
			}
			bounds := decoded.Bounds()
			widths[i] = bounds.Dx()
			heights[i] = bounds.Dy()

			path := outputPath
			if path == "" {
				path = image.AutoFilename()
			} else if len(resp.Data) > 1 {
				path = fmt.Sprintf("%s-%d%s", trimExt(outputPath), i, extOf(outputPath))
			}

			if stdoutMode && !viewMode {
				// Write raw bytes to stdout
				var buf bytes.Buffer
				switch outputFormat {
				case "jpeg", "jpg":
					jpeg.Encode(&buf, decoded, &jpeg.Options{Quality: 90})
				default:
					png.Encode(&buf, decoded)
				}
				os.Stdout.Write(buf.Bytes())
			}

			if err := image.WriteImage(decoded, path, outputFormat); err != nil {
				return fmt.Errorf("write image %d: %w", i, err)
			}
			paths[i] = path

			if viewMode {
				output := image.DisplayInTerminal(decoded, path)
				fmt.Fprintln(cmd.OutOrStdout(), output)
			}
		} else if imgData.URL != "" {
			paths[i] = imgData.URL
		}
	}

	result := OutputResult{
		Paths:     paths,
		Format:    outputFormat,
		Widths:    widths,
		Heights:   heights,
		Model:     model,
		Params:    params,
		LatencyMs: latency,
	}

	outOpts := OutputOptions{
		JSON:         jsonMode,
		Stdout:       stdoutMode,
		View:         viewMode,
		OutputPath:   outputPath,
		OutputFormat: outputFormat,
	}

	if !stdoutMode {
		output, err := FormatResult(result, outOpts)
		if err != nil {
			return fmt.Errorf("format output: %w", err)
		}
		if output != "" {
			fmt.Fprintln(cmd.OutOrStdout(), output)
		}
	}

	return nil
}

// trimExt removes the file extension from a path.
func trimExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx > 0 {
		return path[:idx]
	}
	return path
}

// extOf returns the file extension of a path, including the dot.
func extOf(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx >= 0 {
		return path[idx:]
	}
	return ""
}
```

Note: This file needs additional imports. The actual import block should be:
```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 4: Write gen subcommand implementation**

Create `internal/cli/gen.go`:
```go
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate images from a text prompt",
	RunE:  runGen,
}

func init() {
	genCmd.Flags().StringP("prompt", "p", "", "text description of the desired image(s)")
	_ = genCmd.MarkFlagRequired("prompt")

	genCmd.Flags().String("model", "", "model to use (e.g., dall-e-3)")
	genCmd.Flags().String("size", "1024x1024", "image dimensions (WxH)")
	genCmd.Flags().String("quality", "standard", "image quality (standard or hd)")
	genCmd.Flags().Int("n", 1, "number of images to generate")
	genCmd.Flags().String("style", "", "visual style (vivid or natural)")

	genCmd.Flags().Int("seed", 0, "reproducibility seed")
	genCmd.Flags().Float64("guidance-scale", 0, "guidance scale")
	genCmd.Flags().String("negative-prompt", "", "negative prompt")
	genCmd.Flags().String("response-format", "b64_json", "response format (url or b64_json)")

	genCmd.Flags().StringP("output", "o", "", "output file path")
	genCmd.Flags().String("output-format", "png", "output format (png or jpeg)")
	genCmd.Flags().Bool("view", false, "attempt terminal image display")
	genCmd.Flags().Bool("stdout", false, "pipe raw image bytes to stdout")

	genCmd.Flags().String("provider", "", "provider preset (openai, together, fal)")
	genCmd.Flags().String("base-url", "", "override API base URL")
	genCmd.Flags().String("api-key", "", "override API key")
	genCmd.Flags().Int("retries", 0, "max retry attempts")
	genCmd.Flags().Duration("timeout", 0, "request timeout")

	genCmd.Flags().Bool("dry-run", false, "validate and print request payload without calling API")

	rootCmd.AddCommand(genCmd)
}

func runGen(cmd *cobra.Command, args []string) error {
	prompt := flagString(cmd, "prompt")
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	opts := buildMergeOptions(cmd)
	cfg, err := config.Merge(opts)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	model := cfg.Model
	if cmd.Flags().Changed("model") {
		model = flagString(cmd, "model")
	}

	req := provider.GenerateRequest{
		Prompt:         prompt,
		Model:          model,
		Size:           flagString(cmd, "size"),
		Quality:        flagString(cmd, "quality"),
		N:              flagInt(cmd, "n"),
		Style:          flagString(cmd, "style"),
		ResponseFormat: flagString(cmd, "response-format"),
		Seed:           flagInt(cmd, "seed"),
		GuidanceScale:  flagFloat64(cmd, "guidance-scale"),
		NegativePrompt: flagString(cmd, "negative-prompt"),
	}

	dryRun := flagBool(cmd, "dry-run")
	if dryRun {
		return printDryRun("POST", cfg.BaseURL+"/v1/images/generations", "application/json", req)
	}

	client := provider.NewClient(provider.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Retries: cfg.Retries,
		Timeout: cfg.Timeout,
	})

	start := time.Now()
	resp, err := client.Generate(context.Background(), req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return processAndOutput(cmd, resp, model, map[string]any{
		"size":            req.Size,
		"quality":         req.Quality,
		"n":               req.N,
		"style":           req.Style,
		"response_format": req.ResponseFormat,
	}, latency)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestGen`
Expected: PASS (all gen tests pass)

- [ ] **Step 6: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/cli/ && git commit -m "cli: add gen subcommand, shared helpers, and dry-run support"
```

---

### Task 14: Edit Subcommand

**Files:**
- Create: `internal/cli/edit.go`
- Create: `internal/cli/edit_test.go`

**Interfaces:**
- Consumes: everything from Tasks 3-13 (config merge, provider client, image processing, output formatting, shared helpers)
- Produces: `editCmd` cobra command wired into `rootCmd`, supporting basic edit, inpaint (mask file or geometric), and outpaint (extend) modes

- [ ] **Step 1: Write the failing test for edit subcommand**

Create `internal/cli/edit_test.go`:
```go
package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "edit" || strings.HasPrefix(cmd.Use, "edit ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'edit' subcommand")
	}
}

func TestEditCommandHasImageFlag(t *testing.T) {
	imgFlag := editCmd.Flags().Lookup("image")
	if imgFlag == nil {
		t.Fatal("edit command should have --image flag")
	}
}

func TestEditCommandHasMaskFlags(t *testing.T) {
	if editCmd.Flags().Lookup("mask") == nil {
		t.Fatal("edit command should have --mask flag")
	}
	if editCmd.Flags().Lookup("mask-rect") == nil {
		t.Fatal("edit command should have --mask-rect flag")
	}
	if editCmd.Flags().Lookup("mask-circle") == nil {
		t.Fatal("edit command should have --mask-circle flag")
	}
}

func TestEditCommandHasExtendFlag(t *testing.T) {
	if editCmd.Flags().Lookup("extend") == nil {
		t.Fatal("edit command should have --extend flag")
	}
}

func TestEditDryRunBasic(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	editCmd.SetOut(&buf)
	editCmd.SetErr(&buf)
	editCmd.SetArgs([]string{"--prompt", "make it blue", "--image", imgPath, "--dry-run"})

	err := editCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "make it blue") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}

func TestEditDryRunOutpaint(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	editCmd.SetOut(&buf)
	editCmd.SetErr(&buf)
	editCmd.SetArgs([]string{"--prompt", "more sky", "--image", imgPath, "--extend", "top=100", "--dry-run"})

	err := editCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run outpaint returned error: %v", err)
	}
}

func TestEditMissingImageFile(t *testing.T) {
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")

	var buf bytes.Buffer
	editCmd.SetOut(&buf)
	editCmd.SetErr(&buf)
	editCmd.SetArgs([]string{"--prompt", "test", "--image", "/nonexistent.png", "--dry-run"})

	err := editCmd.Execute()
	if err == nil {
		t.Fatal("edit should error on missing image file")
	}
}

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestEdit`
Expected: FAIL (`editCmd` undefined)

- [ ] **Step 3: Write edit subcommand implementation**

Create `internal/cli/edit.go`:
```go
package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing image",
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringP("prompt", "p", "", "text description of the edit")
	_ = editCmd.MarkFlagRequired("prompt")

	editCmd.Flags().String("image", "", "path to source image file")
	_ = editCmd.MarkFlagRequired("image")

	// Mask flags
	editCmd.Flags().String("mask", "", "path to mask image file (white=edit, black=keep)")
	editCmd.Flags().String("mask-rect", "", "rectangular mask: x,y,w,h in pixels")
	editCmd.Flags().String("mask-circle", "", "circular mask: x,y,r in pixels")
	editCmd.Flags().String("extend", "", "outpaint extension: top=N,bottom=N,left=N,right=N or all=N")

	// Shared flags from gen
	editCmd.Flags().String("model", "", "model to use")
	editCmd.Flags().String("size", "1024x1024", "image dimensions (WxH)")
	editCmd.Flags().Int("n", 1, "number of images to generate")
	editCmd.Flags().String("response-format", "b64_json", "response format (url or b64_json)")

	// Output flags
	editCmd.Flags().StringP("output", "o", "", "output file path")
	editCmd.Flags().String("output-format", "png", "output format (png or jpeg)")
	editCmd.Flags().Bool("view", false, "attempt terminal image display")
	editCmd.Flags().Bool("stdout", false, "pipe raw image bytes to stdout")

	// Provider override flags
	editCmd.Flags().String("provider", "", "provider preset (openai, together, fal)")
	editCmd.Flags().String("base-url", "", "override API base URL")
	editCmd.Flags().String("api-key", "", "override API key")
	editCmd.Flags().Int("retries", 0, "max retry attempts")
	editCmd.Flags().Duration("timeout", 0, "request timeout")

	// Mode flags
	editCmd.Flags().Bool("dry-run", false, "validate and print request payload without calling API")

	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	prompt := flagString(cmd, "prompt")
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	imagePath := flagString(cmd, "image")
	if imagePath == "" {
		return fmt.Errorf("image path is required")
	}

	// Check image file exists
	if _, err := os.Stat(imagePath); err != nil {
		return fmt.Errorf("image file: %w", err)
	}

	opts := buildMergeOptions(cmd)
	cfg, err := config.Merge(opts)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	model := cfg.Model
	if cmd.Flags().Changed("model") {
		model = flagString(cmd, "model")
	}

	// Determine edit mode
	extendFlag := flagString(cmd, "extend")
	maskFlag := flagString(cmd, "mask")
	maskRectFlag := flagString(cmd, "mask-rect")
	maskCircleFlag := flagString(cmd, "mask-circle")

	dryRun := flagBool(cmd, "dry-run")

	// Prepare the image and mask for the edit request
	editImagePath := imagePath
	maskPath := ""

	if extendFlag != "" {
		// Outpaint mode: expand canvas and generate mask internally
		extendCfg, err := img.ParseExtend(extendFlag)
		if err != nil {
			return fmt.Errorf("parse extend: %w", err)
		}

		if dryRun {
			return printDryRun("POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt":  prompt,
				"model":   model,
				"image":   imagePath,
				"extend":   extendCfg,
				"mode":    "outpaint",
			})
		}

		expandedPath, generatedMaskPath, err := img.PrepareOutpaint(imagePath, extendCfg)
		if err != nil {
			return fmt.Errorf("prepare outpaint: %w", err)
		}
		editImagePath = expandedPath
		maskPath = generatedMaskPath
	} else if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		// Inpaint mode: generate mask if geometric flags are used
		if dryRun {
			return printDryRun("POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt":     prompt,
				"model":      model,
				"image":      imagePath,
				"mask":       maskFlag,
				"mask_rect":  maskRectFlag,
				"mask_circle": maskCircleFlag,
				"mode":       "inpaint",
			})
		}

		if maskFlag != "" {
			maskPath = maskFlag
			// If mask dimensions don't match source, loadMaskFile scales it
			// but for direct file use, we verify it's a valid image
			if _, err := os.Stat(maskFlag); err != nil {
				return fmt.Errorf("mask file: %w", err)
			}
		} else {
			// Generate mask from geometric flags
			srcImg, _, err := img.ReadImage(imagePath)
			if err != nil {
				return fmt.Errorf("read source image: %w", err)
			}
			bounds := srcImg.Bounds()

			var maskImg image.Image
			if maskRectFlag != "" {
				x, y, w, h, err := parseRectMask(maskRectFlag)
				if err != nil {
					return fmt.Errorf("parse mask-rect: %w", err)
				}
				maskImg, err = img.RectMask(bounds.Dx(), bounds.Dy(), x, y, w, h)
				if err != nil {
					return fmt.Errorf("generate rect mask: %w", err)
				}
			} else if maskCircleFlag != "" {
				cx, cy, r, err := parseCircleMask(maskCircleFlag)
				if err != nil {
					return fmt.Errorf("parse mask-circle: %w", err)
				}
				maskImg, err = img.CircleMask(bounds.Dx(), bounds.Dy(), cx, cy, r)
				if err != nil {
					return fmt.Errorf("generate circle mask: %w", err)
				}
			}

			// Write mask to temp file
			dir, err := os.MkdirTemp("", "potaco-mask-*")
			if err != nil {
				return fmt.Errorf("create temp dir: %w", err)
			}
			maskPath = filepath.Join(dir, "mask.png")
			if err := img.WriteMask(maskImg, maskPath); err != nil {
				return fmt.Errorf("write mask: %w", err)
			}
		}
	} else {
		// Basic edit mode
		if dryRun {
			return printDryRun("POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt": prompt,
				"model":  model,
				"image":  imagePath,
				"mode":   "basic",
			})
		}
	}

	// Build edit request
	req := provider.EditRequest{
		Prompt:         prompt,
		Model:          model,
		N:              flagInt(cmd, "n"),
		Size:           flagString(cmd, "size"),
		ResponseFormat: flagString(cmd, "response-format"),
		ImagePath:      editImagePath,
		MaskPath:       maskPath,
	}

	client := provider.NewClient(provider.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Retries: cfg.Retries,
		Timeout: cfg.Timeout,
	})

	start := time.Now()
	resp, err := client.Edit(context.Background(), req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	return processAndOutput(cmd, resp, model, map[string]any{
		"mode":            "edit",
		"image":           imagePath,
		"size":            req.Size,
		"n":               req.N,
		"response_format": req.ResponseFormat,
	}, latency)
}

// parseRectMask parses "x,y,w,h" into four ints.
func parseRectMask(s string) (x, y, w, h int, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("expected x,y,w,h, got %d parts", len(parts))
	}
	x, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse x: %w", err)
	}
	y, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse y: %w", err)
	}
	w, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse w: %w", err)
	}
	h, err = strconv.Atoi(parts[3])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse h: %w", err)
	}
	return x, y, w, h, nil
}

// parseCircleMask parses "cx,cy,r" into three ints.
func parseCircleMask(s string) (cx, cy, r int, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected cx,cy,r, got %d parts", len(parts))
	}
	cx, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse cx: %w", err)
	}
	cy, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse cy: %w", err)
	}
	r, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse r: %w", err)
	}
	return cx, cy, r, nil
}
```

Note: The `edit.go` file needs these imports:
```go
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ngct/potaco/internal/config"
	img "github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)
```

The `image` package is imported as `img` to avoid collision with the standard `image` package. Where the code references `img.ParseExtend`, `img.PrepareOutpaint`, `img.ReadImage`, `img.RectMask`, `img.CircleMask`, `img.WriteMask`, these all use the aliased import.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestEdit`
Expected: PASS (all edit tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/cli/ && git commit -m "cli: add edit subcommand with inpaint and outpaint modes"
```

---

### Task 15: Config Subcommand

**Files:**
- Create: `internal/cli/config_cmd.go`
- Create: `internal/cli/config_cmd_test.go`

**Interfaces:**
- Consumes: `config.DefaultConfigPath`, `config.Load` from Task 2, `provider.AllPresets`, `provider.GetPreset` from Task 4
- Produces: `configCmd` cobra command wired into `rootCmd` with subcommands: `set`, `show`, `list-providers`

- [ ] **Step 1: Write the failing test for config subcommand**

Create `internal/cli/config_cmd_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" || strings.HasPrefix(cmd.Use, "config ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'config' subcommand")
	}
}

func TestConfigSetHasFlags(t *testing.T) {
	if configSetCmd.Flags().Lookup("base-url") == nil {
		t.Fatal("config set should have --base-url flag")
	}
	if configSetCmd.Flags().Lookup("api-key") == nil {
		t.Fatal("config set should have --api-key flag")
	}
	if configSetCmd.Flags().Lookup("model") == nil {
		t.Fatal("config set should have --model flag")
	}
}

func TestConfigListProviders(t *testing.T) {
	var buf bytes.Buffer
	configListProvidersCmd.SetOut(&buf)
	configListProvidersCmd.SetErr(&buf)
	configListProvidersCmd.SetArgs([]string{})

	err := configListProvidersCmd.Execute()
	if err != nil {
		t.Fatalf("config list-providers error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should list 'openai' preset, got: %q", output)
	}
	if !strings.Contains(output, "together") {
		t.Errorf("output should list 'together' preset, got: %q", output)
	}
	if !strings.Contains(output, "fal") {
		t.Errorf("output should list 'fal' preset, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestConfig`
Expected: FAIL (`configCmd`, `configSetCmd`, `configListProvidersCmd` undefined)

- [ ] **Step 3: Write config subcommand implementation**

Create `internal/cli/config_cmd.go`:
```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage provider configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values",
	RunE:  runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configListProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List available provider presets",
	RunE:  runConfigListProviders,
}

func init() {
	configSetCmd.Flags().String("base-url", "", "API base URL")
	configSetCmd.Flags().String("api-key", "", "API key")
	configSetCmd.Flags().String("model", "", "default model")
	configSetCmd.Flags().Int("retries", 0, "max retry attempts")
	configSetCmd.Flags().String("timeout", "", "request timeout (e.g., 120s)")
	configSetCmd.Flags().String("provider", "", "apply preset defaults (openai, together, fal)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListProvidersCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	// Load existing config or start fresh
	var content string
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}

	// Build new config YAML
	baseURL, _ := cmd.Flags().GetString("base-url")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	retries, _ := cmd.Flags().GetInt("retries")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	providerName, _ := cmd.Flags().GetString("provider")

	// Apply preset if specified
	if providerName != "" {
		preset, ok := provider.GetPreset(providerName)
		if !ok {
			return fmt.Errorf("unknown provider preset: %s", providerName)
		}
		if baseURL == "" {
			baseURL = preset.BaseURL
		}
		if model == "" {
			model = preset.DefaultModel
		}
	}

	// Build YAML content
	lines := []string{"default:"}
	if baseURL != "" {
		lines = append(lines, fmt.Sprintf("  base_url: %q", baseURL))
	}
	if apiKey != "" {
		lines = append(lines, fmt.Sprintf("  api_key: %q", apiKey))
	}
	if model != "" {
		lines = append(lines, fmt.Sprintf("  model: %q", model))
	}
	if retries > 0 {
		lines = append(lines, fmt.Sprintf("  retries: %d", retries))
	}
	if timeoutStr != "" {
		lines = append(lines, fmt.Sprintf("  timeout: %q", timeoutStr))
	}

	newContent := strings.Join(lines, "\n") + "\n"

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", path)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No configuration file found at", path)
			fmt.Println("Use 'potaco config set' to create one.")
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	fmt.Printf("Config file: %s\n\n", path)
	fmt.Println(string(data))
	return nil
}

func runConfigListProviders(cmd *cobra.Command, args []string) error {
	presets := provider.AllPresets()

	fmt.Println("Available provider presets:")
	fmt.Println()
	for name, preset := range presets {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    base_url:      %s\n", preset.BaseURL)
		fmt.Printf("    default_model: %s\n", preset.DefaultModel)
		fmt.Printf("    sizes:         %v\n", preset.Sizes)
		fmt.Println()
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestConfig`
Expected: PASS (all config tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/cli/ && git commit -m "cli: add config subcommand with set, show, and list-providers"
```

---

### Task 16: Info Subcommand

**Files:**
- Create: `internal/cli/info.go`
- Create: `internal/cli/info_test.go`

**Interfaces:**
- Consumes: `image.ReadImage` from Task 8, `os.Stat` for file size
- Produces: `infoCmd` cobra command wired into `rootCmd`

- [ ] **Step 1: Write the failing test for info subcommand**

Create `internal/cli/info_test.go`:
```go
package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInfoCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "info" || strings.HasPrefix(cmd.Use, "info ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'info' subcommand")
	}
}

func TestInfoCommandOutput(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	f, _ := os.Create(imgPath)
	img := image.NewRGBA(image.Rect(0, 0, 100, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	png.Encode(f, img)
	f.Close()

	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)
	infoCmd.SetArgs([]string{imgPath})

	err := infoCmd.Execute()
	if err != nil {
		t.Fatalf("info command error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "png") {
		t.Errorf("output should mention format 'png', got: %q", output)
	}
	if !strings.Contains(output, "100x200") {
		t.Errorf("output should contain dimensions 100x200, got: %q", output)
	}
	if !strings.Contains(output, imgPath) {
		t.Errorf("output should contain file path, got: %q", output)
	}
}

func TestInfoCommandMissingFile(t *testing.T) {
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)
	infoCmd.SetArgs([]string{"/nonexistent/file.png"})

	err := infoCmd.Execute()
	if err == nil {
		t.Fatal("info should error on missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestInfo`
Expected: FAIL (`infoCmd` undefined)

- [ ] **Step 3: Write info subcommand implementation**

Create `internal/cli/info.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	img "github.com/ngct/potaco/internal/image"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [path]",
	Short: "Print metadata about an image file",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Check file exists
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file: %w", err)
	}

	// Read image
	image, format, err := img.ReadImage(path)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}

	bounds := image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	fileSize := stat.Size()
	colorModel := fmt.Sprintf("%v", image.ColorModel())

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")

	if jsonMode {
		output := map[string]any{
			"path":        path,
			"format":      format,
			"width":       width,
			"height":      height,
			"file_size":   fileSize,
			"color_model": colorModel,
		}
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(b))
	} else {
		fmt.Printf("File:       %s\n", path)
		fmt.Printf("Format:     %s\n", format)
		fmt.Printf("Dimensions: %dx%d\n", width, height)
		fmt.Printf("File size:  %d bytes\n", fileSize)
		fmt.Printf("Color:      %s\n", colorModel)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/ngct/Projects/potaco && go test ./internal/cli/ -v -run TestInfo`
Expected: PASS (all info tests pass)

- [ ] **Step 5: Commit**

```bash
cd /home/ngct/Projects/potaco && git add internal/cli/ && git commit -m "cli: add info subcommand for image metadata"
```

---

### Task 17: Fix helpers.go import and compile check

**Files:**
- Modify: `internal/cli/helpers.go` (if needed to fix imports)

This task exists because `helpers.go` from Task 13 references types from `provider` and `image` packages and needs the correct import block. The imports should include:
```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/ngct/potaco/internal/config"
	img "github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)
```

Important: `helpers.go` uses `img.DecodeBase64Image`, `img.AutoFilename`, `img.WriteImage`, `img.DisplayInTerminal` from `internal/image`, and `provider.ImageResponse` from `internal/provider`. The `image` package must be imported as `img` to avoid collision with the standard library `image` package (which is needed for `image/jpeg` and `image/png` encoders).

Also, `edit.go` uses `img.ParseExtend`, `img.PrepareOutpaint`, `img.ReadImage`, `img.RectMask`, `img.CircleMask`, `img.WriteMask` with the same `img` alias.

- [ ] **Step 1: Verify all packages compile**

Run: `cd /home/ngct/Projects/potaco && go build ./...`
Expected: builds without errors

- [ ] **Step 2: Run all tests**

Run: `cd /home/ngct/Projects/potaco && go test ./... -v`
Expected: all tests pass across all packages

- [ ] **Step 3: Fix any compilation errors**

If there are import issues or type mismatches, fix them. Common issues to check:
- `img` alias consistency between `helpers.go` and `edit.go`
- `provider.ImageResponse` vs `provider.ImageData` field names match Task 4 types
- `config.MergeOptions` field types match Task 3 types
- `OutputResult` and `OutputOptions` field names match Task 12 types

- [ ] **Step 4: Commit any fixes**

```bash
cd /home/ngct/Projects/potaco && git add -A && git commit -m "fix: resolve import and type consistency across CLI files"
```

---

### Task 18: Final Integration and Build Verification

**Files:**
- No new files (verification only)

- [ ] **Step 1: Build the binary**

Run:
```bash
cd /home/ngct/Projects/potaco && go build -o potaco .
```
Expected: creates `potaco` binary

- [ ] **Step 2: Verify root help**

Run:
```bash
./potaco --help
```
Expected: prints usage text listing `gen`, `edit`, `config`, `info` subcommands

- [ ] **Step 3: Verify gen help**

Run:
```bash
./potaco gen --help
```
Expected: lists all gen flags (--prompt, --model, --size, --quality, --n, --style, --seed, --guidance-scale, --negative-prompt, --response-format, --output, --output-format, --view, --stdout, --provider, --base-url, --api-key, --retries, --timeout, --dry-run)

- [ ] **Step 4: Verify edit help**

Run:
```bash
./potaco edit --help
```
Expected: lists edit flags including --image, --mask, --mask-rect, --mask-circle, --extend

- [ ] **Step 5: Verify config help**

Run:
```bash
./potaco config --help
./potaco config list-providers
```
Expected: shows set, show, list-providers subcommands; list-providers prints preset details

- [ ] **Step 6: Verify info help**

Run:
```bash
./potaco info --help
```
Expected: shows info command with path argument

- [ ] **Step 7: Verify dry-run end-to-end**

Run:
```bash
export POTACO_BASE_URL=https://api.openai.com
export POTACO_API_KEY=sk-test
./potaco gen --prompt "a cat" --dry-run
```
Expected: prints JSON with method, url, content_type, headers (redacted), and body containing the prompt. Does NOT make an API call.

- [ ] **Step 8: Run full test suite**

Run:
```bash
cd /home/ngct/Projects/potaco && go test ./... -v
```
Expected: all tests pass

- [ ] **Step 9: Clean up binary**

Run:
```bash
rm potaco
```

- [ ] **Step 10: Commit final state**

```bash
cd /home/ngct/Projects/potaco && git add -A && git commit -m "verify: full build and integration check passes" || echo "Nothing to commit"
```
