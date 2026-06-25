package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

// maxResponseBytes bounds the response body size. It is a variable so
// tests can lower it without allocating huge fixtures.
var maxResponseBytes int64 = 128 << 20

// parseResponse reads the HTTP response and returns a GenerateResponse or
// an error.
func parseResponse(resp *http.Response) (*adapter.GenerateResponse, error) {
	respBody, err := readLimitedBody(resp.Body, maxResponseBytes, "provider response")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Type    string `json:"type"`
				Code    string `json:"code,omitempty"`
				Message string `json:"message"`
				Param   string `json:"param,omitempty"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp adapter.GenerateResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &imgResp, nil
}

// readLimitedBody reads up to limit+1 bytes from r. If the body exceeds
// limit bytes, it returns an error mentioning the label. The +1 byte
// lets us detect overflow without buffering the entire stream.
func readLimitedBody(r io.Reader, limit int64, label string) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s too large: limit is %d bytes", label, limit)
	}
	return data, nil
}
