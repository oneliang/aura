// Package memory provides memory implementations for agents.
package memory

import (
	"sync"
	"testing"

	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// extractTextFromBlocks extracts the text content from ContentBlocks for testing.
func extractTextFromBlocks(blocks []sharedmemory.ContentBlock) string {
	for _, block := range blocks {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

func TestNewConversationMemoryWithConfig(t *testing.T) {
	// Test with positive maxLen
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})
	if mem == nil {
		t.Fatal("NewConversationMemoryWithConfig() returned nil")
	}
	if mem.maxLen != 10 {
		t.Errorf("maxLen = %d, want 10", mem.maxLen)
	}
	if len(mem.messages) != 0 {
		t.Errorf("initial messages length = %d, want 0", len(mem.messages))
	}

	// Test with zero maxLen (should default to 50)
	mem = NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 0,
	})
	if mem.maxLen != 50 {
		t.Errorf("maxLen with 0 input = %d, want 50", mem.maxLen)
	}

	// Test with negative maxLen (should default to 50)
	mem = NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: -10,
	})
	if mem.maxLen != 50 {
		t.Errorf("maxLen with negative input = %d, want 50", mem.maxLen)
	}
}

func TestConversationMemoryAdd(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})

	mem.Add("user", "Hello")
	if len(mem.messages) != 1 {
		t.Errorf("After Add: len = %d, want 1", len(mem.messages))
	}
	if mem.messages[0].Role != "user" {
		t.Errorf("Role = %v, want 'user'", mem.messages[0].Role)
	}
	if extractTextFromBlocks(mem.messages[0].GetContentBlocks()) != "Hello" {
		t.Errorf("Content = %v, want 'Hello'", extractTextFromBlocks(mem.messages[0].GetContentBlocks()))
	}

	mem.Add("assistant", "Hi there!")
	if len(mem.messages) != 2 {
		t.Errorf("After second Add: len = %d, want 2", len(mem.messages))
	}
}

func TestConversationMemoryMaxLen(t *testing.T) {
	maxLen := 5
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: maxLen,
	})

	// Add more messages than maxLen
	for i := 0; i < 10; i++ {
		mem.Add("user", string(rune('A'+i)))
	}

	// Should be trimmed to maxLen
	if len(mem.messages) != maxLen {
		t.Errorf("After exceeding maxLen: len = %d, want %d", len(mem.messages), maxLen)
	}

	// First message should be the 6th one we added (index 5), not the 1st
	if extractTextFromBlocks(mem.messages[0].GetContentBlocks()) != string(rune('A'+5)) {
		t.Errorf("First message after trim = %v, want %v", extractTextFromBlocks(mem.messages[0].GetContentBlocks()), string(rune('A'+5)))
	}
}

func TestConversationMemoryGet(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})
	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi")

	msgs := mem.Get()
	if len(msgs) != 2 {
		t.Errorf("Get() length = %d, want 2", len(msgs))
	}

	// Verify it returns a copy (modifying result shouldn't affect internal state)
	msgs[0].SetContentBlocks([]sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Modified"}})
	originalMsgs := mem.Get()
	if extractTextFromBlocks(originalMsgs[0].GetContentBlocks()) == "Modified" {
		t.Error("Get() should return a copy, not the original slice")
	}
}

func TestConversationMemoryClear(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})
	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi")

	if len(mem.messages) != 2 {
		t.Errorf("Before Clear: len = %d, want 2", len(mem.messages))
	}

	mem.Clear()

	if len(mem.messages) != 0 {
		t.Errorf("After Clear: len = %d, want 0", len(mem.messages))
	}
}

func TestConversationMemoryLast(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})
	mem.Add("user", "First")
	mem.Add("assistant", "Second")
	mem.Add("user", "Third")
	mem.Add("assistant", "Fourth")

	// Get last 2 messages
	last2 := mem.Last(2)
	if len(last2) != 2 {
		t.Errorf("Last(2) length = %d, want 2", len(last2))
	}
	if extractTextFromBlocks(last2[0].GetContentBlocks()) != "Third" {
		t.Errorf("Last(2)[0] = %v, want 'Third'", extractTextFromBlocks(last2[0].GetContentBlocks()))
	}
	if extractTextFromBlocks(last2[1].GetContentBlocks()) != "Fourth" {
		t.Errorf("Last(2)[1] = %v, want 'Fourth'", extractTextFromBlocks(last2[1].GetContentBlocks()))
	}

	// Get last 10 messages (more than available)
	last10 := mem.Last(10)
	if len(last10) != 4 {
		t.Errorf("Last(10) length = %d, want 4", len(last10))
	}

	// Get last 0 messages
	last0 := mem.Last(0)
	if len(last0) != 0 {
		t.Errorf("Last(0) length = %d, want 0", len(last0))
	}
}

func TestConversationMemoryLen(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})

	if mem.Len() != 0 {
		t.Errorf("Initial Len() = %d, want 0", mem.Len())
	}

	mem.Add("user", "Hello")
	if mem.Len() != 1 {
		t.Errorf("After Add: Len() = %d, want 1", mem.Len())
	}

	mem.Add("assistant", "Hi")
	if mem.Len() != 2 {
		t.Errorf("After second Add: Len() = %d, want 2", mem.Len())
	}

	mem.Clear()
	if mem.Len() != 0 {
		t.Errorf("After Clear: Len() = %d, want 0", mem.Len())
	}
}

func TestConversationMemoryConcurrent(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 100,
	})
	var wg sync.WaitGroup

	// Start multiple goroutines that access the memory concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				mem.Add("user", string(rune('A'+id*10+j)))
				_ = mem.Get()
				_ = mem.Len()
				_ = mem.Last(5)
			}
		}(i)
	}

	wg.Wait()

	// Should have 100 messages (10 goroutines * 10 adds each)
	expectedLen := 100
	if len(mem.messages) != expectedLen {
		t.Errorf("Concurrent adds: len = %d, want %d", len(mem.messages), expectedLen)
	}
}

func TestConversationMemoryConcurrentWithMaxLen(t *testing.T) {
	maxLen := 50
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: maxLen,
	})
	var wg sync.WaitGroup

	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				mem.Add("user", string(rune('A'+id*10+j)))
			}
		}(i)
	}

	wg.Wait()

	// Should be trimmed to maxLen
	if len(mem.messages) != maxLen {
		t.Errorf("Concurrent adds with maxLen: len = %d, want %d", len(mem.messages), maxLen)
	}
}

func TestConversationMemoryGetEmpty(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})

	msgs := mem.Get()
	if len(msgs) != 0 {
		t.Errorf("Get() on empty memory: len = %d, want 0", len(msgs))
	}
}

func TestConversationMemoryClearEmpty(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})

	// Clear on empty memory should not panic
	mem.Clear()

	if len(mem.messages) != 0 {
		t.Errorf("Clear() on empty memory: len = %d, want 0", len(mem.messages))
	}
}

func TestConversationMemoryLastEmpty(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxLen: 10,
	})

	last := mem.Last(5)
	if len(last) != 0 {
		t.Errorf("Last() on empty memory: len = %d, want 0", len(last))
	}
}

func TestBaseMemoryAddWithBlocks(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: 10,
	})

	// Test adding a message with text blocks
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello, world!"},
	}
	mem.AddWithBlocks("user", blocks, sharedmemory.MessageTypeUser)

	// Verify the message was added
	msgs := mem.Get()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Verify ContentBlocks contain the text
	if extractTextFromBlocks(msgs[0].GetContentBlocks()) != "Hello, world!" {
		t.Errorf("Content = %q, want %q", extractTextFromBlocks(msgs[0].GetContentBlocks()), "Hello, world!")
	}

	// Verify ContentBlocks are stored
	storedBlocks := msgs[0].GetContentBlocks()
	if len(storedBlocks) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(storedBlocks))
	}
	if tb, ok := storedBlocks[0].(sharedmemory.TextBlock); !ok {
		t.Errorf("Expected TextBlock, got %T", storedBlocks[0])
	} else if tb.Text != "Hello, world!" {
		t.Errorf("TextBlock.Text = %q, want %q", tb.Text, "Hello, world!")
	}
}

func TestBaseMemoryAddWithBlocksMultipleBlocks(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: 10,
	})

	// Test with multiple blocks including non-text blocks
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "First text"},
		sharedmemory.ThinkingBlock{Type: sharedmemory.BlockTypeThinking, Thinking: "Thinking..."},
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Second text"},
	}
	mem.AddWithBlocks("assistant", blocks, sharedmemory.MessageTypeAssistant)

	msgs := mem.Get()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Content should be extracted from the first TextBlock
	if extractTextFromBlocks(msgs[0].GetContentBlocks()) != "First text" {
		t.Errorf("Content = %q, want %q", extractTextFromBlocks(msgs[0].GetContentBlocks()), "First text")
	}

	// All blocks should be preserved
	storedBlocks := msgs[0].GetContentBlocks()
	if len(storedBlocks) != 3 {
		t.Fatalf("Expected 3 content blocks, got %d", len(storedBlocks))
	}
}

func TestBaseMemoryAddWithBlocksNoTextBlock(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: 10,
	})

	// Test with no text blocks (only thinking block)
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.ThinkingBlock{Type: sharedmemory.BlockTypeThinking, Thinking: "Deep thought"},
	}
	mem.AddWithBlocks("assistant", blocks, sharedmemory.MessageTypeAssistant)

	msgs := mem.Get()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Content should be empty since no TextBlock
	if extractTextFromBlocks(msgs[0].GetContentBlocks()) != "" {
		t.Errorf("Content = %q, want empty string", extractTextFromBlocks(msgs[0].GetContentBlocks()))
	}

	// Blocks should still be preserved
	storedBlocks := msgs[0].GetContentBlocks()
	if len(storedBlocks) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(storedBlocks))
	}
}

func TestBaseMemoryAddWithBlocksEmptyBlocks(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: 10,
	})

	// Test with empty blocks slice
	mem.AddWithBlocks("user", []sharedmemory.ContentBlock{}, sharedmemory.MessageTypeUser)

	msgs := mem.Get()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Content should be empty
	if extractTextFromBlocks(msgs[0].GetContentBlocks()) != "" {
		t.Errorf("Content = %q, want empty string", extractTextFromBlocks(msgs[0].GetContentBlocks()))
	}
}

func TestBaseMemoryAddWithBlocksTokenCounting(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxTokens: 100,
		Tokenizer: NewSimpleEstimator(),
	})

	// Add message with blocks
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello, world!"},
	}
	mem.AddWithBlocks("user", blocks, sharedmemory.MessageTypeUser)

	// Verify token count was updated
	if mem.GetTokenCount() == 0 {
		t.Error("Token count should be greater than 0 after adding message")
	}
}

func TestBaseMemoryAddWithBlocksTrimming(t *testing.T) {
	maxLen := 3
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: maxLen,
	})

	// Add more messages than maxLen
	for i := 0; i < 5; i++ {
		blocks := []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: string(rune('A' + i))},
		}
		mem.AddWithBlocks("user", blocks, sharedmemory.MessageTypeUser)
	}

	// Should be trimmed to maxLen
	msgs := mem.Get()
	if len(msgs) != maxLen {
		t.Errorf("After exceeding maxLen: len = %d, want %d", len(msgs), maxLen)
	}

	// First message should be the 3rd one added (index 2), not the 1st
	if extractTextFromBlocks(msgs[0].GetContentBlocks()) != string(rune('C')) {
		t.Errorf("First message after trim = %v, want %v", extractTextFromBlocks(msgs[0].GetContentBlocks()), string(rune('C')))
	}
}

func TestBaseMemoryAddWithBlocksUpdatesLastActiveTime(t *testing.T) {
	mem := NewBaseMemory(BaseMemoryConfig{
		MaxLen: 10,
	})

	initialTime := mem.GetLastActiveTime()

	// Add a small delay to ensure time difference
	// (Note: in practice this is very fast, but the update should still work)
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "test"},
	}
	mem.AddWithBlocks("user", blocks, sharedmemory.MessageTypeUser)

	afterTime := mem.GetLastActiveTime()

	// LastActiveTime should be updated (afterTime >= initialTime)
	if afterTime.Before(initialTime) {
		t.Error("LastActiveTime should be updated after AddWithBlocks")
	}
}
