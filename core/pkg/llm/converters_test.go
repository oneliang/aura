package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to create a message with text content
func newTextMessageForConverter(role, text string) Message {
	msg := Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

func TestAnthropicMessage(t *testing.T) {
	t.Run("creates AnthropicMessage with content blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
		}
		msg := AnthropicMessage{
			Role:    "user",
			Content: blocks,
		}

		if msg.Role != "user" {
			t.Errorf("expected role user, got %s", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Errorf("expected 1 content block, got %d", len(msg.Content))
		}
	})

	t.Run("marshals to JSON correctly", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
			memory.ToolUseBlock{Type: memory.BlockTypeToolUse, ID: "tool1", Name: "test_tool"},
		}
		msg := AnthropicMessage{
			Role:    "assistant",
			Content: blocks,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Verify it strings.Contains expected fields
		if !strings.Contains(string(data), `"role":"assistant"`) {
			t.Errorf("expected role in JSON, got %s", string(data))
		}
		if !strings.Contains(string(data), `"type":"text"`) {
			t.Errorf("expected text type in JSON, got %s", string(data))
		}
	})
}

func TestOpenAIMessage(t *testing.T) {
	t.Run("creates OpenAIMessage with string content", func(t *testing.T) {
		msg := OpenAIMessage{
			Role:    "user",
			Content: "Hello",
		}

		if msg.Role != "user" {
			t.Errorf("expected role user, got %s", msg.Role)
		}
		if msg.Content != "Hello" {
			t.Errorf("expected content Hello, got %s", msg.Content)
		}
	})

	t.Run("creates OpenAIMessage with tool calls", func(t *testing.T) {
		msg := OpenAIMessage{
			Role:    "assistant",
			Content: "",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: OpenAIFunction{
						Name:      "test_tool",
						Arguments: `{"arg1": "value1"}`,
					},
				},
			},
		}

		if len(msg.ToolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(msg.ToolCalls))
		}
		if msg.ToolCalls[0].Function.Name != "test_tool" {
			t.Errorf("expected tool name test_tool, got %s", msg.ToolCalls[0].Function.Name)
		}
	})

	t.Run("marshals to JSON correctly", func(t *testing.T) {
		msg := OpenAIMessage{
			Role:    "assistant",
			Content: "Hello",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: OpenAIFunction{
						Name:      "test_tool",
						Arguments: `{"arg1": "value1"}`,
					},
				},
			},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if !strings.Contains(string(data), `"role":"assistant"`) {
			t.Errorf("expected role in JSON, got %s", string(data))
		}
		if !strings.Contains(string(data), `"tool_calls"`) {
			t.Errorf("expected tool_calls in JSON, got %s", string(data))
		}
	})
}

func TestConvertToAnthropic(t *testing.T) {
	t.Run("converts simple text message", func(t *testing.T) {
		msg := newTextMessageForConverter("user", "Hello, world!")

		result := ConvertToAnthropic(msg)

		if result.Role != "user" {
			t.Errorf("expected role user, got %s", result.Role)
		}
		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(result.Content))
		}

		tb, ok := result.Content[0].(memory.TextBlock)
		if !ok {
			t.Fatalf("expected TextBlock, got %T", result.Content[0])
		}
		if tb.Text != "Hello, world!" {
			t.Errorf("expected text 'Hello, world!', got %s", tb.Text)
		}
	})

	t.Run("converts message with content blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "thinking..."},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToAnthropic(msg)

		if result.Role != "assistant" {
			t.Errorf("expected role assistant, got %s", result.Role)
		}
		if len(result.Content) != 2 {
			t.Errorf("expected 2 content blocks, got %d", len(result.Content))
		}
	})

	t.Run("converts message with tool use blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Let me help you"},
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "tool_123",
				Name:  "search",
				Input: json.RawMessage(`{"query": "test"}`),
			},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToAnthropic(msg)

		if len(result.Content) != 2 {
			t.Errorf("expected 2 content blocks, got %d", len(result.Content))
		}

		// Check tool use block
		tub, ok := result.Content[1].(memory.ToolUseBlock)
		if !ok {
			t.Fatalf("expected ToolUseBlock, got %T", result.Content[1])
		}
		if tub.ID != "tool_123" {
			t.Errorf("expected ID tool_123, got %s", tub.ID)
		}
		if tub.Name != "search" {
			t.Errorf("expected name search, got %s", tub.Name)
		}
	})

	t.Run("converts message with tool result blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.ToolResultBlock{
				Type:      memory.BlockTypeToolResult,
				ToolUseID: "tool_123",
				Content: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "result"},
				},
			},
		}
		msg := Message{Role: "user"}
		msg.SetContentBlocks(blocks)

		result := ConvertToAnthropic(msg)

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(result.Content))
		}

		trb, ok := result.Content[0].(memory.ToolResultBlock)
		if !ok {
			t.Fatalf("expected ToolResultBlock, got %T", result.Content[0])
		}
		if trb.ToolUseID != "tool_123" {
			t.Errorf("expected ToolUseID tool_123, got %s", trb.ToolUseID)
		}
	})

	t.Run("handles empty message", func(t *testing.T) {
		msg := Message{Role: "user"}

		result := ConvertToAnthropic(msg)

		if result.Role != "user" {
			t.Errorf("expected role user, got %s", result.Role)
		}
		if len(result.Content) != 0 {
			t.Errorf("expected 0 content blocks, got %d", len(result.Content))
		}
	})
}

func TestConvertToOpenAI(t *testing.T) {
	t.Run("converts simple text message", func(t *testing.T) {
		msg := newTextMessageForConverter("user", "Hello, world!")

		result := ConvertToOpenAI(msg)

		if result.Role != "user" {
			t.Errorf("expected role user, got %s", result.Role)
		}
		if result.Content != "Hello, world!" {
			t.Errorf("expected content 'Hello, world!', got %s", result.Content)
		}
		if len(result.ToolCalls) != 0 {
			t.Errorf("expected 0 tool calls, got %d", len(result.ToolCalls))
		}
	})

	t.Run("extracts tool calls from tool use blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Let me help"},
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "call_123",
				Name:  "search",
				Input: json.RawMessage(`{"query": "test"}`),
			},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToOpenAI(msg)

		if result.Content != "Let me help" {
			t.Errorf("expected content 'Let me help', got %s", result.Content)
		}
		if len(result.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
		}

		tc := result.ToolCalls[0]
		if tc.ID != "call_123" {
			t.Errorf("expected ID call_123, got %s", tc.ID)
		}
		if tc.Type != "function" {
			t.Errorf("expected type function, got %s", tc.Type)
		}
		if tc.Function.Name != "search" {
			t.Errorf("expected function name search, got %s", tc.Function.Name)
		}
		if tc.Function.Arguments != `{"query": "test"}` {
			t.Errorf("expected arguments, got %s", tc.Function.Arguments)
		}
	})

	t.Run("handles multiple tool use blocks", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "call_1",
				Name:  "tool1",
				Input: json.RawMessage(`{}`),
			},
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "call_2",
				Name:  "tool2",
				Input: json.RawMessage(`{}`),
			},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToOpenAI(msg)

		if len(result.ToolCalls) != 2 {
			t.Errorf("expected 2 tool calls, got %d", len(result.ToolCalls))
		}
	})

	t.Run("concatenates multiple text blocks for content", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello "},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "world!"},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToOpenAI(msg)

		if result.Content != "Hello world!" {
			t.Errorf("expected content 'Hello world!', got %s", result.Content)
		}
	})

	t.Run("skips thinking blocks for content", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "internal thought"},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Response"},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		result := ConvertToOpenAI(msg)

		if result.Content != "Response" {
			t.Errorf("expected content 'Response', got %s", result.Content)
		}
	})

	t.Run("skips tool result blocks for content", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.ToolResultBlock{
				Type:      memory.BlockTypeToolResult,
				ToolUseID: "tool_123",
				Content: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "result"},
				},
			},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Final answer"},
		}
		msg := Message{Role: "user"}
		msg.SetContentBlocks(blocks)

		result := ConvertToOpenAI(msg)

		if result.Content != "Final answer" {
			t.Errorf("expected content 'Final answer', got %s", result.Content)
		}
	})

	t.Run("handles empty message", func(t *testing.T) {
		msg := Message{Role: "user"}

		result := ConvertToOpenAI(msg)

		if result.Role != "user" {
			t.Errorf("expected role user, got %s", result.Role)
		}
		if result.Content != "" {
			t.Errorf("expected empty content, got %s", result.Content)
		}
	})
}

func TestConvertMessagesToAnthropic(t *testing.T) {
	t.Run("converts multiple messages", func(t *testing.T) {
		messages := []Message{
			newTextMessageForConverter("system", "You are helpful."),
			newTextMessageForConverter("user", "Hello"),
			newTextMessageForConverter("assistant", "Hi there!"),
		}

		result := ConvertMessagesToAnthropic(messages)

		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}

		if result[0].Role != "system" {
			t.Errorf("expected role system, got %s", result[0].Role)
		}
		if result[1].Role != "user" {
			t.Errorf("expected role user, got %s", result[1].Role)
		}
		if result[2].Role != "assistant" {
			t.Errorf("expected role assistant, got %s", result[2].Role)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := ConvertMessagesToAnthropic([]Message{})

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		result := ConvertMessagesToAnthropic(nil)

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})
}

func TestConvertMessagesToOpenAI(t *testing.T) {
	t.Run("converts multiple messages", func(t *testing.T) {
		messages := []Message{
			newTextMessageForConverter("system", "You are helpful."),
			newTextMessageForConverter("user", "Hello"),
			newTextMessageForConverter("assistant", "Hi there!"),
		}

		result := ConvertMessagesToOpenAI(messages)

		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}

		if result[0].Role != "system" {
			t.Errorf("expected role system, got %s", result[0].Role)
		}
		if result[0].Content != "You are helpful." {
			t.Errorf("expected content 'You are helpful.', got %s", result[0].Content)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := ConvertMessagesToOpenAI([]Message{})

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		result := ConvertMessagesToOpenAI(nil)

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})

	t.Run("extracts tool calls from messages with tool use", func(t *testing.T) {
		blocks := []memory.ContentBlock{
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "call_1",
				Name:  "test",
				Input: json.RawMessage(`{}`),
			},
		}
		msg := Message{Role: "assistant"}
		msg.SetContentBlocks(blocks)

		messages := []Message{msg}
		result := ConvertMessagesToOpenAI(messages)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		if len(result[0].ToolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(result[0].ToolCalls))
		}
	})
}

func TestConvertToolResultToOpenAI(t *testing.T) {
	t.Run("converts tool result with text content", func(t *testing.T) {
		trb := memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "call_123",
			Content: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: "Tool result"},
			},
		}

		result := ConvertToolResultToOpenAI(trb)

		if result.Role != "tool" {
			t.Errorf("expected role 'tool', got %s", result.Role)
		}
		if result.Content != "Tool result" {
			t.Errorf("expected content 'Tool result', got %s", result.Content)
		}
		if result.ToolCallID != "call_123" {
			t.Errorf("expected ToolCallID 'call_123', got %s", result.ToolCallID)
		}
	})

	t.Run("concatenates multiple content blocks", func(t *testing.T) {
		trb := memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "call_123",
			Content: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: "Part 1 "},
				memory.TextBlock{Type: memory.BlockTypeText, Text: "Part 2"},
			},
		}

		result := ConvertToolResultToOpenAI(trb)

		if result.Content != "Part 1 Part 2" {
			t.Errorf("expected content 'Part 1 Part 2', got %s", result.Content)
		}
	})

	t.Run("handles error result", func(t *testing.T) {
		trb := memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "call_123",
			Content: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: "Error: something went wrong"},
			},
			IsError: true,
		}

		result := ConvertToolResultToOpenAI(trb)

		if result.Role != "tool" {
			t.Errorf("expected role 'tool', got %s", result.Role)
		}
		// Note: IsError is not explicitly represented in OpenAI format,
		// but the content should still be preserved
		if !strings.Contains(result.Content, "Error:") {
			t.Errorf("expected content to contain 'Error:', got %s", result.Content)
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		trb := memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "call_123",
			Content:   []memory.ContentBlock{},
		}

		result := ConvertToolResultToOpenAI(trb)

		if result.Content != "" {
			t.Errorf("expected empty content, got %s", result.Content)
		}
	})

	t.Run("handles nil content", func(t *testing.T) {
		trb := memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "call_123",
			Content:   nil,
		}

		result := ConvertToolResultToOpenAI(trb)

		if result.Content != "" {
			t.Errorf("expected empty content, got %s", result.Content)
		}
	})
}

func TestOpenAIToolCallJSON(t *testing.T) {
	t.Run("marshals OpenAIToolCall correctly", func(t *testing.T) {
		tc := OpenAIToolCall{
			ID:   "call_123",
			Type: "function",
			Function: OpenAIFunction{
				Name:      "search",
				Arguments: `{"query": "test"}`,
			},
		}

		data, err := json.Marshal(tc)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if !strings.Contains(string(data), `"id":"call_123"`) {
			t.Errorf("expected ID in JSON, got %s", string(data))
		}
	})
}