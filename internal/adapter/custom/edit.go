package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
	img "github.com/ncxton/potaco/internal/image"
)

// Edit calls POST /v1/images/edits with a JSON body. The image is sent
// as a base64-encoded data URL inside the "images" array; the optional
// mask is sent as a top-level "mask" data URL so that OpenAI-compatible
// servers can validate the MIME type of each entry.
func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (*adapter.GenerateResponse, error) {
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}

	imageDataURI, err := img.FileToDataURI(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	imageEntry := map[string]any{"image_url": imageDataURI}

	body := map[string]any{
		"prompt": req.Prompt,
		"images": []map[string]any{imageEntry},
	}
	if req.MaskPath != "" {
		maskDataURI, err := img.FileToDataURI(req.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("encode mask: %w", err)
		}
		body["mask"] = maskDataURI
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
	if req.ResponseFormat != "" {
		body["response_format"] = req.ResponseFormat
	}
	if req.User != "" {
		body["user"] = req.User
	}
	for k, v := range req.ExtraParams {
		body[k] = v
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := a.editURL()
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
