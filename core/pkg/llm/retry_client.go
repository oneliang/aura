// Package llm provides LLM client abstraction.
package llm

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/llm/internal"
	"github.com/oneliang/aura/shared/pkg/config"
)

// RetryConfig is an alias to config.RetryConfig for convenience.
type RetryConfig = config.RetryConfig

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
	}
}

// RetryClient wraps an LLM client to retry on transient errors.
type RetryClient struct {
	client Client
	cfg    RetryConfig
}

// NewRetryClient creates a new RetryClient.
func NewRetryClient(client Client, cfg RetryConfig) Client {
	if cfg.MaxRetries <= 0 {
		return client
	}
	return &RetryClient{
		client: client,
		cfg:    cfg,
	}
}

// Complete implements the Client interface.
func (r *RetryClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error
	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		resp, err := r.client.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if !r.isRetryable(err) {
			return nil, err
		}

		delay := r.computeDelay(attempt, err)
		if !r.waitWithCancel(ctx, delay) {
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// Stream implements the Client interface.
// Retry only if the error occurs before any chunk is received
// (i.e., the error is returned from the Stream call itself).
func (r *RetryClient) Stream(ctx context.Context, req *Request) (<-chan Chunk, error) {
	var lastErr error
	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		ch, err := r.client.Stream(ctx, req)
		if err == nil {
			return ch, nil
		}

		lastErr = err

		if !r.isRetryable(err) {
			return nil, err
		}

		delay := r.computeDelay(attempt, err)
		if !r.waitWithCancel(ctx, delay) {
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// Embed implements the Client interface.
func (r *RetryClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var lastErr error
	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		embeddings, err := r.client.Embed(ctx, texts)
		if err == nil {
			return embeddings, nil
		}

		lastErr = err

		if !r.isRetryable(err) {
			return nil, err
		}

		delay := r.computeDelay(attempt, err)
		if !r.waitWithCancel(ctx, delay) {
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// isRetryable determines if an error is transient and worth retrying.
func (r *RetryClient) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var httpErr *internal.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusTooManyRequests:
			return true
		case http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "EOF") {
		return true
	}

	return false
}

// computeDelay calculates the backoff delay for the given attempt.
// For 429 errors, respects the Retry-After header if present.
func (r *RetryClient) computeDelay(attempt int, err error) time.Duration {
	// Check for Retry-After header on 429 responses.
	var httpErr *internal.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusTooManyRequests {
		if retryAfter := r.parseRetryAfter(httpErr.Header); retryAfter > 0 {
			return retryAfter
		}
	}

	delay := r.cfg.InitialDelay * time.Duration(1<<uint(attempt))
	if delay > r.cfg.MaxDelay {
		delay = r.cfg.MaxDelay
	}

	jitter := r.jitter(delay)
	return delay + jitter // final range: [delay*0.75, delay*1.25)
}

// parseRetryAfter extracts the Retry-After header value as a time.Duration.
// Supports both seconds (integer) and HTTP-date (RFC 1123) formats.
func (r *RetryClient) parseRetryAfter(header http.Header) time.Duration {
	val := header.Get("Retry-After")
	if val == "" {
		return 0
	}

	// Try as seconds (integer).
	if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
		d := time.Duration(seconds) * time.Second
		if d > r.cfg.MaxDelay {
			return r.cfg.MaxDelay
		}
		return d
	}

	// Try as HTTP-date (HTTP time format, e.g. "Sun, 06 Nov 1994 08:49:37 GMT").
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 && d <= r.cfg.MaxDelay {
			return d
		}
	}

	return 0
}

// jitter adds randomized delay to prevent thundering herd.
// Returns a value in the range [-delay/4, +delay/4).
func (r *RetryClient) jitter(delay time.Duration) time.Duration {
	half := delay / 2
	if half <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(half)) - int64(half)/2)
}

// waitWithCancel sleeps for the given duration or until context is done.
// Returns false if context was cancelled before the delay elapsed.
func (r *RetryClient) waitWithCancel(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
