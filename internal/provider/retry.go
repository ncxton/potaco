package provider

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// defaultBackoff returns the exponential backoff duration for a given attempt.
// Attempt 0 = 1s, 1 = 2s, 2+ = 4s. Jitter of 0-500ms is added.
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

// doWithRetry executes the given request, retrying on 429 and 5xx
// with exponential backoff up to maxRetries times. The ctx allows
// cancelling an in-progress retry sequence.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; ; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				c.backoffSleep(ctx, attempt)
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

		if attempt >= maxRetries {
			break
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				req.Body = body
			}
		}

		c.backoffSleep(ctx, attempt)
	}

	if lastResp != nil {
		return lastResp, nil // let parseResponse handle the error
	}
	return nil, lastErr
}

func (c *Client) backoffSleep(ctx context.Context, attempt int) {
	d := defaultBackoff(attempt)
	if c.backoff != nil {
		d = c.backoff(attempt)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}
