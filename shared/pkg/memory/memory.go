// Package memory provides memory interfaces and types for the Aura system.
// This package is the single source of truth for memory-related interfaces,
// allowing lower-level modules (like storage) to implement them without
// creating circular dependencies.
package memory

import (
	"encoding/json"
)

// Role constants for message roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// MessageType defines the type of a message for filtering and business logic.
type MessageType string

const (
	// Conversation messages (persisted to session storage)
	MessageTypeUser      MessageType = "user"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeSystem    MessageType = "system"

	// Internal reasoning messages (not persisted to session storage)
	MessageTypeThought     MessageType = "thought"
	MessageTypeAction      MessageType = "action"
	MessageTypeObservation MessageType = "observation"

	// Tool execution messages (not persisted by default)
	MessageTypeToolStart  MessageType = "tool_start"
	MessageTypeToolEnd    MessageType = "tool_end"
	MessageTypeToolResult MessageType = "tool_result"

	// Error messages (not persisted)
	MessageTypeError MessageType = "error"

	// Compact messages (summarized/compacted content)
	MessageTypeCompact MessageType = "compact"
)

// Message represents a chat message in the conversation.
type Message struct {
	Role          string         `json:"role"`            // system, user, assistant
	ContentBlocks []ContentBlock `json:"content_blocks"`  // Content blocks for structured content
	Type          MessageType    `json:"type,omitempty"`  // Optional: for internal tracking
	Parts         []MessagePart  `json:"parts,omitempty"` // Optional: for multi-modal messages
}

// rawMessage is used for JSON marshaling/unmarshaling to handle polymorphic ContentBlocks.
type rawMessage struct {
	Role          string            `json:"role"`
	ContentBlocks []RawContentBlock `json:"content_blocks,omitempty"`
	Type          MessageType       `json:"type,omitempty"`
	Parts         []MessagePart     `json:"parts,omitempty"`
}

// GetContentBlocks returns the content blocks for this message.
func (m *Message) GetContentBlocks() []ContentBlock {
	return m.ContentBlocks
}

// SetContentBlocks sets the content blocks for this message.
func (m *Message) SetContentBlocks(blocks []ContentBlock) {
	m.ContentBlocks = blocks
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
// Required to handle polymorphic ContentBlock interface types.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw rawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.Role = raw.Role
	m.Type = raw.Type
	m.Parts = raw.Parts

	// Convert RawContentBlocks to ContentBlocks
	if len(raw.ContentBlocks) > 0 {
		blocks := make([]ContentBlock, 0, len(raw.ContentBlocks))
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
		Role:  m.Role,
		Type:  m.Type,
		Parts: m.Parts,
	}

	// Convert ContentBlocks to RawContentBlocks
	if m.ContentBlocks != nil {
		raw.ContentBlocks = ContentBlocksToRaw(m.ContentBlocks)
	}

	return json.Marshal(raw)
}

// MessagePart represents a part of a multi-modal message.
type MessagePart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL or dataURI.
type ImageURL struct {
	URL string `json:"url"` // Can be a URL or dataURI (data:image/jpeg;base64,...)
}

// Memory defines the interface for conversation memory management.
// Implementations can be in-memory, persistent, or hybrid.
type Memory interface {
	// Add adds a message to the memory.
	Add(role, content string)
	// AddWithType adds a message to the memory with a specific type.
	// The type determines if the message should be persisted to storage.
	AddWithType(role, content string, msgType MessageType)
	// AddWithParts adds a multi-modal message to the memory.
	// Used for messages containing images or other non-text content.
	AddWithParts(role string, parts []MessagePart, msgType MessageType)
	// AddWithBlocks adds a message with structured content blocks.
	// Used for messages with thinking, tool use, or tool result content.
	AddWithBlocks(role string, blocks []ContentBlock, msgType MessageType)
	// Get returns all messages in the memory.
	Get() []Message
	// Clear removes all messages from the memory.
	Clear()
}

// SummarizingMemory extends Memory with conversation summarization support.
type SummarizingMemory interface {
	Memory
	// GetMessagesWithSummary returns messages with summary prepended if available.
	GetMessagesWithSummary() []Message
	// GetSummary returns the current conversation summary text.
	GetSummary() string
	// ClearPreserveSummary clears messages but preserves the conversation summary.
	// Useful for staleness cleanup where preserving context summary is valuable.
	ClearPreserveSummary()
}

// TokenCountingMemory extends Memory with token counting capability.
type TokenCountingMemory interface {
	Memory
	// GetTokenCount returns the current total token count.
	GetTokenCount() int
}

// RecentMemory extends Memory with recent message retrieval.
type RecentMemory interface {
	Memory
	// Last returns the most recent n messages.
	Last(n int) []Message
}