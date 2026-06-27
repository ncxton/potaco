package fal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

func (a *Adapter) Generate(ctx context.Context, req adapter.GenerateRequest) (result *adapter.GenerateResponse, err error) {
	body := a.buildGenerateBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.generateURL(req.Model), bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", a.AuthHeader(a.apiKey))

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
		"prompt":    req.Prompt,
		"sync_mode": true,
	}
	if req.N > 0 {
		body["num_images"] = req.N
	}
	if req.Size != "" {
		body["image_size"] = req.Size
	}
	if req.Seed != 0 {
		body["seed"] = req.Seed
	}
	if req.GuidanceScale != 0 {
		body["guidance_scale"] = req.GuidanceScale
	}
	if req.NegativePrompt != "" {
		body["negative_prompt"] = req.NegativePrompt
	}
	for key, value := range req.ExtraParams {
		body[key] = value
	}
	return body
}
