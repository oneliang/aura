// Package memory provides token-based message trimming utilities.
package memory

import (
	"github.com/oneliang/aura/core/pkg/llm"
)

// TrimResult holds the result of token-based message trimming.
type TrimResult struct {
	Messages    []llm.Message
	TotalTokens int
	CutoffIndex int
}

// TrimMessagesByTokens trims messages to fit within maxTokens using reverse traversal.
// This function provides unified trimming logic with proper edge case handling:
// - Single message exceeding limit: keeps the last message (avoids complete truncation)
// - Assistant message at cutoff boundary: searches backward for corresponding user message
// - Code block integrity: delegates to adjustForCodeBlocks
//
// Parameters:
// - messages: the message list to trim
// - maxTokens: maximum token limit
// - tokenizer: token estimator for counting
//
// Returns TrimResult with trimmed messages and updated token count.
func TrimMessagesByTokens(messages []llm.Message, maxTokens int, tokenizer TokenEstimator) TrimResult {
	if tokenizer == nil || len(messages) == 0 {
		return TrimResult{
			Messages:    messages,
			TotalTokens: 0,
			CutoffIndex: 0,
		}
	}

	totalTokens := tokenizer.EstimateMessages(messages)
	if totalTokens <= maxTokens {
		return TrimResult{
			Messages:    messages,
			TotalTokens: totalTokens,
			CutoffIndex: 0,
		}
	}

	// Target 95% of maxTokens to leave buffer for new messages
	targetTokens := int(float64(maxTokens) * 0.95)

	// Reverse traversal: accumulate tokens from newest messages
	runningTotal := 0
	cutoffIdx := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := tokenizer.EstimateMessages([]llm.Message{messages[i]})
		runningTotal += msgTokens

		if runningTotal > targetTokens {
			cutoffIdx = i + 1
			break
		}
	}

	// Edge case: single message exceeds limit
	// Keep the last message to avoid complete truncation
	if cutoffIdx >= len(messages) {
		cutoffIdx = len(messages) - 1
		if cutoffIdx < 0 {
			cutoffIdx = 0
		}
	}

	// Assistant message handling: search backward for corresponding user message
	if cutoffIdx > 0 && cutoffIdx < len(messages) && messages[cutoffIdx].Role == "assistant" {
		for i := cutoffIdx - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				cutoffIdx = i
				break
			}
		}
	}

	// Code block boundary adjustment
	cutoffIdx = adjustForCodeBlocks(messages, cutoffIdx)

	// Apply trimming
	var trimmedMessages []llm.Message
	if cutoffIdx > 0 && cutoffIdx <= len(messages) {
		trimmedMessages = messages[cutoffIdx:]
	} else {
		trimmedMessages = messages
	}

	newTotalTokens := tokenizer.EstimateMessages(trimmedMessages)

	return TrimResult{
		Messages:    trimmedMessages,
		TotalTokens: newTotalTokens,
		CutoffIndex: cutoffIdx,
	}
}
