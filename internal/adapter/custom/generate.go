package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// Generate calls POST /v1/images/generations and returns the response.
func (a *Adapter) Generate(ctx context.Context, req adapter.GenerateRequest) (*adapter.GenerateResponse, error) {
	body := a.buildGenerateBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := a.generateURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// buildGenerateBody converts the normalized GenerateRequest to the OpenAI
// JSON schema. Provider-specific fields pass through ExtraParams.
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
	if req.GuidanceScale != 0 {
		body["guidance_scale"] = req.GuidanceScale
	}
	if req.NegativePrompt != "" {
		body["negative_prompt"] = req.NegativePrompt
	}
	for k, v := range req.ExtraParams {
		body[k] = v
	}
	return body
}
