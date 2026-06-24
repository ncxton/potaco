package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ClientConfig holds the parameters for constructing a Client.
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Retries int
	Timeout time.Duration
}

// Client is the HTTP client for an OpenAI-compatible image provider.
type Client struct {
	baseURL string
	apiKey  string
	retries int
	http    *http.Client
	backoff func(attempt int) time.Duration // override for testing
}

// NewClient creates a provider Client from the given config.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		retries: cfg.Retries,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Generate calls POST /v1/images/generations and returns the response.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (*ImageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/v1/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.doWithRetry(httpReq, c.retries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// parseResponse reads the HTTP response and returns an ImageResponse or an error.
func parseResponse(resp *http.Response) (*ImageResponse, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp ImageResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &imgResp, nil
}

// Edit calls POST /v1/images/edits with multipart form data.
func (c *Client) Edit(ctx context.Context, req EditRequest) (*ImageResponse, error) {
	// Validate image file exists
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

	// Add image file part
	imgPart, err := writer.CreateFormFile("image", filepath.Base(req.ImagePath))
	if err != nil {
		return nil, fmt.Errorf("create image part: %w", err)
	}
	if _, err := io.Copy(imgPart, imgFile); err != nil {
		return nil, fmt.Errorf("copy image data: %w", err)
	}

	// Add mask file part (optional)
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

	// Add text fields
	if req.Prompt != "" {
		writer.WriteField("prompt", req.Prompt)
	}
	if req.Model != "" {
		writer.WriteField("model", req.Model)
	}
	if req.N > 0 {
		writer.WriteField("n", strconv.Itoa(req.N))
	}
	if req.Size != "" {
		writer.WriteField("size", req.Size)
	}
	if req.ResponseFormat != "" {
		writer.WriteField("response_format", req.ResponseFormat)
	}
	if req.User != "" {
		writer.WriteField("user", req.User)
	}

	writer.Close()

	url := c.baseURL + "/v1/images/edits"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.doWithRetry(httpReq, c.retries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}
