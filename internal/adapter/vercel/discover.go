package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

func (a *Adapter) DiscoverModels(ctx context.Context) (models []adapter.Model, err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.modelsURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("discover models: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("discover models failed (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	for _, model := range result.Data {
		if model.Type != "image" {
			continue
		}
		models = append(models, adapter.Model{
			ID:           model.ID,
			DisplayName:  stripProviderPrefix(model.ID),
			SupportsGen:  true,
			SupportsEdit: false,
		})
	}
	if len(models) == 0 {
		return nil, adapter.ErrDiscoveryFailed
	}
	return models, nil
}

func (a *Adapter) Verify(ctx context.Context) error {
	if err := a.verifyReachable(ctx); err != nil {
		return err
	}
	if err := a.verifyKey(ctx); err != nil {
		return err
	}
	return nil
}

func (a *Adapter) verifyReachable(ctx context.Context) (err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.modelsURL(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify reachability: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("verification failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) verifyKey(ctx context.Context) (err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.endpointsURL("openai/gpt-image-2"), nil)
	if err != nil {
		return fmt.Errorf("create key validation request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify key: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	return nil
}
