// Package internal provides internal utilities for LLM clients.
package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPError wraps an HTTP error with status code and response headers.
type HTTPError struct {
	StatusCode int
	Header     http.Header
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// HTTPClient is an interface for HTTP requests, allowing for testing with mocks.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BuildHTTPRequest builds an HTTP request with common headers.
func BuildHTTPRequest(ctx context.Context, method, url, contentType string, body []byte, headers map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

// SendRequest sends an HTTP request and returns the response.
func SendRequest(client HTTPClient, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// ReadResponseBody reads the response body.
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

// CheckStatusCode checks if the response status code is OK.
func CheckStatusCode(resp *http.Response, expected int) error {
	if resp.StatusCode == expected {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Message:    fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)),
	}
}

// CheckStatusWithAPIError checks status code and tries to parse API error response.
func CheckStatusWithAPIError(resp *http.Response, url string, parseError func([]byte) (string, error)) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Message:    fmt.Sprintf("unexpected status %d: failed to read body (url=%s)", resp.StatusCode, url),
		}
	}

	errMsg, parseErr := parseError(body)
	if parseErr == nil && errMsg != "" {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Message:    fmt.Sprintf("API error: %s (url=%s, status=%d)", errMsg, url, resp.StatusCode),
		}
	}

	return &HTTPError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Message:    fmt.Sprintf("unexpected status %d: %s (url=%s)", resp.StatusCode, string(body), url),
	}
}

// StreamSSE reads SSE stream from response and calls handler for each chunk.
// Deprecated: Use StreamSSEWithContext for context-aware streaming.
func StreamSSE(resp *http.Response, handler func(data []byte) error) error {
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip non-data lines
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)

		// Check for stream end marker
		if data == "[DONE]" {
			return nil
		}

		if err := handler([]byte(data)); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// StreamSSEWithContext reads SSE stream from response and calls handler for each chunk.
// Context-aware: returns ctx.Err() immediately when context is cancelled.
// This prevents blocking on scanner.Scan() when waiting for network data.
// idleTimeout: max duration between consecutive chunks. If no chunk arrives within
// this window, the stream is considered stalled and an error is returned.
// Pass 0 to disable idle timeout (not recommended for production).
func StreamSSEWithContext(ctx context.Context, resp *http.Response, idleTimeout time.Duration, handler func(data []byte) error) error {
	defer resp.Body.Close()

	// Create a channel to receive scanner lines
	lineCh := make(chan string, 100)
	errCh := make(chan error, 1)

	// Start scanner goroutine
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case lineCh <- scanner.Text():
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
		if scanner.Err() != nil {
			select {
			case errCh <- scanner.Err():
			default:
			}
		}
	}()

	// Set up idle timeout timer
	var idleTimer *time.Timer
	var idleCh <-chan time.Time
	if idleTimeout > 0 {
		idleTimer = time.NewTimer(idleTimeout)
		defer idleTimer.Stop()
		idleCh = idleTimer.C
	}

	// Process lines with context and idle timeout checks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idleCh:
			return fmt.Errorf("stream idle timeout: no data received for %v", idleTimeout)
		case err, ok := <-errCh:
			if ok && err != nil {
				return err
			}
			return nil // scanner finished normally
		case line, ok := <-lineCh:
			if !ok {
				return nil // channel closed, scanner done
			}
			// Reset idle timer on each chunk received
			if idleTimer != nil {
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleTimeout)
			}
			// Skip empty lines
			if line == "" {
				continue
			}
			// Skip non-data lines
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			data = strings.TrimSpace(data)
			// Check for stream end marker
			if data == "[DONE]" {
				return nil
			}
			if err := handler([]byte(data)); err != nil {
				return err
			}
		}
	}
}

// MarshalJSON marshals a value to JSON.
func MarshalJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	return data, nil
}

// UnmarshalJSON unmarshals JSON data.
func UnmarshalJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}
	return nil
}

// DecodeJSON decodes JSON from response body.
func DecodeJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
