package provider

import (
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

// doWithRetry executes the given function, retrying on 429 and 5xx
// with exponential backoff up to maxRetries times.
func (c *Client) doWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; ; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			// Network errors: retry once
			if attempt < 1 && maxRetries > 0 {
				c.backoffSleep(attempt)
				// Recreate the request body reader if needed
				continue
			}
			return nil, fmt.Errorf("http request: %w", err)
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp

		if attempt >= maxRetries {
			// Exhausted retries, return the error response
			break
		}

		// Drain the body before retrying
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		c.backoffSleep(attempt)
	}

	if lastResp != nil {
		return lastResp, nil // let parseResponse handle the error
	}
	return nil, lastErr
}

func (c *Client) backoffSleep(attempt int) {
	if c.backoff != nil {
		time.Sleep(c.backoff(attempt))
	} else {
		time.Sleep(defaultBackoff(attempt))
	}
}
