// Package ollama provides an Ollama API client.
package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/llm/internal"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/httpclient"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// Client implements llm.Client for Ollama.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// Option is a client configuration option.
type Option func(*Client)

// WithBaseURL sets the base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithModel sets the model.
func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// WithTimeout sets the HTTP response header timeout (TTFB).
// Note: this sets Transport.ResponseHeaderTimeout, NOT http.Client.Timeout,
// because Client.Timeout covers the entire request lifecycle including reading
// the response body, which kills active streaming connections.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.ResponseHeaderTimeout = timeout
		}
	}
}

// WithHTTPClient injects an external HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// New creates a new Ollama client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    constants.DefaultLLMBaseURL,
		model:      constants.DefaultOllamaModel,
		httpClient: httpclient.DefaultLLMClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ollamaRequest represents an Ollama chat request.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	ID        int            `json:"id,omitempty"`
	Function  string         `json:"function"`
	Arguments map[string]any `json:"arguments"`
}

// ollamaResponse represents an Ollama chat response.
type ollamaResponse struct {
	Model     string                `json:"model"`
	CreatedAt string                `json:"created_at"`
	Message   ollamaResponseMessage `json:"message"`
	Done      bool                  `json:"done"`
}

// ollamaResponseMessage is like ollamaMessage but with tool calls.
type ollamaResponseMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

// buildThinkingOptions applies thinking config to Ollama request options.
// Ollama uses "num_think" for max thinking tokens (model-dependent support).
func buildThinkingOptions(opts map[string]any, thinking *llm.ThinkingConfig) {
	if thinking == nil || !thinking.Enabled {
		return
	}
	if thinking.BudgetTokens > 0 {
		opts["num_think"] = thinking.BudgetTokens
	}
}

// Complete implements llm.Client.
func (c *Client) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	ollamaReq := ollamaRequest{
		Model:   c.model,
		Stream:  false,
		Options: make(map[string]any),
	}

	// Handle SystemBlocks: Ollama doesn't support system blocks array,
	// so we expand them into a single system message at the beginning.
	if req.PromptCache != nil && len(req.PromptCache.SystemBlocks) > 0 {
		var systemText string
		for _, block := range req.PromptCache.SystemBlocks {
			systemText += block.Text + "\n\n"
		}
		if systemText != "" {
			ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	for _, msg := range req.Messages {
		// Extract text content from ContentBlocks
		var textContent string
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
			Role:    msg.Role,
			Content: textContent,
		})
	}

	// Forward tool definitions (Ollama supports OpenAI-compatible tools API)
	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]ollamaTool, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = ollamaTool{
				Type: "function",
				Function: ollamaFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	// Apply thinking configuration
	buildThinkingOptions(ollamaReq.Options, req.Thinking)

	body, err := internal.MarshalJSON(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", c.baseURL+"/api/chat", "application/json", body, nil)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := internal.CheckStatusCode(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var ollamaResp ollamaResponse
	if err := internal.DecodeJSON(resp, &ollamaResp); err != nil {
		return nil, err
	}

	response := &llm.Response{
		Message: llm.Message{
			Role:          ollamaResp.Message.Role,
			ContentBlocks: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: ollamaResp.Message.Content},
			},
		},
		Model: ollamaResp.Model,
	}

	// Extract tool calls
	if len(ollamaResp.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]llm.ToolCall, len(ollamaResp.Message.ToolCalls))
		for i, tc := range ollamaResp.Message.ToolCalls {
			response.ToolCalls[i] = llm.ToolCall{
				ID:         string(rune(tc.ID)),
				Name:       tc.Function,
				Parameters: tc.Arguments,
			}
		}
	}

	return response, nil
}

// Stream implements llm.Client.
func (c *Client) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	ollamaReq := ollamaRequest{
		Model:   c.model,
		Stream:  true,
		Options: make(map[string]any),
	}

	// Handle SystemBlocks: Ollama doesn't support system blocks array,
	// so we expand them into a single system message at the beginning.
	if req.PromptCache != nil && len(req.PromptCache.SystemBlocks) > 0 {
		var systemText string
		for _, block := range req.PromptCache.SystemBlocks {
			systemText += block.Text + "\n\n"
		}
		if systemText != "" {
			ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	for _, msg := range req.Messages {
		// Extract text content from ContentBlocks
		var textContent string
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
			Role:    msg.Role,
			Content: textContent,
		})
	}

	// Forward tool definitions
	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]ollamaTool, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = ollamaTool{
				Type: "function",
				Function: ollamaFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	// Apply thinking configuration
	buildThinkingOptions(ollamaReq.Options, req.Thinking)

	body, err := internal.MarshalJSON(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", c.baseURL+"/api/chat", "application/json", body, nil)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}

	if err := internal.CheckStatusCode(resp, http.StatusOK); err != nil {
		resp.Body.Close()
		return nil, err
	}

	ch := make(chan llm.Chunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Separate decode goroutine: json.Decoder.Decode() blocks on resp.Body.Read(),
		// so we run it in its own goroutine and communicate via channel.
		type decodeResult struct {
			resp *ollamaResponse
			err  error
		}
		decodeCh := make(chan decodeResult, 1)
		go func() {
			decoder := json.NewDecoder(resp.Body)
			for {
				var ollamaResp ollamaResponse
				err := decoder.Decode(&ollamaResp)
				decodeCh <- decodeResult{resp: &ollamaResp, err: err}
				if err != nil || ollamaResp.Done {
					return
				}
			}
		}()

		idleTimer := time.NewTimer(constants.DefaultStreamIdleTimeout)
		defer idleTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-idleTimer.C:
				ch <- llm.Chunk{Done: true}
				return
			case result := <-decodeCh:
				// Reset idle timer on each decoded chunk
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(constants.DefaultStreamIdleTimeout)

				if result.err != nil {
					if result.err == io.EOF {
						return
					}
					ch <- llm.Chunk{Done: true}
					return
				}

				chunk := llm.Chunk{
					Content: result.resp.Message.Content,
					Done:    result.resp.Done,
				}

				// Extract tool call delta if present
				if len(result.resp.Message.ToolCalls) > 0 {
					tc := result.resp.Message.ToolCalls[0]
					chunk.ToolCallDelta = &llm.ToolCall{
						ID:         string(rune(tc.ID)),
						Name:       tc.Function,
						Parameters: tc.Arguments,
					}
				}

				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}

				if result.resp.Done {
					return
				}
			}
		}
	}()

	return ch, nil
}

// ollamaEmbedRequest represents an embedding request.
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse represents an embedding response.
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed implements llm.Client.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		req := ollamaEmbedRequest{
			Model:  c.model,
			Prompt: text,
		}

		body, err := internal.MarshalJSON(req)
		if err != nil {
			return nil, err
		}

		httpReq, err := internal.BuildHTTPRequest(ctx, "POST", c.baseURL+"/api/embeddings", "application/json", body, nil)
		if err != nil {
			return nil, err
		}

		resp, err := internal.SendRequest(c.httpClient, httpReq)
		if err != nil {
			return nil, err
		}

		if err := internal.CheckStatusCode(resp, http.StatusOK); err != nil {
			resp.Body.Close()
			return nil, err
		}

		var embedResp ollamaEmbedResponse
		if err := internal.DecodeJSON(resp, &embedResp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		embeddings[i] = embedResp.Embedding
	}

	return embeddings, nil
}

// buildChatRequest builds a chat request body.
func (c *Client) buildChatRequest(stream bool, messages []llm.Message) ([]byte, error) {
	ollamaReq := ollamaRequest{
		Model:  c.model,
		Stream: stream,
	}

	for _, msg := range messages {
		// Extract text content from ContentBlocks
		var textContent string
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
			Role:    msg.Role,
			Content: textContent,
		})
	}

	return internal.MarshalJSON(ollamaReq)
}

// sendChatRequest sends a chat request and returns the response.
func (c *Client) sendChatRequest(ctx context.Context, body []byte, stream bool) (*http.Response, error) {
	endpoint := "/api/chat"
	if !stream {
		endpoint = "/api/chat"
	}

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", c.baseURL+endpoint, "application/json", body, nil)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}

	if err := internal.CheckStatusCode(resp, http.StatusOK); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}
