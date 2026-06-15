// Package anthropic provides an Anthropic API client with extended thinking support.
package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// chatRequest represents an Anthropic chat request.
type chatRequest struct {
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens"`
	Messages    []chatMessage  `json:"messages"`
	System      any            `json:"system,omitempty"` // string or []systemBlock for caching
	Stream      bool           `json:"stream"`
	Thinking    *thinkingBlock `json:"thinking,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Tools       []toolDef      `json:"tools,omitempty"`
	ToolChoice  any            `json:"tool_choice,omitempty"`
}

// systemBlock represents a system prompt block with cache_control.
type systemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

// cacheControl represents Anthropic cache control marker.
type cacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// thinkingBlock represents Anthropic's extended thinking configuration.
type thinkingBlock struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// chatMessage represents a single message in the conversation.
type chatMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// contentBlock represents a content block within a message.
// Supports all Anthropic content block types: text, image, thinking, tool_use, tool_result.
type contentBlock struct {
	Type       string         `json:"type"`
	Text       string         `json:"text,omitempty"`
	Source     *imageSource   `json:"source,omitempty"`
	Thinking   string         `json:"thinking,omitempty"`
	Signature  string         `json:"signature,omitempty"`
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolUseID  string         `json:"tool_use_id,omitempty"`
	Content    []contentBlock `json:"content,omitempty"`
	IsError    bool           `json:"is_error,omitempty"`
}

// imageSource represents an image content source.
type imageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// toolDef represents a tool definition for function calling.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// toolChoice represents tool choice configuration.
type toolChoiceAny struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// chatResponse represents an Anthropic chat response.
type chatResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        *usageBlock    `json:"usage"`
}

// usageBlock represents token usage in a response.
type usageBlock struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// streamEvent represents a parsed SSE event from Anthropic.
type streamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	Delta        *contentDelta `json:"delta,omitempty"`
	ContentBlock *contentBlock `json:"content_block,omitempty"`
	Usage        *usageBlock   `json:"usage,omitempty"`
	Message      *streamMsg    `json:"message,omitempty"`
}

// contentDelta represents a delta in a content_block_delta event.
type contentDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"` // tool call input JSON delta
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// streamMsg represents message-level metadata in streaming.
type streamMsg struct {
	Usage *usageBlock `json:"usage,omitempty"`
}

// extractTextContent extracts text from Anthropic response content blocks.
// Thinking blocks are skipped; only text blocks are concatenated.
func extractTextContent(blocks []contentBlock) string {
	var sb strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}

// convertMessages converts llm.Messages to Anthropic chat messages.
// All system messages are collected into a slice; user/assistant messages become chat messages.
// ContentBlocks are converted to Anthropic's content block format.
func convertMessages(messages []llm.Message) (systems []string, chatMsgs []chatMessage) {
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// Collect ALL system messages (RAG, summary, skill bodies, etc.)
			for _, block := range msg.GetContentBlocks() {
				if tb, ok := block.(memory.TextBlock); ok {
					systems = append(systems, tb.Text)
					break
				}
			}
		case "user", "assistant":
			// Convert ContentBlocks to Anthropic content blocks
			blocks := convertContentBlocks(msg.GetContentBlocks())
			// Handle multi-part messages (images etc.)
			for _, part := range msg.Parts {
				switch part.Type {
				case "text":
					blocks = append(blocks, contentBlock{Type: "text", Text: part.Text})
				case "image_url":
					if part.ImageURL != nil {
						blocks = append(blocks, contentBlock{
							Type: "image",
							Source: &imageSource{
								Type:      "base64",
								MediaType: "image/png",
								Data:      part.ImageURL.URL,
							},
						})
					}
				}
			}
			chatMsgs = append(chatMsgs, chatMessage{
				Role:    msg.Role,
				Content: blocks,
			})
		}
	}
	return
}

// convertContentBlocks converts memory.ContentBlocks to Anthropic content blocks.
func convertContentBlocks(blocks []memory.ContentBlock) []contentBlock {
	if blocks == nil || len(blocks) == 0 {
		// If no blocks, return a single text block from Content string
		return []contentBlock{{Type: "text", Text: ""}}
	}

	result := make([]contentBlock, 0, len(blocks))
	for _, block := range blocks {
		switch b := block.(type) {
		case memory.TextBlock:
			result = append(result, contentBlock{
				Type: "text",
				Text: b.Text,
			})
		case memory.ThinkingBlock:
			result = append(result, contentBlock{
				Type:      "thinking",
				Thinking:  b.Thinking,
				Signature: b.Signature,
			})
		case memory.ToolUseBlock:
			// Parse Input JSON to map for Anthropic format
			var input map[string]any
			if len(b.Input) > 0 {
				if err := json.Unmarshal(b.Input, &input); err != nil {
					input = map[string]any{"raw": string(b.Input)}
				}
			}
			result = append(result, contentBlock{
				Type:  "tool_use",
				ID:    b.ID,
				Name:  b.Name,
				Input: input,
			})
		case memory.ToolResultBlock:
			// Convert nested Content blocks
			contentBlocks := convertContentBlocks(b.Content)
			result = append(result, contentBlock{
				Type:      "tool_result",
				ToolUseID: b.ToolUseID,
				Content:   contentBlocks,
				IsError:   b.IsError,
			})
		}
	}
	return result
}

// buildThinkingBlock creates an Anthropic thinking configuration block.
// Returns nil if thinking is disabled.
func buildThinkingBlock(thinking *llm.ThinkingConfig) *thinkingBlock {
	if thinking == nil || !thinking.Enabled {
		return nil
	}
	budget := thinking.BudgetTokens
	if budget <= 0 {
		budget = 2048
	}
	return &thinkingBlock{
		Type:         "enabled",
		BudgetTokens: budget,
	}
}
