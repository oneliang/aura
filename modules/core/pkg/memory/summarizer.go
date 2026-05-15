// Package memory provides conversation summarization for context optimization.
package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/i18n"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// getSummarizationPrompt returns the summarization prompt, with i18n support.
func getSummarizationPrompt() string {
	prompt := i18n.T("memory.summarization_prompt")
	if prompt == "memory.summarization_prompt" || prompt == "" {
		// Fallback to English
		return `Please summarize the following conversation, preserving:
1. Key user instructions and preferences
2. Important code snippets or file paths
3. Completed tasks and conclusions
4. Pending action items

Provide a concise summary in about 200 words or less. Output only the summary, no additional commentary.

Conversation:
%s

Summary:`
	}
	return prompt
}

// SummarizerConfig holds configuration for the summarizer.
type SummarizerConfig struct {
	Threshold   int     // Trigger summary when message count exceeds this
	Window      int     // Number of recent messages to summarize
	MaxTokens   int     // Maximum tokens for the summary
	Temperature float64 // Temperature for summarization
}

// DefaultSummarizerConfig returns the default summarizer configuration.
func DefaultSummarizerConfig() SummarizerConfig {
	return SummarizerConfig{
		Threshold:   20, // Use a reasonable default for message count
		Window:      constants.DefaultSummaryWindow,
		MaxTokens:   constants.DefaultSummaryMaxTokens,
		Temperature: constants.DefaultSummaryTemperature,
	}
}

// Summarizer generates summaries of conversation history.
type Summarizer struct {
	client llm.Client
	config SummarizerConfig
}

// NewSummarizer creates a new Summarizer instance.
func NewSummarizer(client llm.Client, config SummarizerConfig) *Summarizer {
	if config.Threshold <= 0 {
		config.Threshold = DefaultSummarizerConfig().Threshold
	}
	if config.Window <= 0 {
		config.Window = DefaultSummarizerConfig().Window
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = DefaultSummarizerConfig().MaxTokens
	}

	return &Summarizer{
		client: client,
		config: config,
	}
}

// ShouldSummarize checks if summarization should be triggered.
func (s *Summarizer) ShouldSummarize(messageCount int) bool {
	return messageCount >= s.config.Threshold
}

// GenerateSummary generates a summary of the given messages.
func (s *Summarizer) GenerateSummary(ctx context.Context, messages []llm.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Build conversation text for summarization
	var sb strings.Builder
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "Aura"
		}
		// Extract text from ContentBlocks
		var textContent string
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, textContent))
	}

	// Generate summary using LLM
	prompt := fmt.Sprintf(getSummarizationPrompt(), sb.String())

	resp, err := s.client.Complete(ctx, &llm.Request{
		Model: s.getModel(),
		Messages: []llm.Message{
			{
				Role:          "user",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: prompt},
				},
			},
		},
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	})
	if err != nil {
		return "", fmt.Errorf("generate summary: %w", err)
	}

	// Extract text from response ContentBlocks
	var result string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			result = tb.Text
			break
		}
	}
	return strings.TrimSpace(result), nil
}

// GenerateSummaryWithPrevious combines previous summary with new messages.
func (s *Summarizer) GenerateSummaryWithPrevious(
	ctx context.Context,
	previousSummary string,
	newMessages []llm.Message,
) (string, error) {
	if len(newMessages) == 0 && previousSummary == "" {
		return "", nil
	}

	if len(newMessages) == 0 {
		return previousSummary, nil
	}

	// Build conversation text
	var sb strings.Builder
	if previousSummary != "" {
		sb.WriteString("Previous summary:\n")
		sb.WriteString(previousSummary)
		sb.WriteString("\n\nNew conversation:\n")
	}

	for _, msg := range newMessages {
		role := msg.Role
		if role == "assistant" {
			role = "Aura"
		}
		// Extract text from ContentBlocks
		var textContent string
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, textContent))
	}

	// Generate updated summary
	prompt := fmt.Sprintf(getSummarizationPrompt(), sb.String())

	resp, err := s.client.Complete(ctx, &llm.Request{
		Model: s.getModel(),
		Messages: []llm.Message{
			{
				Role:          "user",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: prompt},
				},
			},
		},
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	})
	if err != nil {
		return "", fmt.Errorf("generate updated summary: %w", err)
	}

	// Extract text from response ContentBlocks
	var result string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			result = tb.Text
			break
		}
	}
	return strings.TrimSpace(result), nil
}

// getModel returns the model to use for summarization.
// Uses the client's current model.
func (s *Summarizer) getModel() string {
	// Try to get model from client if available
	// For now, use a reasonable default
	return "qwen3:8b"
}

// GetConfig returns the summarizer configuration.
func (s *Summarizer) GetConfig() SummarizerConfig {
	return s.config
}
