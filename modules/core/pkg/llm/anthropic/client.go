package anthropic

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/llm/internal"
	"github.com/oneliang/aura/shared/pkg/httpclient"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// Client implements llm.Client for Anthropic APIs.
type Client struct {
	baseURL    string
	model      string
	apiKey     string
	apiVersion string
	httpClient *http.Client
}

// Option is a client configuration option.
type Option func(*Client)

// WithBaseURL sets the base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithModel sets the model.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = timeout }
}

// WithHTTPClient injects an external HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

// New creates a new Anthropic client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    "https://api.anthropic.com/v1",
		model:      "claude-sonnet-4-20250514",
		apiVersion: "2024-10-22",
		httpClient: httpclient.DefaultLLMClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// buildSystemValue builds the system value for Anthropic API requests.
// Returns string format by default, or []systemBlock for caching when enabled.
func buildSystemValue(system string, req *llm.Request) any {
	if req.PromptCache == nil || !req.PromptCache.Enabled || len(req.PromptCache.SystemBlocks) == 0 {
		return system // default string format
	}

	// Use Anthropic-style system blocks with cache_control
	blocks := make([]systemBlock, len(req.PromptCache.SystemBlocks))
	for i, sb := range req.PromptCache.SystemBlocks {
		blocks[i] = systemBlock{
			Type: sb.Type,
			Text: sb.Text,
		}
		if sb.CacheControl != nil {
			blocks[i].CacheControl = &cacheControl{Type: sb.CacheControl.Type}
		}
	}
	return blocks
}

// Complete implements llm.Client.
func (c *Client) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	system, msgs := convertMessages(req.Messages)

	anthReq := chatRequest{
		Model:     c.model,
		MaxTokens: 4096,
		Messages:  msgs,
		System:    buildSystemValue(system, req),
		Stream:    false,
		Thinking:  buildThinkingBlock(req.Thinking),
	}

	// Anthropic requires temperature=1.0 when thinking is enabled
	if req.Thinking != nil && req.Thinking.Enabled {
		anthReq.Temperature = 1.0
		// Ensure max_tokens > thinking budget (Anthropic API requirement)
		if req.Thinking.BudgetTokens > 0 && anthReq.MaxTokens <= req.Thinking.BudgetTokens {
			anthReq.MaxTokens = req.Thinking.BudgetTokens * 2
		}
	} else if req.Temperature > 0 {
		anthReq.Temperature = req.Temperature
	}

	if len(req.Tools) > 0 {
		anthReq.Tools = make([]toolDef, len(req.Tools))
		for i, tool := range req.Tools {
			anthReq.Tools[i] = toolDef{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			}
		}
		if req.ToolChoice != "" {
			anthReq.ToolChoice = toolChoiceAny{Type: "tool", Name: req.ToolChoice}
		}
	}

	body, err := internal.MarshalJSON(anthReq)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/messages"
	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", url, "application/json", body, c.buildHeaders())
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

	var anthResp chatResponse
	if err := internal.DecodeJSON(resp, &anthResp); err != nil {
		return nil, err
	}

	response := &llm.Response{
		Message: llm.Message{
			Role:          anthResp.Role,
			ContentBlocks: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: extractTextContent(anthResp.Content)},
			},
		},
		Model: anthResp.Model,
	}

	if anthResp.Usage != nil {
		response.Usage = llm.Usage{
			PromptTokens:            anthResp.Usage.InputTokens,
			CompletionTokens:        anthResp.Usage.OutputTokens,
			TotalTokens:             anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
			CacheCreationInputTokens: anthResp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     anthResp.Usage.CacheReadInputTokens,
		}
	}

	return response, nil
}

// Stream implements llm.Client.
func (c *Client) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	system, msgs := convertMessages(req.Messages)

	anthReq := chatRequest{
		Model:     c.model,
		MaxTokens: 4096,
		Messages:  msgs,
		System:    buildSystemValue(system, req),
		Stream:    true,
		Thinking:  buildThinkingBlock(req.Thinking),
	}

	if req.Thinking != nil && req.Thinking.Enabled {
		anthReq.Temperature = 1.0
		// Ensure max_tokens > thinking budget (Anthropic API requirement)
		if req.Thinking.BudgetTokens > 0 && anthReq.MaxTokens <= req.Thinking.BudgetTokens {
			anthReq.MaxTokens = req.Thinking.BudgetTokens * 2
		}
	} else if req.Temperature > 0 {
		anthReq.Temperature = req.Temperature
	}

	if len(req.Tools) > 0 {
		anthReq.Tools = make([]toolDef, len(req.Tools))
		for i, tool := range req.Tools {
			anthReq.Tools[i] = toolDef{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			}
		}
		if req.ToolChoice != "" {
			anthReq.ToolChoice = toolChoiceAny{Type: "tool", Name: req.ToolChoice}
		}
	}

	body, err := internal.MarshalJSON(anthReq)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/messages"
	httpReq, err := internal.BuildHTTPRequest(ctx, "POST", url, "application/json", body, c.buildHeaders())
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

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- llm.Chunk{Done: true}
				return
			}

			var evt streamEvent
			if err := internal.UnmarshalJSON([]byte(data), &evt); err != nil {
				continue
			}

			switch evt.Type {
			case "content_block_delta":
				if evt.Delta != nil && evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
					select {
					case ch <- llm.Chunk{Content: evt.Delta.Text}:
					case <-ctx.Done():
						return
					}
				}
			case "message_delta":
				if evt.Delta != nil && evt.Delta.Type == "text" && evt.Delta.Text != "" {
					select {
					case ch <- llm.Chunk{Content: evt.Delta.Text}:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		select {
		case ch <- llm.Chunk{Done: true}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// Embed implements llm.Client.
// Anthropic does not have a native embedding API, so this returns an error.
func (c *Client) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, fmt.Errorf("anthropic does not support embeddings")
}

// buildHeaders builds HTTP headers for Anthropic API requests.
func (c *Client) buildHeaders() map[string]string {
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": c.apiVersion,
		"content-type":      "application/json",
	}
	return headers
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

	return &internal.HTTPError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Message:    fmt.Sprintf("unexpected status %d: %s (url=%s)", resp.StatusCode, string(body), url),
	}
}
