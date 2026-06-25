package vercel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ncxton/potaco/internal/adapter"
)

var maxResponseBytes int64 = 128 << 20

func parseResponse(resp *http.Response) (*adapter.GenerateResponse, error) {
	respBody, err := readLimitedBody(resp.Body, maxResponseBytes, "provider response")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
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
