package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
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
