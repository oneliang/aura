// Package llm provides LLM client abstraction.
package llm

import (
	"context"

	"github.com/oneliang/aura/shared/pkg/memory"
)

// Message represents a chat message.
// This is a type alias to shared/memory.Message for backward compatibility.
type Message = memory.Message

// ThinkingConfig configures native LLM thinking/reasoning mode.
// Each provider maps this to their own API:
//   - OpenAI: reasoning_effort + max_completion_tokens
//   - Anthropic: thinking.enabled + budget_tokens (forces temperature=1.0)
//   - Ollama: options.num_think
type ThinkingConfig struct {
	Enabled         bool   `json:"enabled"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // low/medium/high (OpenAI)
	BudgetTokens    int    `json:"budget_tokens,omitempty"`    // max thinking tokens (Anthropic)
}

// CacheControl marks content for prompt caching.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// SystemBlock represents a system prompt block with optional cache control.
// Used for Anthropic-style caching (system as array of blocks).
type SystemBlock struct {
	Type         string        `json:"type"`                    // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"` // Anthropic caching marker
}

// PromptCacheConfig configures prompt caching behavior.
type PromptCacheConfig struct {
	// Enabled enables prompt caching for this request.
	Enabled bool `json:"enabled"`

	// SystemBlocks provides Anthropic-style system blocks with cache markers.
	// When non-empty, Anthropic client uses this instead of string system.
	SystemBlocks []SystemBlock `json:"system_blocks,omitempty"`

	// CacheType for OpenAI-style request-level caching ("ephemeral").
	// OpenAI client adds this to the request body when enabled.
	CacheType string `json:"cache_type,omitempty"`
}

// Request represents an LLM request.
type Request struct {
	Messages       []Message         `json:"messages"`
	Model          string            `json:"model"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Temperature    float64           `json:"temperature,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	Tools          []ToolSchema      `json:"tools,omitempty"`
	ToolChoice     string            `json:"tool_choice,omitempty"` // "auto"|"required"|"none" or specific tool
	ResponseFormat *string           `json:"response_format,omitempty"`
	Thinking       *ThinkingConfig   `json:"thinking,omitempty"`

	// PromptCache enables prompt caching with provider-specific format.
	PromptCache *PromptCacheConfig `json:"prompt_cache,omitempty"`
}

// ToolSchema describes a tool available to the LLM.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON schema for input parameters
}

// ToolCall represents a tool call from the LLM response.
type ToolCall struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// Response represents an LLM response.
type Response struct {
	Message      Message    `json:"message"`
	Model        string     `json:"model"`
	Usage        Usage      `json:"usage"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// Cache metrics (Anthropic/OpenAI format)
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Chunk represents a streaming response chunk.
type Chunk struct {
	Content          string    `json:"content"`
	ReasoningContent string    `json:"reasoning_content,omitempty"`
	FinishReason     string    `json:"finish_reason"`
	Done             bool      `json:"done"`
	ToolCallDelta    *ToolCall `json:"tool_call_delta,omitempty"`
}

// Client defines the LLM client interface.
type Client interface {
	// Complete sends a request and returns the response.
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Stream sends a request and returns a channel of chunks.
	Stream(ctx context.Context, req *Request) (<-chan Chunk, error)

	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
