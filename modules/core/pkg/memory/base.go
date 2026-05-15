// Package memory provides base memory functionality.
package memory

import (
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// Verify BaseMemory implements the shared memory interfaces.
var (
	_ sharedmemory.Memory              = (*BaseMemory)(nil)
	_ sharedmemory.SummarizingMemory   = (*BaseMemory)(nil)
	_ sharedmemory.TokenCountingMemory = (*BaseMemory)(nil)
	_ sharedmemory.RecentMemory        = (*BaseMemory)(nil)
)

// BaseMemory provides common memory management functionality.
// It can be embedded in other memory types to avoid code duplication.
type BaseMemory struct {
	mu              sync.RWMutex
	messages        []llm.Message
	totalTokens     int
	maxLen          int
	maxTokens       int
	tokenizer       TokenEstimator
	summarizer      *Summarizer
	summaryText     string
	lastSummaryAt   int
	archiveOriginal bool
	lastActiveTime  time.Time // For staleness detection
}

// BaseMemoryConfig holds configuration for BaseMemory.
type BaseMemoryConfig struct {
	MaxLen          int
	MaxTokens       int
	Tokenizer       TokenEstimator
	Summarizer      *Summarizer
	ArchiveOriginal bool
	InitialMessages []llm.Message
}

// NewBaseMemory creates a new BaseMemory instance.
func NewBaseMemory(cfg BaseMemoryConfig) *BaseMemory {
	if cfg.MaxLen <= 0 && cfg.MaxTokens <= 0 {
		cfg.MaxLen = 50
	}

	if cfg.MaxTokens > 0 && cfg.Tokenizer == nil {
		cfg.Tokenizer = NewSimpleEstimator()
	}

	m := &BaseMemory{
		messages:        make([]llm.Message, 0),
		maxLen:          cfg.MaxLen,
		maxTokens:       cfg.MaxTokens,
		tokenizer:       cfg.Tokenizer,
		summarizer:      cfg.Summarizer,
		archiveOriginal: cfg.ArchiveOriginal,
		lastSummaryAt:   0,
		lastActiveTime:  time.Now(),
	}

	if len(cfg.InitialMessages) > 0 {
		m.messages = cfg.InitialMessages
		if m.tokenizer != nil {
			m.totalTokens = m.tokenizer.EstimateMessages(m.messages)
		}
	}

	return m
}

// Add adds a message to the memory.
func (m *BaseMemory) Add(role, content string) {
	m.AddWithType(role, content, sharedmemory.MessageTypeUser)
}

// AddWithType adds a message to the memory with a specific type.
// BaseMemory doesn't filter by type - all messages are added.
func (m *BaseMemory) AddWithType(role, content string, msgType sharedmemory.MessageType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateLastActiveLocked()

	msg := llm.Message{
		Role:          role,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
	}

	msgTokens := 0
	if m.tokenizer != nil {
		msgTokens = m.tokenizer.EstimateMessages([]llm.Message{msg})
	}

	m.messages = append(m.messages, msg)
	m.totalTokens += msgTokens

	if m.maxTokens > 0 && m.tokenizer != nil {
		m.trimByTokens()
	} else if m.maxLen > 0 {
		m.trimByCount()
	}
}

// AddWithParts adds a multi-modal message to the memory.
func (m *BaseMemory) AddWithParts(role string, parts []sharedmemory.MessagePart, msgType sharedmemory.MessageType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateLastActiveLocked()

	msg := llm.Message{
		Role:  role,
		Parts: parts,
	}

	// Build ContentBlocks from text parts
	var textContent strings.Builder
	for _, part := range parts {
		if part.Type == "text" {
			textContent.WriteString(part.Text)
		}
	}
	if textContent.Len() > 0 {
		msg.ContentBlocks = []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: textContent.String()},
		}
	}

	msgTokens := 0
	if m.tokenizer != nil {
		msgTokens = m.tokenizer.EstimateMessages([]llm.Message{msg})
	}

	m.messages = append(m.messages, msg)
	m.totalTokens += msgTokens

	if m.maxTokens > 0 && m.tokenizer != nil {
		m.trimByTokens()
	} else if m.maxLen > 0 {
		m.trimByCount()
	}
}

// AddWithBlocks adds a message with content blocks to the memory.
func (m *BaseMemory) AddWithBlocks(role string, blocks []sharedmemory.ContentBlock, msgType sharedmemory.MessageType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateLastActiveLocked()

	msg := llm.Message{
		Role:          role,
		ContentBlocks: blocks,
	}

	msgTokens := 0
	if m.tokenizer != nil {
		msgTokens = m.tokenizer.EstimateMessages([]llm.Message{msg})
	}

	m.messages = append(m.messages, msg)
	m.totalTokens += msgTokens

	if m.maxTokens > 0 && m.tokenizer != nil {
		m.trimByTokens()
	} else if m.maxLen > 0 {
		m.trimByCount()
	}
}

// AddMessage adds a pre-constructed message to the memory.
func (m *BaseMemory) AddMessage(msg llm.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateLastActiveLocked()

	msgTokens := 0
	if m.tokenizer != nil {
		msgTokens = m.tokenizer.EstimateMessages([]llm.Message{msg})
	}

	m.messages = append(m.messages, msg)
	m.totalTokens += msgTokens

	if m.maxTokens > 0 && m.tokenizer != nil {
		m.trimByTokens()
	} else if m.maxLen > 0 {
		m.trimByCount()
	}
}

// trimByCount trims messages to the last maxLen messages.
// Caller must hold the memory mutex.
func trimByCount(messages []llm.Message, maxLen int, tokenizer TokenEstimator) (int, []llm.Message) {
	if len(messages) <= maxLen {
		return 0, messages
	}
	messages = messages[len(messages)-maxLen:]
	totalTokens := 0
	if tokenizer != nil {
		totalTokens = tokenizer.EstimateMessages(messages)
	}
	return totalTokens, messages
}

// getTextFromMessage extracts text content from a message.
func getTextFromMessage(msg llm.Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// adjustForCodeBlocks ensures we don't cut in the middle of a code block.
func adjustForCodeBlocks(messages []llm.Message, cutoffIdx int) int {
	if cutoffIdx >= len(messages) || cutoffIdx == 0 {
		return cutoffIdx
	}

	lastCompleteIdx := cutoffIdx - 1
	content := getTextFromMessage(messages[lastCompleteIdx])

	codeBlockCount := 0
	inCodeBlock := false
	for i := 0; i < len(content); i++ {
		if i+2 < len(content) && content[i:i+3] == "```" {
			inCodeBlock = !inCodeBlock
			codeBlockCount++
			i += 2
		}
	}

	if codeBlockCount%2 != 0 {
		for i := lastCompleteIdx - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" {
				return i + 1
			}
		}
	}

	return cutoffIdx
}

// trimByCount trims messages based on message count (legacy FIFO).
func (m *BaseMemory) trimByCount() {
	totalTokens, msgs := trimByCount(m.messages, m.maxLen, m.tokenizer)
	// Only update if there's a change (optimization)
	if len(msgs) != len(m.messages) {
		m.messages = msgs
		m.totalTokens = totalTokens
	}
}

// trimByTokens trims messages based on token count with integrity preservation.
func (m *BaseMemory) trimByTokens() {
	result := TrimMessagesByTokens(m.messages, m.maxTokens, m.tokenizer)
	m.messages = result.Messages
	m.totalTokens = result.TotalTokens
}

// adjustForCodeBlocks ensures we don't cut in the middle of a code block.
func (m *BaseMemory) adjustForCodeBlocks(cutoffIdx int) int {
	return adjustForCodeBlocks(m.messages, cutoffIdx)
}

// Get returns all messages.
func (m *BaseMemory) Get() []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]llm.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetTokenCount returns the current total token count.
func (m *BaseMemory) GetTokenCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalTokens
}

// Clear clears all messages.
func (m *BaseMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]llm.Message, 0)
	m.totalTokens = 0
}

// ClearPreserveSummary clears messages but preserves the conversation summary.
// Useful for staleness cleanup where preserving context summary is valuable.
func (m *BaseMemory) ClearPreserveSummary() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]llm.Message, 0)
	m.totalTokens = 0
	// Preserve summaryText - don't clear it
}

// Last returns the last n messages.
func (m *BaseMemory) Last(n int) []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n >= len(m.messages) {
		result := make([]llm.Message, len(m.messages))
		copy(result, m.messages)
		return result
	}

	result := make([]llm.Message, n)
	copy(result, m.messages[len(m.messages)-n:])
	return result
}

// Len returns the number of messages.
func (m *BaseMemory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// LoadMessages loads messages and calculates token count.
// Updates lastActiveTime to prevent newly loaded session from being marked stale.
func (m *BaseMemory) LoadMessages(messages []llm.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = messages
	m.lastActiveTime = time.Now() // Mark as active on load

	if m.tokenizer != nil {
		m.totalTokens = m.tokenizer.EstimateMessages(messages)
	}
}

// SetSummarizer sets the summarizer.
func (m *BaseMemory) SetSummarizer(s *Summarizer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summarizer = s
}

// GetSummary returns the current conversation summary.
func (m *BaseMemory) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.summaryText
}

// ClearSummary clears the conversation summary.
func (m *BaseMemory) ClearSummary() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaryText = ""
	m.lastSummaryAt = 0
}

// GetMessagesWithSummary returns messages with summary prepended as a system message.
func (m *BaseMemory) GetMessagesWithSummary() []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []llm.Message

	if m.summaryText != "" {
		result = append(result, llm.Message{
			Role:          "system",
			ContentBlocks: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Previous conversation summary:\n" + m.summaryText},
			},
		})
	}

	result = append(result, m.messages...)

	return result
}

// GetSummarizer returns the summarizer.
func (m *BaseMemory) GetSummarizer() *Summarizer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.summarizer
}

// GetMessagesRaw returns the internal messages slice (for read-only access).
// Caller must hold the lock.
func (m *BaseMemory) GetMessagesRaw() []llm.Message {
	return m.messages
}

// GetMutex returns the mutex for external locking.
func (m *BaseMemory) GetMutex() *sync.RWMutex {
	return &m.mu
}

// VerifyTokenCount verifies that the cached token count matches actual calculation.
func (m *BaseMemory) VerifyTokenCount() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tokenizer == nil {
		return nil
	}

	calculated := m.tokenizer.EstimateMessages(m.messages)
	if calculated != m.totalTokens {
		return &TokenCountMismatchError{
			Cached:     m.totalTokens,
			Calculated: calculated,
		}
	}
	return nil
}

// UpdateTokenCount recalculates and updates the token count.
func (m *BaseMemory) UpdateTokenCount() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tokenizer != nil {
		m.totalTokens = m.tokenizer.EstimateMessages(m.messages)
	}
}

// IsStale checks if the memory is stale based on a threshold duration.
// A memory is considered stale if it hasn't been updated within the threshold.
func (m *BaseMemory) IsStale(threshold time.Duration) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if threshold <= 0 {
		return false // No threshold means never stale
	}

	return time.Since(m.lastActiveTime) > threshold
}

// GetLastActiveTime returns the last active time of the memory.
func (m *BaseMemory) GetLastActiveTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastActiveTime
}

// UpdateLastActive updates the last active time to current time.
func (m *BaseMemory) UpdateLastActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActiveTime = time.Now()
}

// updateLastActiveLocked updates the last active time (caller must hold lock).
func (m *BaseMemory) updateLastActiveLocked() {
	m.lastActiveTime = time.Now()
}