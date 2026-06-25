package fal

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

	if resp.StatusCode >= 400 {
		var errResp struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
			message := errResp.Detail
			if message == "" {
				message = errResp.Message
			}
			if message != "" {
				return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, message)
			}
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var falResp struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(respBody, &falResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	imgResp := &adapter.GenerateResponse{}
	for _, image := range falResp.Images {
		imgResp.Data = append(imgResp.Data, adapter.ImageData{
			URL: image.URL,
		})
	}
	return imgResp, nil
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
