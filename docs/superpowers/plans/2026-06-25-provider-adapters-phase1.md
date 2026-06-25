# Provider Adapters Migration - Phase 1: Adapter Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat `internal/provider/` package with an interface-based adapter system, extracting the OpenAI adapter as the first implementation, and migrating `gen` and `edit` commands to use the adapter interface.

**Architecture:** Create `internal/adapter/` with a shared `Adapter` interface, a provider registry, and an `internal/adapter/openai/` sub-package that wraps the existing HTTP client logic. The CLI commands (`gen`, `edit`) switch from calling `provider.NewClient` + `provider.Client` to using `adapter.Get("openai")` + the `Adapter` interface. The old `internal/provider/` package is removed once all references are migrated.

**Tech Stack:** Go 1.26, Cobra CLI, standard library `net/http`, `httptest` for tests

## Global Constraints

- Go 1.26, pure Go only (no CGO)
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping
- Table-driven tests preferred. Test files sit alongside source: `foo.go` / `foo_test.go`
- Internal image package is imported as `img` in CLI files to avoid collision with stdlib `image`
- CLI tests dispatch via `rootCmd.SetArgs([]string{"subcommand", ...})` and `rootCmd.Execute()`
- Provider tests use `httptest.Server` mocks and override `client.backoff` to 1ms for fast retry tests
- Keep files focused: one responsibility per file, one subcommand per file in `cli/`
- Exit codes: 0 success, 2 config error, 3 API error, 4 image error (defined in `internal/cli/errors.go`)

---

### Task 1: Adapter Interface and Shared Types

**Files:**
- Create: `internal/adapter/adapter.go`
- Create: `internal/adapter/adapter_test.go`

**Interfaces:**
- Produces: `adapter.Adapter` interface, `adapter.GenerateRequest`, `adapter.EditRequest`, `adapter.GenerateResponse`, `adapter.ImageData`, `adapter.Model`, `adapter.Param`, adapter error vars

- [ ] **Step 1: Write the failing test**

```go
// internal/adapter/adapter_test.go
package adapter

import (
	"context"
	"errors"
	"testing"
)

func TestAdapterInterfaceCompile(t *testing.T) {
	// Compile-time check that a minimal struct implements Adapter
	var _ Adapter = &mockAdapter{}
}

type mockAdapter struct{}

func (m *mockAdapter) Name() string                                    { return "mock" }
func (m *mockAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{}, nil
}
func (m *mockAdapter) Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error) {
	return &GenerateResponse{}, nil
}
func (m *mockAdapter) DiscoverModels(ctx context.Context) ([]Model, error) { return nil, nil }
func (m *mockAdapter) Verify(ctx context.Context) error                    { return nil }
func (m *mockAdapter) ModelParams(ctx context.Context, modelID string) ([]Param, error) {
	return nil, nil
}
func (m *mockAdapter) AuthHeader(apiKey string) string { return "Bearer " + apiKey }

func TestAdapterErrors(t *testing.T) {
	if !errors.Is(ErrEditNotSupported, ErrEditNotSupported) {
		t.Error("ErrEditNotSupported should be a sentinel error")
	}
	if !errors.Is(ErrModelNotFound, ErrModelNotFound) {
		t.Error("ErrModelNotFound should be a sentinel error")
	}
	if !errors.Is(ErrVerificationFailed, ErrVerificationFailed) {
		t.Error("ErrVerificationFailed should be a sentinel error")
	}
	if !errors.Is(ErrDiscoveryFailed, ErrDiscoveryFailed) {
		t.Error("ErrDiscoveryFailed should be a sentinel error")
	}
}

func TestGenerateRequestFields(t *testing.T) {
	req := GenerateRequest{
		Prompt:        "a cat",
		Model:         "gpt-image-2",
		N:             1,
		Size:          "1024x1024",
		Quality:       "auto",
		Style:         "vivid",
		ResponseFormat: "b64_json",
		Seed:          42,
		GuidanceScale: 7.5,
		NegativePrompt: "blurry",
		ExtraParams:   map[string]any{"background": "transparent"},
	}
	if req.Prompt != "a cat" {
		t.Errorf("Prompt = %q", req.Prompt)
	}
	if req.ExtraParams["background"] != "transparent" {
		t.Errorf("ExtraParams not set correctly")
	}
}

func TestEditRequestFields(t *testing.T) {
	req := EditRequest{
		Prompt:    "make it blue",
		Model:     "gpt-image-2",
		ImagePath: "/tmp/test.png",
		MaskPath:  "/tmp/mask.png",
		N:         1,
		Size:      "1024x1024",
		ExtraParams: map[string]any{"strength": 0.8},
	}
	if req.ImagePath != "/tmp/test.png" {
		t.Errorf("ImagePath = %q", req.ImagePath)
	}
	if req.ExtraParams["strength"] != 0.8 {
		t.Errorf("ExtraParams not set correctly")
	}
}

func TestGenerateResponseFields(t *testing.T) {
	resp := GenerateResponse{
		Created: 1234567890,
		Data: []ImageData{
			{B64JSON: "aGVsbG8=", URL: "", RevisedPrompt: "a fluffy cat"},
		},
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q", resp.Data[0].B64JSON)
	}
	if resp.Data[0].RevisedPrompt != "a fluffy cat" {
		t.Errorf("RevisedPrompt = %q", resp.Data[0].RevisedPrompt)
	}
}

func TestModelFields(t *testing.T) {
	m := Model{
		ID:           "gpt-image-2",
		DisplayName:  "GPT Image 2",
		SupportsGen:  true,
		SupportsEdit: true,
		Capabilities: []string{"quality", "background", "output_format"},
	}
	if !m.SupportsGen {
		t.Error("SupportsGen should be true")
	}
	if !m.SupportsEdit {
		t.Error("SupportsEdit should be true")
	}
	if len(m.Capabilities) != 3 {
		t.Errorf("Capabilities len = %d, want 3", len(m.Capabilities))
	}
}

func TestParamFields(t *testing.T) {
	p := Param{
		Name:        "size",
		Type:        "enum",
		Description: "Image dimensions",
		Default:     "1024x1024",
		EnumValues:  []string{"1024x1024", "1536x1024", "1024x1536"},
		Required:    false,
	}
	if p.Default != "1024x1024" {
		t.Errorf("Default = %q", p.Default)
	}
	if len(p.EnumValues) != 3 {
		t.Errorf("EnumValues len = %d, want 3", len(p.EnumValues))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/ -v`
Expected: FAIL with "package adapter is not in std" or "no such package"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/adapter/adapter.go
package adapter

import (
	"context"
	"errors"
)

// Adapter is the interface that each provider implements to abstract over
// their API differences for image generation, editing, model discovery,
// and verification.
type Adapter interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error)
	DiscoverModels(ctx context.Context) ([]Model, error)
	Verify(ctx context.Context) error
	ModelParams(ctx context.Context, modelID string) ([]Param, error)
	AuthHeader(apiKey string) string
}

// GenerateRequest is the normalized request for image generation.
// Provider-specific fields pass through ExtraParams.
type GenerateRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	Quality        string
	Style          string
	ResponseFormat string
	Seed           int
	GuidanceScale  float64
	NegativePrompt string
	ExtraParams    map[string]any
}

// EditRequest is the normalized request for image editing.
// Provider-specific fields pass through ExtraParams.
type EditRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	ResponseFormat string
	ImagePath      string
	MaskPath       string
	User           string
	ExtraParams    map[string]any
}

// GenerateResponse is the normalized response from both generate and edit.
type GenerateResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents a single generated or edited image.
type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// Model represents an image-generation-capable model from a provider.
type Model struct {
	ID           string
	DisplayName  string
	SupportsGen  bool
	SupportsEdit bool
	Capabilities []string
}

// Param describes a supported parameter for a specific model.
type Param struct {
	Name        string
	Type        string // "string", "int", "float", "bool", "enum"
	Description string
	Default     string
	EnumValues  []string
	Required    bool
}

// Sentinel errors for adapter operations.
var (
	ErrEditNotSupported    = errors.New("image editing not supported by this provider")
	ErrModelNotFound       = errors.New("model not found")
	ErrVerificationFailed  = errors.New("provider verification failed")
	ErrDiscoveryFailed     = errors.New("model discovery failed")
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/adapter.go internal/adapter/adapter_test.go
git commit -m "adapter: add Adapter interface and shared types"
```

---

### Task 2: Provider Registry

**Files:**
- Create: `internal/adapter/registry.go`
- Create: `internal/adapter/registry_test.go`

**Interfaces:**
- Consumes: `adapter.Adapter` (from Task 1)
- Produces: `adapter.Register(name, factory)`, `adapter.Get(name, apiKey, opts) (Adapter, error)`, `adapter.List() []string`

- [ ] **Step 1: Write the failing test**

```go
// internal/adapter/registry_test.go
package adapter

import (
	"testing"
)

func TestRegistryGetUnknownProvider(t *testing.T) {
	_, err := Get("nonexistent", "sk-test", AdapterOpts{})
	if err == nil {
		t.Fatal("Get should error for unknown provider")
	}
}

func TestRegistryListContainsRegisteredProviders(t *testing.T) {
	// The test-only adapter registered in TestRegistryRegisterAndGet
	names := List()
	// After init, openai should be registered (registered in openai package init)
	// But in the registry test, we only test what we explicitly add
	found := false
	for _, n := range names {
		if n == "test-provider" {
			found = true
			break
		}
	}
	if !found {
		// test-provider is registered in the test below via TestRegistryRegisterAndGet
		// If tests run in any order, we need to register here
		Register("test-provider", func(apiKey string, opts AdapterOpts) (Adapter, error) {
			return &mockAdapter{}, nil
		})
		_, err := Get("test-provider", "sk-test", AdapterOpts{})
		if err != nil {
			t.Fatalf("Get test-provider after register: %v", err)
		}
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	Register("test-provider", func(apiKey string, opts AdapterOpts) (Adapter, error) {
		return &mockAdapter{}, nil
	})

	a, err := Get("test-provider", "sk-test", AdapterOpts{})
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if a.Name() != "mock" {
		t.Errorf("Name = %q, want 'mock'", a.Name())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/ -run TestRegistry -v`
Expected: FAIL with undefined `Get`, `Register`, `List`, `AdapterOpts`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/adapter/registry.go
package adapter

import (
	"fmt"
	"sort"
	"sync"
)

// AdapterOpts holds options for constructing an adapter instance.
type AdapterOpts struct {
	BaseURL string // override the adapter's default base URL
	Timeout string // override the adapter's default timeout
}

// AdapterFactory creates an Adapter instance for a given API key and options.
type AdapterFactory func(apiKey string, opts AdapterOpts) (Adapter, error)

var (
	registry   = make(map[string]AdapterFactory)
	registryMu sync.RWMutex
)

// Register adds a provider adapter factory to the registry.
// Called by each adapter package's init() function.
func Register(name string, factory AdapterFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get retrieves and instantiates the adapter for the named provider.
func Get(name string, apiKey string, opts AdapterOpts) (Adapter, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %v)", name, List())
	}
	return factory(apiKey, opts)
}

// List returns the names of all registered providers, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/ -run TestRegistry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/registry.go internal/adapter/registry_test.go
git commit -m "adapter: add provider registry with Get, List, Register"
```

---

### Task 3: OpenAI Adapter - Generate Method

**Files:**
- Create: `internal/adapter/openai/openai.go`
- Create: `internal/adapter/openai/openai_test.go`

**Interfaces:**
- Consumes: `adapter.Adapter`, `adapter.GenerateRequest`, `adapter.GenerateResponse`, `adapter.AdapterOpts` (from Tasks 1-2)
- Produces: `openai.New(apiKey, opts) *Adapter`, `openai.Adapter` implementing `adapter.Adapter`, the openai package `init()` calling `adapter.Register("openai", ...)`.

Note: This task extracts the existing `provider.Client.Generate` logic (from `internal/provider/client.go`) into the OpenAI adapter. The shared HTTP client, response parsing, and retry logic are copied from `internal/provider/`.

The `openai.New()` function takes `adapter.AdapterOpts` (defined in Task 2's `registry.go`), not a separate `openai.AdapterOpts`. The `SetBackoff` and `SetSleep` methods are test helpers on the concrete `*Adapter` type.

- [ ] **Step 1: Write the failing test**

```go
// internal/adapter/openai/openai_test.go
package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/openai/ -v`
Expected: FAIL with "no such package" or compilation errors

- [ ] **Step 3: Write minimal implementation**

```go
// internal/adapter/openai/openai.go
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

// defaultBaseURL is the default OpenAI API base URL.
const defaultBaseURL = "https://api.openai.com/v1"

// Adapter implements adapter.Adapter for the OpenAI Images API.
type Adapter struct {
	apiKey  string
	baseURL string
	retries int
	timeout time.Duration
	http    *http.Client
	backoff func(attempt int) time.Duration
	sleep   func(ctx context.Context, d time.Duration)
}

// New creates an OpenAI adapter with the given API key and options.
func New(apiKey string, opts adapter.AdapterOpts) adapter.Adapter {
	baseURL := defaultBaseURL
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	timeout := 120 * time.Second
	if opts.Timeout != "" {
		if d, err := time.ParseDuration(opts.Timeout); err == nil {
			timeout = d
		}
	}
	return &Adapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		retries: 2,
		timeout: timeout,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetBackoff overrides the backoff function (for testing).
func (a *Adapter) SetBackoff(fn func(int) time.Duration) {
	a.backoff = fn
}

// SetSleep overrides the sleep function (for testing).
func (a *Adapter) SetSleep(fn func(context.Context, time.Duration)) {
	a.sleep = fn
}

func (a *Adapter) Name() string { return "openai" }

func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

// maxResponseBytes bounds the response body size.
var maxResponseBytes int64 = 128 << 20

func (a *Adapter) Generate(ctx context.Context, req adapter.GenerateRequest) (*adapter.GenerateResponse, error) {
	body := a.buildGenerateBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := a.generateURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

func (a *Adapter) generateURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/generations"
	}
	return a.baseURL + "/v1/images/generations"
}

func (a *Adapter) editURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/edits"
	}
	return a.baseURL + "/v1/images/edits"
}

// buildGenerateBody converts the normalized GenerateRequest to the OpenAI JSON schema.
func (a *Adapter) buildGenerateBody(req adapter.GenerateRequest) map[string]any {
	body := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Model != "" {
		body["model"] = req.Model
	}
	if req.N > 0 {
		body["n"] = req.N
	}
	if req.Size != "" {
		body["size"] = req.Size
	}
	if req.Quality != "" {
		body["quality"] = req.Quality
	}
	if req.Style != "" {
		body["style"] = req.Style
	}
	if req.ResponseFormat != "" {
		body["response_format"] = req.ResponseFormat
	}
	if req.Seed != 0 {
		body["seed"] = req.Seed
	}
	if req.GuidanceScale != 0 {
		body["guidance_scale"] = req.GuidanceScale
	}
	if req.NegativePrompt != "" {
		body["negative_prompt"] = req.NegativePrompt
	}
	for k, v := range req.ExtraParams {
		body[k] = v
	}
	return body
}

// parseResponse reads the HTTP response and returns a GenerateResponse or an error.
func parseResponse(resp *http.Response) (*adapter.GenerateResponse, error) {
	respBody, err := readLimitedBody(resp.Body, maxResponseBytes, "provider response")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Type    string `json:"type"`
				Code    string `json:"code,omitempty"`
				Message string `json:"message"`
				Param   string `json:"param,omitempty"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp adapter.GenerateResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &imgResp, nil
}

// readLimitedBody reads up to limit+1 bytes from r.
func readLimitedBody(r io.Reader, limit int64, label string) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s too large: limit is %d bytes", label, limit)
	}
	return data, nil
}
```

Also need the retry logic. Add it to the same file since it is tightly coupled with the adapter's HTTP calls:

```go
// Add to internal/adapter/openai/openai.go (continue)

import (
	"math/rand"
	"strconv"
)

var maxRetryDrainBytes int64 = 1 << 20

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

func shouldRetry(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

func retryDelay(resp *http.Response, attempt int, fallback func(int) time.Duration) time.Duration {
	if resp != nil {
		if raw := resp.Header.Get("Retry-After"); raw != "" {
			if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	return fallback(attempt)
}

func (a *Adapter) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; ; attempt++ {
		resp, err := a.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt < a.retries {
				a.backoffSleep(ctx, retryDelay(nil, attempt, a.backoffOrDefault))
				if req.GetBody != nil {
					body, err := req.GetBody()
					if err == nil {
						req.Body = body
					}
				}
				continue
			}
			return nil, fmt.Errorf("http request: %w", err)
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp

		if attempt >= a.retries {
			break
		}

		_, _ = io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes)
		resp.Body.Close()

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				req.Body = body
			}
		}

		a.backoffSleep(ctx, retryDelay(resp, attempt, a.backoffOrDefault))
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

func (a *Adapter) backoffOrDefault(attempt int) time.Duration {
	if a.backoff != nil {
		return a.backoff(attempt)
	}
	return defaultBackoff(attempt)
}

func (a *Adapter) backoffSleep(ctx context.Context, d time.Duration) {
	if a.sleep != nil {
		a.sleep(ctx, d)
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}
```

And the init function to register the adapter:

```go
// Add to internal/adapter/openai/openai.go (continue)

func init() {
	adapter.Register("openai", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
```

Note: The `strings` import is needed. Adjust the import block in the real file to include all imports used.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/openai/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/openai/openai.go internal/adapter/openai/openai_test.go
git commit -m "adapter: add OpenAI adapter with Generate method and retry logic"
```

---

### Task 4: OpenAI Adapter - Edit Method

**Files:**
- Modify: `internal/adapter/openai/openai.go` (add Edit method)
- Modify: `internal/adapter/openai/openai_test.go` (add Edit tests)

**Interfaces:**
- Consumes: `adapter.EditRequest` (from Task 1)
- Produces: `Adapter.Edit()` method on the OpenAI adapter

Note: This extracts the existing `provider.Client.Edit` multipart logic from `internal/provider/client.go`.

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/adapter/openai/openai_test.go

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
```

Also add the helper function at the top of the test file (if not already present):

```go
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

Run: `go test ./internal/adapter/openai/ -run TestOpenAIEdit -v`
Expected: FAIL with "Adapter.Edit undefined" or compilation error

- [ ] **Step 3: Write minimal implementation**

```go
// Add to internal/adapter/openai/openai.go

import (
	"mime/multipart"
	"os"
	"path/filepath"
)

func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (*adapter.GenerateResponse, error) {
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

	imgPart, err := writer.CreateFormFile("image", filepath.Base(req.ImagePath))
	if err != nil {
		return nil, fmt.Errorf("create image part: %w", err)
	}
	if _, err := io.Copy(imgPart, imgFile); err != nil {
		return nil, fmt.Errorf("copy image data: %w", err)
	}

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

	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			return nil, fmt.Errorf("write prompt field: %w", err)
		}
	}
	if req.Model != "" {
		if err := writer.WriteField("model", req.Model); err != nil {
			return nil, fmt.Errorf("write model field: %w", err)
		}
	}
	if req.N > 0 {
		if err := writer.WriteField("n", strconv.Itoa(req.N)); err != nil {
			return nil, fmt.Errorf("write n field: %w", err)
		}
	}
	if req.Size != "" {
		if err := writer.WriteField("size", req.Size); err != nil {
			return nil, fmt.Errorf("write size field: %w", err)
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			return nil, fmt.Errorf("write response_format field: %w", err)
		}
	}
	if req.User != "" {
		if err := writer.WriteField("user", req.User); err != nil {
			return nil, fmt.Errorf("write user field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := a.editURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/openai/ -run TestOpenAIEdit -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/openai/openai.go internal/adapter/openai/openai_test.go
git commit -m "adapter: add OpenAI adapter Edit method with multipart support"
```

---

### Task 5: OpenAI Adapter - DiscoverModels, Verify, and ModelParams

**Files:**
- Modify: `internal/adapter/openai/openai.go` (add DiscoverModels, Verify, ModelParams)
- Create: `internal/adapter/openai/models.go` (hardcoded model parameter defaults)
- Modify: `internal/adapter/openai/openai_test.go` (add tests for new methods)

**Interfaces:**
- Consumes: `adapter.Model`, `adapter.Param` (from Task 1)
- Produces: `Adapter.DiscoverModels()`, `Adapter.Verify()`, `Adapter.ModelParams()` methods; `models.go` hardcoded defaults

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/adapter/openai/openai_test.go

func TestOpenAIDiscoverModelsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-image-2", "object": "model", "owned_by": "openai"},
				{"id": "gpt-image-1", "object": "model", "owned_by": "openai"},
				{"id": "dall-e-3", "object": "model", "owned_by": "openai"},
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
	if !ids["dall-e-3"] {
		t.Error("should include dall-e-3")
	}
	if ids["text-davinci-003"] {
		t.Error("should not include text model")
	}

	// Check SupportsEdit is set for gpt-image-2 and dall-e-2
	for _, m := range models {
		if m.ID == "gpt-image-2" && !m.SupportsEdit {
			t.Error("gpt-image-2 should have SupportsEdit=true")
		}
		if m.ID == "dall-e-3" && m.SupportsEdit {
			t.Error("dall-e-3 should have SupportsEdit=false")
		}
	}
}

func TestOpenAIDiscoverModelsFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	a := New("sk-test", adapter.AdapterOpts{BaseURL: server.URL})

	models, err := a.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels should fall back to hardcoded, got error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("should return fallback models")
	}

	// Should include hardcoded defaults
	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
	}
	if !ids["gpt-image-2"] {
		t.Error("fallback should include gpt-image-2")
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

func TestOpenAIModelParamsGPTImage2(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})

	params, err := a.ModelParams(context.Background(), "gpt-image-2")
	if err != nil {
		t.Fatalf("ModelParams error: %v", err)
	}

	names := make(map[string]bool)
	for _, p := range params {
		names[p.Name] = true
	}
	if !names["size"] {
		t.Error("should include size param")
	}
	if !names["quality"] {
		t.Error("should include quality param")
	}
	if !names["n"] {
		t.Error("should include n param")
	}
	// dall-e-3 only params should not be present
	if names["style"] {
		t.Error("gpt-image-2 should not have style param")
	}
}

func TestOpenAIModelParamsDalE3(t *testing.T) {
	a := New("sk-test", adapter.AdapterOpts{})

	params, err := a.ModelParams(context.Background(), "dall-e-3")
	if err != nil {
		t.Fatalf("ModelParams error: %v", err)
	}

	names := make(map[string]bool)
	for _, p := range params {
		names[p.Name] = true
	}
	if !names["style"] {
		t.Error("dall-e-3 should have style param")
	}
	if !names["quality"] {
		t.Error("dall-e-3 should have quality param")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/openai/ -run "TestOpenAIDiscover|TestOpenAIVerify|TestOpenAIModelParams" -v`
Expected: FAIL with undefined methods

- [ ] **Step 3: Write the models.go file with hardcoded defaults**

```go
// internal/adapter/openai/models.go
package openai

import "github.com/ncxton/potaco/internal/adapter"

// imageModelIDs is the set of OpenAI model IDs that support image generation.
var imageModelIDs = map[string]bool{
	"gpt-image-2":      true,
	"gpt-image-1":      true,
	"gpt-image-1-mini": true,
	"dall-e-3":         true,
	"dall-e-2":         true,
}

// editCapableModels is the set of image model IDs that support image editing.
var editCapableModels = map[string]bool{
	"gpt-image-2":      true,
	"gpt-image-1":      true,
	"gpt-image-1-mini": true,
	"dall-e-2":         true,
}

// fallbackModels is the hardcoded list used when API discovery fails.
var fallbackModels = []adapter.Model{
	{ID: "gpt-image-2", DisplayName: "GPT Image 2", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n", "background", "output_format", "output_compression", "moderation"}},
	{ID: "gpt-image-1", DisplayName: "GPT Image 1", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
	{ID: "gpt-image-1-mini", DisplayName: "GPT Image 1 Mini", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
	{ID: "dall-e-3", DisplayName: "DALL-E 3", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "style", "n"}},
	{ID: "dall-e-2", DisplayName: "DALL-E 2", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
}

// hardcodedModelParams maps model IDs to their supported parameters.
var hardcodedModelParams = map[string][]adapter.Param{
	"gpt-image-2": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536", "auto"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
		{Name: "background", Type: "enum", Description: "Background type", Default: "auto", EnumValues: []string{"transparent", "opaque", "auto"}},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "output_compression", Type: "int", Description: "Output compression (0-100)", Default: "0"},
		{Name: "moderation", Type: "enum", Description: "Moderation level", Default: "auto", EnumValues: []string{"auto", "low"}},
	},
	"gpt-image-1": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"gpt-image-1-mini": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"dall-e-3": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1792x1024", "1024x1792"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "standard", EnumValues: []string{"standard", "hd"}},
		{Name: "style", Type: "enum", Description: "Visual style", Default: "vivid", EnumValues: []string{"vivid", "natural"}},
		{Name: "n", Type: "int", Description: "Number of images (always 1 for dall-e-3)", Default: "1"},
	},
	"dall-e-2": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"256x256", "512x512", "1024x1024"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "standard", EnumValues: []string{"standard"}},
		{Name: "n", Type: "int", Description: "Number of images (1-10)", Default: "1"},
	},
}
```

Now add the methods to `openai.go`:

```go
// Add to internal/adapter/openai/openai.go

func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.baseURL + "/v1/models"
	if strings.HasSuffix(a.baseURL, "/v1") {
		url = a.baseURL + "/models"
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fallbackModels, nil // fall back silently
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fallbackModels, nil
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fallbackModels, nil
	}

	var models []adapter.Model
	for _, m := range result.Data {
		if !imageModelIDs[m.ID] {
			continue
		}
		models = append(models, adapter.Model{
			ID:           m.ID,
			DisplayName:  m.ID,
			SupportsGen:  true,
			SupportsEdit: editCapableModels[m.ID],
			Capabilities: modelCapabilities(m.ID),
		})
	}
	if len(models) == 0 {
		return fallbackModels, nil
	}
	return models, nil
}

func (a *Adapter) Verify(ctx context.Context) error {
	url := a.baseURL + "/v1/models"
	if strings.HasSuffix(a.baseURL, "/v1") {
		url = a.baseURL + "/models"
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("verification failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) ModelParams(ctx context.Context, modelID string) ([]adapter.Param, error) {
	// OpenAI does not expose a per-model parameter schema via API.
	// Falls back to hardcoded defaults.
	params, ok := hardcodedModelParams[modelID]
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}

// modelCapabilities returns capability strings for a model ID.
func modelCapabilities(modelID string) []string {
	if params, ok := hardcodedModelParams[modelID]; ok {
		caps := make([]string, len(params))
		for i, p := range params {
			caps[i] = p.Name
		}
		return caps
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/openai/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/openai/openai.go internal/adapter/openai/models.go internal/adapter/openai/openai_test.go
git commit -m "adapter: add OpenAI DiscoverModels, Verify, ModelParams methods"
```

---

### Task 6: Migrate CLI `gen` Command to Use Adapter

**Files:**
- Modify: `internal/cli/gen.go`
- Modify: `internal/cli/helpers.go`
- Modify: `internal/cli/gen_test.go`
- Modify: `internal/cli/output.go` (replace `provider.ImageResponse` with `adapter.GenerateResponse`)

**Interfaces:**
- Consumes: `adapter.Adapter`, `adapter.Get`, `adapter.GenerateRequest`, `adapter.GenerateResponse` (from Tasks 1-5)
- Produces: `gen` command that uses `adapter.Get("openai", apiKey, opts)` instead of `provider.NewClient(config)`

Note: At this point `gen` still uses the `--provider` flag and `config.Merge` for backward compat. The provider flag will be wired through to `adapter.Get`. In Phase 2, `--provider` is removed and replaced with the active provider from config.

- [ ] **Step 1: Write the failing test**

The existing test `TestGenCommandDryRunNoAPI` should still pass after migration. Add a new test that verifies gen uses the adapter path:

```go
// Add to internal/cli/gen_test.go

func TestGenCommandUsesAdapter(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")
	t.Setenv("POTACO_MODEL", "gpt-image-2")

	// Verify the gen command can resolve to an adapter via dry-run
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("dry-run should contain model, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify current state**

Run: `go test ./internal/cli/ -run TestGenCommand -v`
Expected: Existing tests PASS (they should still work since gen.go hasn't changed yet)

- [ ] **Step 3: Migrate `helpers.go` to use adapter**

Replace the `provider` import and usage in `helpers.go`. The `buildMergeOptions` function needs to return config that includes the base URL and API key, which will be passed to `adapter.Get`. Update the `outputContext` struct to use `adapter.GenerateResponse` instead of `provider.ImageResponse`:

In `internal/cli/helpers.go`, change:
- Import: `"github.com/ncxton/potaco/internal/provider"` -> `"github.com/ncxton/potaco/internal/adapter"`
- `outputContext` struct: `resp *provider.ImageResponse` -> `resp *adapter.GenerateResponse`
- In `processAndOutput`: `octx.resp.Data` stays the same since `adapter.GenerateResponse` has the same `Data []ImageData` field with same `B64JSON`/`URL` fields
- The `buildMergeOptions` function's `provider.GetPreset` call needs replacement. For now (Phase 1), keep using `config.Merge` for config resolution. The provider name is available via `opts.Provider`. Map the provider name to adapter base URL using a temporary helper that will be replaced in Phase 2.

Add a helper function to `helpers.go`:

```go
// adapterForProvider resolves the adapter for the given provider name and config.
// In Phase 1, this is a transition helper that maps the old config model to the adapter.
func adapterForProvider(cfg *config.Config) (adapter.Adapter, error) {
	providerName := cfg.Provider
	if providerName == "" {
		providerName = "openai" // default to openai for backward compat
	}

	opts := adapter.AdapterOpts{
		BaseURL: cfg.BaseURL,
	}
	if cfg.Timeout > 0 {
		opts.Timeout = cfg.Timeout.String()
	}

	return adapter.Get(providerName, cfg.APIKey, opts)
}
```

- [ ] **Step 4: Migrate `gen.go` to use adapter**

In `internal/cli/gen.go`, replace the `provider.NewClient` call with `adapterForProvider`:

Replace:
```go
client := provider.NewClient(provider.ClientConfig{
    BaseURL: cfg.BaseURL,
    APIKey:  cfg.APIKey,
    Retries: cfg.Retries,
    Timeout: cfg.Timeout,
})

start := time.Now()
resp, err := client.Generate(context.Background(), req)
```

With:
```go
ad, err := adapterForProvider(cfg)
if err != nil {
    return configError(fmt.Errorf("adapter: %w", err))
}

req := adapter.GenerateRequest{
    Prompt:         prompt,
    Model:          model,
    Size:           flagString(cmd, "size"),
    Quality:        flagString(cmd, "quality"),
    N:              flagInt(cmd, "n"),
    ResponseFormat: flagString(cmd, "response-format"),
    Seed:           flagInt(cmd, "seed"),
    GuidanceScale:  flagFloat64(cmd, "guidance-scale"),
    NegativePrompt: flagString(cmd, "negative-prompt"),
}

start := time.Now()
resp, err := ad.Generate(context.Background(), req)
```

Also update the import: replace `"github.com/ncxton/potaco/internal/provider"` with `"github.com/ncxton/potaco/internal/adapter"`.

The `provider.GenerateRequest` becomes `adapter.GenerateRequest`. The `outputContext` now uses `*adapter.GenerateResponse`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestGenCommand -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/gen.go internal/cli/helpers.go internal/cli/gen_test.go internal/cli/output.go
git commit -m "cli: migrate gen command to use adapter interface"
```

---

### Task 7: Migrate CLI `edit` Command to Use Adapter

**Files:**
- Modify: `internal/cli/edit.go`
- Modify: `internal/cli/edit_mask.go`
- Modify: `internal/cli/edit_test.go`

**Interfaces:**
- Consumes: `adapter.Adapter`, `adapter.EditRequest`, `adapter.GenerateResponse` (from Tasks 1-6)
- Produces: `edit` command uses `adapterForProvider` + `ad.Edit()` instead of `provider.NewClient` + `client.Edit()`

- [ ] **Step 1: Write the failing test**

The existing edit tests use httptest servers. After migration, the edit command should still work with the mock server through the adapter. Verify the existing dry-run test passes:

```go
// Add to internal/cli/edit_test.go

func TestEditCommandDryRunUsesAdapter(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("POTACO_BASE_URL", "https://api.example.com")
	t.Setenv("POTACO_API_KEY", "sk-test")
	t.Setenv("POTACO_MODEL", "gpt-image-2")

	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	writeMinimalPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "make it blue", "--image", imgPath, "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "make it blue") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify current state**

Run: `go test ./internal/cli/ -run TestEditCommandDryRun -v`
Expected: Existing tests may PASS or need adjustment

- [ ] **Step 3: Migrate `edit.go` to use adapter**

In `internal/cli/edit.go`, replace the `provider.NewClient` call:

Replace:
```go
client := provider.NewClient(provider.ClientConfig{
    BaseURL: cfg.BaseURL,
    APIKey:  cfg.APIKey,
    Retries: cfg.Retries,
    Timeout: cfg.Timeout,
})

start := time.Now()
resp, err := client.Edit(context.Background(), req)
```

With:
```go
ad, err := adapterForProvider(cfg)
if err != nil {
    return configError(fmt.Errorf("adapter: %w", err))
}

req := adapter.EditRequest{
    Prompt:         prompt,
    Model:          model,
    N:              flagInt(cmd, "n"),
    Size:           flagString(cmd, "size"),
    ResponseFormat: flagString(cmd, "response-format"),
    ImagePath:      editImagePath,
    MaskPath:       maskPath,
}

start := time.Now()
resp, err := ad.Edit(context.Background(), req)
```

Update imports: replace `"github.com/ncxton/potaco/internal/provider"` with `"github.com/ncxton/potaco/internal/adapter"`. Change `provider.EditRequest` to `adapter.EditRequest`.

- [ ] **Step 4: Migrate `edit_mask.go` references**

In `internal/cli/edit_mask.go`, the `printEditDryRun` function references `cfg.BaseURL` which is fine since `config.Config` still has that field. Check for any `provider` package imports and remove them if unused. The file should not import the `provider` package anymore.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/cli/ -run "TestEdit" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/edit.go internal/cli/edit_mask.go internal/cli/edit_test.go
git commit -m "cli: migrate edit command to use adapter interface"
```

---

### Task 8: Remove Old `internal/provider/` Package

**Files:**
- Delete: `internal/provider/client.go`
- Delete: `internal/provider/client_test.go`
- Delete: `internal/provider/presets.go`
- Delete: `internal/provider/presets_test.go`
- Delete: `internal/provider/retry.go`
- Delete: `internal/provider/retry_test.go`
- Delete: `internal/provider/types.go`
- Modify: `internal/cli/config_cmd.go` (remove `provider.GetPreset` / `provider.AllPresets` usage)
- Modify: `internal/cli/config_cmd_test.go` (update tests that reference presets)

**Interfaces:**
- Consumes: All adapter migrations complete (Tasks 1-7)
- Produces: No remaining references to `internal/provider/` package

- [ ] **Step 1: Search for remaining provider references**

Run: `rg "internal/provider" --type go`
Expected: List of files still importing the old package. Fix each one.

The main remaining references will be in:
- `internal/cli/config_cmd.go` - uses `provider.GetPreset()` and `provider.AllPresets()`
- `internal/cli/helpers.go` - already migrated in Task 6, but verify

- [ ] **Step 2: Migrate `config_cmd.go` to remove provider dependency**

For `config set --provider`, replace `provider.GetPreset` with a hardcoded mapping or a call to `adapter.Get` to get the default base URL. For `config list-providers`, replace `provider.AllPresets()` with `adapter.List()` and get defaults from adapters.

In `config_cmd.go`, the `runConfigListProviders` function should use `adapter.List()` to show available adapters. For each adapter, it can create a temporary instance with a dummy key to get its base URL (or we add a `BaseURL()` method to the adapter interface, but that is out of scope for Phase 1). For Phase 1, hardcode the provider info:

```go
func runConfigListProviders(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	providers := adapter.List()

	fmt.Fprintln(out, "Available provider adapters:")
	fmt.Fprintln(out)
	for _, name := range providers {
		fmt.Fprintf(out, "  %s\n", name)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use 'potaco config set --provider <name>' to apply default settings.")
	return nil
}
```

For `config set --provider`, map the provider name to a base URL using a temporary lookup. In Phase 1, this is a transitional step before the full auth system replaces it:

```go
// providerBaseURLs is a temporary lookup for Phase 1 backward compat.
// This is removed in Phase 2 when the auth system manages provider config.
var providerBaseURLs = map[string]string{
	"openai": "https://api.openai.com/v1",
	"together": "https://api.together.ai",
	"fal": "https://fal.run",
}
```

Replace `provider.GetPreset(providerName)` with `providerBaseURLs[providerName]` lookup. Replace `preset.BaseURL` and `preset.DefaultModel` accordingly. For the model, keep a similar lookup:

```go
var providerDefaultModels = map[string]string{
	"openai":  "gpt-image-2",
	"together": "black-forest-labs/flux-1",
	"fal":     "fal-ai/flux",
}
```

- [ ] **Step 3: Update config_cmd_test.go**

The test `TestConfigListProviders` should now check for adapter names instead of preset names. The test `TestConfigSetProviderPreservesExplicitValues` should still work since it tests the config file write, not the provider lookup directly.

Update `TestConfigListProviders`:
```go
func TestConfigListProviders(t *testing.T) {
	_, buf := newConfigTest(t)
	rootCmd.SetArgs([]string{"config", "list-providers"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config list-providers error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should list 'openai' adapter, got: %q", output)
	}
	// "together" is no longer a registered adapter (only openai in Phase 1)
	// "fal" is not yet registered (comes in Phase 3)
}
```

- [ ] **Step 4: Delete old provider package files**

```bash
rm internal/provider/client.go
rm internal/provider/client_test.go
rm internal/provider/presets.go
rm internal/provider/presets_test.go
rm internal/provider/retry.go
rm internal/provider/retry_test.go
rm internal/provider/types.go
```

- [ ] **Step 5: Run all tests to verify nothing breaks**

Run: `go test ./... -v 2>&1 | head -50`
Expected: All tests PASS

Run: `go build -o potaco .`
Expected: Builds successfully

- [ ] **Step 6: Run go vet and gofmt**

Run: `go vet ./... && gofmt -l .`
Expected: Clean (no output from gofmt, no errors from go vet)

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "adapter: remove old internal/provider package, migrate config_cmd to adapter.List"
```

---

### Task 9: Final Verification and Integration Test

**Files:**
- No new files; verify all existing tests pass and the binary builds

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 2: Build the binary**

Run: `go build -o potaco .`
Expected: Successful build, `potaco` binary exists

- [ ] **Step 3: Run dry-run smoke test**

Run: `POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test ./potaco gen --prompt "a cat" --dry-run`
Expected: JSON output with method POST, URL containing `/v1/images/generations`, prompt "a cat"

- [ ] **Step 4: Run edit dry-run smoke test**

Run:
```bash
# Create a minimal test image
echo -n '\x89PNG\r\n\x1a\n' > /tmp/test.png
POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test ./potaco edit --prompt "make it blue" --image /tmp/test.png --dry-run
```
Expected: JSON output with method POST, URL containing `/images/edits`

- [ ] **Step 5: Verify gofmt and go vet are clean**

Run: `gofmt -l . && go vet ./...`
Expected: No output (clean)

- [ ] **Step 6: Final commit (if any formatting fixes needed)**

```bash
git add -A
git commit -m "adapter: Phase 1 complete - all tests pass, binary builds"
```

If nothing to commit, skip this step.
