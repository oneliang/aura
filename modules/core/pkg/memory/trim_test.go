package memory

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// mockEstimator is a simple mock for testing.
type mockEstimator struct {
	counts map[string]int
}

func newMockEstimator() *mockEstimator {
	return &mockEstimator{
		counts: make(map[string]int),
	}
}

func (e *mockEstimator) Estimate(text string) int {
	if e.counts != nil {
		if c, ok := e.counts[text]; ok {
			return c
		}
	}
	// Default: 1.5 chars ≈ 1 token
	return (len([]rune(text))*2)/3 + 1
}

func (e *mockEstimator) EstimateMessages(msgs []llm.Message) int {
	total := 0
	for _, msg := range msgs {
		total += 10 // overhead
		for _, block := range msg.GetContentBlocks() { if tb, ok := block.(sharedmemory.TextBlock); ok { total += e.Estimate(tb.Text); break } }
	}
	return total
}

// TestTrimMessagesByTokens_Normal tests normal trimming case.
func TestTrimMessagesByTokens_Normal(t *testing.T) {
	estimator := newMockEstimator()
	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "msg1"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "msg2"}}},
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "msg3"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "msg4"}}},
	}

	// maxTokens low enough to trigger trimming
	result := TrimMessagesByTokens(messages, 30, estimator)

	if len(result.Messages) >= len(messages) {
		t.Errorf("Expected trimmed messages, got len=%d", len(result.Messages))
	}
	if result.TotalTokens > 30 {
		t.Errorf("Expected totalTokens <= 30 after trim, got %d", result.TotalTokens)
	}
}

// TestTrimMessagesByTokens_NoTrim tests case where no trimming needed.
func TestTrimMessagesByTokens_NoTrim(t *testing.T) {
	estimator := newMockEstimator()
	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "short"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "reply"}}},
	}

	// maxTokens high enough, no trimming
	result := TrimMessagesByTokens(messages, 1000, estimator)

	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages (no trim), got %d", len(result.Messages))
	}
	if result.CutoffIndex != 0 {
		t.Errorf("Expected cutoffIndex=0 (no trim), got %d", result.CutoffIndex)
	}
}

// TestTrimMessagesByTokens_SingleExceeds tests single message exceeding limit.
func TestTrimMessagesByTokens_SingleExceeds(t *testing.T) {
	estimator := newMockEstimator()
	estimator.counts["verylongmessage"] = 1000

	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "verylongmessage"}}},
	}

	// maxTokens less than single message
	result := TrimMessagesByTokens(messages, 100, estimator)

	// Should keep the last message (avoid complete truncation)
	if len(result.Messages) != 1 {
		t.Errorf("Expected 1 message (kept last), got %d", len(result.Messages))
	}
	if extractTextFromBlocks(result.Messages[0].GetContentBlocks()) != "verylongmessage" {
		t.Errorf("Expected 'verylongmessage', got '%s'", extractTextFromBlocks(result.Messages[0].GetContentBlocks()))
	}
}

// TestTrimMessagesByTokens_AssistantBoundary tests assistant message at cutoff.
func TestTrimMessagesByTokens_AssistantBoundary(t *testing.T) {
	estimator := newMockEstimator()

	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "u1"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "a1"}}},
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "u2"}}},
		{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "a2"}}}, // This might be at cutoff
	}

	// Trigger trimming that might cut at assistant
	result := TrimMessagesByTokens(messages, 25, estimator)

	// Verify assistant handling: should not start with assistant
	if len(result.Messages) > 0 && result.Messages[0].Role == "assistant" {
		t.Errorf("First message should not be assistant after trim")
	}
}

// TestTrimMessagesByTokens_EmptyMessages tests empty message list.
func TestTrimMessagesByTokens_EmptyMessages(t *testing.T) {
	estimator := newMockEstimator()
	messages := []llm.Message{}

	result := TrimMessagesByTokens(messages, 100, estimator)

	if len(result.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(result.Messages))
	}
}

// TestTrimMessagesByTokens_NilEstimator tests nil tokenizer.
func TestTrimMessagesByTokens_NilEstimator(t *testing.T) {
	messages := []llm.Message{
		{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "test"}}},
	}

	result := TrimMessagesByTokens(messages, 100, nil)

	if len(result.Messages) != 1 {
		t.Errorf("Expected 1 message (nil estimator returns unchanged), got %d", len(result.Messages))
	}
	if result.TotalTokens != 0 {
		t.Errorf("Expected 0 tokens (nil estimator), got %d", result.TotalTokens)
	}
}
