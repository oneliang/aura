package tui

import (
	"time"
)

// MessageStore manages messages.
// Update() is single-threaded (guaranteed by Bubble Tea), so no mutex needed.
// Note: MessageStore is pure data management - rendering is done by view.renderMessage().
type MessageStore struct {
	messages    []*Message
	maxMessages int
	userName    string
}

// NewMessageStore creates a new message store.
func NewMessageStore(maxMessages int, userName string) *MessageStore {
	return &MessageStore{
		messages:    make([]*Message, 0),
		maxMessages: maxMessages,
		userName:    userName,
	}
}

// Add adds a new message and pre-renders it.
// Returns the rendered message content for printing to stdout.
func (s *MessageStore) Add(msgType MessageType, content string, extra map[string]any, renderFunc func(*Message, *MarkdownRenderer, UIStyles, string) string, renderer *MarkdownRenderer, styles UIStyles) string {
	return s.AddWithTimestamp(msgType, content, extra, time.Now(), renderFunc, renderer, styles)
}

// AddWithTimestamp adds a new message with a specific timestamp and pre-renders it.
// Returns the rendered message content for printing to stdout.
func (s *MessageStore) AddWithTimestamp(msgType MessageType, content string, extra map[string]any, timestamp time.Time, renderFunc func(*Message, *MarkdownRenderer, UIStyles, string) string, renderer *MarkdownRenderer, styles UIStyles) string {
	msg := &Message{
		ID:        generateID(),
		Type:      msgType,
		Content:   content,
		Timestamp: timestamp,
		Extra:     extra,
	}

	// Only render if renderFunc is provided
	if renderFunc != nil {
		msg.Rendered = renderFunc(msg, renderer, styles, s.userName)
		if renderer != nil {
			msg.RenderedWidth = renderer.width
		}
	}
	s.messages = append(s.messages, msg)

	// Trim if over limit
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}

	return msg.Rendered
}

// AddRaw adds a pre-rendered message directly.
// Returns the rendered content for printing to stdout.
func (s *MessageStore) AddRaw(rendered string) string {
	msg := &Message{
		ID:        generateID(),
		Type:      MessageTypeSystem,
		Content:   rendered,
		Rendered:  rendered,
		Timestamp: time.Now(),
	}

	s.messages = append(s.messages, msg)

	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}

	return rendered
}

// AppendToLast appends content to the last assistant message.
// Only accumulates content, does not trigger rendering.
func (s *MessageStore) AppendToLast(content string) {
	// Check if last message is an assistant message
	if len(s.messages) > 0 && s.messages[len(s.messages)-1].Type == MessageTypeAssistant {
		// Append to existing message
		lastMsg := s.messages[len(s.messages)-1]
		lastMsg.Content += content
		return
	}

	// Create new assistant message (without rendering)
	msg := &Message{
		ID:        generateID(),
		Type:      MessageTypeAssistant,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.messages = append(s.messages, msg)

	// Trim if over limit
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}
}

// AddEmpty creates an empty message for streaming content.
// Used by response_start to create placeholder for accumulating chunks.
func (s *MessageStore) AddEmpty(msgType MessageType) {
	msg := &Message{
		ID:        generateID(),
		Type:      msgType,
		Content:   "",
		Timestamp: time.Now(),
		Complete:  false, // Streaming in progress
	}
	s.messages = append(s.messages, msg)

	// Trim if over limit
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}
}

// MarkLastAssistantComplete marks the last assistant message as complete.
// Called by response_end to signal streaming ended (no cursor displayed).
func (s *MessageStore) MarkLastAssistantComplete() {
	msg := s.GetLastAssistantMessage()
	if msg != nil {
		msg.Complete = true
	}
}

// AppendToLastTyped appends content to the last message of the given type.
// Creates a new message if the last message is not of that type.
// Only accumulates content, does not trigger rendering.
func (s *MessageStore) AppendToLastTyped(content string, msgType MessageType) {
	// Check if last message matches the desired type
	if len(s.messages) > 0 && s.messages[len(s.messages)-1].Type == msgType {
		s.messages[len(s.messages)-1].Content += content
		return
	}
	// Create new message of the desired type
	msg := &Message{
		ID:        generateID(),
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.messages = append(s.messages, msg)

	// Trim if over limit
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}
}

// RenderLastWithType renders the last message of the given type.
// Returns empty string if no matching message exists or content is empty.
func (s *MessageStore) RenderLastWithType(msgType MessageType, renderer *MarkdownRenderer, styles UIStyles, renderFunc func(*Message, *MarkdownRenderer, UIStyles, string) string) string {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Type == msgType {
			msg := s.messages[i]
			if msg.Content == "" || msg.Rendered != "" {
				return msg.Rendered
			}
			msg.Rendered = renderFunc(msg, renderer, styles, s.userName)
			if renderer != nil {
				msg.RenderedWidth = renderer.width
			}
			return msg.Rendered
		}
	}
	return ""
}

// RenderLastWithTypeAndComplete renders the last message of the given type and marks it as complete.
// This is used for thinking messages that should not display a cursor after streaming ends.
func (s *MessageStore) RenderLastWithTypeAndComplete(msgType MessageType, renderer *MarkdownRenderer, styles UIStyles, renderFunc func(*Message, *MarkdownRenderer, UIStyles, string) string) string {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Type == msgType {
			msg := s.messages[i]
			msg.Complete = true // Mark as complete to hide cursor
			if msg.Content == "" || msg.Rendered != "" {
				return msg.Rendered
			}
			msg.Rendered = renderFunc(msg, renderer, styles, s.userName)
			if renderer != nil {
				msg.RenderedWidth = renderer.width
			}
			return msg.Rendered
		}
	}
	return ""
}

// MergeToolBlock finds the corresponding ToolStart message by executionID and merges the result.
// This creates a complete IN/OUT block for each tool instead of separate IN and OUT messages.
// Uses executionID for precise matching of concurrent tool executions.
func (s *MessageStore) MergeToolBlockByExecID(executionID, toolName, result string, duration time.Duration, styles UIStyles) {
	startMsg := s.findToolStartByExecID(executionID)
	if startMsg == nil {
		// No matching ToolStart found - add as standalone ToolEnd
		extra := map[string]any{"duration": duration, "tool": toolName}
		s.Add(MessageTypeToolEnd, result, extra, renderMessage, nil, styles)
		return
	}

	// Merge: update the ToolStart message to include OUT content
	params := s.extractParams(startMsg)
	startMsg.Rendered = renderToolBlockComplete(toolName, params, result, duration, styles)
	// Mark as merged so renderMessage won't re-render as IN-only block on width change
	if startMsg.Extra == nil {
		startMsg.Extra = make(map[string]any)
	}
	startMsg.Extra["merged"] = true
}

// findToolStartByExecID finds a ToolStart message by executionID for precise matching.
func (s *MessageStore) findToolStartByExecID(executionID string) *Message {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Type == MessageTypeToolStart {
			if s.messages[i].Extra != nil {
				if id, ok := s.messages[i].Extra["execution_id"].(string); ok && id == executionID {
					return s.messages[i]
				}
			}
		}
	}
	return nil
}

// extractParams extracts the params string from a message's Extra field.
func (s *MessageStore) extractParams(msg *Message) string {
	if msg.Extra != nil {
		if p, ok := msg.Extra["params"].(string); ok {
			return p
		}
	}
	return ""
}

// RenderLast renders the last assistant message.
// Returns empty string if no assistant message exists or content is empty.
// renderFunc is typically view.renderMessage from view.go.
func (s *MessageStore) RenderLast(renderer *MarkdownRenderer, styles UIStyles, renderFunc func(*Message, *MarkdownRenderer, UIStyles, string) string) string {
	msg := s.GetLastAssistantMessage()
	if msg == nil || msg.Content == "" {
		return ""
	}
	// Skip if already rendered
	if msg.Rendered != "" {
		return msg.Rendered
	}
	msg.Rendered = renderFunc(msg, renderer, styles, s.userName)
	if renderer != nil {
		msg.RenderedWidth = renderer.width
	}
	return msg.Rendered
}

// GetLastAssistantMessage returns the last assistant message if it exists.
// Used to retrieve buffered streaming content for final display.
func (s *MessageStore) GetLastAssistantMessage() *Message {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Type == MessageTypeAssistant {
			return s.messages[i]
		}
	}
	return nil
}

// GetMessages returns a copy of all messages.
func (s *MessageStore) GetMessages() []*Message {
	result := make([]*Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Clear clears all messages.
func (s *MessageStore) Clear() {
	s.messages = make([]*Message, 0)
}

// Count returns the number of messages.
func (s *MessageStore) Count() int {
	return len(s.messages)
}

// SetUserName sets the user name.
func (s *MessageStore) SetUserName(name string) {
	s.userName = name
}

// generateID generates a unique ID for a message.
func generateID() string {
	return time.Now().Format("20060102150405.000")
}
