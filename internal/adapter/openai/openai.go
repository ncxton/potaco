// Package openai implements the adapter.Adapter interface for the
// OpenAI Images API.
package openai

import (
	"context"
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

// Name returns the provider name.
func (a *Adapter) Name() string { return "openai" }

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
	adapter.Register("openai", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
