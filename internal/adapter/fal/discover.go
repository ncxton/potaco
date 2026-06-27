package fal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/observability"
)

func (a *Adapter) DiscoverModels(ctx context.Context) (models []adapter.Model, err error) {
	url := a.modelsURL() + "?category=image"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

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
		Models []struct {
			ID       string `json:"id"`
			Metadata struct {
				DisplayName string `json:"display_name"`
			} `json:"metadata"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	for _, model := range result.Models {
		displayName := model.Metadata.DisplayName
		if displayName == "" {
			displayName = model.ID
		}
		models = append(models, adapter.Model{
			ID:           model.ID,
			DisplayName:  displayName,
			SupportsGen:  true,
			SupportsEdit: true,
		})
	}
	if len(models) == 0 {
		return nil, adapter.ErrDiscoveryFailed
	}
	return models, nil
}

func (a *Adapter) Verify(ctx context.Context) (err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.modelsURL(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("verification failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}
