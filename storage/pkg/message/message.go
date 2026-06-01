// Package message provides the core message data structure for storage.
package message

import (
	"encoding/json"

	"github.com/oneliang/aura/shared/pkg/memory"
)

// Usage represents token usage statistics for a message.
type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

// CompactMetadata contains metadata about message compression.
type CompactMetadata struct {
	CompactedAt     int64   `json:"compacted_at,omitempty"`     // Unix timestamp when compacted
	CompactionRatio float64 `json:"compaction_ratio,omitempty"` // Ratio of original to compacted size
	Summary         string  `json:"summary,omitempty"`          // Summary of compacted content
	Trigger         string  `json:"trigger,omitempty"`          // Compression trigger: "token_limit", "manual", "auto"
	PreTokens       int     `json:"pre_tokens,omitempty"`       // Token count before compression
	PostTokens      int     `json:"post_tokens,omitempty"`      // Token count after compression
	Strategy        string  `json:"strategy,omitempty"`         // Compression strategy: "selective", "full_summarize"
}

// Message represents a single message in a session.
type Message struct {
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id,omitempty"` // User ID for multi-user isolation
	Type          memory.MessageType     `json:"type,omitempty"`    // Message type for filtering
	Role          string                 `json:"role"`              // user/assistant/system
	ContentBlocks []memory.ContentBlock  `json:"content_blocks"`    // Content blocks for structured content
	Timestamp     int64                  `json:"timestamp"`         // Unix timestamp in milliseconds
	Source        string                 `json:"source,omitempty"`  // cli/feishu/email/api

	// Metadata fields
	UUID        string `json:"uuid,omitempty"`
	ParentUUID  string `json:"parent_uuid,omitempty"`
	Subtype     string `json:"subtype,omitempty"`
	IsSidechain bool   `json:"is_sidechain,omitempty"`
	CWD         string `json:"cwd,omitempty"`
	GitBranch   string `json:"git_branch,omitempty"`

	// Token usage
	Usage Usage `json:"usage,omitempty"`

	// Compaction metadata
	CompactMetadata CompactMetadata `json:"compact_metadata,omitempty"`
}

// rawMessage is used for JSON marshaling/unmarshaling to handle polymorphic ContentBlocks.
type rawMessage struct {
	SessionID       string                   `json:"session_id,omitempty"`
	UserID          string                   `json:"user_id,omitempty"`
	Type            memory.MessageType       `json:"type,omitempty"`
	Role            string                   `json:"role"`
	ContentBlocks   []memory.RawContentBlock `json:"content_blocks,omitempty"`
	Timestamp       int64                    `json:"timestamp"`
	Source          string                   `json:"source,omitempty"`
	UUID            string                   `json:"uuid,omitempty"`
	ParentUUID      string                   `json:"parent_uuid,omitempty"`
	Subtype         string                   `json:"subtype,omitempty"`
	IsSidechain     bool                     `json:"is_sidechain,omitempty"`
	CWD             string                   `json:"cwd,omitempty"`
	GitBranch       string                   `json:"git_branch,omitempty"`
	Usage           Usage                    `json:"usage,omitempty"`
	CompactMetadata CompactMetadata          `json:"compact_metadata,omitempty"`
}

// GetContentBlocks returns the content blocks for this message.
func (m *Message) GetContentBlocks() []memory.ContentBlock {
	return m.ContentBlocks
}

// SetContentBlocks sets the content blocks for this message.
func (m *Message) SetContentBlocks(blocks []memory.ContentBlock) {
	m.ContentBlocks = blocks
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
// Required to handle polymorphic ContentBlock interface types.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw rawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.SessionID = raw.SessionID
	m.UserID = raw.UserID
	m.Type = raw.Type
	m.Role = raw.Role
	m.Timestamp = raw.Timestamp
	m.Source = raw.Source
	m.UUID = raw.UUID
	m.ParentUUID = raw.ParentUUID
	m.Subtype = raw.Subtype
	m.IsSidechain = raw.IsSidechain
	m.CWD = raw.CWD
	m.GitBranch = raw.GitBranch
	m.Usage = raw.Usage
	m.CompactMetadata = raw.CompactMetadata

	// Convert RawContentBlocks to ContentBlocks
	if len(raw.ContentBlocks) > 0 {
		blocks := make([]memory.ContentBlock, 0, len(raw.ContentBlocks))
		for _, rc := range raw.ContentBlocks {
			cb, err := rc.ToContentBlock()
			if err != nil {
				return err
			}
			blocks = append(blocks, cb)
		}
		m.ContentBlocks = blocks
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Message.
// Required to handle polymorphic ContentBlock interface types.
func (m Message) MarshalJSON() ([]byte, error) {
	raw := rawMessage{
		SessionID:       m.SessionID,
		UserID:          m.UserID,
		Type:            m.Type,
		Role:            m.Role,
		Timestamp:       m.Timestamp,
		Source:          m.Source,
		UUID:            m.UUID,
		ParentUUID:      m.ParentUUID,
		Subtype:         m.Subtype,
		IsSidechain:     m.IsSidechain,
		CWD:             m.CWD,
		GitBranch:       m.GitBranch,
		Usage:           m.Usage,
		CompactMetadata: m.CompactMetadata,
	}

	// Convert ContentBlocks to RawContentBlocks
	if m.ContentBlocks != nil {
		raw.ContentBlocks = memory.ContentBlocksToRaw(m.ContentBlocks)
	}

	return json.Marshal(raw)
}