package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/observability"
)

func (a *Adapter) DiscoverModels(ctx context.Context) (models []adapter.Model, err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.modelsURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fallbackModels, nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
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

	for _, model := range result.Data {
		if model.Type != "image" {
			continue
		}
		models = append(models, adapter.Model{
			ID:           model.ID,
			DisplayName:  stripProviderPrefix(model.ID),
			SupportsGen:  true,
			SupportsEdit: false,
			Capabilities: modelCapabilities(model.ID),
		})
	}
	if len(models) == 0 {
		return fallbackModels, nil
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

func (a *Adapter) ModelParams(_ context.Context, modelID string) ([]adapter.Param, error) {
	params, ok := lookupModelParams(modelID)
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}

func (a *Adapter) verifyReachable(ctx context.Context) (err error) {
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
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

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
