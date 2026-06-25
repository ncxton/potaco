package fal

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var maxRetryDrainBytes int64 = 1 << 20

func defaultBackoff(attempt int) time.Duration {
	base := time.Second
	switch attempt {
	case 0:
		base = time.Second
	case 1:
		base = 2 * time.Second
	default:
		base = 4 * time.Second
	}
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	return base + jitter
}

func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func retryDelay(resp *http.Response, attempt int, fallback func(int) time.Duration) time.Duration {
	if resp != nil {
		raw := resp.Header.Get("Retry-After")
		if raw != "" {
			seconds, err := strconv.Atoi(raw)
			if err == nil && seconds >= 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	return fallback(attempt)
}

func (a *Adapter) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastResp *http.Response

	for attempt := 0; ; attempt++ {
		resp, err := a.http.Do(req)
		if err != nil {
			if attempt >= a.retries {
				return nil, fmt.Errorf("http request: %w", err)
			}
			if err := a.sleepBeforeRetry(ctx, retryDelay(nil, attempt, a.backoffOrDefault)); err != nil {
				return nil, err
			}
			if err := resetRequestBody(req); err != nil {
				return nil, err
			}
			continue
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}
		lastResp = resp
		if attempt >= a.retries {
			return lastResp, nil
		}

		if err := drainAndClose(resp); err != nil {
			return nil, err
		}
		if err := resetRequestBody(req); err != nil {
			return nil, err
		}
		if err := a.sleepBeforeRetry(ctx, retryDelay(resp, attempt, a.backoffOrDefault)); err != nil {
			return nil, err
		}
	}
}

func drainAndClose(resp *http.Response) error {
	if _, err := io.CopyN(io.Discard, resp.Body, maxRetryDrainBytes); err != nil && err != io.EOF {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return fmt.Errorf("drain retry response: %w; close response body: %w", err, closeErr)
		}
		return fmt.Errorf("drain retry response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("close retry response body: %w", err)
	}
	return nil
}

func resetRequestBody(req *http.Request) error {
	if req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("reset request body: %w", err)
	}
	req.Body = body
	return nil
}

func (a *Adapter) backoffOrDefault(attempt int) time.Duration {
	if a.backoff != nil {
		return a.backoff(attempt)
	}
	return defaultBackoff(attempt)
}

func (a *Adapter) sleepBeforeRetry(ctx context.Context, d time.Duration) error {
	a.backoffSleep(ctx, d)
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("retry canceled: %w", err)
	}
	return nil
}

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
