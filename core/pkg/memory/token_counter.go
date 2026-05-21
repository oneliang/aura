// Package memory provides memory implementations for agents.
package memory

import (
	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// TokenEstimator defines the interface for estimating token counts.
type TokenEstimator interface {
	// Estimate estimates the number of tokens in a text string.
	Estimate(text string) int
	// EstimateMessages estimates the total number of tokens in a list of messages.
	EstimateMessages(msgs []llm.Message) int
}

// SimpleEstimator is a lightweight token estimator using an accurate heuristic.
// It uses 1.5 characters ≈ 1 token, which better accounts for Chinese characters
// (1 Hanzi ≈ 1.5 tokens) and provides more accurate token estimation for LLM APIs.
type SimpleEstimator struct {
	// overheadPerMessage is the estimated token overhead per message (role, formatting, etc.)
	overheadPerMessage int
}

// NewSimpleEstimator creates a new SimpleEstimator with default settings.
func NewSimpleEstimator() *SimpleEstimator {
	return &SimpleEstimator{
		overheadPerMessage: 10, // ~10 tokens per message for role and formatting
	}
}

// Estimate estimates the number of tokens in a text string.
// Uses 1.5 characters ≈ 1 token for accurate estimation.
// Formula: (runeCount * 2) / 3 + 1, which is equivalent to runeCount / 1.5 + 1.
func (e *SimpleEstimator) Estimate(text string) int {
	if text == "" {
		return 0
	}
	// Use rune count for proper Unicode support (important for Chinese characters)
	runeCount := len([]rune(text))
	// 1.5 chars ≈ 1 token: (runeCount * 2) / 3 gives us runeCount / 1.5
	return (runeCount*2)/3 + 1
}

// EstimateMessages estimates the total number of tokens in a list of messages.
func (e *SimpleEstimator) EstimateMessages(msgs []llm.Message) int {
	if len(msgs) == 0 {
		return 0
	}

	total := 0
	for _, msg := range msgs {
		// Add per-message overhead
		total += e.overheadPerMessage
		// Extract text content from ContentBlocks
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				total += e.Estimate(tb.Text)
				break
			}
		}
	}
	return total
}

// Ensure SimpleEstimator implements TokenEstimator interface.
var _ TokenEstimator = (*SimpleEstimator)(nil)
