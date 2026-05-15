// Package llm provides LLM client abstraction and API format converters.
package llm

import (
	"strings"

	"github.com/oneliang/aura/shared/pkg/memory"
)

// AnthropicMessage represents a message in Anthropic API format.
// Anthropic uses content blocks array for structured content.
type AnthropicMessage struct {
	Role    string              `json:"role"`              // "user" or "assistant"
	Content []memory.ContentBlock `json:"content"`          // Array of content blocks
}

// OpenAIMessage represents a message in OpenAI API format.
// OpenAI uses string content and separate tool_calls array.
type OpenAIMessage struct {
	Role      string         `json:"role"`                // "system", "user", "assistant", or "tool"
	Content   string         `json:"content,omitempty"`  // String content (can be empty for tool responses)
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"` // Tool calls from assistant
	ToolCallID string        `json:"tool_call_id,omitempty"` // For tool role messages, references the tool call ID
}

// OpenAIToolCall represents a tool call in OpenAI API format.
type OpenAIToolCall struct {
	ID       string        `json:"id"`               // Unique identifier for the tool call
	Type     string        `json:"type"`             // Always "function"
	Function OpenAIFunction `json:"function"`        // Function details
}

// OpenAIFunction represents function details in a tool call.
type OpenAIFunction struct {
	Name      string `json:"name"`      // Function name
	Arguments string `json:"arguments"` // JSON string of arguments
}

// ConvertToAnthropic converts a Message to Anthropic API format.
// Anthropic messages use content blocks directly, so this conversion
// is straightforward - just use GetContentBlocks().
func ConvertToAnthropic(msg Message) AnthropicMessage {
	return AnthropicMessage{
		Role:    msg.Role,
		Content: msg.GetContentBlocks(),
	}
}

// ConvertToOpenAI converts a Message to OpenAI API format.
// OpenAI splits content into:
// - Content (string): concatenated text from TextBlocks
// - ToolCalls: extracted from ToolUseBlocks
//
// ThinkingBlocks and ToolResultBlocks are not included in OpenAI format
// (they are handled differently by OpenAI API).
func ConvertToOpenAI(msg Message) OpenAIMessage {
	blocks := msg.GetContentBlocks()

	// Extract text content and tool calls
	var textContent strings.Builder
	var toolCalls []OpenAIToolCall

	for _, block := range blocks {
		switch b := block.(type) {
		case memory.TextBlock:
			textContent.WriteString(b.Text)
		case memory.ToolUseBlock:
			// Convert ToolUseBlock to OpenAIToolCall
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   b.ID,
				Type: "function",
				Function: OpenAIFunction{
					Name:      b.Name,
					Arguments: string(b.Input),
				},
			})
		// ThinkingBlocks and ToolResultBlocks are skipped
		// - ThinkingBlocks: OpenAI uses different reasoning format
		// - ToolResultBlocks: These become separate "tool" role messages
		default:
			// Skip unknown block types
		}
	}

	return OpenAIMessage{
		Role:      msg.Role,
		Content:   textContent.String(),
		ToolCalls: toolCalls,
	}
}

// ConvertMessagesToAnthropic converts a slice of Messages to Anthropic format.
func ConvertMessagesToAnthropic(messages []Message) []AnthropicMessage {
	if messages == nil || len(messages) == 0 {
		return nil
	}

	result := make([]AnthropicMessage, len(messages))
	for i, msg := range messages {
		result[i] = ConvertToAnthropic(msg)
	}
	return result
}

// ConvertMessagesToOpenAI converts a slice of Messages to OpenAI format.
// Note: ToolResultBlocks within messages should be converted to separate
// "tool" role messages using ConvertToolResultToOpenAI.
func ConvertMessagesToOpenAI(messages []Message) []OpenAIMessage {
	if messages == nil || len(messages) == 0 {
		return nil
	}

	result := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		result[i] = ConvertToOpenAI(msg)
	}
	return result
}

// ConvertToolResultToOpenAI converts a ToolResultBlock to OpenAI format.
// In OpenAI API, tool results are represented as separate messages with:
// - Role: "tool"
// - Content: the result text
// - ToolCallID: reference to the original tool call
//
// This function concatenates text from all Content blocks in the ToolResultBlock.
func ConvertToolResultToOpenAI(trb memory.ToolResultBlock) OpenAIMessage {
	// Concatenate text content
	var textContent strings.Builder
	for _, block := range trb.Content {
		if tb, ok := block.(memory.TextBlock); ok {
			textContent.WriteString(tb.Text)
		}
	}

	return OpenAIMessage{
		Role:       "tool",
		Content:    textContent.String(),
		ToolCallID: trb.ToolUseID,
	}
}