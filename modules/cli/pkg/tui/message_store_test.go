package tui

import (
	"sync"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/utils"
)

// mockUIStyles returns minimal styles for testing.
func mockUIStyles() UIStyles {
	return UIStyles{}
}

func TestNewMessageStore(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	if store == nil {
		t.Fatal("NewMessageStore() returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("New message store should be empty, got %d messages", store.Count())
	}
}

func TestMessageStore_Add(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	tests := []struct {
		name     string
		msgType  MessageType
		content  string
		wantType MessageType
	}{
		{
			name:     "user message",
			msgType:  MessageTypeUser,
			content:  "Hello",
			wantType: MessageTypeUser,
		},
		{
			name:     "assistant message",
			msgType:  MessageTypeAssistant,
			content:  "Hi there",
			wantType: MessageTypeAssistant,
		},
		{
			name:     "tool start message",
			msgType:  MessageTypeToolStart,
			content:  "file_read",
			wantType: MessageTypeToolStart,
		},
		{
			name:     "tool end message",
			msgType:  MessageTypeToolEnd,
			content:  "result",
			wantType: MessageTypeToolEnd,
		},
		{
			name:     "error message",
			msgType:  MessageTypeError,
			content:  "Something went wrong",
			wantType: MessageTypeError,
		},
		{
			name:     "system message",
			msgType:  MessageTypeSystem,
			content:  "System notification",
			wantType: MessageTypeSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialCount := store.Count()
			store.Add(tt.msgType, tt.content, nil, nil, nil, UIStyles{})

			if store.Count() != initialCount+1 {
				t.Errorf("Expected %d messages, got %d", initialCount+1, store.Count())
			}

			msgs := store.GetMessages()
			lastMsg := msgs[len(msgs)-1]

			if lastMsg.Type != tt.wantType {
				t.Errorf("Expected type %v, got %v", tt.wantType, lastMsg.Type)
			}
			if lastMsg.Content != tt.content {
				t.Errorf("Expected content %q, got %q", tt.content, lastMsg.Content)
			}
		})
	}
}

func TestMessageStore_AddWithTimestamp(t *testing.T) {
	store := NewMessageStore(100, "TestUser")
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	store.AddWithTimestamp(MessageTypeUser, "Test message", nil, ts, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	if !msgs[0].Timestamp.Equal(ts) {
		t.Errorf("Expected timestamp %v, got %v", ts, msgs[0].Timestamp)
	}
}

func TestMessageStore_AddRaw(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	rendered := store.AddRaw("Pre-rendered content")

	if rendered != "Pre-rendered content" {
		t.Errorf("Expected rendered content to be returned, got %q", rendered)
	}

	msgs := store.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Type != MessageTypeSystem {
		t.Errorf("Expected MessageTypeSystem, got %v", msgs[0].Type)
	}
}

func TestMessageStore_MaxMessages(t *testing.T) {
	maxMessages := 5
	store := NewMessageStore(maxMessages, "TestUser")

	// Add more messages than the limit
	for i := 0; i < 10; i++ {
		store.Add(MessageTypeUser, "message", nil, nil, nil, UIStyles{})
	}

	if store.Count() > maxMessages {
		t.Errorf("Expected at most %d messages, got %d", maxMessages, store.Count())
	}
}

func TestMessageStore_Clear(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	store.Add(MessageTypeUser, "message 1", nil, nil, nil, UIStyles{})
	store.Add(MessageTypeUser, "message 2", nil, nil, nil, UIStyles{})

	if store.Count() != 2 {
		t.Fatalf("Expected 2 messages before clear, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", store.Count())
	}
}

func TestMessageStore_Render(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	store.Add(MessageTypeUser, "Hello", nil, nil, nil, UIStyles{})
	store.Add(MessageTypeAssistant, "Hi there", nil, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(msgs))
	}

	// Check that both messages have correct content
	if msgs[0].Content != "Hello" {
		t.Error("First message should contain 'Hello'")
	}
	if msgs[1].Content != "Hi there" {
		t.Error("Second message should contain 'Hi there'")
	}
}

func TestMessageStore_SetUserName(t *testing.T) {
	store := NewMessageStore(100, "")

	store.SetUserName("Alice")

	// Add a user message and check the rendered output
	store.Add(MessageTypeUser, "Test message", nil, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Test message" {
		t.Errorf("Expected content 'Test message', got %q", msgs[0].Content)
	}
}

func TestMessageStore_AppendToLast(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	// Append without existing assistant message
	store.AppendToLast("Hello")
	msgs := store.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Type != MessageTypeAssistant {
		t.Errorf("Expected MessageTypeAssistant, got %v", msgs[0].Type)
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %q", msgs[0].Content)
	}

	// Append to existing message
	store.AppendToLast(" World")
	msgs = store.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message (appended), got %d", len(msgs))
	}
	if msgs[0].Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got %q", msgs[0].Content)
	}
}

func TestMessageStore_AppendToLast_WithUserMessage(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	// Add a user message first
	store.Add(MessageTypeUser, "User question", nil, nil, nil, UIStyles{})

	// Now append to create assistant message
	store.AppendToLast("Assistant response")

	msgs := store.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(msgs))
	}

	// Last message should be assistant
	if msgs[1].Type != MessageTypeAssistant {
		t.Errorf("Expected last message to be MessageTypeAssistant, got %v", msgs[1].Type)
	}
}

func TestMessageStore_Concurrent(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				store.Add(MessageTypeUser, "concurrent message", nil, nil, nil, UIStyles{})
			}
		}()
	}

	wg.Wait()

	// Should have at most maxMessages (100) due to trimming
	if store.Count() > 100 {
		t.Errorf("Expected at most 100 messages, got %d", store.Count())
	}
}

func TestMessageStore_ToolStartWithParams(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	// Add tool start with params
	store.Add(MessageTypeToolStart, "file_read", map[string]any{
		"params": `{"path": "/test/file.go"}`,
	}, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) == 0 {
		t.Fatal("Expected at least 1 message")
	}
	// Check that extra params are stored
	if msgs[0].Extra == nil {
		t.Error("Expected extra params to be stored")
	}
}

func TestMessageStore_ToolEndWithDuration(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	// Add tool end with duration
	store.Add(MessageTypeToolEnd, "result", map[string]any{
		"duration": 150 * time.Millisecond,
	}, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) == 0 {
		t.Fatal("Expected at least 1 message")
	}
	// Check that extra params are stored
	if msgs[0].Extra == nil {
		t.Error("Expected extra params to be stored")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateID() should return unique IDs")
	}
}

func TestFormatTimestamp(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	formatted := utils.FormatTimestamp(ts)

	expected := "10:30:45"
	if formatted != expected {
		t.Errorf("formatTimestamp() = %q, want %q", formatted, expected)
	}
}

func TestMessageStore_UpdateRenderer(t *testing.T) {
	store := NewMessageStore(100, "TestUser")

	// Add a message
	store.Add(MessageTypeAssistant, "Test", nil, nil, nil, UIStyles{})

	// Verify we can still add messages
	store.Add(MessageTypeAssistant, "New message", nil, nil, nil, UIStyles{})

	msgs := store.GetMessages()
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}
}

func TestMessageStore_GetMessages_Copy(t *testing.T) {
	store := NewMessageStore(100, "TestUser")
	store.Add(MessageTypeUser, "Test", nil, nil, nil, UIStyles{})

	msgs1 := store.GetMessages()
	msgs2 := store.GetMessages()

	// Should be different slices (copy)
	if &msgs1[0] == &msgs2[0] {
		t.Error("GetMessages() should return a copy, not the same slice")
	}
}
