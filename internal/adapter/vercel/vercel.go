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

// SetBackoff overrides the backoff function for testing.
func (a *Adapter) SetBackoff(fn func(int) time.Duration) {
	a.backoff = fn
}

// SetSleep overrides the sleep function for testing.
func (a *Adapter) SetSleep(fn func(context.Context, time.Duration)) {
	a.sleep = fn
}

// Name returns the provider name.
func (a *Adapter) Name() string { return "vercel" }

// SupportsGenerate reports whether this provider supports image generation.
func (a *Adapter) SupportsGenerate() bool { return true }

// SupportsEdit reports whether this provider supports image editing.
func (a *Adapter) SupportsEdit() bool { return false }

// AuthHeader returns the Authorization header value.
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

func (a *Adapter) generateURL() string {
	return a.baseURL + "/images/generations"
}

func (a *Adapter) modelsURL() string {
	return a.baseURL + "/models"
}

func (a *Adapter) endpointsURL(modelID string) string {
	return a.baseURL + "/models/" + modelID + "/endpoints"
}

func init() {
	adapter.Register("vercel", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
