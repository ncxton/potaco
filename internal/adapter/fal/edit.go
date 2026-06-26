package fal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/observability"
)

func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (result *adapter.GenerateResponse, err error) {
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}

	imageDataURI, err := fileToDataURI(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	body := map[string]any{
		"prompt":    req.Prompt,
		"image_url": imageDataURI,
		"sync_mode": true,
	}
	if req.MaskPath != "" {
		maskDataURI, err := fileToDataURI(req.MaskPath)
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

func fileToDataURI(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:image/png;base64," + encoded, nil
}
