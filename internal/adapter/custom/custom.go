// Package custom implements the adapter.Adapter interface for an
// OpenAI-compatible API endpoint. It is intended for providers such as
// Together, Groq, or local vLLM servers that expose the same Images API
// as OpenAI.
package custom

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

// Adapter implements adapter.Adapter for an OpenAI-compatible endpoint.
type Adapter struct {
	apiKey  string
	baseURL string
	retries int
	timeout time.Duration
	http    *http.Client
	backoff func(attempt int) time.Duration
	sleep   func(ctx context.Context, d time.Duration)
}

// New creates a custom adapter with the given API key and options.
func New(apiKey string, opts adapter.AdapterOpts) adapter.Adapter {
	baseURL := opts.BaseURL
	timeout := 120 * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
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

// SetBackoff overrides the backoff function (for testing).
func (a *Adapter) SetBackoff(fn func(int) time.Duration) {
	a.backoff = fn
}

// SetSleep overrides the sleep function (for testing).
func (a *Adapter) SetSleep(fn func(context.Context, time.Duration)) {
	a.sleep = fn
}

// Name returns the provider name.
func (a *Adapter) Name() string { return "custom" }

// SupportsGenerate reports whether this provider supports image generation.
func (a *Adapter) SupportsGenerate() bool { return true }

// SupportsEdit reports whether this provider supports image editing.
func (a *Adapter) SupportsEdit() bool { return true }

// AuthHeader returns the Authorization header value for the given API key.
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

// generateURL returns the full URL for the generations endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) generateURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/generations"
	}
	return a.baseURL + "/v1/images/generations"
}

// editURL returns the full URL for the edits endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) editURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/edits"
	}
	return a.baseURL + "/v1/images/edits"
}

// modelsURL returns the full URL for the models endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) modelsURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/models"
	}
	return a.baseURL + "/v1/models"
}

func init() {
	adapter.Register("custom", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
