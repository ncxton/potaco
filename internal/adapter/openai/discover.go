package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/observability"
)

// DiscoverModels calls GET /v1/models and filters for known image model
// IDs. On any API failure it falls back to a hardcoded list of models.
func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

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

// Verify calls GET /v1/models to check whether the API key is valid and
// the endpoint is reachable. A 401/403 indicates an invalid key; any
// other 4xx indicates verification failed.
func (a *Adapter) Verify(ctx context.Context) error {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

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
// OpenAI does not expose a per-model parameter schema via API, so this
// returns hardcoded defaults. It returns adapter.ErrModelNotFound when
// the model ID is not in the hardcoded map.
func (a *Adapter) ModelParams(ctx context.Context, modelID string) ([]adapter.Param, error) {
	params, ok := hardcodedModelParams[modelID]
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}

// modelCapabilities returns the capability strings for a model ID,
// derived from the hardcoded parameter names.
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
