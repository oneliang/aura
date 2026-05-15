package llm

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/llm/internal"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to create a message with text content
func newTextMessageForRetry(role, text string) Message {
	msg := Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

// Helper function to extract text content from message
func getTextContentForRetry(msg Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// mockClient is a mock LLM client for testing.
type mockClient struct {
	completeCalls int
	streamCalls   int
	embedCalls    int

	completeErr error
	streamErr   error
	embedErr    error

	completeResp *Response
	embedResp    [][]float32
}

func (m *mockClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	m.completeCalls++
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return m.completeResp, nil
}

func (m *mockClient) Stream(ctx context.Context, req *Request) (<-chan Chunk, error) {
	m.streamCalls++
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: "hello", Done: true}
	return ch, nil
}

func (m *mockClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	m.embedCalls++
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return m.embedResp, nil
}

// TestRetryClient_Complete_SuccessOnFirstTry tests that a successful first attempt does not retry.
func TestRetryClient_Complete_SuccessOnFirstTry(t *testing.T) {
	mock := &mockClient{completeResp: &Response{Message: newTextMessageForRetry("assistant", "ok")}}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	resp, err := client.Complete(context.Background(), &Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getTextContentForRetry(resp.Message) != "ok" {
		t.Fatalf("expected 'ok', got %q", getTextContentForRetry(resp.Message))
	}
	if mock.completeCalls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.completeCalls)
	}
}

// TestRetryClient_Complete_SuccessAfterRetries tests retry on transient errors.
func TestRetryClient_Complete_SuccessAfterRetries(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		completeFn: func(ctx context.Context, req *Request) (*Response, error) {
			callCount++
			if callCount <= 2 {
				return nil, &internal.HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "503"}
			}
			return &Response{Message: newTextMessageForRetry("assistant", "ok")}, nil
		},
	}, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	resp, err := client.Complete(context.Background(), &Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getTextContentForRetry(resp.Message) != "ok" {
		t.Fatalf("expected 'ok', got %q", getTextContentForRetry(resp.Message))
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

// TestRetryClient_Complete_NonRetryableError tests that 4xx errors are not retried.
func TestRetryClient_Complete_NonRetryableError(t *testing.T) {
	err400 := &internal.HTTPError{StatusCode: http.StatusBadRequest, Message: "400"}
	mock := &mockClient{completeErr: err400}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	_, err := client.Complete(context.Background(), &Request{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.completeCalls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", mock.completeCalls)
	}
}

// TestRetryClient_Complete_AuthErrorNotRetried tests that 401 is not retried.
func TestRetryClient_Complete_AuthErrorNotRetried(t *testing.T) {
	err401 := &internal.HTTPError{StatusCode: http.StatusUnauthorized, Message: "401"}
	mock := &mockClient{completeErr: err401}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	_, err := client.Complete(context.Background(), &Request{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.completeCalls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", mock.completeCalls)
	}
}

// TestRetryClient_Complete_ContextCancelled tests that context cancellation is not retried.
func TestRetryClient_Complete_ContextCancelled(t *testing.T) {
	mock := &mockClient{completeErr: context.Canceled}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
	})

	_, err := client.Complete(context.Background(), &Request{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if mock.completeCalls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", mock.completeCalls)
	}
}

// TestRetryClient_Complete_NetworkError tests that network errors are retried.
func TestRetryClient_Complete_NetworkError(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		completeFn: func(ctx context.Context, req *Request) (*Response, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}, RetryConfig{
		MaxRetries:   2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	_, err := client.Complete(context.Background(), &Request{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls (1 + 2 retries), got %d", callCount)
	}
}

// TestRetryClient_Complete_ExhaustedRetries tests that retries are exhausted and error returned.
func TestRetryClient_Complete_ExhaustedRetries(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		completeFn: func(ctx context.Context, req *Request) (*Response, error) {
			callCount++
			return nil, &internal.HTTPError{StatusCode: http.StatusInternalServerError, Message: "500"}
		},
	}, RetryConfig{
		MaxRetries:   2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	_, err := client.Complete(context.Background(), &Request{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls (1 + 2 retries), got %d", callCount)
	}
}

// TestRetryClient_Stream_SuccessBeforeChunks tests that Stream returns immediately on success.
func TestRetryClient_Stream_SuccessBeforeChunks(t *testing.T) {
	mock := &mockClient{}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	ch, err := client.Stream(context.Background(), &Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
	if mock.streamCalls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.streamCalls)
	}
}

// TestRetryClient_Stream_ErrorBeforeChannel tests retry when Stream() itself fails.
func TestRetryClient_Stream_ErrorBeforeChannel(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		streamFn: func(ctx context.Context, req *Request) (<-chan Chunk, error) {
			callCount++
			if callCount <= 1 {
				return nil, &internal.HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "503"}
			}
			ch := make(chan Chunk, 1)
			ch <- Chunk{Content: "hello", Done: true}
			return ch, nil
		},
	}, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	ch, err := client.Stream(context.Background(), &Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
	// Read from channel to verify
	select {
	case chunk := <-ch:
		if chunk.Content != "hello" {
			t.Fatalf("expected 'hello', got %q", chunk.Content)
		}
	default:
		t.Fatal("expected chunk in channel")
	}
}

// TestRetryClient_Embed_SuccessAfterRetries tests retry on Embed.
func TestRetryClient_Embed_SuccessAfterRetries(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		embedFn: func(ctx context.Context, texts []string) ([][]float32, error) {
			callCount++
			if callCount <= 1 {
				return nil, &internal.HTTPError{StatusCode: http.StatusBadGateway, Message: "502"}
			}
			return [][]float32{{0.1, 0.2}}, nil
		},
	}, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	embeddings, err := client.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
}

// TestRetryClient_Embed_NonRetryableError tests that 400 on Embed is not retried.
func TestRetryClient_Embed_NonRetryableError(t *testing.T) {
	callCount := 0
	client := NewRetryClient(&retryTestWrapper{
		embedFn: func(ctx context.Context, texts []string) ([][]float32, error) {
			callCount++
			return nil, &internal.HTTPError{StatusCode: http.StatusBadRequest, Message: "400"}
		},
	}, RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	_, err := client.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", callCount)
	}
}

// TestRetryClient_Disabled_WhenMaxRetriesZero tests that MaxRetries=0 returns underlying client.
func TestRetryClient_Disabled_WhenMaxRetriesZero(t *testing.T) {
	mock := &mockClient{}
	client := NewRetryClient(mock, RetryConfig{
		MaxRetries:   0,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
	})

	// Should return the underlying client directly
	if _, ok := client.(*mockClient); !ok {
		t.Fatal("expected underlying client returned when MaxRetries=0")
	}
}

// TestRetryClient_MaxDelayCap tests that delay never exceeds MaxDelay with jitter.
func TestRetryClient_MaxDelayCap(t *testing.T) {
	client := &RetryClient{
		cfg: RetryConfig{
			MaxRetries:   5,
			InitialDelay: 1 * time.Second,
			MaxDelay:     5 * time.Second,
		},
	}

	for attempt := 0; attempt <= 10; attempt++ {
		err := &internal.HTTPError{StatusCode: http.StatusInternalServerError, Message: "500"}
		delay := client.computeDelay(attempt, err)
		if delay > client.cfg.MaxDelay+client.cfg.MaxDelay/4 {
			t.Fatalf("attempt %d: delay %v exceeds MaxDelay+jitter cap", attempt, delay)
		}
	}
}

// TestRetryClient_IsRetryable_NetworkError tests network error detection.
func TestRetryClient_IsRetryable_NetworkError(t *testing.T) {
	client := &RetryClient{cfg: DefaultRetryConfig()}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"no such host", errors.New("dial tcp: no such host"), true},
		{"EOF", errors.New("unexpected EOF"), true},
		{"net.Error timeout", &net.DNSError{IsTimeout: true}, true},
		{"4xx error", &internal.HTTPError{StatusCode: 400}, false},
		{"401 error", &internal.HTTPError{StatusCode: 401}, false},
		{"429 error", &internal.HTTPError{StatusCode: 429}, true},
		{"500 error", &internal.HTTPError{StatusCode: 500}, true},
		{"503 error", &internal.HTTPError{StatusCode: 503}, true},
		{"context cancelled", context.Canceled, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryable(%v) = %v; want %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestRetryClient_ParseRetryAfter tests Retry-After header parsing.
func TestRetryClient_ParseRetryAfter(t *testing.T) {
	client := &RetryClient{cfg: DefaultRetryConfig()}

	tests := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{"empty", http.Header{}, 0},
		{"seconds", http.Header{"Retry-After": []string{"5"}}, 5 * time.Second},
		{"zero seconds", http.Header{"Retry-After": []string{"0"}}, 0},
		{"invalid", http.Header{"Retry-After": []string{"abc"}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.parseRetryAfter(tt.header)
			// Allow 1s tolerance for HTTP-date tests.
			if tt.want > 0 {
				if got < tt.want-time.Second || got > tt.want+time.Second {
					t.Errorf("parseRetryAfter() = %v; want ~%v", got, tt.want)
				}
			} else if got != 0 {
				t.Errorf("parseRetryAfter() = %v; want 0", got)
			}
		})
	}
}

// TestRetryClient_ParseRetryAfter_HTTPDate tests Retry-After HTTP-date parsing with a future timestamp.
func TestRetryClient_ParseRetryAfter_HTTPDate(t *testing.T) {
	client := &RetryClient{cfg: RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
	}}

	future := time.Now().Add(5 * time.Second).UTC()
	hdr := http.Header{"Retry-After": []string{future.Format(http.TimeFormat)}}

	got := client.parseRetryAfter(hdr)
	if got < 4*time.Second || got > 6*time.Second {
		t.Errorf("parseRetryAfter() = %v; want ~5s", got)
	}
}

// TestRetryClient_Jitter tests that jitter adds randomness.
func TestRetryClient_Jitter(t *testing.T) {
	client := &RetryClient{cfg: DefaultRetryConfig()}

	delay := 100 * time.Millisecond
	results := make(map[time.Duration]bool)
	for i := 0; i < 20; i++ {
		j := client.jitter(delay)
		results[j] = true
		if j < -delay/4 || j >= delay/4 {
			t.Errorf("jitter(%v) = %v; expected [-%v, +%v)", delay, j, delay/4, delay/4)
		}
	}

	if len(results) < 2 {
		t.Errorf("jitter produced only %d unique values in 20 runs", len(results))
	}
}

// retryTestWrapper is a test helper that wraps a mock client with custom function overrides.
type retryTestWrapper struct {
	completeFn func(ctx context.Context, req *Request) (*Response, error)
	streamFn   func(ctx context.Context, req *Request) (<-chan Chunk, error)
	embedFn    func(ctx context.Context, texts []string) ([][]float32, error)
}

func (w *retryTestWrapper) Complete(ctx context.Context, req *Request) (*Response, error) {
	if w.completeFn != nil {
		return w.completeFn(ctx, req)
	}
	return nil, nil
}

func (w *retryTestWrapper) Stream(ctx context.Context, req *Request) (<-chan Chunk, error) {
	if w.streamFn != nil {
		return w.streamFn(ctx, req)
	}
	ch := make(chan Chunk, 1)
	ch <- Chunk{Content: "ok", Done: true}
	return ch, nil
}

func (w *retryTestWrapper) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if w.embedFn != nil {
		return w.embedFn(ctx, texts)
	}
	return nil, nil
}