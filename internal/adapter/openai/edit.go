package openai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ncxton/potaco/internal/adapter"
)

// Edit calls POST /v1/images/edits with multipart form data.
func (a *Adapter) Edit(ctx context.Context, req adapter.EditRequest) (*adapter.GenerateResponse, error) {
	if req.ImagePath == "" {
		return nil, fmt.Errorf("image file path is required")
	}
	imgFile, err := os.Open(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("image file: %w", err)
	}
	defer imgFile.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	imgPart, err := writer.CreateFormFile("image", filepath.Base(req.ImagePath))
	if err != nil {
		return nil, fmt.Errorf("create image part: %w", err)
	}
	if _, err := io.Copy(imgPart, imgFile); err != nil {
		return nil, fmt.Errorf("copy image data: %w", err)
	}

	if req.MaskPath != "" {
		maskFile, err := os.Open(req.MaskPath)
		if err != nil {
			return nil, fmt.Errorf("mask file: %w", err)
		}
		maskPart, err := writer.CreateFormFile("mask", filepath.Base(req.MaskPath))
		if err != nil {
			maskFile.Close()
			return nil, fmt.Errorf("create mask part: %w", err)
		}
		if _, err := io.Copy(maskPart, maskFile); err != nil {
			maskFile.Close()
			return nil, fmt.Errorf("copy mask data: %w", err)
		}
		maskFile.Close()
	}

	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			return nil, fmt.Errorf("write prompt field: %w", err)
		}
	}
	if req.Model != "" {
		if err := writer.WriteField("model", req.Model); err != nil {
			return nil, fmt.Errorf("write model field: %w", err)
		}
	}
	if req.N > 0 {
		if err := writer.WriteField("n", strconv.Itoa(req.N)); err != nil {
			return nil, fmt.Errorf("write n field: %w", err)
		}
	}
	if req.Size != "" {
		if err := writer.WriteField("size", req.Size); err != nil {
			return nil, fmt.Errorf("write size field: %w", err)
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			return nil, fmt.Errorf("write response_format field: %w", err)
		}
	}
	if req.User != "" {
		if err := writer.WriteField("user", req.User); err != nil {
			return nil, fmt.Errorf("write user field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := a.editURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.doWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}
