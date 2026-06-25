package provider

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// maxRetryDrainBytes bounds the number of bytes discarded from a retry
// response body before closing it. It is a variable so tests can lower it.
var maxRetryDrainBytes int64 = 1 << 20

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
				c.backoffSleep(ctx, retryDelay(nil, attempt, c.backoffOrDefault))
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

		drainRetryBody(resp)

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				req.Body = body
			}
		}

		c.backoffSleep(ctx, retryDelay(resp, attempt, c.backoffOrDefault))
	}

	if lastResp != nil {
		return lastResp, nil // let parseResponse handle the error
	}
	return nil, lastErr
}

// drainRetryBody discards up to maxRetryDrainBytes from resp.Body and then
// closes it, bounding memory and fd usage on retry paths.
func drainRetryBody(resp *http.Response) {
	_, _ = io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes)
	resp.Body.Close()
}

// backoffOrDefault returns the backoff duration for an attempt, using the
// test-overridable backoff function when set, otherwise the default.
func (c *Client) backoffOrDefault(attempt int) time.Duration {
	if c.backoff != nil {
		return c.backoff(attempt)
	}
	return defaultBackoff(attempt)
}

func (c *Client) backoffSleep(ctx context.Context, d time.Duration) {
	if c.sleep != nil {
		c.sleep(ctx, d)
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
