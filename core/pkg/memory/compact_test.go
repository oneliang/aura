package memory

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize i18n with English locale for consistent test results
	i18n.Init("", "en")
}

func TestSelectiveCompressorConfigDefaults(t *testing.T) {
	cfg := DefaultSelectiveCompressorConfig()

	assert.Equal(t, 10, cfg.RecentFullCount, "RecentFullCount default should be 10")
	assert.Equal(t, 3, cfg.RecentToolResults, "RecentToolResults default should be 3")
	assert.Equal(t, 2, cfg.RecentThinking, "RecentThinking default should be 2")
}

func TestNewSelectiveCompressor(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		assert.NotNil(t, compressor)
		assert.Equal(t, cfg, compressor.config)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   5,
			RecentToolResults: 2,
			RecentThinking:    1,
		}
		compressor := NewSelectiveCompressor(cfg)

		assert.NotNil(t, compressor)
		assert.Equal(t, 5, compressor.config.RecentFullCount)
		assert.Equal(t, 2, compressor.config.RecentToolResults)
		assert.Equal(t, 1, compressor.config.RecentThinking)
	})

	t.Run("applies defaults for zero values", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{} // All zeros
		compressor := NewSelectiveCompressor(cfg)

		assert.Equal(t, 10, compressor.config.RecentFullCount)
		assert.Equal(t, 3, compressor.config.RecentToolResults)
		assert.Equal(t, 2, compressor.config.RecentThinking)
	})
}

func TestSelectiveCompressorCompress(t *testing.T) {
	t.Run("empty messages", func(t *testing.T) {
		compressor := NewSelectiveCompressor(DefaultSelectiveCompressorConfig())
		result := compressor.Compress([]llm.Message{})

		assert.Empty(t, result.Messages)
		assert.Equal(t, 0, result.PreTokens)
		assert.Equal(t, 0, result.PostTokens)
	})

	t.Run("fewer messages than RecentFullCount keeps all", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   5,
			RecentToolResults: 2,
			RecentThinking:    1,
		}
		compressor := NewSelectiveCompressor(cfg)

		messages := []llm.Message{
			{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"}}},
			{Role: "assistant", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Hi there"}}},
		}
		result := compressor.Compress(messages)

		assert.Len(t, result.Messages, 2)
		assert.Equal(t, messages, result.Messages)
	})

	t.Run("compresses old messages", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   2,
			RecentToolResults: 1,
			RecentThinking:    1,
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create messages with thinking blocks in old messages
		oldMsg := llm.Message{
			Role: "assistant",
		}
		oldMsg.SetContentBlocks([]memory.ContentBlock{
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "This is a long thinking process that should be truncated to about fifty characters or so"},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Response text"},
		})

		recentMsg := llm.Message{
			Role: "assistant",
		}
		recentMsg.SetContentBlocks([]memory.ContentBlock{
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Recent thinking"},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Recent response"},
		})

		messages := []llm.Message{
			{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"}}},
			oldMsg,
			recentMsg,
		}

		result := compressor.Compress(messages)

		// Result should have: boundary + compressed old + recent messages
		// 3 input messages, RecentFullCount=2 means last 2 are "recent"
		// Old = [user "Hello"] (1 message), Recent = [oldMsg, recentMsg] (2 messages)
		// Result: boundary(1) + compressed_old(1) + recent(2) = 4 messages
		assert.Len(t, result.Messages, 4)
		// First message should be the boundary marker
		assert.Equal(t, "system", result.Messages[0].Role)
	})

	t.Run("compresses tool results in old messages", func(t *testing.T) {
		// Use config with RecentToolResults=1 so that with 2 tool results, first 1 gets summarized
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   1,
			RecentToolResults: 1, // Only last 1 tool result is kept intact
			RecentThinking:    1,
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create a message with 2 tool results - first should be summarized
		oldMsg := llm.Message{
			Role: "user",
		}
		oldMsg.SetContentBlocks([]memory.ContentBlock{
			memory.ToolResultBlock{
				Type:       memory.BlockTypeToolResult,
				ToolUseID:  "tool_1",
				Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "First tool result"}},
				IsError:    false,
			},
			memory.ToolResultBlock{
				Type:       memory.BlockTypeToolResult,
				ToolUseID:  "tool_2",
				Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Second tool result"}},
				IsError:    false,
			},
		})

		recentMsg := llm.Message{Role: "assistant", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Recent response"}}}

		messages := []llm.Message{
			oldMsg,
			recentMsg,
		}

		result := compressor.Compress(messages)

		// 2 input messages, RecentFullCount=1 means last 1 is "recent"
		// Old = [oldMsg] (1 message), Recent = [recentMsg] (1 message)
		// Result: boundary(1) + compressed_old(1) + recent(1) = 3 messages
		assert.Len(t, result.Messages, 3)
		// First message should be the boundary marker
		assert.Equal(t, "system", result.Messages[0].Role)
		// Verify old message was compressed - first tool result should be summarized
		blocks := result.Messages[1].GetContentBlocks()
		assert.Len(t, blocks, 2)
		tb, ok := blocks[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Contains(t, tb.Text, "[tool_result")
		// Second tool result should be kept intact
		trb, ok := blocks[1].(memory.ToolResultBlock)
		assert.True(t, ok)
		assert.Equal(t, "tool_2", trb.ToolUseID)
	})
}

func TestCompressOldMessages(t *testing.T) {
	t.Run("keeps text blocks unchanged", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		msg := llm.Message{Role: "assistant"}
		msg.SetContentBlocks([]memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "This is text"},
		})

		messages := []llm.Message{msg}
		compressed := compressor.compressOldMessages(messages)

		assert.Len(t, compressed, 1)
		blocks := compressed[0].GetContentBlocks()
		assert.Len(t, blocks, 1)
		tb, ok := blocks[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Equal(t, "This is text", tb.Text)
	})

	t.Run("keeps tool use blocks unchanged", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		msg := llm.Message{Role: "assistant"}
		msg.SetContentBlocks([]memory.ContentBlock{
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "tool_123",
				Name:  "Read",
				Input: []byte(`{"file": "test.go"}`),
			},
		})

		messages := []llm.Message{msg}
		compressed := compressor.compressOldMessages(messages)

		assert.Len(t, compressed, 1)
		blocks := compressed[0].GetContentBlocks()
		assert.Len(t, blocks, 1)
		tu, ok := blocks[0].(memory.ToolUseBlock)
		assert.True(t, ok)
		assert.Equal(t, "Read", tu.Name)
	})

	t.Run("truncates thinking blocks", func(t *testing.T) {
		// Use config with RecentThinking=1 so with 2 thinking blocks, first 1 gets truncated
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   10,
			RecentToolResults: 3,
			RecentThinking:    1, // Only last 1 thinking block is kept intact
		}
		compressor := NewSelectiveCompressor(cfg)

		msg := llm.Message{Role: "assistant"}
		msg.SetContentBlocks([]memory.ContentBlock{
			memory.ThinkingBlock{
				Type:     memory.BlockTypeThinking,
				Thinking: "This is a very long thinking process that exceeds fifty characters and should be truncated",
			},
			memory.ThinkingBlock{
				Type:     memory.BlockTypeThinking,
				Thinking: "This is recent thinking",
			},
		})

		messages := []llm.Message{msg}
		compressed := compressor.compressOldMessages(messages)

		blocks := compressed[0].GetContentBlocks()
		assert.Len(t, blocks, 2)
		tb, ok := blocks[0].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(tb.Thinking), 53) // ~50 chars + "..."
		// Second thinking block should be kept intact
		tb2, ok := blocks[1].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.Equal(t, "This is recent thinking", tb2.Thinking)
	})

	t.Run("keeps short thinking blocks unchanged", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		msg := llm.Message{Role: "assistant"}
		msg.SetContentBlocks([]memory.ContentBlock{
			memory.ThinkingBlock{
				Type:     memory.BlockTypeThinking,
				Thinking: "Short thinking",
			},
		})

		messages := []llm.Message{msg}
		compressed := compressor.compressOldMessages(messages)

		blocks := compressed[0].GetContentBlocks()
		tb, ok := blocks[0].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.Equal(t, "Short thinking", tb.Thinking)
	})
}

func TestCompressBlocks(t *testing.T) {
	t.Run("compresses tool result blocks when more than recent count", func(t *testing.T) {
		// Use config with RecentToolResults=2, so with 3 tool results, first 1 will be summarized
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   10,
			RecentToolResults: 2,
			RecentThinking:    2,
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create 3 tool result blocks - first 1 should be summarized
		blocks := []memory.ContentBlock{
			memory.ToolResultBlock{
				Type:       memory.BlockTypeToolResult,
				ToolUseID:  "tool_1",
				Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Long content 1"}},
				IsError:    false,
			},
			memory.ToolResultBlock{
				Type:       memory.BlockTypeToolResult,
				ToolUseID:  "tool_2",
				Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Content 2"}},
				IsError:    false,
			},
			memory.ToolResultBlock{
				Type:       memory.BlockTypeToolResult,
				ToolUseID:  "tool_3",
				Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Content 3"}},
				IsError:    false,
			},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 3)

		// First should be a text block with summary
		tb, ok := compressed[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Contains(t, tb.Text, "[tool_result tool_1:")

		// Last 2 should be kept intact
		trb2, ok := compressed[1].(memory.ToolResultBlock)
		assert.True(t, ok)
		assert.Equal(t, "tool_2", trb2.ToolUseID)

		trb3, ok := compressed[2].(memory.ToolResultBlock)
		assert.True(t, ok)
		assert.Equal(t, "tool_3", trb3.ToolUseID)
	})

	t.Run("keeps recent tool result blocks intact", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   10,
			RecentToolResults: 2,
			RecentThinking:    2,
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create 4 tool result blocks - last 2 should be kept intact
		blocks := []memory.ContentBlock{
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t1", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 1"}}},
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t2", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 2"}}},
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t3", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 3"}}},
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t4", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 4"}}},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 4)

		// First 2 should be summarized (text blocks)
		tb1, ok := compressed[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Contains(t, tb1.Text, "[tool_result")

		tb2, ok := compressed[1].(memory.TextBlock)
		assert.True(t, ok)
		assert.Contains(t, tb2.Text, "[tool_result")

		// Last 2 should be kept intact (ToolResultBlock)
		trb3, ok := compressed[2].(memory.ToolResultBlock)
		assert.True(t, ok)
		assert.Equal(t, "t3", trb3.ToolUseID)

		trb4, ok := compressed[3].(memory.ToolResultBlock)
		assert.True(t, ok)
		assert.Equal(t, "t4", trb4.ToolUseID)
	})

	t.Run("keeps recent thinking blocks intact", func(t *testing.T) {
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   10,
			RecentToolResults: 2,
			RecentThinking:    2,
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create 4 thinking blocks - last 2 should be kept intact
		blocks := []memory.ContentBlock{
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 1 that is quite long and should be truncated normally"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 2 that is quite long and should be truncated normally"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 3 that is recent and should be kept intact"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 4 that is recent and should be kept intact"},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 4)

		// First 2 should be truncated
		tb1, ok := compressed[0].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(tb1.Thinking), 53)

		tb2, ok := compressed[1].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(tb2.Thinking), 53)

		// Last 2 should be kept intact
		tb3, ok := compressed[2].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.Equal(t, "Thinking 3 that is recent and should be kept intact", tb3.Thinking)

		tb4, ok := compressed[3].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.Equal(t, "Thinking 4 that is recent and should be kept intact", tb4.Thinking)
	})

	t.Run("keeps tool use blocks unchanged", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		blocks := []memory.ContentBlock{
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "tool_123",
				Name:  "Read",
				Input: []byte(`{"file": "test.go"}`),
			},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 1)

		tu, ok := compressed[0].(memory.ToolUseBlock)
		assert.True(t, ok)
		assert.Equal(t, "Read", tu.Name)
	})

	t.Run("keeps text blocks unchanged", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 1)

		tb, ok := compressed[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Equal(t, "Hello", tb.Text)
	})

	t.Run("handles mixed blocks", func(t *testing.T) {
		// Use config with small recent counts so blocks get compressed
		cfg := SelectiveCompressorConfig{
			RecentFullCount:   10,
			RecentToolResults: 1, // Only last 1 tool result is kept
			RecentThinking:    1, // Only last 1 thinking block is kept
		}
		compressor := NewSelectiveCompressor(cfg)

		// Create blocks with 2 thinking and 2 tool results
		blocks := []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Text content"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 1 that is quite long and should be truncated"},
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking 2 is recent kept intact"},
			memory.ToolUseBlock{Type: memory.BlockTypeToolUse, ID: "t1", Name: "Read"},
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t1", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 1"}}},
			memory.ToolResultBlock{Type: memory.BlockTypeToolResult, ToolUseID: "t2", Content: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result 2"}}},
		}

		compressed := compressor.compressBlocks(blocks)
		assert.Len(t, compressed, 6)

		// Block 0: Text block unchanged
		tb0, ok := compressed[0].(memory.TextBlock)
		assert.True(t, ok)
		assert.Equal(t, "Text content", tb0.Text)

		// Block 1: First thinking block truncated (not in recent)
		tb1, ok := compressed[1].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(tb1.Thinking), 53)

		// Block 2: Second thinking block kept intact (is recent)
		tb2, ok := compressed[2].(memory.ThinkingBlock)
		assert.True(t, ok)
		assert.Equal(t, "Thinking 2 is recent kept intact", tb2.Thinking)

		// Block 3: Tool use unchanged
		_, ok = compressed[3].(memory.ToolUseBlock)
		assert.True(t, ok)

		// Block 4: First tool result summarized (not in recent)
		tb4, ok := compressed[4].(memory.TextBlock)
		assert.True(t, ok)
		assert.Contains(t, tb4.Text, "[tool_result")

		// Block 5: Second tool result kept intact (is recent)
		_, ok = compressed[5].(memory.ToolResultBlock)
		assert.True(t, ok)
	})
}

func TestSummarizeToolResult(t *testing.T) {
	t.Run("basic summary", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		trb := memory.ToolResultBlock{
			Type:       memory.BlockTypeToolResult,
			ToolUseID:  "tool_123",
			Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Result content"}},
			IsError:    false,
		}

		summary := compressor.summarizeToolResult(trb)
		assert.Contains(t, summary, "[tool_result tool_123:")
		assert.Contains(t, summary, "chars]")
	})

	t.Run("error tool result", func(t *testing.T) {
		cfg := DefaultSelectiveCompressorConfig()
		compressor := NewSelectiveCompressor(cfg)

		trb := memory.ToolResultBlock{
			Type:       memory.BlockTypeToolResult,
			ToolUseID:  "tool_err",
			Content:    []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "Error message"}},
			IsError:    true,
		}

		summary := compressor.summarizeToolResult(trb)
		assert.Contains(t, summary, "[tool_result tool_err:")
		assert.Contains(t, summary, "ERROR")
	})
}

func TestCompactBoundaryMessage(t *testing.T) {
	t.Run("creates boundary message", func(t *testing.T) {
		msg := CompactBoundaryMessage{
			Type:     "system",
			Subtype:  "compact_boundary",
			Content:  "Conversation compacted",
			Metadata: CompactMetadata{Trigger: "token_limit", PreTokens: 1000, PostTokens: 500, Strategy: "selective"},
		}

		assert.Equal(t, "system", msg.Type)
		assert.Equal(t, "compact_boundary", msg.Subtype)
		assert.Equal(t, "Conversation compacted", msg.Content)
		assert.Equal(t, "token_limit", msg.Metadata.Trigger)
		assert.Equal(t, 1000, msg.Metadata.PreTokens)
		assert.Equal(t, 500, msg.Metadata.PostTokens)
		assert.Equal(t, "selective", msg.Metadata.Strategy)
	})

	t.Run("converts to llm.Message", func(t *testing.T) {
		msg := CompactBoundaryMessage{
			Type:     "system",
			Subtype:  "compact_boundary",
			Content:  "Conversation compacted",
			Metadata: CompactMetadata{Trigger: "token_limit", PreTokens: 1000, PostTokens: 500, Strategy: "selective"},
		}

		llmMsg := msg.ToMessage()
		assert.Equal(t, "system", llmMsg.Role)
		assert.NotEmpty(t, llmMsg.GetContentBlocks())
	})
}

func TestCompressResult(t *testing.T) {
	t.Run("stores token counts", func(t *testing.T) {
		result := CompressResult{
			Messages:  []llm.Message{{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "test"}}}},
			PreTokens:  100,
			PostTokens: 50,
		}

		assert.Equal(t, 100, result.PreTokens)
		assert.Equal(t, 50, result.PostTokens)
		assert.Len(t, result.Messages, 1)
	})
}

func TestTruncateThinking(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short text unchanged",
			input:    "Short",
			expected: "Short",
		},
		{
			name:     "exactly 50 chars unchanged",
			input:    "This is exactly fifty characters long text here!!",
			expected: "This is exactly fifty characters long text here!!",
		},
		{
			name:     "long text truncated",
			input:    "This is a very long thinking process that definitely exceeds fifty characters",
			expected: "This is a very long thinking process that definite...",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateThinking(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}