// Package llm provides LLM client abstraction with optional request/response logging.
package llm

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/logger"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// LoggingClient wraps an LLM client to log all interactions.
type LoggingClient struct {
	client    Client
	provider  string
	model     string
	sessionID string
}

// clientInfo caches client type information.
type clientInfo struct {
	provider string
	model    string
}

var (
	clientInfoCache sync.Map // map[uintptr]clientInfo
)

// NewLoggingClient creates a new LoggingClient that wraps the given client.
// The provider and model parameters identify the LLM configuration.
// The sessionID is used for tracing requests within the same session.
func NewLoggingClient(client Client, provider, model, sessionID string) *LoggingClient {
	return &LoggingClient{
		client:    client,
		provider:  provider,
		model:     model,
		sessionID: sessionID,
	}
}

// Complete implements the Client interface.
func (lc *LoggingClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	resp, err := lc.client.Complete(ctx, req)

	durationMs := time.Since(startTime).Milliseconds()
	logLLMComplete(requestID, lc.sessionID, lc.provider, req, resp, durationMs, err)

	return resp, err
}

// Stream implements the Client interface.
func (lc *LoggingClient) Stream(ctx context.Context, req *Request) (<-chan Chunk, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	ch, err := lc.client.Stream(ctx, req)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		logLLMStream(requestID, lc.sessionID, lc.provider, req, "", durationMs, err)
		return ch, err
	}

	// Wrap the channel to capture the full response
	wrappedCh := make(chan Chunk, constants.LLMStreamBufferSize)
	go func() {
		defer close(wrappedCh)

		var fullContent string
		var seenError error

		for chunk := range ch {
			wrappedCh <- chunk
			if chunk.Content != "" {
				fullContent += chunk.Content
			}
			if chunk.Done {
				break
			}
		}

		// Log the complete stream
		durationMs := time.Since(startTime).Milliseconds()
		logLLMStream(requestID, lc.sessionID, lc.provider, req, fullContent, durationMs, seenError)
	}()

	return wrappedCh, nil
}

// Embed implements the Client interface.
func (lc *LoggingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	embeddings, err := lc.client.Embed(ctx, texts)

	durationMs := time.Since(startTime).Milliseconds()
	logLLMEmbed(requestID, lc.sessionID, lc.provider, lc.model, texts, embeddings, durationMs, err)

	return embeddings, err
}

// logLLMComplete logs a Complete method call via the shared logger's LLM audit logger.
func logLLMComplete(requestID, sessionID, provider string, req *Request, resp *Response, durationMs int64, err error) {
	auditLogger := logger.GetLLMAuditLogger()
	if auditLogger == nil {
		return
	}

	entry := logger.LLMLogEntry{
		Timestamp:  time.Now().UnixNano(),
		RequestID:  requestID,
		SessionID:  sessionID,
		Method:     "Complete",
		Provider:   provider,
		Model:      req.Model,
		Messages:   req.Messages,
		DurationMs: durationMs,
	}

	if err != nil {
		entry.Error = err.Error()
	} else if resp != nil {
		// Extract text content from ContentBlocks
		var textContent string
		for _, block := range resp.Message.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		respData := map[string]any{
			"content":       textContent,
			"model":         resp.Model,
			"usage":         resp.Usage,
			"finish_reason": resp.FinishReason,
		}
		if len(resp.ToolCalls) > 0 {
			respData["tool_calls"] = resp.ToolCalls
		}
		entry.Response = respData
	}

	auditLogger.Log(entry)
}

// logLLMStream logs a Stream method call via the shared logger's LLM audit logger.
func logLLMStream(requestID, sessionID, provider string, req *Request, fullContent string, durationMs int64, err error) {
	auditLogger := logger.GetLLMAuditLogger()
	if auditLogger == nil {
		return
	}

	entry := logger.LLMLogEntry{
		Timestamp:  time.Now().UnixNano(),
		RequestID:  requestID,
		SessionID:  sessionID,
		Method:     "Stream",
		Provider:   provider,
		Model:      req.Model,
		Messages:   req.Messages,
		DurationMs: durationMs,
	}

	if err != nil {
		entry.Error = err.Error()
	} else {
		entry.Response = map[string]any{
			"content": fullContent,
			"model":   req.Model,
		}
	}

	auditLogger.Log(entry)
}

// logLLMEmbed logs an Embed method call via the shared logger's LLM audit logger.
func logLLMEmbed(requestID, sessionID, provider string, model string, inputTexts []string, embeddings [][]float32, durationMs int64, err error) {
	auditLogger := logger.GetLLMAuditLogger()
	if auditLogger == nil {
		return
	}

	entry := logger.LLMLogEntry{
		Timestamp:  time.Now().UnixNano(),
		RequestID:  requestID,
		SessionID:  sessionID,
		Method:     "Embed",
		Provider:   provider,
		Model:      model,
		InputTexts: inputTexts,
		DurationMs: durationMs,
	}

	if err != nil {
		entry.Error = err.Error()
	} else {
		entry.Response = map[string]any{
			"embeddings": embeddings,
		}
	}

	auditLogger.Log(entry)
}

// CloseLLMLogger closes the global LLM audit logger.
func CloseLLMLogger() error {
	return logger.CloseLLMAuditLogger()
}
