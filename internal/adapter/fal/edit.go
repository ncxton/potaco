package fal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	img "github.com/ncxton/potaco/internal/image"
)

func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (result *adapter.GenerateResponse, err error) {
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}

	imageDataURI, err := img.FileToDataURI(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	body := map[string]any{
		"prompt":    req.Prompt,
		"image_url": imageDataURI,
		"sync_mode": true,
	}
	if req.MaskPath != "" {
		maskDataURI, err := img.FileToDataURI(req.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("encode mask: %w", err)
		}
		body["mask_url"] = maskDataURI
	}
	if req.N > 0 {
		body["num_images"] = req.N
	}
	if req.Size != "" {
		body["image_size"] = req.Size
	}
	for key, value := range req.ExtraParams {
		body[key] = value
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.editURL(req.Model), bytes.NewReader(bodyJSON))
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
