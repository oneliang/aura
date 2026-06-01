// Package memory provides session-based memory implementation.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	"github.com/oneliang/aura/storage/pkg/message"
)

// defaultMaxMessages is the default max messages to load from store.
const defaultMaxMessages = 1000

// MessageSource represents the source of a message for persistence.
type MessageSource string

const (
	SourceCLI    MessageSource = "cli"
	SourceTUI    MessageSource = "tui"
	SourceAPI    MessageSource = "api"
	SourceWeb    MessageSource = "web"
	SourceFeishu MessageSource = "feishu"
)

// SessionMemory implements the agent.Memory interface with JSONL persistence.
type SessionMemory struct {
	*BaseMemory
	sessionID           string
	userID              string
	source              MessageSource
	store               *jsonl.MessageStore
	hookEngine          *hooks.Engine
	selectiveCompressor *SelectiveCompressor
	persistWg           sync.WaitGroup // WaitGroup for pending persistence operations
}

// SetHookEngine sets the hooks engine for this memory instance.
func (m *SessionMemory) SetHookEngine(hookEngine *hooks.Engine) {
	m.hookEngine = hookEngine
}

// SessionMemoryConfig holds configuration for SessionMemory.
type SessionMemoryConfig struct {
	MaxLen              int
	MaxTokens           int
	Tokenizer           TokenEstimator
	Summarizer          *Summarizer
	ArchiveOriginal     bool
	Source              MessageSource
	SelectiveCompressor *SelectiveCompressor
}

// NewSessionMemoryWithConfig creates a new session memory instance with advanced configuration.
func NewSessionMemoryWithConfig(sessionID, userID string, store *jsonl.MessageStore, cfg SessionMemoryConfig) (*SessionMemory, error) {
	m := &SessionMemory{
		BaseMemory: NewBaseMemory(BaseMemoryConfig{
			MaxLen:          cfg.MaxLen,
			MaxTokens:       cfg.MaxTokens,
			Tokenizer:       cfg.Tokenizer,
			Summarizer:      cfg.Summarizer,
			ArchiveOriginal: cfg.ArchiveOriginal,
		}),
		sessionID:           sessionID,
		userID:              userID,
		source:              cfg.Source,
		store:               store,
		selectiveCompressor: cfg.SelectiveCompressor,
	}

	if err := m.loadFromStore(); err != nil {
		return nil, errors.New(fmt.Sprintf(i18n.T("error.memory.load_failed"), err))
	}

	return m, nil
}

// AddWithType adds a message to memory and persists it based on type.
// All messages are added to memory for LLM context, but only conversation
// messages (User/Assistant/System) are persisted to session storage.
func (m *SessionMemory) AddWithType(role, content string, msgType sharedmemory.MessageType) {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

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

	// Persist to storage based on type (async with WaitGroup for shutdown sync)
	if m.store != nil && shouldPersistByType(msgType) {
		m.persistWg.Add(1)
		go func() {
			defer m.persistWg.Done()
			m.persistWithType(role, content, msgType)
		}()
	}
}

// shouldPersistByType determines if a message type should be persisted.
// Persists conversation messages (user/assistant/system) and tool execution records (action/observation).
func shouldPersistByType(msgType sharedmemory.MessageType) bool {
	switch msgType {
	case sharedmemory.MessageTypeUser, sharedmemory.MessageTypeAssistant, sharedmemory.MessageTypeSystem:
		return true
	case sharedmemory.MessageTypeAction:     // Tool call records
		return true
	case sharedmemory.MessageTypeObservation: // Tool result records
		return true
	default:
		return false
	}
}

// persistWithType persists a message to JSONL storage with the given type.
func (m *SessionMemory) persistWithType(role, content string, msgType sharedmemory.MessageType) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessionMsg := message.Message{
		SessionID: m.sessionID,
		Type:      msgType,
		Role:      role,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
		Timestamp: time.Now().UnixMilli(),
		Source:    string(m.source),
		UserID:    m.userID,
	}
	if err := m.store.Append(ctx, &sessionMsg); err != nil {
		logger.Default().Warn().Err(err).Str("sessionID", m.sessionID).Msg("Failed to persist session message")
	}
}

// AddWithParts adds a multi-modal message to memory.
// Only text parts are persisted to session storage.
func (m *SessionMemory) AddWithParts(role string, parts []sharedmemory.MessagePart, msgType sharedmemory.MessageType) {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	// Build ContentBlocks from text parts
	var textContent strings.Builder
	var blocks []sharedmemory.ContentBlock
	for _, part := range parts {
		if part.Type == "text" {
			textContent.WriteString(part.Text)
			blocks = append(blocks, sharedmemory.TextBlock{
				Type: sharedmemory.BlockTypeText,
				Text: part.Text,
			})
		}
	}

	msg := llm.Message{
		Role:          role,
		Parts:         parts,
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

// AddWithBlocks adds a message with content blocks to memory and persists to storage.
// This overrides BaseMemory.AddWithBlocks to add persistence functionality.
func (m *SessionMemory) AddWithBlocks(role string, blocks []sharedmemory.ContentBlock, msgType sharedmemory.MessageType) {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

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

	// Persist to storage based on type (async with WaitGroup for shutdown sync)
	if m.store != nil && shouldPersistByType(msgType) {
		m.persistWg.Add(1)
		go func() {
			defer m.persistWg.Done()
			m.persistWithBlocks(role, blocks, msgType)
		}()
	}
}

// persistWithBlocks persists a message with content blocks to JSONL storage.
func (m *SessionMemory) persistWithBlocks(role string, blocks []sharedmemory.ContentBlock, msgType sharedmemory.MessageType) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessionMsg := message.Message{
		SessionID:     m.sessionID,
		Type:          msgType,
		Role:          role,
		ContentBlocks: blocks,
		Timestamp:     time.Now().UnixMilli(),
		Source:        string(m.source),
		UserID:        m.userID,
	}

	if err := m.store.Append(ctx, &sessionMsg); err != nil {
		logger.Default().Warn().Err(err).Str("sessionID", m.sessionID).Msg("Failed to persist session message with blocks")
	}
}

// Shutdown waits for all pending persistence operations to complete.
// This should be called before program exit to ensure all messages are persisted.
func (m *SessionMemory) Shutdown() {
	m.persistWg.Wait()
}

// Clear clears all messages from memory and truncates the JSONL file.
func (m *SessionMemory) Clear() {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	m.messages = make([]llm.Message, 0)
	m.totalTokens = 0

	if m.store != nil {
		if err := m.store.TruncateSession(m.sessionID); err != nil {
			logger.Default().Warn().Err(err).Str("sessionID", m.sessionID).Msg("Failed to truncate session file")
		}
	}
}

// loadFromStore loads messages from the JSONL file into cache.
func (m *SessionMemory) loadFromStore() error {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	if m.store == nil {
		m.messages = make([]llm.Message, 0)
		m.totalTokens = 0
		return nil
	}

	ctx := context.Background()
	limit := m.maxLen
	if limit <= 0 {
		limit = defaultMaxMessages
	}
	messages, err := m.store.Get(ctx, m.sessionID, limit, m.userID)
	if err != nil {
		return err
	}

	m.messages = make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		// Skip compact boundary messages - they're metadata markers only
		if msg.Type == sharedmemory.MessageTypeCompact {
			continue
		}

		llmMsg := llm.Message{
			Role:          msg.Role,
			ContentBlocks: msg.ContentBlocks,
		}
		m.messages = append(m.messages, llmMsg)
	}

	if m.tokenizer != nil {
		m.totalTokens = m.tokenizer.EstimateMessages(m.messages)
	}

	return nil
}

// MaybeSummarize generates a summary of the conversation if conditions are met.
func (m *SessionMemory) MaybeSummarize(ctx context.Context) error {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	if m.summarizer == nil {
		return errors.New(i18n.T("error.memory.summarizer_not_configured"))
	}

	if len(m.messages) < m.summarizer.config.Threshold {
		return errors.New(fmt.Sprintf(i18n.T("error.memory.not_enough_messages"), len(m.messages), m.summarizer.config.Threshold))
	}

	startIdx := m.lastSummaryAt
	endIdx := len(m.messages) - m.summarizer.config.Window

	if endIdx <= startIdx {
		return errors.New(i18n.T("error.memory.no_new_messages"))
	}

	toSummarize := m.messages[startIdx:endIdx]

	// Fire PreCompact hook (non-blocking)
	m.hookEngine.Fire(ctx, hooks.EventPreCompact, map[string]any{
		"message_count": len(m.messages),
	})

	var newSummary string
	var err error
	if m.summaryText == "" {
		newSummary, err = m.summarizer.GenerateSummary(ctx, toSummarize)
	} else {
		newSummary, err = m.summarizer.GenerateSummaryWithPrevious(ctx, m.summaryText, toSummarize)
	}
	if err != nil {
		return errors.New(fmt.Sprintf(i18n.T("error.memory.summary_failed"), err))
	}

	// Fire PostCompact hook (non-blocking)
	m.hookEngine.Fire(ctx, hooks.EventPostCompact, map[string]any{
		"new_summary": newSummary,
		"old_count":   len(m.messages),
		"new_count":   len(m.messages) - endIdx + startIdx,
	})

	m.summaryText = newSummary
	m.lastSummaryAt = endIdx

	if m.archiveOriginal {
		m.messages = m.messages[endIdx:]
		m.totalTokens = 0
		for _, msg := range m.messages {
			if m.tokenizer != nil {
				// Extract text content from ContentBlocks for token estimation
				for _, block := range msg.GetContentBlocks() {
					if tb, ok := block.(sharedmemory.TextBlock); ok {
						m.totalTokens += m.tokenizer.Estimate(tb.Text)
						break
					}
				}
			}
		}
		if m.tokenizer != nil {
			m.totalTokens += m.tokenizer.Estimate(m.summaryText)
		}
	}

	return nil
}

// MaybeCompact performs selective compression when token threshold is exceeded.
// Returns compression metadata if compression occurred, nil otherwise.
func (m *SessionMemory) MaybeCompact(ctx context.Context) (*CompactMetadata, error) {
	mu := m.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	// Check if compression is needed
	if m.maxTokens <= 0 || m.totalTokens < m.maxTokens {
		return nil, nil // No compression needed
	}

	// Fire PreCompact hook (non-blocking)
	if m.hookEngine != nil {
		m.hookEngine.Fire(ctx, hooks.EventPreCompact, map[string]any{
			"token_count":   m.totalTokens,
			"max_tokens":    m.maxTokens,
			"message_count": len(m.messages),
		})
	}

	// Get compressor from config or create default
	compressor := m.selectiveCompressor
	if compressor == nil {
		compressor = NewSelectiveCompressor(DefaultSelectiveCompressorConfig())
	}

	// Perform compression
	result := compressor.Compress(m.messages)

	// Update memory state
	m.messages = result.Messages
	m.totalTokens = result.PostTokens

	// Persist CompactBoundary to storage
	m.persistCompactBoundary(ctx, result.PreTokens, result.PostTokens)

	// Fire PostCompact hook (non-blocking)
	if m.hookEngine != nil {
		m.hookEngine.Fire(ctx, hooks.EventPostCompact, map[string]any{
			"pre_tokens":    result.PreTokens,
			"post_tokens":   result.PostTokens,
			"message_count": len(result.Messages),
			"strategy":      "selective",
		})
	}

	// Return metadata
	return &CompactMetadata{
		Trigger:    "token_limit",
		PreTokens:  result.PreTokens,
		PostTokens: result.PostTokens,
		Strategy:   "selective",
	}, nil
}

// persistCompactBoundary persists the compression boundary marker to JSONL storage.
func (m *SessionMemory) persistCompactBoundary(ctx context.Context, preTokens, postTokens int) {
	if m.store == nil {
		return
	}

	boundaryMsg := message.Message{
		SessionID: m.sessionID,
		Type:      sharedmemory.MessageTypeCompact,
		Role:      sharedmemory.RoleSystem,
		Subtype:   "compact_boundary",
		Timestamp: time.Now().UnixMilli(),
		UserID:    m.userID,
		Source:    string(m.source),
		CompactMetadata: message.CompactMetadata{
			CompactedAt:     time.Now().UnixMilli(),
			CompactionRatio: float64(postTokens) / float64(preTokens),
			Trigger:         "token_limit",
			PreTokens:       preTokens,
			PostTokens:      postTokens,
			Strategy:        "selective",
		},
	}

	if err := m.store.Append(ctx, &boundaryMsg); err != nil {
		logger.Default().Warn().Err(err).Str("sessionID", m.sessionID).Msg("Failed to persist compact boundary")
	}
}
