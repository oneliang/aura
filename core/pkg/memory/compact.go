// Package memory provides memory implementations for agents.
package memory

import (
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/i18n"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// SelectiveCompressorConfig holds configuration for the selective compressor.
type SelectiveCompressorConfig struct {
	// RecentFullCount is the number of recent messages to keep fully intact.
	RecentFullCount int
	// RecentToolResults is the number of recent tool results to keep fully intact.
	RecentToolResults int
	// RecentThinking is the number of recent thinking blocks to keep fully intact.
	RecentThinking int
}

// DefaultSelectiveCompressorConfig returns the default configuration for SelectiveCompressor.
func DefaultSelectiveCompressorConfig() SelectiveCompressorConfig {
	return SelectiveCompressorConfig{
		RecentFullCount:   10,
		RecentToolResults: 3,
		RecentThinking:    2,
	}
}

// SelectiveCompressor selectively compresses conversation history.
// It keeps recent messages fully intact while summarizing older content.
type SelectiveCompressor struct {
	config       SelectiveCompressorConfig
	tokenCounter TokenEstimator
}

// NewSelectiveCompressor creates a new SelectiveCompressor with the given configuration.
// Zero values in the config are replaced with defaults.
func NewSelectiveCompressor(cfg SelectiveCompressorConfig) *SelectiveCompressor {
	// Apply defaults for zero values
	if cfg.RecentFullCount <= 0 {
		cfg.RecentFullCount = DefaultSelectiveCompressorConfig().RecentFullCount
	}
	if cfg.RecentToolResults <= 0 {
		cfg.RecentToolResults = DefaultSelectiveCompressorConfig().RecentToolResults
	}
	if cfg.RecentThinking <= 0 {
		cfg.RecentThinking = DefaultSelectiveCompressorConfig().RecentThinking
	}

	return &SelectiveCompressor{
		config:       cfg,
		tokenCounter: NewSimpleEstimator(),
	}
}

// CompressResult contains the result of compression.
type CompressResult struct {
	// Messages is the compressed message list.
	Messages []llm.Message
	// PreTokens is the token count before compression.
	PreTokens int
	// PostTokens is the token count after compression.
	PostTokens int
}

// CompactMetadata contains metadata about the compression.
type CompactMetadata struct {
	// Trigger is what triggered the compression (e.g., "token_limit", "manual").
	Trigger string `json:"trigger"`
	// PreTokens is the token count before compression.
	PreTokens int `json:"pre_tokens"`
	// PostTokens is the token count after compression.
	PostTokens int `json:"post_tokens"`
	// Strategy is the compression strategy used (e.g., "selective").
	Strategy string `json:"strategy"`
}

// CompactBoundaryMessage represents a system message that marks the boundary
// between compacted and non-compacted messages.
type CompactBoundaryMessage struct {
	// Type is always "system".
	Type string `json:"type"`
	// Subtype is always "compact_boundary".
	Subtype string `json:"subtype"`
	// Content is the human-readable description.
	Content string `json:"content"`
	// Metadata contains compression details.
	Metadata CompactMetadata `json:"metadata"`
}

// ToMessage converts the CompactBoundaryMessage to an llm.Message.
func (m CompactBoundaryMessage) ToMessage() llm.Message {
	content := i18n.T("memory.compact.boundary", m.Content, m.Metadata.PreTokens, m.Metadata.PostTokens, m.Metadata.Strategy)

	msg := llm.Message{
		Role:          m.Type,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
	}
	return msg
}

// Compress compresses the given messages using selective compression.
// It keeps recent messages fully intact and summarizes older content.
func (c *SelectiveCompressor) Compress(messages []llm.Message) CompressResult {
	if len(messages) == 0 {
		return CompressResult{
			Messages:   []llm.Message{},
			PreTokens:  0,
			PostTokens: 0,
		}
	}

	// Calculate pre-compression token count
	preTokens := c.tokenCounter.EstimateMessages(messages)

	// If we have fewer messages than RecentFullCount, return as-is
	if len(messages) <= c.config.RecentFullCount {
		return CompressResult{
			Messages:   messages,
			PreTokens:  preTokens,
			PostTokens: preTokens,
		}
	}

	// Split into old and recent messages
	recentStartIdx := len(messages) - c.config.RecentFullCount
	oldMessages := messages[:recentStartIdx]
	recentMessages := messages[recentStartIdx:]

	// Compress old messages
	compressedOld := c.compressOldMessages(oldMessages)

	// Create boundary message
	boundary := CompactBoundaryMessage{
		Type:     "system",
		Subtype:  "compact_boundary",
		Content:  i18n.T("memory.compact.compressed"),
		Metadata: CompactMetadata{Trigger: "token_limit", PreTokens: preTokens, Strategy: "selective"},
	}

	// Combine: boundary message + compressed old + recent
	result := make([]llm.Message, 0, len(compressedOld)+len(recentMessages)+1)
	result = append(result, boundary.ToMessage())
	result = append(result, compressedOld...)
	result = append(result, recentMessages...)

	// Calculate post-compression token count
	postTokens := c.tokenCounter.EstimateMessages(result)

	// Update boundary message with actual post-tokens
	boundary.Metadata.PostTokens = postTokens
	result[0] = boundary.ToMessage()

	return CompressResult{
		Messages:   result,
		PreTokens:  preTokens,
		PostTokens: postTokens,
	}
}

// compressOldMessages compresses old messages by summarizing thinking blocks and tool results.
func (c *SelectiveCompressor) compressOldMessages(messages []llm.Message) []llm.Message {
	result := make([]llm.Message, 0, len(messages))

	for _, msg := range messages {
		blocks := msg.GetContentBlocks()

		// If no blocks, keep message as-is (simple text message)
		if len(blocks) == 0 {
			result = append(result, msg)
			continue
		}

		// Compress blocks
		compressedBlocks := c.compressBlocks(blocks)

		// Create new message with compressed blocks
		newMsg := llm.Message{
			Role: msg.Role,
			Type: msg.Type,
		}
		newMsg.SetContentBlocks(compressedBlocks)
		result = append(result, newMsg)
	}

	return result
}

// compressBlocks compresses content blocks with consideration for recent counts.
// It tracks the count of each block type and keeps recent ones intact.
// - Text blocks: kept unchanged
// - Tool use blocks: kept unchanged
// - Thinking blocks: truncated to ~50 chars (except recent RecentThinking)
// - Tool result blocks: summarized with character count (except recent RecentToolResults)
func (c *SelectiveCompressor) compressBlocks(blocks []sharedmemory.ContentBlock) []sharedmemory.ContentBlock {
	result := make([]sharedmemory.ContentBlock, 0, len(blocks))

	// Track counts for recent blocks (count from the end of the blocks list)
	toolResultCount := 0
	thinkingCount := 0

	// Count recent blocks from the end
	for i := len(blocks) - 1; i >= 0; i-- {
		switch blocks[i].(type) {
		case sharedmemory.ToolResultBlock:
			toolResultCount++
		case sharedmemory.ThinkingBlock:
			thinkingCount++
		}
	}

	// Track how many we've processed (from the end perspective)
	toolResultsSeen := 0
	thinkingSeen := 0

	for _, block := range blocks {
		switch b := block.(type) {
		case sharedmemory.TextBlock:
			// Keep text blocks unchanged
			result = append(result, b)

		case sharedmemory.ThinkingBlock:
			// Check if this is a recent thinking block (should be kept intact)
			// Recent blocks are counted from the end, so we check if we're in the recent range
			thinkingSeen++
			isRecent := (thinkingCount - thinkingSeen) < c.config.RecentThinking

			if isRecent {
				// Keep recent thinking blocks unchanged
				result = append(result, b)
			} else {
				// Truncate old thinking blocks
				compressed := sharedmemory.ThinkingBlock{
					Type:      b.Type,
					Thinking:  truncateThinking(b.Thinking),
					Signature: b.Signature,
				}
				result = append(result, compressed)
			}

		case sharedmemory.ToolUseBlock:
			// Keep tool use blocks unchanged
			result = append(result, b)

		case sharedmemory.ToolResultBlock:
			// Check if this is a recent tool result (should be kept intact)
			toolResultsSeen++
			isRecent := (toolResultCount - toolResultsSeen) < c.config.RecentToolResults

			if isRecent {
				// Keep recent tool results unchanged
				result = append(result, b)
			} else {
				// Summarize old tool result blocks
				summary := c.summarizeToolResult(b)
				result = append(result, sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: summary,
				})
			}

		default:
			// Unknown block type, keep as-is
			result = append(result, block)
		}
	}

	return result
}

// summarizeToolResult creates a summary string for a tool result block.
func (c *SelectiveCompressor) summarizeToolResult(trb sharedmemory.ToolResultBlock) string {
	// Calculate total character count from all content blocks
	charCount := 0
	for _, block := range trb.Content {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			charCount += len(tb.Text)
		}
	}

	// Build summary string
	suffix := ""
	if trb.IsError {
		suffix = " ERROR"
	}

	return i18n.T("memory.compact.tool_result_format", trb.ToolUseID, charCount, suffix)
}

// truncateThinking truncates a thinking string to approximately 50 characters.
// If the string is longer than 50 characters, it's truncated and "..." is appended.
func truncateThinking(s string) string {
	const maxLength = 50
	if len(s) <= maxLength {
		return s
	}
	// Truncate to maxLength and add ellipsis
	return s[:maxLength] + "..."
}