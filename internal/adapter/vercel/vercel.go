// Package vercel implements the adapter.Adapter interface for the
// Vercel AI Gateway.
package vercel

import (
	"context"
	"fmt"
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

// AuthHeader returns the Authorization header value.
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

func (a *Adapter) generateURL() string {
	return a.baseURL + "/images/generations"
}

func (a *Adapter) Edit(context.Context, adapter.EditRequest) (*adapter.GenerateResponse, error) {
	return nil, fmt.Errorf("vercel edit: %w", adapter.ErrEditNotSupported)
}

func (a *Adapter) DiscoverModels(context.Context) ([]adapter.Model, error) {
	return nil, fmt.Errorf("vercel discover models: %w", adapter.ErrDiscoveryFailed)
}

func (a *Adapter) Verify(context.Context) error {
	return fmt.Errorf("vercel verify: %w", adapter.ErrVerificationFailed)
}

func (a *Adapter) ModelParams(context.Context, string) ([]adapter.Param, error) {
	return nil, fmt.Errorf("vercel model params: %w", adapter.ErrModelNotFound)
}

func init() {
	adapter.Register("vercel", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
