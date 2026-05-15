// Package memory provides memory implementations for agents.
package memory

import (
	"context"

	"github.com/oneliang/aura/core/pkg/llm"
)

// ConversationMemory is an in-memory conversation store.
type ConversationMemory struct {
	*BaseMemory
}

// ConversationMemoryConfig holds configuration for ConversationMemory.
type ConversationMemoryConfig struct {
	MaxLen          int
	MaxTokens       int
	Tokenizer       TokenEstimator
	Summarizer      *Summarizer
	ArchiveOriginal bool
}

// NewConversationMemoryWithConfig creates a new conversation memory with advanced configuration.
func NewConversationMemoryWithConfig(cfg ConversationMemoryConfig) *ConversationMemory {
	return &ConversationMemory{
		BaseMemory: NewBaseMemory(BaseMemoryConfig{
			MaxLen:          cfg.MaxLen,
			MaxTokens:       cfg.MaxTokens,
			Tokenizer:       cfg.Tokenizer,
			Summarizer:      cfg.Summarizer,
			ArchiveOriginal: cfg.ArchiveOriginal,
		}),
	}
}

// MaybeSummarize checks if summarization should be triggered and generates a summary.
func (m *ConversationMemory) MaybeSummarize(ctx context.Context) error {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	summarizer := m.GetSummarizer()
	if summarizer == nil {
		return nil
	}

	messages := m.GetMessagesRaw()
	if !summarizer.ShouldSummarize(len(messages)) {
		return nil
	}

	summary := m.GetSummary()
	startIdx := m.lastSummaryAt
	if startIdx < 0 {
		startIdx = 0
	}

	var messagesToSummarize []llm.Message
	if startIdx < len(messages) {
		messagesToSummarize = messages[startIdx:]
	}

	if len(messagesToSummarize) == 0 {
		return nil
	}

	summary, err := summarizer.GenerateSummaryWithPrevious(ctx, summary, messagesToSummarize)
	if err != nil {
		return err
	}

	m.summaryText = summary
	m.lastSummaryAt = len(messages)

	if m.archiveOriginal && startIdx > 0 {
		m.messages = m.messages[startIdx:]
		m.lastSummaryAt = 0
		if m.tokenizer != nil {
			m.totalTokens = m.tokenizer.EstimateMessages(m.messages)
		}
	}

	return nil
}

// TokenCountMismatchError is returned when VerifyTokenCount detects a mismatch.
type TokenCountMismatchError struct {
	Cached     int
	Calculated int
}

func (e *TokenCountMismatchError) Error() string {
	return "token count mismatch"
}
