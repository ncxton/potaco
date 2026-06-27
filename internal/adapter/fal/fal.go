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
	if opts.Timeout > 0 {
		timeout = opts.Timeout
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

// SupportsGenerate reports whether this provider supports image generation.
func (a *Adapter) SupportsGenerate() bool { return true }

// SupportsEdit reports whether this provider supports image editing.
func (a *Adapter) SupportsEdit() bool { return true }

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
