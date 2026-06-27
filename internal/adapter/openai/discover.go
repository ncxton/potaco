package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// DiscoverModels calls GET /v1/models and filters for known image model
// IDs. On any API failure it returns an error instead of falling back to
// a hardcoded list.
func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("discover models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("discover models failed (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
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
		})
	}
	if len(models) == 0 {
		return nil, adapter.ErrDiscoveryFailed
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
