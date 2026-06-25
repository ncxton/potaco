// Package openai implements the adapter.Adapter interface for the
// OpenAI Images API.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
)

// defaultBaseURL is the default OpenAI API base URL.
const defaultBaseURL = "https://api.openai.com/v1"

// maxResponseBytes bounds the response body size. It is a variable so
// tests can lower it without allocating huge fixtures.
var maxResponseBytes int64 = 128 << 20

// maxRetryDrainBytes bounds the number of bytes discarded from a retry
// response body before closing it. It is a variable so tests can lower it.
var maxRetryDrainBytes int64 = 1 << 20

// Adapter implements adapter.Adapter for the OpenAI Images API.
type Adapter struct {
	apiKey  string
	baseURL string
	retries int
	timeout time.Duration
	http    *http.Client
	backoff func(attempt int) time.Duration
	sleep   func(ctx context.Context, d time.Duration)
}

// New creates an OpenAI adapter with the given API key and options.
func New(apiKey string, opts adapter.AdapterOpts) adapter.Adapter {
	baseURL := defaultBaseURL
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	timeout := 120 * time.Second
	if opts.Timeout != "" {
		if d, err := time.ParseDuration(opts.Timeout); err == nil {
			timeout = d
		}
	}
	return &Adapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		retries: 2,
		timeout: timeout,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetBackoff overrides the backoff function (for testing).
func (a *Adapter) SetBackoff(fn func(int) time.Duration) {
	a.backoff = fn
}

// SetSleep overrides the sleep function (for testing).
func (a *Adapter) SetSleep(fn func(context.Context, time.Duration)) {
	a.sleep = fn
}

// Name returns the provider name.
func (a *Adapter) Name() string { return "openai" }

// AuthHeader returns the Authorization header value for the given API key.
func (a *Adapter) AuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

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

// generateURL returns the full URL for the generations endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) generateURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/generations"
	}
	return a.baseURL + "/v1/images/generations"
}

// editURL returns the full URL for the edits endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) editURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/images/edits"
	}
	return a.baseURL + "/v1/images/edits"
}

// modelsURL returns the full URL for the models endpoint,
// handling whether baseURL already includes /v1.
func (a *Adapter) modelsURL() string {
	if strings.HasSuffix(a.baseURL, "/v1") {
		return a.baseURL + "/models"
	}
	return a.baseURL + "/v1/models"
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

// defaultBackoff returns the exponential backoff duration for a given
// attempt. Attempt 0 = 1s, 1 = 2s, 2+ = 4s. Jitter of 0-500ms is added.
func defaultBackoff(attempt int) time.Duration {
	base := time.Second
	switch attempt {
	case 0:
		base = 1 * time.Second
	case 1:
		base = 2 * time.Second
	default:
		base = 4 * time.Second
	}
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	return base + jitter
}

// shouldRetry returns true if the status code warrants a retry.
// Retries on 429 (rate limit) and 5xx (server errors).
func shouldRetry(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

// retryDelay returns the delay before the next retry attempt. If the
// response carries a parseable Retry-After header (non-negative integer
// seconds), that value takes precedence over the fallback backoff.
func retryDelay(resp *http.Response, attempt int, fallback func(int) time.Duration) time.Duration {
	if resp != nil {
		if raw := resp.Header.Get("Retry-After"); raw != "" {
			if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	return fallback(attempt)
}

// doWithRetry executes the given request, retrying on 429 and 5xx with
// exponential backoff up to a.retries times. The ctx allows cancelling
// an in-progress retry sequence.
func (a *Adapter) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; ; attempt++ {
		resp, err := a.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt < a.retries {
				a.backoffSleep(ctx, retryDelay(nil, attempt, a.backoffOrDefault))
				if req.GetBody != nil {
					body, err := req.GetBody()
					if err == nil {
						req.Body = body
					}
				}
				continue
			}
			return nil, fmt.Errorf("http request: %w", err)
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp

		if attempt >= a.retries {
			break
		}

		_, _ = io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes)
		resp.Body.Close()

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				req.Body = body
			}
		}

		a.backoffSleep(ctx, retryDelay(resp, attempt, a.backoffOrDefault))
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

// backoffOrDefault returns the backoff duration for an attempt, using the
// test-overridable backoff function when set, otherwise the default.
func (a *Adapter) backoffOrDefault(attempt int) time.Duration {
	if a.backoff != nil {
		return a.backoff(attempt)
	}
	return defaultBackoff(attempt)
}

// backoffSleep sleeps for d, respecting ctx cancellation. Uses the
// test-overridable sleep function when set.
func (a *Adapter) backoffSleep(ctx context.Context, d time.Duration) {
	if a.sleep != nil {
		a.sleep(ctx, d)
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}

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

// DiscoverModels calls GET /v1/models and filters for known image model
// IDs. On any API failure it falls back to a hardcoded list of models.
func (a *Adapter) DiscoverModels(ctx context.Context) ([]adapter.Model, error) {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fallbackModels, nil // fall back silently
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fallbackModels, nil
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fallbackModels, nil
	}

	var models []adapter.Model
	for _, m := range result.Data {
		if !imageModelIDs[m.ID] {
			continue
		}
		models = append(models, adapter.Model{
			ID:           m.ID,
			DisplayName:  m.ID,
			SupportsGen:  true,
			SupportsEdit: editCapableModels[m.ID],
			Capabilities: modelCapabilities(m.ID),
		})
	}
	if len(models) == 0 {
		return fallbackModels, nil
	}
	return models, nil
}

// Verify calls GET /v1/models to check whether the API key is valid and
// the endpoint is reachable. A 401/403 indicates an invalid key; any
// other 4xx indicates verification failed.
func (a *Adapter) Verify(ctx context.Context) error {
	url := a.modelsURL()
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("verification failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// ModelParams returns the supported parameters for the given model ID.
// OpenAI does not expose a per-model parameter schema via API, so this
// returns hardcoded defaults. It returns adapter.ErrModelNotFound when
// the model ID is not in the hardcoded map.
func (a *Adapter) ModelParams(ctx context.Context, modelID string) ([]adapter.Param, error) {
	params, ok := hardcodedModelParams[modelID]
	if !ok {
		return nil, adapter.ErrModelNotFound
	}
	return params, nil
}

// modelCapabilities returns the capability strings for a model ID,
// derived from the hardcoded parameter names.
func modelCapabilities(modelID string) []string {
	if params, ok := hardcodedModelParams[modelID]; ok {
		caps := make([]string, len(params))
		for i, p := range params {
			caps[i] = p.Name
		}
		return caps
	}
	return nil
}

func init() {
	adapter.Register("openai", func(apiKey string, opts adapter.AdapterOpts) (adapter.Adapter, error) {
		return New(apiKey, opts), nil
	})
}
