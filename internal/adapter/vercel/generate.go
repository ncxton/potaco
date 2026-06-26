package vercel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/observability"
)

// Generate calls POST /v1/images/generations with an OpenAI-compatible body.
func (a *Adapter) Generate(ctx context.Context, req adapter.GenerateRequest) (result *adapter.GenerateResponse, err error) {
	body := a.buildGenerateBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.generateURL(), bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))
	if rid := observability.RequestIDFromContext(ctx); rid != "" {
		httpReq.Header.Set("X-Request-ID", rid)
	}

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	return parseResponse(resp)
}

func (a *Adapter) buildGenerateBody(req adapter.GenerateRequest) map[string]any {
	body := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Model != "" {
		body["model"] = req.Model
	}
	if req.N > 0 {
		body["n"] = req.N
	}
	if req.Size != "" {
		body["size"] = req.Size
	}
	if req.Quality != "" {
		body["quality"] = req.Quality
	}
	if req.Style != "" {
		body["style"] = req.Style
	}
	if req.ResponseFormat != "" {
		body["response_format"] = req.ResponseFormat
	}
	if req.Seed != 0 {
		body["seed"] = req.Seed
	}
	for key, value := range req.ExtraParams {
		body[key] = value
	}
	return body
}
