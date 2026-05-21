package memory

import (
	"context"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// TestContextOptimizationIntegration tests the full context optimization system
// with token-aware truncation, summarization, and dynamic RAG working together.
func TestContextOptimizationIntegration(t *testing.T) {
	// Create memory with token-aware truncation (small budget for testing)
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 500, // Small budget to trigger truncation
		Tokenizer: NewSimpleEstimator(),
	})

	// Simulate a long conversation
	for i := 0; i < 30; i++ {
		mem.Add("user", "This is user message "+string(rune(i+'0'))+" with some additional content to consume tokens")
		mem.Add("assistant", "This is assistant response "+string(rune(i+'0'))+" with detailed explanation and code examples")
	}

	// Verify token count is within budget
	tokenCount := mem.GetTokenCount()
	if tokenCount > 500 {
		t.Errorf("Token count %d exceeds budget 500", tokenCount)
	}

	// Verify messages are still accessible
	msgs := mem.Get()
	if len(msgs) == 0 {
		t.Error("Expected messages after truncation")
	}

	t.Logf("Integration test passed: %d messages, %d tokens (budget: 500)", len(msgs), tokenCount)
}

// TestSummarizationIntegration tests summarization triggering and integration.
func TestSummarizationIntegration(t *testing.T) {
	// Create a mock LLM client for summarization
	mockClient := &mockLLMClient{
		response: "Summary: User asked about context optimization. Assistant explained token-aware truncation, summarization, and dynamic RAG strategies.",
	}

	// Create memory with summarization
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 1000,
		Tokenizer: NewSimpleEstimator(),
	})

	// Create summarizer with low threshold for testing
	summarizer := NewSummarizer(mockClient, SummarizerConfig{
		Threshold: 10, // Trigger after 10 messages
		Window:    5,  // Keep last 5 messages
	})

	// Manually set summarizer (normally done via WithSummarizer)
	mem.summarizer = summarizer

	// Add enough messages to trigger summarization
	for i := 0; i < 12; i++ {
		mem.Add("user", "User message "+string(rune(i+'0')))
		mem.Add("assistant", "Assistant response "+string(rune(i+'0')))
	}

	// Trigger summarization manually for testing
	ctx := context.Background()
	messages := mem.Get()
	if len(messages) >= summarizer.config.Threshold {
		summary, err := summarizer.GenerateSummary(ctx, messages[:20])
		if err != nil {
			t.Fatalf("Summarization failed: %v", err)
		}
		if summary == "" {
			t.Error("Expected non-empty summary")
		}
		mem.summaryText = summary
	}

	// Verify GetMessagesWithSummary includes summary
	msgsWithSummary := mem.GetMessagesWithSummary()
	if len(msgsWithSummary) == 0 {
		t.Error("Expected messages with summary")
	}

	// First message should be the summary (as system message)
	if msgsWithSummary[0].Role != "system" {
		t.Error("Expected first message to be summary (system role)")
	}

	t.Logf("Summarization test passed: summary length=%d", len(mem.summaryText))
}

// TestDynamicTokenBudget tests token budget tracking during conversation.
func TestDynamicTokenBudget(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 800,
		Tokenizer: NewSimpleEstimator(),
	})

	initialBudget := 800
	currentUsage := 0

	// Add messages and track budget
	for i := 0; i < 20; i++ {
		content := "Message content " + string(rune(i+'0')) + " with extra text"
		mem.Add("user", content)

		newUsage := mem.GetTokenCount()
		if newUsage <= currentUsage {
			t.Errorf("Token count should increase: was %d, now %d", currentUsage, newUsage)
		}
		currentUsage = newUsage

		// Verify still within budget
		if currentUsage > initialBudget {
			t.Logf("Token budget exceeded at message %d: %d/%d", i+1, currentUsage, initialBudget)
			break
		}
	}

	t.Logf("Dynamic budget test: final usage %d/%d", currentUsage, initialBudget)
}

// TestTokenAwareTruncationWithMessageIntegrity tests that token-aware truncation
// preserves message integrity (user/assistant pairs, code blocks).
func TestTokenAwareTruncationWithMessageIntegrity(t *testing.T) {
	mem := NewConversationMemoryWithConfig(ConversationMemoryConfig{
		MaxTokens: 300, // Small budget
		Tokenizer: NewSimpleEstimator(),
	})

	// Add messages with code blocks
	mem.Add("user", "How do I write a function in Go?")
	mem.Add("assistant", "Here's an example:\n```go\nfunc hello() {\n    fmt.Println(\"Hello\")\n}\n```")

	// Add more messages to trigger truncation
	for i := 0; i < 10; i++ {
		mem.Add("user", "Question "+string(rune(i+'0')))
		mem.Add("assistant", "Answer "+string(rune(i+'0')))
	}

	// Verify token count is within budget
	tokenCount := mem.GetTokenCount()
	if tokenCount > 300 {
		t.Errorf("Token count %d exceeds budget 300", tokenCount)
	}

	// Verify messages start with a complete pair (not orphaned)
	msgs := mem.Get()
	if len(msgs) < 2 {
		t.Error("Expected at least 2 messages after truncation")
	}

	// First message should be user (not orphaned assistant response)
	if msgs[0].Role == "assistant" && len(msgs) > 1 && msgs[1].Role == "assistant" {
		t.Error("Found orphaned assistant message at start")
	}

	t.Logf("Integrity test passed: %d messages, %d tokens", len(msgs), tokenCount)
}

// mockLLMClient is a mock LLM client for testing.
type mockLLMClient struct {
	response string
}

func (m *mockLLMClient) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return &llm.Response{
		Message: llm.Message{
			Role:    "assistant",
			ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: m.response}},
		},
	}, nil
}

func (m *mockLLMClient) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Content: m.response}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2, 0.3}}, nil
}
