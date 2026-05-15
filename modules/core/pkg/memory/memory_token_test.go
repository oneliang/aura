package memory

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

func TestConversationMemory_TokenAware(t *testing.T) {
	tests := []struct {
		name          string
		maxTokens     int
		messages      []llm.Message
		wantRemaining int
		wantTokenMin  int // minimum expected tokens after trim
		wantTokenMax  int // maximum expected tokens after trim
	}{
		{
			name:      "under token limit",
			maxTokens: 1000,
			messages: []llm.Message{
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hi there!"}}},
			},
			wantRemaining: 2,
			wantTokenMin:  20,
			wantTokenMax:  100,
		},
		{
			name:      "exceeds token limit",
			maxTokens: 80, // Limit to force trim (each msg pair ~34 tokens with new 1.5 chars/token)
			messages: []llm.Message{
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Message 1"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Response 1"}}},
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Message 2"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Response 2"}}},
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Message 3"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Response 3"}}},
			},
			wantRemaining: 4, // Keeps 2 pairs (user/assistant integrity)
			wantTokenMin:  50,
			wantTokenMax:  90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
				MaxTokens: tt.maxTokens,
				Tokenizer: NewSimpleEstimator(),
			})

			for _, msg := range tt.messages {
				blocks := msg.GetContentBlocks(); for _, block := range blocks { if tb, ok := block.(sharedmemory.TextBlock); ok { mem.Add(msg.Role, tb.Text); break } }
			}

			got := mem.Get()
			if len(got) != tt.wantRemaining {
				t.Errorf("Got %d messages, want %d", len(got), tt.wantRemaining)
			}

			tokenCount := mem.GetTokenCount()
			if tokenCount < tt.wantTokenMin || tokenCount > tt.wantTokenMax {
				t.Errorf("Token count = %d, want [%d, %d]", tokenCount, tt.wantTokenMin, tt.wantTokenMax)
			}
		})
	}
}

func TestConversationMemory_TokenPreservation(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 200,
		Tokenizer: NewSimpleEstimator(),
	})

	// Add messages with code blocks
	mem.Add("user", "Write a Go function")
	mem.Add("assistant", "```go\nfunc hello() {\n\tprintln(\"Hello\")\n}\n```")
	mem.Add("user", "Thanks")
	mem.Add("assistant", "You're welcome!")
	mem.Add("user", "Another request")
	mem.Add("assistant", "Sure, here's another response")

	// Verify user/assistant pairs are preserved
	got := mem.Get()

	// Check that we have complete pairs (even number of messages)
	if len(got)%2 != 0 {
		t.Errorf("Got odd number of messages (%d), expected even", len(got))
	}

	// Verify token count is within limit
	tokenCount := mem.GetTokenCount()
	if tokenCount > 200 {
		t.Errorf("Token count %d exceeds max 200", tokenCount)
	}
}

func TestConversationMemory_VerifyTokenCount(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 1000,
		Tokenizer: NewSimpleEstimator(),
	})

	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi there!")

	// Verify should pass
	if err := mem.VerifyTokenCount(); err != nil {
		t.Errorf("VerifyTokenCount() failed: %v", err)
	}

	// Corrupt the token count to test error detection
	mem.totalTokens = 9999
	if err := mem.VerifyTokenCount(); err == nil {
		t.Error("VerifyTokenCount() should detect corrupted token count")
	}
}

func TestConversationMemory_LoadMessages(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 1000,
		Tokenizer: NewSimpleEstimator(),
	})

	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hi"}}},
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "How are you?"}}},
	}

	mem.LoadMessages(messages)

	if len(mem.Get()) != 3 {
		t.Errorf("LoadMessages() loaded %d messages, want 3", len(mem.Get()))
	}

	tokenCount := mem.GetTokenCount()
	if tokenCount == 0 {
		t.Error("LoadMessages() should calculate token count")
	}
}

func TestConversationMemory_TrimByTokens_CodeBlock(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 150, // Low limit to force trim
		Tokenizer: NewSimpleEstimator(),
	})

	// Add messages where one contains an unclosed code block
	mem.Add("user", "Test 1")
	mem.Add("assistant", "Response 1")
	mem.Add("user", "Test 2")
	mem.Add("assistant", "```go\nfunc test() {\n") // Unclosed code block

	got := mem.Get()
	if len(got) == 0 {
		t.Error("Trim should preserve at least some messages")
	}
}

func TestConversationMemory_FallbackToMaxLen(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen:    3,
		MaxTokens: 0, // Disabled, should use MaxLen
	})

	for i := 0; i < 5; i++ {
		mem.Add("user", "Message")
		mem.Add("assistant", "Response")
	}

	got := mem.Get()
	if len(got) > 3 {
		t.Errorf("Got %d messages, want <= 3 (fallback to MaxLen)", len(got))
	}
}
