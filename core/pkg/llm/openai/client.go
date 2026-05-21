// Package openai provides an OpenAI API compatible client.
package openai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/llm/internal"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/httpclient"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Client implements llm.Client for OpenAI compatible APIs.
type Client struct {
	baseURL    string
	model      string
	apiKey     string
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

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient injects an external HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// New creates a new OpenAI compatible client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    "https://api.openai.com/v1",
		model:      constants.DefaultOpenAIModel,
		httpClient: httpclient.DefaultLLMClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// openaiRequest represents an OpenAI chat request.
type openaiRequest struct {
	Model               string          `json:"model"`
	Messages            []openaiMessage `json:"messages"`
	Stream              bool            `json:"stream"`
	Tools               []openaiTool    `json:"tools,omitempty"`
	ToolChoice          any             `json:"tool_choice,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`

	// OpenAI Prompt Caching (request-level)
	CacheControl *openaiCacheControl `json:"cache_control,omitempty"`
}

// openaiCacheControl represents OpenAI cache control for request-level caching.
type openaiCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // Can be string or []openaiContentPart for multi-modal
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // For "tool" role messages
}

// openaiContentPart represents a content part in a multi-modal message.
type openaiContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openaiImageURL `json:"image_url,omitempty"`
}

// openaiImageURL represents an image URL for multi-modal messages.
type openaiImageURL struct {
	URL string `json:"url"`
}

// convertContent converts a Message to OpenAI format.
// Returns either a string (text-only) or []openaiContentPart (multi-modal).
// Extracts text from TextBlocks and tool calls from ToolUseBlocks.
func convertContent(msg llm.Message) interface{} {
	blocks := msg.GetContentBlocks()

	// If no blocks, check for multi-modal Parts
	if len(blocks) == 0 {
		// Check for multi-modal Parts
		if len(msg.Parts) > 0 {
			parts := make([]openaiContentPart, 0, len(msg.Parts))
			for _, part := range msg.Parts {
				switch part.Type {
				case "text":
					parts = append(parts, openaiContentPart{
						Type: "text",
						Text: part.Text,
					})
				case "image_url":
					if part.ImageURL != nil {
						parts = append(parts, openaiContentPart{
							Type:     "image_url",
							ImageURL: &openaiImageURL{URL: part.ImageURL.URL},
						})
					}
				}
			}
			return parts
		}
		return ""
	}

	// Extract text content and check for tool calls
	var textParts []string
	var hasToolCalls bool
	for _, block := range blocks {
		switch b := block.(type) {
		case memory.TextBlock:
			textParts = append(textParts, b.Text)
		case memory.ToolUseBlock:
			hasToolCalls = true
		}
	}

	// If tool calls present, return string content (tool_calls handled separately)
	if hasToolCalls {
		return strings.Join(textParts, "")
	}

	// If multi-modal Parts exist, use array format
	if len(msg.Parts) > 0 {
		parts := make([]openaiContentPart, 0, len(blocks)+len(msg.Parts))
		for _, block := range blocks {
			if tb, ok := block.(memory.TextBlock); ok {
				parts = append(parts, openaiContentPart{
					Type: "text",
					Text: tb.Text,
				})
			}
		}
		for _, part := range msg.Parts {
			switch part.Type {
			case "text":
				parts = append(parts, openaiContentPart{
					Type: "text",
					Text: part.Text,
				})
			case "image_url":
				if part.ImageURL != nil {
					parts = append(parts, openaiContentPart{
						Type:     "image_url",
						ImageURL: &openaiImageURL{URL: part.ImageURL.URL},
					})
				}
			}
		}
		return parts
	}

	// Simple text-only: return string
	return strings.Join(textParts, "")
}

// convertToolCalls extracts tool calls from ContentBlocks for OpenAI format.
func convertToolCalls(blocks []memory.ContentBlock) []openaiToolCall {
	var toolCalls []openaiToolCall
	for _, block := range blocks {
		if tub, ok := block.(memory.ToolUseBlock); ok {
			toolCalls = append(toolCalls, openaiToolCall{
				ID:   tub.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tub.Name,
					Arguments: string(tub.Input),
				},
			})
		}
	}
	return toolCalls
}

// convertToolResult converts a ToolResultBlock to OpenAI "tool" role message.
func convertToolResult(trb memory.ToolResultBlock) openaiMessage {
	// Concatenate text content from nested blocks
	var textContent strings.Builder
	for _, block := range trb.Content {
		if tb, ok := block.(memory.TextBlock); ok {
			textContent.WriteString(tb.Text)
		}
	}

	return openaiMessage{
		Role:       "tool",
		Content:    textContent.String(),
		ToolCallID: trb.ToolUseID,
	}
}

// convertMessages converts llm.Messages to OpenAI format.
// Handles ContentBlocks including ToolUseBlocks and ToolResultBlocks.
func convertMessages(messages []llm.Message) []openaiMessage {
	var result []openaiMessage

	for _, msg := range messages {
		blocks := msg.GetContentBlocks()

		// Check for ToolResultBlock - convert to "tool" role message
		for _, block := range blocks {
			if trb, ok := block.(memory.ToolResultBlock); ok {
				result = append(result, convertToolResult(trb))
				continue
			}
		}

		// Build normal message
		om := openaiMessage{
			Role:    msg.Role,
			Content: convertContent(msg),
		}

		// Add tool calls if present (for assistant messages)
		toolCalls := convertToolCalls(blocks)
		if len(toolCalls) > 0 {
			om.ToolCalls = toolCalls
		}

		// Skip if this was a tool result only message
		hasOnlyToolResult := true
		for _, block := range blocks {
			if _, ok := block.(memory.ToolResultBlock); !ok {
				hasOnlyToolResult = false
				break
			}
		}
		if hasOnlyToolResult && len(blocks) > 0 {
			continue
		}

		result = append(result, om)
	}

	return result
}

// openaiResponse represents an OpenAI chat response.
type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		Delta        openaiDelta   `json:"delta"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

// openaiDelta represents a streaming delta response.
type openaiDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          interface{}      `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []openaiToolCall `json:"tool_calls,omitempty"`
}

// openaiError represents an OpenAI error response.
type openaiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Complete implements llm.Client.
func (c *Client) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	openaiReq := openaiRequest{
		Model:   c.model,
		Stream:  false,
		Messages: convertMessages(req.Messages),
	}

	// Forward tool definitions
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]openaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openaiTool{
				Type: "function",
				Function: openaiFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
		// Skip tool_choice when thinking is enabled — some OpenAI-compatible
		// providers (e.g. DashScope) reject tool_choice in thinking mode.
		if req.ToolChoice != "" && (req.Thinking == nil || !req.Thinking.Enabled) {
			openaiReq.ToolChoice = req.ToolChoice
		}
	}

	// Apply thinking configuration
	buildThinkingParams(&openaiReq, req.Thinking)

	// Apply prompt caching configuration
	if req.PromptCache != nil && req.PromptCache.Enabled && req.PromptCache.CacheType != "" {
		openaiReq.CacheControl = &openaiCacheControl{Type: req.PromptCache.CacheType}
	}

	body, err := internal.MarshalJSON(openaiReq)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/chat/completions"
	headers := c.buildHeaders()

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", url, "application/json", body, headers)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkStatus(resp, url); err != nil {
		return nil, err
	}

	var openaiResp openaiResponse
	if err := internal.DecodeJSON(resp, &openaiResp); err != nil {
		return nil, err
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	// Extract text content from response
	content := ""
	if str, ok := openaiResp.Choices[0].Message.Content.(string); ok {
		content = str
	}

	response := &llm.Response{
		Message: llm.Message{
			Role:          openaiResp.Choices[0].Message.Role,
			ContentBlocks: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: content},
			},
		},
		Model: openaiResp.Model,
	}

	// Extract tool calls
	if len(openaiResp.Choices[0].Message.ToolCalls) > 0 {
		response.ToolCalls = make([]llm.ToolCall, len(openaiResp.Choices[0].Message.ToolCalls))
		for i, tc := range openaiResp.Choices[0].Message.ToolCalls {
			params := make(map[string]any)
			if tc.Function.Arguments != "" {
				_ = internal.UnmarshalJSON([]byte(tc.Function.Arguments), &params)
			}
			response.ToolCalls[i] = llm.ToolCall{
				ID:         tc.ID,
				Name:       tc.Function.Name,
				Parameters: params,
			}
		}
	}

	if openaiResp.Usage != nil {
		response.Usage = llm.Usage{
			PromptTokens:            openaiResp.Usage.PromptTokens,
			CompletionTokens:        openaiResp.Usage.CompletionTokens,
			TotalTokens:             openaiResp.Usage.TotalTokens,
			CacheCreationInputTokens: openaiResp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     openaiResp.Usage.CacheReadInputTokens,
		}
	}

	return response, nil
}

// Stream implements llm.Client.
func (c *Client) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	openaiReq := openaiRequest{
		Model:   c.model,
		Stream:  true,
		Messages: convertMessages(req.Messages),
	}

	// Forward tool definitions
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]openaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openaiTool{
				Type: "function",
				Function: openaiFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
		// Skip tool_choice when thinking is enabled — some OpenAI-compatible
		// providers (e.g. DashScope) reject tool_choice in thinking mode.
		if req.ToolChoice != "" && (req.Thinking == nil || !req.Thinking.Enabled) {
			openaiReq.ToolChoice = req.ToolChoice
		}
	}

	// Apply thinking configuration
	buildThinkingParams(&openaiReq, req.Thinking)

	// Apply prompt caching configuration
	if req.PromptCache != nil && req.PromptCache.Enabled && req.PromptCache.CacheType != "" {
		openaiReq.CacheControl = &openaiCacheControl{Type: req.PromptCache.CacheType}
	}

	body, err := internal.MarshalJSON(openaiReq)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/chat/completions"
	headers := c.buildHeaders()

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", url, "application/json", body, headers)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}

	if err := c.checkStatus(resp, url); err != nil {
		resp.Body.Close()
		return nil, err
	}

	ch := make(chan llm.Chunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		err := internal.StreamSSE(resp, func(data []byte) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var openaiResp openaiResponse
			if err := internal.UnmarshalJSON(data, &openaiResp); err != nil {
				return nil // Skip malformed chunks
			}

			if len(openaiResp.Choices) > 0 {
				choice := openaiResp.Choices[0]
				// Extract text content from delta (streaming responses return text)
				content := ""
				if str, ok := choice.Delta.Content.(string); ok {
					content = str
				}

				chunk := llm.Chunk{
					Content:          content,
					ReasoningContent: choice.Delta.ReasoningContent,
					FinishReason:     choice.FinishReason,
				}

				// Extract tool call delta if present
				if len(choice.Delta.ToolCalls) > 0 {
					tc := choice.Delta.ToolCalls[0]
					params := make(map[string]any)
					if tc.Function.Arguments != "" {
						_ = internal.UnmarshalJSON([]byte(tc.Function.Arguments), &params)
					}
					chunk.ToolCallDelta = &llm.ToolCall{
						ID:         tc.ID,
						Name:       tc.Function.Name,
						Parameters: params,
					}
				}

				select {
				case ch <- chunk:
				case <-ctx.Done():
					return ctx.Err()
				}

				if choice.FinishReason != "" {
					select {
					case ch <- llm.Chunk{Done: true}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		})

		// Always send Done signal at the end (if not already sent)
		select {
		case ch <- llm.Chunk{Done: true}:
		case <-ctx.Done():
		}

		if err != nil && err != ctx.Err() {
			// Error already signaled via Done, just log
		}
	}()

	return ch, nil
}

// openaiEmbedRequest represents an embedding request.
type openaiEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openaiEmbedResponse represents an embedding response.
type openaiEmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed implements llm.Client.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := openaiEmbedRequest{
		Model: c.model,
		Input: texts,
	}

	body, err := internal.MarshalJSON(req)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/embeddings"
	headers := c.buildHeaders()

	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", url, "application/json", body, headers)
	if err != nil {
		return nil, err
	}

	resp, err := internal.SendRequest(c.httpClient, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkStatus(resp, url); err != nil {
		return nil, err
	}

	var embedResp openaiEmbedResponse
	if err := internal.DecodeJSON(resp, &embedResp); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(texts))
	for _, data := range embedResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

// buildHeaders builds HTTP headers for API requests.
func (c *Client) buildHeaders() map[string]string {
	if c.apiKey == "" {
		return nil
	}
	return map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}
}

// buildThinkingParams applies thinking config to the OpenAI request.
// Only sets reasoning_effort when non-empty (non-reasoning models may error).
// Only sets max_completion_tokens when > 0.
func buildThinkingParams(req *openaiRequest, thinking *llm.ThinkingConfig) {
	if thinking == nil || !thinking.Enabled {
		return
	}
	if thinking.ReasoningEffort != "" {
		req.ReasoningEffort = thinking.ReasoningEffort
	}
	if thinking.BudgetTokens > 0 {
		req.MaxCompletionTokens = thinking.BudgetTokens
	}
}

// checkStatus checks response status and returns error if not OK.
func (c *Client) checkStatus(resp *http.Response, url string) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, err := internal.ReadResponseBody(resp)
	if err != nil {
		return &internal.HTTPError{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Message:    err.Error(),
		}
	}

	var apiErr openaiError
	if err := internal.UnmarshalJSON(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return &internal.HTTPError{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Message:    fmt.Sprintf("API error: %s (url=%s, status=%d)", apiErr.Error.Message, url, resp.StatusCode),
		}
	}

	return &internal.HTTPError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Message:    fmt.Sprintf("unexpected status %d: %s (url=%s)", resp.StatusCode, string(body), url),
	}
}
