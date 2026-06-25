# Phase 3: fal & Vercel Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the fal and Vercel AI Gateway adapters following the established OpenAI adapter pattern, register them in the adapter registry, and update the provider presets map so the CLI recognizes all three providers.

**Architecture:** Each adapter lives in its own sub-package (`internal/adapter/fal/`, `internal/adapter/vercel/`) and implements the `adapter.Adapter` interface. The fal adapter handles fal's per-model endpoint URLs, `Key` auth header, JSON-based image editing, and `images[]` response format. The Vercel adapter handles the OpenAI-compatible `/v1/images/generations` endpoint with provider-prefixed model IDs, `providerOptions` passthrough, and returns `ErrEditNotSupported` for edit. Both adapters register themselves via `init()`. The CLI `helpers.go` imports the new adapter packages for their side effects and updates `providerPresets` with the new providers.

**Tech Stack:** Go 1.26, `net/http`, `encoding/json`, `encoding/base64`, `os`, `image/png` (for fal image encoding), `github.com/ncxton/potaco/internal/adapter`

## Global Constraints

- Go 1.26, pure Go only (no CGO)
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping
- No `_ = err` (every (T, error) must be checked)
- `context.Context` as first param where applicable
- Keep files under 250 pure LOC
- Table-driven tests preferred. Test files sit alongside source: `foo.go` / `foo_test.go`
- Adapter tests use `httptest.Server` mocks, override `backoff` to 1ms for fast retry tests
- Exit codes: 0 success, 2 config error, 3 API error, 4 image error
- Module path: `github.com/ncxton/potaco`
- `internal/adapter/` package already exists with `Adapter` interface, `Get(name, apiKey, opts)`, `List()`, `Register(name, factory)`, `AdapterOpts{BaseURL, Timeout, Retries}`
- `internal/adapter/openai/` is the reference implementation with files: `openai.go`, `generate.go`, `edit.go`, `discover.go`, `response.go`, `retry.go`, `models.go`
- OpenAI adapter uses `SetBackoff(fn)` and `SetSleep(fn)` for test-overridable retry timing
- `internal/cli/helpers.go` imports `_ "github.com/ncxton/potaco/internal/adapter/openai"` for registration
- `internal/cli/helpers.go` has `providerPresets` map with `"openai"` and `"fal"` entries (no `"vercel"` yet)
- `internal/cli/resolve.go` has `resolveAdapterForCommand` that uses `getProviderPreset(name)` for base URL fallback
- `internal/auth/auth.go` has `defaultModelForProvider` map with `"openai": "gpt-image-2"`, `"fal": "fal-ai/flux/dev"`, `"vercel": "openai/gpt-image-2"`
- `printDryRun` in `helpers.go` hardcodes `Authorization: Bearer [REDACTED]` header display (needs to handle `Key` auth for fal)
- `gen.go` dry-run hardcodes `resolved.BaseURL+"/v1/images/generations"` URL (needs per-adapter dry-run URL support)
- `edit.go` dry-run calls `printEditDryRun(cmd, resolved.BaseURL, ...)` which also hardcodes the OpenAI edit URL

---

## File Structure

### New Files

| File | Responsibility |
|------|----------------|
| `internal/adapter/fal/fal.go` | Adapter struct, `New()`, `Name()`, `AuthHeader()`, URL helpers, `init()` registration, `SetBackoff()`/`SetSleep()` |
| `internal/adapter/fal/generate.go` | `Generate()` method, `buildGenerateBody()` (maps normalized request to fal schema) |
| `internal/adapter/fal/edit.go` | `Edit()` method (JSON body with base64 image data URI), `buildEditBody()` |
| `internal/adapter/fal/discover.go` | `DiscoverModels()`, `Verify()`, `ModelParams()` |
| `internal/adapter/fal/response.go` | `parseResponse()` (normalizes `images[]` to `GenerateResponse`), `readLimitedBody()` |
| `internal/adapter/fal/retry.go` | `doWithRetry()`, `backoffSleep()`, `shouldRetry()`, `retryDelay()` |
| `internal/adapter/fal/models.go` | Hardcoded model params, fallback models, edit endpoint mappings |
| `internal/adapter/fal/fal_test.go` | Tests for Generate, Edit, DiscoverModels, Verify, ModelParams |
| `internal/adapter/fal/retry_test.go` | Retry tests (ported from openai pattern) |
| `internal/adapter/vercel/vercel.go` | Adapter struct, `New()`, `Name()`, `AuthHeader()`, URL helpers, `init()` registration |
| `internal/adapter/vercel/generate.go` | `Generate()` method, `buildGenerateBody()` (OpenAI-compatible with `providerOptions`) |
| `internal/adapter/vercel/edit.go` | `Edit()` returns `ErrEditNotSupported` |
| `internal/adapter/vercel/discover.go` | `DiscoverModels()` (filter `type=="image"`), `Verify()` (two-step), `ModelParams()` |
| `internal/adapter/vercel/response.go` | `parseResponse()` (OpenAI-compatible), `readLimitedBody()` |
| `internal/adapter/vercel/models.go` | Hardcoded model params, fallback models |
| `internal/adapter/vercel/vercel_test.go` | Tests for Generate, Edit (not supported), DiscoverModels, Verify, ModelParams |

### Modified Files

| File | Changes |
|------|---------|
| `internal/cli/helpers.go` | Add `_ "github.com/ncxton/potaco/internal/adapter/fal"` and `_ "github.com/ncxton/potaco/internal/adapter/vercel"` imports; add `"vercel"` to `providerPresets`; update `printDryRun` to accept auth header type |
| `internal/cli/gen.go` | Update dry-run to use adapter-specific URL and auth header |
| `internal/cli/edit.go` | Update dry-run to use adapter-specific URL and auth header |
| `internal/cli/edit_mask.go` | Update `printEditDryRun` to accept adapter-specific URL and auth header |

---

## Task 1: fal Adapter - Struct, Constructor, Registration

**Files:**
- Create: `internal/adapter/fal/fal.go`
- Test: `internal/adapter/fal/fal_test.go`

**Interfaces:**
- Consumes: `github.com/ncxton/potaco/internal/adapter` (Adapter interface, AdapterOpts, shared types)
- Produces: `fal.New(apiKey string, opts adapter.AdapterOpts) adapter.Adapter`, `fal.Adapter` struct with `SetBackoff(fn)`/`SetSleep(fn)` methods, registers `"fal"` in adapter registry via `init()`

- [ ] **Step 1: Write the failing test**

```go
package fal

import (
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalAdapterName(t *testing.T) {
	ad := New("test-key", adapter.AdapterOpts{})
	if ad.Name() != "fal" {
		t.Errorf("Name() = %q, want %q", ad.Name(), "fal")
	}
}

func TestFalAuthHeader(t *testing.T) {
	ad := New("my-key", adapter.AdapterOpts{})
	got := ad.AuthHeader("my-key")
	want := "Key my-key"
	if got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestFalNewWithDefaults(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	fa, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *fal.Adapter, got %T", ad)
	}
	if fa.baseURL != "https://fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://fal.run")
	}
	if fa.retries != 2 {
		t.Errorf("retries = %d, want 2", fa.retries)
	}
}

func TestFalNewWithOverrides(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{
		BaseURL: "https://custom.fal.run",
		Retries: 5,
	})
	fa := ad.(*Adapter)
	if fa.baseURL != "https://custom.fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://custom.fal.run")
	}
	if fa.retries != 5 {
		t.Errorf("retries = %d, want 5", fa.retries)
	}
}

func TestFalRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("fal", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"fal\") failed: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/fal/ -v`
Expected: FAIL with "package not found" or "undefined: New"

- [ ] **Step 3: Write minimal implementation**

```go
// Package fal implements the adapter.Adapter interface for the fal API.
package fal

import (
	"context"
	"net/http"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

const (
	defaultBaseURL    = "https://fal.run"
	defaultAPIBaseURL = "https://api.fal.ai"
)

// Adapter implements adapter.Adapter for the fal API.
type Adapter struct {
	apiKey     string
	baseURL    string // inference base URL (https://fal.run)
	apiBaseURL string // discovery/verify base URL (https://api.fal.ai)
	retries    int
	timeout    time.Duration
	http       *http.Client
	backoff    func(attempt int) time.Duration
	sleep      func(ctx context.Context, d time.Duration)
}

// New creates a fal adapter with the given API key and options.
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
	retries := opts.Retries
	if retries == 0 {
		retries = 2
	}
	return &Adapter{
		apiKey:     apiKey,
		baseURL:    baseURL,
		apiBaseURL: defaultAPIBaseURL,
		retries:    retries,
		timeout:    timeout,
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

// Name returns the provider name.
func (a *Adapter) Name() string { return "fal" }

// AuthHeader returns the Authorization header value for the given API key.
// fal uses "Key" prefix instead of "Bearer".
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Key " + apiKey
}

// generateURL returns the full URL for a generate request to a model endpoint.
// The model ID is used as the path (e.g., fal-ai/flux/dev -> https://fal.run/fal-ai/flux/dev).
func (a *Adapter) generateURL(modelID string) string {
	return a.baseURL + "/" + modelID
}

// editURL returns the full URL for an edit request by appending /image-to-image
// to the model endpoint.
func (a *Adapter) editURL(modelID string) string {
	return a.baseURL + "/" + modelID + "/image-to-image"
}

// modelsURL returns the full URL for the models listing endpoint on the API host.
func (a *Adapter) modelsURL() string {
	return a.apiBaseURL + "/v1/models"
}

func init() {
	adapter.Register("fal", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/fal/ -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/fal/fal.go internal/adapter/fal/fal_test.go
git commit -m "fal: add adapter struct, constructor, and registry registration"
```

---

## Task 2: fal Adapter - Generate Method

**Files:**
- Create: `internal/adapter/fal/generate.go`
- Modify: `internal/adapter/fal/fal_test.go` (add generate tests)
- Create: `internal/adapter/fal/response.go`
- Create: `internal/adapter/fal/retry.go`

**Interfaces:**
- Consumes: `fal.Adapter` struct from Task 1, `adapter.GenerateRequest`, `adapter.GenerateResponse`, `adapter.ImageData`
- Produces: `Adapter.Generate(ctx, req) (*GenerateResponse, error)`, `parseResponse(resp)`, `doWithRetry(ctx, req)`, `readLimitedBody(r, limit, label)`

- [ ] **Step 1: Write the failing tests**

Add to `internal/adapter/fal/fal_test.go`:

```go
import (
	// add to existing imports:
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalGenerate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/fal-ai/flux/dev" {
			t.Errorf("path = %q, want /fal-ai/flux/dev", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/result1.png", "width": 1024, "height": 1024},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "fal-ai/flux/dev",
		N:      1,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://cdn.fal.ai/result1.png" {
		t.Errorf("URL = %q, want cdn url", resp.Data[0].URL)
	}

	// Verify body mapping: N -> num_images, Size -> image_size
	if gotBody["prompt"] != "a cat" {
		t.Errorf("body prompt = %v, want 'a cat'", gotBody["prompt"])
	}
	if gotBody["num_images"] != float64(1) {
		t.Errorf("body num_images = %v, want 1", gotBody["num_images"])
	}
	if gotBody["image_size"] != "1024x1024" {
		t.Errorf("body image_size = %v, want '1024x1024'", gotBody["image_size"])
	}
}

func TestFalGenerateWithExtraParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		json.Unmarshal(body, &gotBody)

		if gotBody["guidance_scale"] != float64(7.5) {
			t.Errorf("guidance_scale = %v, want 7.5", gotBody["guidance_scale"])
		}
		if gotBody["num_inference_steps"] != float64(50) {
			t.Errorf("num_inference_steps = %v, want 50", gotBody["num_inference_steps"])
		}
		if gotBody["output_format"] != "png" {
			t.Errorf("output_format = %v, want 'png'", gotBody["output_format"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/result.png"},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt:        "a cat",
		Model:         "fal-ai/flux/dev",
		GuidanceScale: 7.5,
		ExtraParams: map[string]any{
			"num_inference_steps": 50,
			"output_format":       "png",
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestFalGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"detail": "invalid model",
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "fal-ai/invalid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/fal/ -run TestFalGenerate -v`
Expected: FAIL with "undefined: Generate" or compile error

- [ ] **Step 3: Write the response parser**

Create `internal/adapter/fal/response.go`:

```go
package fal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

var maxResponseBytes int64 = 128 << 20

// parseResponse reads the fal API response and normalizes it to the
// shared GenerateResponse. fal returns images as images[] with url fields,
// which we map to GenerateResponse.Data.
func parseResponse(resp *http.Response) (*adapter.GenerateResponse, error) {
	respBody, err := readLimitedBody(resp.Body, maxResponseBytes, "provider response")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Detail string `json:"detail"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
			msg := errResp.Detail
			if msg == "" {
				msg = errResp.Message
			}
			if msg != "" {
				return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// fal response: { "images": [ { "url": "...", "content_type": "..." } ] }
	var falResp struct {
		Images []struct {
			URL         string `json:"url"`
			Content     []byte `json:"content,omitempty"` // base64 data URI in some responses
			ContentType string `json:"content_type,omitempty"`
		} `json:"images"`
	}
	if err := json.Unmarshal(respBody, &falResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	imgResp := &adapter.GenerateResponse{}
	for _, img := range falResp.Images {
		imgResp.Data = append(imgResp.Data, adapter.ImageData{
			URL: img.URL,
		})
	}
	return imgResp, nil
}

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

- [ ] **Step 4: Write the retry logic**

Create `internal/adapter/fal/retry.go`:

```go
package fal

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
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

		if _, err := io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes); err != nil && err != io.EOF {
			resp.Body.Close()
			return nil, fmt.Errorf("drain retry response: %w", err)
		}
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

- [ ] **Step 5: Write the Generate method**

Create `internal/adapter/fal/generate.go`:

```go
package fal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// Generate calls POST /<model_id> on the fal inference endpoint.
func (a *Adapter) Generate(ctx context.Context, req adapter.GenerateRequest) (*adapter.GenerateResponse, error) {
	body := a.buildGenerateBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := a.generateURL(req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// buildGenerateBody converts the normalized GenerateRequest to the fal
// JSON schema. Key mappings: N -> num_images, Size -> image_size,
// GuidanceScale -> guidance_scale, Seed -> seed.
// Provider-specific fields pass through ExtraParams.
func (a *Adapter) buildGenerateBody(req adapter.GenerateRequest) map[string]any {
	body := map[string]any{
		"prompt": req.Prompt,
	}
	if req.N > 0 {
		body["num_images"] = req.N
	}
	if req.Size != "" {
		body["image_size"] = req.Size
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
	// Enable sync_mode so results are returned in the response body
	// rather than requiring a webhook callback.
	body["sync_mode"] = true
	for k, v := range req.ExtraParams {
		body[k] = v
	}
	return body
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapter/fal/ -v`
Expected: PASS (all tests including generate)

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/fal/generate.go internal/adapter/fal/response.go internal/adapter/fal/retry.go internal/adapter/fal/fal_test.go
git commit -m "fal: add Generate method with response parsing and retry logic"
```

---

## Task 3: fal Adapter - Edit Method

**Files:**
- Create: `internal/adapter/fal/edit.go`
- Modify: `internal/adapter/fal/fal_test.go` (add edit tests)

**Interfaces:**
- Consumes: `fal.Adapter` struct, `adapter.EditRequest`, `adapter.ImageData`
- Produces: `Adapter.Edit(ctx, req) (*GenerateResponse, error)`

- [ ] **Step 1: Write the failing tests**

Add to `internal/adapter/fal/fal_test.go`:

```go
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
		json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/edited.png"},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	fa.SetSleep(func(context.Context, time.Duration) {})

	// Create a small test image file
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

	// Verify body has image_url as data URI
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
		json.Unmarshal(body, &gotBody)

		if _, ok := gotBody["mask_url"]; !ok {
			t.Error("mask_url not found in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://cdn.fal.ai/edited.png"},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL})
	fa := ad.(*Adapter)
	fa.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
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
```

Also add the test helper function:

```go
import (
	// add to imports:
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/fal/ -run TestFalEdit -v`
Expected: FAIL with compile error (undefined: Edit)

- [ ] **Step 3: Write the Edit method**

Create `internal/adapter/fal/edit.go`:

```go
package fal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ncxton/potaco/internal/adapter"
)

// Edit calls POST /<model_id>/image-to-image with a JSON body containing
// the source image as a base64 data URI. The mask, if provided, is also
// encoded as a data URI.
func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (*adapter.GenerateResponse, error) {
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}

	imageDataURI, err := fileToDataURI(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	body := map[string]any{
		"prompt":    req.Prompt,
		"image_url": imageDataURI,
	}
	if req.MaskPath != "" {
		maskURI, err := fileToDataURI(req.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("encode mask: %w", err)
		}
		body["mask_url"] = maskURI
	}
	if req.N > 0 {
		body["num_images"] = req.N
	}
	if req.Size != "" {
		body["image_size"] = req.Size
	}
	body["sync_mode"] = true
	for k, v := range req.ExtraParams {
		body[k] = v
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := a.editURL(req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// fileToDataURI reads a file and returns it as a base64 data URI
// with a PNG content type.
func fileToDataURI(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:image/png;base64," + encoded, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapter/fal/ -v`
Expected: PASS (all tests including edit)

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/fal/edit.go internal/adapter/fal/fal_test.go
git commit -m "fal: add Edit method with base64 data URI image encoding"
```

---

## Task 4: fal Adapter - DiscoverModels, Verify, ModelParams

**Files:**
- Create: `internal/adapter/fal/discover.go`
- Create: `internal/adapter/fal/models.go`
- Modify: `internal/adapter/fal/fal_test.go` (add discovery/verify/params tests)

**Interfaces:**
- Consumes: `fal.Adapter` struct, `adapter.Model`, `adapter.Param`
- Produces: `Adapter.DiscoverModels(ctx)`, `Adapter.Verify(ctx)`, `Adapter.ModelParams(ctx, modelID)`, `fallbackModels`, `hardcodedModelParams`, `editEndpointSuffixes`

- [ ] **Step 1: Write the failing tests**

Add to `internal/adapter/fal/fal_test.go`:

```go
func TestFalDiscoverModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		if qs := r.URL.RawQuery; qs != "category=image" {
			t.Errorf("query = %q, want 'category=image'", qs)
		}
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"id": "fal-ai/flux/dev",
					"metadata": map[string]any{
						"display_name": "Flux Dev",
					},
				},
				{
					"id": "fal-ai/flux/dev/image-to-image",
					"metadata": map[string]any{
						"display_name": "Flux Dev Image-to-Image",
					},
				},
				{
					"id": "fal-ai/nano-banana",
					"metadata": map[string]any{
						"display_name": "Nano Banana",
					},
				},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("models len = %d, want 3", len(models))
	}
	// Check first model
	if models[0].ID != "fal-ai/flux/dev" {
		t.Errorf("model[0] ID = %q, want 'fal-ai/flux/dev'", models[0].ID)
	}
	if models[0].DisplayName != "Flux Dev" {
		t.Errorf("model[0] DisplayName = %q, want 'Flux Dev'", models[0].DisplayName)
	}
	if !models[0].SupportsGen {
		t.Error("model[0] should support gen")
	}
	if models[0].SupportsEdit {
		t.Error("model[0] should not support edit")
	}
	// Check edit-capable model
	if models[1].ID != "fal-ai/flux/dev/image-to-image" {
		t.Errorf("model[1] ID = %q", models[1].ID)
	}
	if !models[1].SupportsEdit {
		t.Error("model[1] should support edit (has 'image-to-image' in ID)")
	}
}

func TestFalDiscoverModelsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels should fall back, got error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("fallback models should not be empty")
	}
}

func TestFalVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Key test-key" {
			t.Errorf("auth = %q, want 'Key test-key'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	if err := ad.Verify(context.Background()); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestFalVerifyInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	ad := New("bad-key", adapter.AdapterOpts{})
	fa := ad.(*Adapter)
	fa.apiBaseURL = srv.URL

	err := ad.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("error should mention invalid key, got: %v", err)
	}
}

func TestFalModelParamsKnownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	params, err := ad.ModelParams(context.Background(), "fal-ai/flux/dev")
	if err != nil {
		t.Fatalf("ModelParams: %v", err)
	}
	if len(params) == 0 {
		t.Fatal("params should not be empty for known model")
	}
}

func TestFalModelParamsUnknownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.ModelParams(context.Background(), "unknown-model")
	if err != adapter.ErrModelNotFound {
		t.Errorf("ModelParams unknown model: got %v, want ErrModelNotFound", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/fal/ -run TestFalDiscover -v`
Expected: FAIL with compile error (undefined: DiscoverModels)

- [ ] **Step 3: Write the models file**

Create `internal/adapter/fal/models.go`:

```go
package fal

import "github.com/ncxton/potaco/internal/adapter"

// editEndpointPatterns identifies edit-capable models by checking if
// the model ID contains any of these substrings.
var editEndpointPatterns = []string{"image-to-image", "edit"}

// isEditEndpoint returns true if the model ID suggests edit support.
func isEditEndpoint(modelID string) bool {
	for _, pattern := range editEndpointPatterns {
		if strings.Contains(modelID, pattern) {
			return true
		}
	}
	return false
}

// fallbackModels is the hardcoded list used when API discovery fails.
var fallbackModels = []adapter.Model{
	{ID: "fal-ai/flux/dev", DisplayName: "Flux Dev", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"guidance_scale", "num_inference_steps", "seed", "output_format", "image_size", "num_images", "enable_safety_checker"}},
	{ID: "fal-ai/flux/schnell", DisplayName: "Flux Schnell", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"num_inference_steps", "seed", "output_format", "image_size", "num_images"}},
	{ID: "fal-ai/nano-banana", DisplayName: "Nano Banana", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"aspect_ratio", "output_format", "safety_tolerance", "system_prompt"}},
}

// hardcodedModelParams maps model family prefixes to their supported parameters.
var hardcodedModelParams = map[string][]adapter.Param{
	"fal-ai/flux/": {
		{Name: "guidance_scale", Type: "float", Description: "Guidance scale for generation", Default: "3.5"},
		{Name: "num_inference_steps", Type: "int", Description: "Number of inference steps", Default: "50"},
		{Name: "seed", Type: "int", Description: "Reproducibility seed", Default: "0"},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "image_size", Type: "string", Description: "Image dimensions (WxH or preset)", Default: "1024x1024"},
		{Name: "num_images", Type: "int", Description: "Number of images", Default: "1"},
		{Name: "enable_safety_checker", Type: "bool", Description: "Enable safety checker", Default: "true"},
	},
	"fal-ai/nano-banana": {
		{Name: "aspect_ratio", Type: "string", Description: "Aspect ratio (e.g., 16:9, 1:1)", Default: "1:1"},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg"}},
		{Name: "safety_tolerance", Type: "int", Description: "Safety tolerance level (1-6)", Default: "2"},
		{Name: "system_prompt", Type: "string", Description: "System prompt", Default: ""},
	},
}

// lookupModelParams finds params for a model ID by matching against
// hardcoded family prefixes.
func lookupModelParams(modelID string) ([]adapter.Param, bool) {
	for prefix, params := range hardcodedModelParams {
		if strings.HasPrefix(modelID, prefix) {
			return params, true
		}
	}
	return nil, false
}
```

Also add `"strings"` to the imports in `models.go`:

```go
import (
	"strings"

	"github.com/ncxton/potaco/internal/adapter"
)
```

- [ ] **Step 4: Write the discover file**

Create `internal/adapter/fal/discover.go`:

```go
package fal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// DiscoverModels calls GET /v1/models?category=image on the fal API host
// and returns a normalized list of image models. On API failure it falls
// back to a hardcoded list.
func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.modelsURL() + "?category=image"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Key "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fallbackModels, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fallbackModels, nil
	}

	var result struct {
		Models []struct {
			ID       string `json:"id"`
			Metadata struct {
				DisplayName string `json:"display_name"`
			} `json:"metadata"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fallbackModels, nil
	}

	var models []adapter.Model
	for _, m := range result.Models {
		displayName := m.Metadata.DisplayName
		if displayName == "" {
			displayName = m.ID
		}
		models = append(models, adapter.Model{
			ID:           m.ID,
			DisplayName:  displayName,
			SupportsGen:  true,
			SupportsEdit: isEditEndpoint(m.ID),
			Capabilities: modelCapabilities(m.ID),
		})
	}
	if len(models) == 0 {
		return fallbackModels, nil
	}
	return models, nil
}

// Verify calls GET /v1/models on the fal API host to check whether the
// API key is valid and the endpoint is reachable.
func (a *Adapter) Verify(ctx context.Context) error {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Key "+a.apiKey)

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

// ModelParams returns the supported parameters for the given model ID.
// fal does not expose a per-model parameter schema via API, so this
// uses hardcoded defaults matched by model family prefix.
func (a *Adapter) ModelParams(ctx context.Context, modelID string) ([]adapter.Param, error) {
	params, ok := lookupModelParams(modelID)
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}

// modelCapabilities returns capability strings derived from the
// hardcoded parameter names for a model ID.
func modelCapabilities(modelID string) []string {
	if params, ok := lookupModelParams(modelID); ok {
		caps := make([]string, len(params))
		for i, p := range params {
			caps[i] = p.Name
		}
		return caps
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/adapter/fal/ -v`
Expected: PASS (all tests)

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/fal/discover.go internal/adapter/fal/models.go internal/adapter/fal/fal_test.go
git commit -m "fal: add DiscoverModels, Verify, and ModelParams with hardcoded fallbacks"
```

---

## Task 5: Vercel Adapter - Struct, Constructor, Generate, Registration

**Files:**
- Create: `internal/adapter/vercel/vercel.go`
- Create: `internal/adapter/vercel/generate.go`
- Create: `internal/adapter/vercel/response.go`
- Create: `internal/adapter/vercel/retry.go`
- Create: `internal/adapter/vercel/vercel_test.go`

**Interfaces:**
- Consumes: `github.com/ncxton/potaco/internal/adapter` (Adapter interface, AdapterOpts, shared types)
- Produces: `vercel.New(apiKey string, opts adapter.AdapterOpts) adapter.Adapter`, `vercel.Adapter` struct, `Adapter.Generate(ctx, req)`, registers `"vercel"` in adapter registry via `init()`

- [ ] **Step 1: Write the failing tests**

Create `internal/adapter/vercel/vercel_test.go`:

```go
package vercel

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

func TestVercelAdapterName(t *testing.T) {
	ad := New("test-key", adapter.AdapterOpts{})
	if ad.Name() != "vercel" {
		t.Errorf("Name() = %q, want %q", ad.Name(), "vercel")
	}
}

func TestVercelAuthHeader(t *testing.T) {
	ad := New("my-key", adapter.AdapterOpts{})
	got := ad.AuthHeader("my-key")
	want := "Bearer my-key"
	if got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestVercelNewWithDefaults(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	va, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *vercel.Adapter, got %T", ad)
	}
	if va.baseURL != "https://ai-gateway.vercel.sh/v1" {
		t.Errorf("baseURL = %q, want %q", va.baseURL, "https://ai-gateway.vercel.sh/v1")
	}
}

func TestVercelRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("vercel", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"vercel\") failed: %v", err)
	}
}

func TestVercelGenerate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want 'Bearer test-key'", auth)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data": []map[string]any{
				{"b64_json": "iVBORw0KGgo="},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	resp, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "openai/gpt-image-2",
		N:      1,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "iVBORw0KGgo=" {
		t.Errorf("b64_json = %q, want 'iVBORw0KGgo='", resp.Data[0].B64JSON)
	}
	if gotBody["model"] != "openai/gpt-image-2" {
		t.Errorf("body model = %v, want 'openai/gpt-image-2'", gotBody["model"])
	}
}

func TestVercelGenerateWithProviderOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		var gotBody map[string]any
		json.Unmarshal(body, &gotBody)

		po, ok := gotBody["providerOptions"]
		if !ok {
			t.Fatal("providerOptions not found in body")
		}
		poMap, ok := po.(map[string]any)
		if !ok {
			t.Fatalf("providerOptions is %T, want map", po)
		}
		if _, ok := poMap["blackForestLabs"]; !ok {
			t.Error("providerOptions.blackForestLabs not found")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"url": "https://example.com/result.png"},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "a cat",
		Model:  "bfl/flux-2-pro",
		ExtraParams: map[string]any{
			"providerOptions": map[string]any{
				"blackForestLabs": map[string]any{
					"outputFormat": "png",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestVercelGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid model id",
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	va := ad.(*Adapter)
	va.SetBackoff(func(int) time.Duration { return 1 * time.Millisecond })
	va.SetSleep(func(context.Context, time.Duration) {})

	_, err := ad.Generate(context.Background(), adapter.GenerateRequest{
		Prompt: "test",
		Model:  "invalid/model",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid model id") {
		t.Errorf("error should contain 'invalid model id', got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/vercel/ -v`
Expected: FAIL with "package not found" or "undefined: New"

- [ ] **Step 3: Write the struct and constructor**

Create `internal/adapter/vercel/vercel.go`:

```go
// Package vercel implements the adapter.Adapter interface for the
// Vercel AI Gateway.
package vercel

import (
	"context"
	"net/http"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

const defaultBaseURL = "https://ai-gateway.vercel.sh/v1"

// Adapter implements adapter.Adapter for the Vercel AI Gateway.
type Adapter struct {
	apiKey  string
	baseURL string
	retries int
	timeout time.Duration
	http    *http.Client
	backoff func(attempt int) time.Duration
	sleep   func(ctx context.Context, d time.Duration)
}

// New creates a Vercel adapter with the given API key and options.
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
	retries := opts.Retries
	if retries == 0 {
		retries = 2
	}
	return &Adapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		retries: retries,
		timeout: timeout,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (a *Adapter) SetBackoff(fn func(int) time.Duration) { a.backoff = fn }
func (a *Adapter) SetSleep(fn func(context.Context, time.Duration)) {
	a.sleep = fn
}

func (a *Adapter) Name() string { return "vercel" }

// AuthHeader returns the Authorization header value.
// Vercel AI Gateway uses Bearer auth (OpenAI-compatible).
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

// generateURL returns the full URL for the images/generations endpoint.
func (a *Adapter) generateURL() string {
	return a.baseURL + "/images/generations"
}

// modelsURL returns the full URL for the models endpoint.
func (a *Adapter) modelsURL() string {
	return a.baseURL + "/models"
}

// endpointsURL returns the full URL for a model's endpoints detail.
func (a *Adapter) endpointsURL(modelID string) string {
	return a.baseURL + "/models/" + modelID + "/endpoints"
}

func init() {
	adapter.Register("vercel", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
```

- [ ] **Step 4: Write the response parser and retry logic**

Create `internal/adapter/vercel/response.go`:

```go
package vercel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

var maxResponseBytes int64 = 128 << 20

// parseResponse reads the Vercel AI Gateway response. Vercel uses the
// OpenAI-compatible response format with data[] and b64_json/url fields.
func parseResponse(resp *http.Response) (*adapter.GenerateResponse, error) {
	respBody, err := readLimitedBody(resp.Body, maxResponseBytes, "provider response")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
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

Create `internal/adapter/vercel/retry.go`:

```go
package vercel

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
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

		if _, err := io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes); err != nil && err != io.EOF {
			resp.Body.Close()
			return nil, fmt.Errorf("drain retry response: %w", err)
		}
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

- [ ] **Step 5: Write the Generate method**

Create `internal/adapter/vercel/generate.go`:

```go
package vercel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// Generate calls POST /v1/images/generations with an OpenAI-compatible
// JSON body. Model IDs are provider-prefixed (e.g., openai/gpt-image-2).
// Provider-specific options pass through ExtraParams with a
// "providerOptions" key.
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

// buildGenerateBody converts the normalized GenerateRequest to the
// Vercel AI Gateway JSON schema (OpenAI-compatible with providerOptions).
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
	for k, v := range req.ExtraParams {
		body[k] = v
	}
	return body
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapter/vercel/ -v`
Expected: PASS (all tests)

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/vercel/vercel.go internal/adapter/vercel/generate.go internal/adapter/vercel/response.go internal/adapter/vercel/retry.go internal/adapter/vercel/vercel_test.go
git commit -m "vercel: add adapter struct, Generate method, and registry registration"
```

---

## Task 6: Vercel Adapter - Edit, DiscoverModels, Verify, ModelParams

**Files:**
- Create: `internal/adapter/vercel/edit.go`
- Create: `internal/adapter/vercel/discover.go`
- Create: `internal/adapter/vercel/models.go`
- Modify: `internal/adapter/vercel/vercel_test.go` (add edit/discover/verify/params tests)

**Interfaces:**
- Consumes: `vercel.Adapter` struct, `adapter.Model`, `adapter.Param`, `adapter.ErrEditNotSupported`, `adapter.ErrModelNotFound`
- Produces: `Adapter.Edit(ctx, req)`, `Adapter.DiscoverModels(ctx)`, `Adapter.Verify(ctx)`, `Adapter.ModelParams(ctx, modelID)`, `fallbackModels`, `hardcodedModelParams`

- [ ] **Step 1: Write the failing tests**

Add to `internal/adapter/vercel/vercel_test.go`:

```go
func TestVercelEditNotSupported(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.Edit(context.Background(), adapter.EditRequest{
		Prompt:    "test",
		Model:     "openai/gpt-image-2",
		ImagePath: "/tmp/test.png",
	})
	if err != adapter.ErrEditNotSupported {
		t.Errorf("Edit error = %v, want ErrEditNotSupported", err)
	}
}

func TestVercelDiscoverModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "openai/gpt-image-2", "type": "image"},
				{"id": "openai/text-embedding-3", "type": "embedding"},
				{"id": "bfl/flux-2-pro", "type": "image"},
				{"id": "meta/llama-3", "type": "chat"},
			},
		})
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d, want 2 (only image type)", len(models))
	}
	if models[0].ID != "openai/gpt-image-2" {
		t.Errorf("model[0] ID = %q, want 'openai/gpt-image-2'", models[0].ID)
	}
	if models[1].ID != "bfl/flux-2-pro" {
		t.Errorf("model[1] ID = %q, want 'bfl/flux-2-pro'", models[1].ID)
	}
	if models[0].DisplayName != "gpt-image-2" {
		t.Errorf("model[0] DisplayName = %q, want 'gpt-image-2' (prefix stripped)", models[0].DisplayName)
	}
}

func TestVercelDiscoverModelsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		t.Fatalf("DiscoverModels should fall back: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("fallback models should not be empty")
	}
}

func TestVercelVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The verify flow makes two requests: GET /v1/models (reachability)
		// and GET /v1/models/openai/gpt-image-2/endpoints (key validation).
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/models/openai/gpt-image-2/endpoints" {
			if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
				t.Errorf("endpoints auth = %q, want 'Bearer test-key'", auth)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ad := New("test-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	if err := ad.Verify(context.Background()); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestVercelVerifyInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/models/openai/gpt-image-2/endpoints" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ad := New("bad-key", adapter.AdapterOpts{BaseURL: srv.URL + "/v1"})
	err := ad.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("error should mention invalid key, got: %v", err)
	}
}

func TestVercelModelParamsKnownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	params, err := ad.ModelParams(context.Background(), "openai/gpt-image-2")
	if err != nil {
		t.Fatalf("ModelParams: %v", err)
	}
	if len(params) == 0 {
		t.Fatal("params should not be empty")
	}
}

func TestVercelModelParamsUnknownModel(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	_, err := ad.ModelParams(context.Background(), "unknown/model")
	if err != adapter.ErrModelNotFound {
		t.Errorf("got %v, want ErrModelNotFound", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/vercel/ -run TestVercelEdit -v`
Expected: FAIL with compile error

- [ ] **Step 3: Write the Edit method**

Create `internal/adapter/vercel/edit.go`:

```go
package vercel

import (
	"context"

	"github.com/ncxton/potaco/internal/adapter"
)

// Edit returns ErrEditNotSupported. The Vercel AI Gateway does not
// support image editing.
func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (*adapter.GenerateResponse, error) {
	return nil, adapter.ErrEditNotSupported
}
```

- [ ] **Step 4: Write the models file**

Create `internal/adapter/vercel/models.go`:

```go
package vercel

import (
	"strings"

	"github.com/ncxton/potaco/internal/adapter"
)

// stripProviderPrefix removes the provider prefix from a model ID for
// display purposes (e.g., "openai/gpt-image-2" -> "gpt-image-2").
func stripProviderPrefix(modelID string) string {
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		return modelID[idx+1:]
	}
	return modelID
}

// providerPrefix extracts the provider prefix from a model ID
// (e.g., "openai/gpt-image-2" -> "openai").
func providerPrefix(modelID string) string {
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		return modelID[:idx]
	}
	return ""
}

// fallbackModels is the hardcoded list used when API discovery fails.
var fallbackModels = []adapter.Model{
	{ID: "openai/gpt-image-2", DisplayName: "gpt-image-2", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "n"}},
	{ID: "openai/dall-e-3", DisplayName: "dall-e-3", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "style", "n"}},
	{ID: "bfl/flux-2-pro", DisplayName: "flux-2-pro", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"outputFormat", "aspectRatio"}},
}

// hardcodedModelParams maps provider prefixes to their supported params.
var hardcodedModelParams = map[string][]adapter.Param{
	"openai": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536", "auto"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"bfl": {
		{Name: "outputFormat", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "aspectRatio", Type: "string", Description: "Aspect ratio", Default: "1:1"},
	},
}

func lookupModelParams(modelID string) ([]adapter.Param, bool) {
	prefix := providerPrefix(modelID)
	if prefix == "" {
		return nil, false
	}
	params, ok := hardcodedModelParams[prefix]
	return params, ok
}

func modelCapabilities(modelID string) []string {
	if params, ok := lookupModelParams(modelID); ok {
		caps := make([]string, len(params))
		for i, p := range params {
			caps[i] = p.Name
		}
		return caps
	}
	return nil
}
```

- [ ] **Step 5: Write the discover file**

Create `internal/adapter/vercel/discover.go`:

```go
package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// DiscoverModels calls GET /v1/models and filters for models with
// type == "image". Strips the provider prefix for display names.
// Falls back to hardcoded list on API failure.
func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fallbackModels, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fallbackModels, nil
	}

	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fallbackModels, nil
	}

	var models []adapter.Model
	for _, m := range result.Data {
		if m.Type != "image" {
			continue
		}
		models = append(models, adapter.Model{
			ID:           m.ID,
			DisplayName:  stripProviderPrefix(m.ID),
			SupportsGen:  true,
			SupportsEdit: false, // Vercel does not support editing
			Capabilities: modelCapabilities(m.ID),
		})
	}
	if len(models) == 0 {
		return fallbackModels, nil
	}
	return models, nil
}

// Verify performs a two-step check:
// 1. GET /v1/models to confirm the gateway is reachable.
// 2. GET /v1/models/openai/gpt-image-2/endpoints with the Bearer key
//    to validate the API key. A 401/403 means invalid key.
func (a *Adapter) Verify(ctx context.Context) error {
	// Step 1: reachability check (no auth needed, but we send it anyway).
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify reachability: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("verification failed (HTTP %d)", resp.StatusCode)
	}

	// Step 2: key validation using a well-known model ID.
	endpointURL := a.endpointsURL("openai/gpt-image-2")
	httpReq2, err := http.NewRequestWithContext(ctx, "GET", endpointURL, nil)
	if err != nil {
		return fmt.Errorf("create key validation request: %w", err)
	}
	httpReq2.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp2, err := a.http.Do(httpReq2)
	if err != nil {
		return fmt.Errorf("verify key: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == 401 || resp2.StatusCode == 403 {
		return fmt.Errorf("invalid API key (HTTP %d)", resp2.StatusCode)
	}
	return nil
}

// ModelParams returns the supported parameters for the given model ID.
// Uses hardcoded defaults based on the provider prefix in the model ID.
func (a *Adapter) ModelParams(ctx context.Context, modelID string) ([]adapter.Param, error) {
	params, ok := lookupModelParams(modelID)
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapter/vercel/ -v`
Expected: PASS (all tests)

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/vercel/edit.go internal/adapter/vercel/discover.go internal/adapter/vercel/models.go internal/adapter/vercel/vercel_test.go
git commit -m "vercel: add Edit (not supported), DiscoverModels, Verify, and ModelParams"
```

---

## Task 7: Register Adapters in CLI and Update Provider Presets

**Files:**
- Modify: `internal/cli/helpers.go`
- Test: `internal/cli/helpers_test.go` (create if not exists, or add to existing test file)

**Interfaces:**
- Consumes: `internal/adapter/fal` and `internal/adapter/vercel` packages (via blank import for `init()` registration)
- Produces: Updated `providerPresets` map with `"vercel"` entry, CLI recognizes all three providers

- [ ] **Step 1: Write the failing test**

Create `internal/cli/helpers_test.go`:

```go
package cli

import (
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestAllThreeProvidersRegistered(t *testing.T) {
	names := adapter.List()
	want := []string{"fal", "openai", "vercel"}
	if len(names) != len(want) {
		t.Fatalf("registered providers = %v, want %v", names, want)
	}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q not registered, got %v", w, names)
		}
	}
}

func TestVercelPresetExists(t *testing.T) {
	preset, ok := getProviderPreset("vercel")
	if !ok {
		t.Fatal("vercel preset not found")
	}
	if preset.BaseURL == "" {
		t.Error("vercel preset BaseURL should not be empty")
	}
	if preset.DefaultModel == "" {
		t.Error("vercel preset DefaultModel should not be empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestAllThreeProviders -v`
Expected: FAIL (vercel not registered because the import is missing)

- [ ] **Step 3: Update helpers.go imports and providerPresets**

In `internal/cli/helpers.go`, add the new adapter imports and the vercel preset:

Change the import block from:
```go
import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/openai" // register openai adapter
	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
)
```

To:
```go
import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/fal"    // register fal adapter
	_ "github.com/ncxton/potaco/internal/adapter/openai" // register openai adapter
	_ "github.com/ncxton/potaco/internal/adapter/vercel" // register vercel adapter
	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
)
```

And add vercel to the `providerPresets` map:

```go
var providerPresets = map[string]providerPreset{
	"openai": {BaseURL: "https://api.openai.com", DefaultModel: "gpt-image-2"},
	"fal":    {BaseURL: "https://fal.run", DefaultModel: "fal-ai/flux/dev"},
	"vercel": {BaseURL: "https://ai-gateway.vercel.sh", DefaultModel: "openai/gpt-image-2"},
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestAllThreeProviders -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (all tests, no regressions)

- [ ] **Step 6: Commit**

```bash
git add internal/cli/helpers.go internal/cli/helpers_test.go
git commit -m "cli: register fal and vercel adapters, add vercel provider preset"
```

---

## Task 8: Update Dry-Run Output for Multi-Provider Support

**Files:**
- Modify: `internal/cli/helpers.go` (update `printDryRun` to accept auth header)
- Modify: `internal/cli/gen.go` (use adapter-specific dry-run URL and auth header)
- Modify: `internal/cli/edit.go` (use adapter-specific dry-run URL and auth header)
- Modify: `internal/cli/edit_mask.go` (update `printEditDryRun` signature)
- Test: `internal/cli/gen_test.go`, `internal/cli/edit_test.go` (verify dry-run output for fal and vercel)

**Interfaces:**
- Consumes: `resolvedConfig.Adapter` (for `AuthHeader()` and `Name()` methods)
- Produces: Updated `printDryRun` and `printEditDryRun` that show the correct auth header type and URL per provider

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/gen_test.go`:

```go
func setupAuthProviderForProvider(t *testing.T, providerName, apiKey, model string) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "add", providerName, "--api-key", apiKey, "--force", "--model", model})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add %s: %v", providerName, err)
	}

	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
}

func TestGenDryRunFalProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "fal", "fal-key", "fal-ai/flux/dev")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--provider", "fal", "--base-url", "https://fal.run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	// fal uses "Key" auth, not "Bearer"
	if !strings.Contains(output, "Key [REDACTED]") {
		t.Errorf("dry-run should show 'Key [REDACTED]' for fal, got: %s", output)
	}
	if !strings.Contains(output, "fal.run") {
		t.Errorf("dry-run should show fal.run URL, got: %s", output)
	}
}

func TestGenDryRunVercelProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "vercel", "vkey", "openai/gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--provider", "vercel", "--base-url", "https://ai-gateway.vercel.sh/v1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Bearer [REDACTED]") {
		t.Errorf("dry-run should show 'Bearer [REDACTED]' for vercel, got: %s", output)
	}
	if !strings.Contains(output, "images/generations") {
		t.Errorf("dry-run should show images/generations URL, got: %s", output)
	}
}
```

Add to `internal/cli/edit_test.go`:

```go
func TestEditDryRunFalProvider(t *testing.T) {
	setupAuthProviderForProvider(t, "fal", "fal-key", "fal-ai/flux/dev")

	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	createTestPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run", "--provider", "fal", "--base-url", "https://fal.run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Key [REDACTED]") {
		t.Errorf("dry-run should show 'Key [REDACTED]' for fal, got: %s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestGenDryRunFal -v`
Expected: FAIL (dry-run still shows "Bearer [REDACTED]" for all providers)

- [ ] **Step 3: Update printDryRun to accept auth header**

In `internal/cli/helpers.go`, change the `printDryRun` function signature and body:

From:
```go
func printDryRun(cmd *cobra.Command, method, url, contentType string, body any) error {
	// ...
		"headers": map[string]string{
			"Authorization": "Bearer [REDACTED]",
		},
	// ...
}
```

To:
```go
func printDryRun(cmd *cobra.Command, method, url, contentType, authHeader string, body any) error {
	bodyJSON, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run body: %w", err)
	}

	dryRunOutput := map[string]any{
		"method":       method,
		"url":          url,
		"content_type": contentType,
		"headers": map[string]string{
			"Authorization": authHeader,
		},
		"body": json.RawMessage(bodyJSON),
	}

	output, err := json.MarshalIndent(dryRunOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run output: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(output))
	return nil
}
```

- [ ] **Step 4: Update gen.go dry-run call**

In `internal/cli/gen.go`, change the dry-run block:

From:
```go
	if dryRun {
		return printDryRun(cmd, "POST", resolved.BaseURL+"/v1/images/generations", "application/json", req)
	}
```

To:
```go
	if dryRun {
		// Build the dry-run URL based on the adapter type.
		// OpenAI and Vercel use /v1/images/generations, fal uses /<model_id>.
		dryRunURL := resolved.BaseURL + "/v1/images/generations"
		if resolved.Adapter.Name() == "fal" {
			dryRunURL = resolved.BaseURL + "/" + model
		}
		if resolved.Adapter.Name() == "fal" && model == "" {
			dryRunURL = resolved.BaseURL + "/<model>"
		}
		authHeader := resolved.Adapter.AuthHeader("[REDACTED]")
		return printDryRun(cmd, "POST", dryRunURL, "application/json", authHeader, req)
	}
```

- [ ] **Step 5: Update edit.go and edit_mask.go dry-run**

In `internal/cli/edit.go`, change the dry-run call:

From:
```go
	if dryRun {
		return printEditDryRun(cmd, resolved.BaseURL, prompt, model, imagePath, cmd.Flags())
	}
```

To:
```go
	if dryRun {
		authHeader := resolved.Adapter.AuthHeader("[REDACTED]")
		return printEditDryRun(cmd, resolved.BaseURL, resolved.Adapter.Name(), authHeader, prompt, model, imagePath, cmd.Flags())
	}
```

In `internal/cli/edit_mask.go`, update the `printEditDryRun` function. The current signature is:

```go
func printEditDryRun(cmd *cobra.Command, baseURL, prompt, model, imagePath string, flags *pflag.FlagSet) error {
```

Change it to:

```go
func printEditDryRun(cmd *cobra.Command, baseURL, providerName, authHeader, prompt, model, imagePath string, flags *pflag.FlagSet) error {
```

Then update the final `printDryRun` call at the end of the function. The current call is:

```go
	return printDryRun(cmd, "POST", baseURL+"/v1/images/edits", "multipart/form-data", body)
```

Change it to build the URL per-provider and pass the auth header:

```go
	editURL := baseURL + "/v1/images/edits"
	if providerName == "fal" {
		editURL = baseURL + "/" + model + "/image-to-image"
	}
	return printDryRun(cmd, "POST", editURL, "application/json", authHeader, body)
```

Note: fal uses `application/json` for edits (not multipart), while OpenAI uses `multipart/form-data`. The content type in the dry-run output should reflect the actual request format.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestGenDryRun -v`
Run: `go test ./internal/cli/ -run TestEditDryRun -v`
Expected: PASS

- [ ] **Step 7: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (all tests, no regressions)

- [ ] **Step 8: Commit**

```bash
git add internal/cli/helpers.go internal/cli/gen.go internal/cli/edit.go internal/cli/edit_mask.go internal/cli/gen_test.go internal/cli/edit_test.go
git commit -m "cli: update dry-run output to show per-provider auth header and URL"
```

---

## Task 9: Final Verification

**Files:**
- No new files. Verification only.

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass across all packages (adapter, adapter/fal, adapter/openai, adapter/vercel, auth, cli, config, credential, image)

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Run gofmt**

Run: `gofmt -l .`
Expected: No output (all files formatted)

- [ ] **Step 4: Verify build**

Run: `go build -o potaco .`
Expected: Success

- [ ] **Step 5: Smoke test - auth add with fal**

```bash
rm -rf /tmp/potaco-test
HOME=/tmp/potaco-test ./potaco auth add fal --api-key fal-test-key --force
HOME=/tmp/potaco-test ./potaco auth list
HOME=/tmp/potaco-test ./potaco gen --prompt "a cat" --dry-run
```
Expected: Provider 'fal' added, listed with model fal-ai/flux/dev, dry-run shows Key auth header and fal.run URL

- [ ] **Step 6: Smoke test - auth add with vercel**

```bash
HOME=/tmp/potaco-test ./potaco auth add vercel --api-key vkey --force
HOME=/tmp/potaco-test ./potaco auth list
HOME=/tmp/potaco-test ./potaco use vercel
HOME=/tmp/potaco-test ./potaco gen --prompt "a cat" --dry-run
```
Expected: Vercel added, listed, switched to vercel, dry-run shows Bearer auth and ai-gateway.vercel.sh URL

- [ ] **Step 7: Smoke test - edit on vercel shows not supported**

```bash
# Create a tiny test image
HOME=/tmp/potaco-test ./potaco edit --prompt "test" --image /tmp/potaco-test/test.png --provider vercel
```
Expected: Error message about edit not supported by Vercel

- [ ] **Step 8: Check all file LOCs are under 250**

Run:
```bash
for f in internal/adapter/fal/*.go internal/adapter/vercel/*.go; do
  loc=$(awk '!/^[[:space:]]*$/ && !/^[[:space:]]*(\/\/)/' "$f" | wc -l)
  echo "$loc  $f"
done
```
Expected: All files under 250 pure LOC

- [ ] **Step 9: Commit any final fixes if needed**

If any issues were found and fixed:
```bash
git add -A
git commit -m "fix: final verification fixes for Phase 3"
```

Otherwise, no commit needed.
