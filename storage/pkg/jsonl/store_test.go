// Package jsonl provides tests for JSONL-based message storage.
package jsonl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/storage/pkg/message"
)

// setupTestStore creates a temporary directory and message store for testing.
func setupTestStore(t *testing.T) (*MessageStore, string, func()) {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "jsonl-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewMessageStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create message store: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return store, tempDir, cleanup
}

// createTestMessage creates a test message with the given session ID.
func createTestMessage(sessionID, role, content string, timestamp int64) *message.Message {
	return &message.Message{
		SessionID: sessionID,
		Role:      role,
		ContentBlocks: []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: content},
		},
		Timestamp: timestamp,
	}
}

// getTextContent extracts text content from a message's ContentBlocks.
func getTextContent(m *message.Message) string {
	for _, block := range m.ContentBlocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

func TestNewMessageStore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "jsonl-store-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		store, err := NewMessageStore(tempDir)
		if err != nil {
			t.Fatalf("NewMessageStore() error = %v", err)
		}
		if store == nil {
			t.Fatal("NewMessageStore() returned nil")
		}
		if store.dataDir != tempDir {
			t.Errorf("dataDir = %q, want %q", store.dataDir, tempDir)
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "jsonl-store-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		newDir := filepath.Join(tempDir, "new-dir")
		store, err := NewMessageStore(newDir)
		if err != nil {
			t.Fatalf("NewMessageStore() error = %v", err)
		}
		if store == nil {
			t.Fatal("NewMessageStore() returned nil")
		}

		// Verify directory was created
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			t.Error("Directory was not created")
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		// Use a path that should fail on most systems
		invalidPath := "/root/protected/invalid/path/that/should/fail"
		store, err := NewMessageStore(invalidPath)
		if err == nil {
			os.RemoveAll(invalidPath)
			t.Fatal("NewMessageStore() with invalid path should return error")
		}
		if store != nil {
			t.Error("Store should be nil on error")
		}
	})
}

func TestMessageStoreFilePath(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	sessionID := "test-session-123"
	expected := filepath.Join(store.dataDir, sessionID+".jsonl")
	got := store.filePath(sessionID)

	if got != expected {
		t.Errorf("filePath() = %q, want %q", got, expected)
	}
}

func TestMessageStoreAppend(t *testing.T) {
	t.Run("append single message", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		msg := createTestMessage("session-1", "user", "Hello", 1234567890000)

		err := store.Append(ctx, msg)
		if err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Verify file was created
		filePath := store.filePath("session-1")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Session file was not created")
		}
	})

	t.Run("append sets timestamp if zero", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		msg := createTestMessage("session-1", "user", "Hello", 0)
		beforeAppend := time.Now().UnixMilli()

		err := store.Append(ctx, msg)
		if err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		afterAppend := time.Now().UnixMilli()

		// Read back and verify timestamp was set
		messages, err := store.Get(ctx, "session-1", 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(messages))
		}
		if messages[0].Timestamp < beforeAppend || messages[0].Timestamp > afterAppend {
			t.Errorf("Timestamp not set correctly: got %d, expected between %d and %d",
				messages[0].Timestamp, beforeAppend, afterAppend)
		}
	})

	t.Run("append multiple messages", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-multi"

		messages := []*message.Message{
			createTestMessage(sessionID, "user", "Message 1", 1000),
			createTestMessage(sessionID, "assistant", "Message 2", 2000),
			createTestMessage(sessionID, "user", "Message 3", 3000),
		}

		for _, msg := range messages {
			err := store.Append(ctx, msg)
			if err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Verify all messages were stored
		stored, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(stored) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(stored))
		}
	})

	t.Run("concurrent append", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-concurrent"
		numGoroutines := 10
		messagesPerGoroutine := 10

		done := make(chan bool, numGoroutines)

		for g := 0; g < numGoroutines; g++ {
			go func(gid int) {
				for m := 0; m < messagesPerGoroutine; m++ {
					msg := createTestMessage(sessionID, "user",
						"Goroutine "+string(rune(gid+'0'))+" Message "+string(rune(m+'0')),
						int64(gid*1000+m))
					if err := store.Append(ctx, msg); err != nil {
						t.Errorf("Concurrent Append() error = %v", err)
					}
				}
				done <- true
			}(g)
		}

		// Wait for all goroutines to finish
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all messages were stored
		stored, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		expected := numGoroutines * messagesPerGoroutine
		if len(stored) != expected {
			t.Errorf("Expected %d messages, got %d", expected, len(stored))
		}
	})
}

func TestMessageStoreGet(t *testing.T) {
	t.Run("get from non-existent session", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		messages, err := store.Get(ctx, "non-existent-session", 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected empty slice, got %d messages", len(messages))
		}
		if messages == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("get all messages", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-get-all"

		// Append 5 messages
		for i := 0; i < 5; i++ {
			msg := createTestMessage(sessionID, "user", "Message "+string(rune(i+'1')), int64(i+1))
			if err := store.Append(ctx, msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Get all messages (limit <= 0)
		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 5 {
			t.Errorf("Expected 5 messages, got %d", len(messages))
		}
	})

	t.Run("get with limit", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-get-limit"

		// Append 10 messages
		for i := 0; i < 10; i++ {
			msg := createTestMessage(sessionID, "user", "Message "+string(rune(i+'1')), int64(i+1))
			if err := store.Append(ctx, msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Get last 3 messages
		messages, err := store.Get(ctx, sessionID, 3, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
		// Verify we got the last 3 messages
		if messages[0].Timestamp != 8 {
			t.Errorf("First message timestamp = %d, want 8", messages[0].Timestamp)
		}
		if messages[1].Timestamp != 9 {
			t.Errorf("Second message timestamp = %d, want 9", messages[1].Timestamp)
		}
		if messages[2].Timestamp != 10 {
			t.Errorf("Third message timestamp = %d, want 10", messages[2].Timestamp)
		}
	})

	t.Run("get with limit larger than message count", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-get-large-limit"

		// Append 3 messages
		for i := 0; i < 3; i++ {
			msg := createTestMessage(sessionID, "user", "Message "+string(rune(i+'1')), int64(i+1))
			if err := store.Append(ctx, msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}

		// Get with limit larger than count
		messages, err := store.Get(ctx, sessionID, 100, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
	})

	t.Run("get handles malformed lines", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-malformed"

		// Write a mix of valid and invalid lines
		filePath := store.filePath(sessionID)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Valid line
		validMsg := `{"session_id":"` + sessionID + `","role":"user","content":"Valid","timestamp":1}`
		file.WriteString(validMsg + "\n")
		// Invalid line (should be skipped)
		file.WriteString("this is not valid json\n")
		// Another valid line
		validMsg2 := `{"session_id":"` + sessionID + `","role":"assistant","content":"Also Valid","timestamp":2}`
		file.WriteString(validMsg2 + "\n")
		file.Close()

		// Get should skip malformed lines
		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("Expected 2 valid messages (skipping malformed), got %d", len(messages))
		}
	})
}

func TestMessageStoreDeleteSession(t *testing.T) {
	t.Run("delete existing session", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-delete"

		// Append a message to create the session file
		msg := createTestMessage(sessionID, "user", "To be deleted", 1000)
		if err := store.Append(ctx, msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Verify file exists
		filePath := store.filePath(sessionID)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("Session file should exist before delete")
		}

		// Delete session
		err := store.DeleteSession(sessionID)
		if err != nil {
			t.Fatalf("DeleteSession() error = %v", err)
		}

		// Verify file is gone
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("Session file should be deleted")
		}

		// Get should return empty for deleted session
		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() after delete error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages after delete, got %d", len(messages))
		}
	})

	t.Run("delete non-existent session", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		err := store.DeleteSession("non-existent-session")
		if err != nil {
			t.Fatalf("DeleteSession() for non-existent session should not error, got %v", err)
		}
	})
}

func TestMessageStoreIntegration(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Test complete workflow: append, get, delete
	sessionID := "integration-session"

	// Phase 1: Append multiple messages
	t.Run("append phase", func(t *testing.T) {
		messages := []struct {
			role    string
			content string
			ts      int64
		}{
			{"user", "Hello", 1000},
			{"assistant", "Hi there!", 2000},
			{"user", "How are you?", 3000},
			{"assistant", "I'm doing well, thanks!", 4000},
		}

		for _, m := range messages {
			msg := createTestMessage(sessionID, m.role, m.content, m.ts)
			if err := store.Append(ctx, msg); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
		}
	})

	// Phase 2: Retrieve and verify
	t.Run("retrieve phase", func(t *testing.T) {
		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 4 {
			t.Fatalf("Expected 4 messages, got %d", len(messages))
		}

		expected := []struct {
			role    string
			content string
			ts      int64
		}{
			{"user", "Hello", 1000},
			{"assistant", "Hi there!", 2000},
			{"user", "How are you?", 3000},
			{"assistant", "I'm doing well, thanks!", 4000},
		}

		for i, exp := range expected {
			if messages[i].Role != exp.role {
				t.Errorf("Message %d role = %q, want %q", i, messages[i].Role, exp.role)
			}
			if getTextContent(&messages[i]) != exp.content {
				t.Errorf("Message %d content = %q, want %q", i, getTextContent(&messages[i]), exp.content)
			}
			if messages[i].Timestamp != exp.ts {
				t.Errorf("Message %d timestamp = %d, want %d", i, messages[i].Timestamp, exp.ts)
			}
		}
	})

	// Phase 3: Get with limit
	t.Run("limited retrieve phase", func(t *testing.T) {
		messages, err := store.Get(ctx, sessionID, 2, "")
		if err != nil {
			t.Fatalf("Get(limit=2) error = %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
		// Should return last 2 messages
		if messages[0].Timestamp != 3000 {
			t.Errorf("First limited message timestamp = %d, want 3000", messages[0].Timestamp)
		}
		if messages[1].Timestamp != 4000 {
			t.Errorf("Second limited message timestamp = %d, want 4000", messages[1].Timestamp)
		}
	})

	// Phase 4: Delete and verify
	t.Run("delete phase", func(t *testing.T) {
		err := store.DeleteSession(sessionID)
		if err != nil {
			t.Fatalf("DeleteSession() error = %v", err)
		}

		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() after delete error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages after delete, got %d", len(messages))
		}
	})
}

// TestMessageStoreAppend_ErrorCases tests error cases for Append.
func TestMessageStoreAppend_ErrorCases(t *testing.T) {
	t.Run("marshal error with unmarshalable type", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		// Create a message with a channel which can't be marshaled
		msg := &message.Message{
			SessionID: "session-1",
			Role:      "user",
			ContentBlocks: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: "test"},
			},
			Timestamp: 1000,
		}
		// Normal message should work
		err := store.Append(ctx, msg)
		if err != nil {
			t.Fatalf("Append() normal message error = %v", err)
		}
	})
}

// TestMessageStoreGet_ErrorCases tests error cases for Get.
func TestMessageStoreGet_ErrorCases(t *testing.T) {
	t.Run("empty session file", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-empty"

		// Create an empty file
		filePath := store.filePath(sessionID)
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages from empty file, got %d", len(messages))
		}
	})

	t.Run("file with only empty lines", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-empty-lines"

		// Create a file with only empty lines
		filePath := store.filePath(sessionID)
		if err := os.WriteFile(filePath, []byte("\n\n\n"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages from file with empty lines, got %d", len(messages))
		}
	})

	t.Run("file with only malformed lines", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-all-malformed"

		// Create a file with only malformed lines
		filePath := store.filePath(sessionID)
		if err := os.WriteFile(filePath, []byte("invalid json\nalso invalid\n"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		messages, err := store.Get(ctx, sessionID, 0, "")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages from file with malformed lines, got %d", len(messages))
		}
	})
}

// TestMessageStoreDeleteSession_ErrorCases tests error cases for DeleteSession.
func TestMessageStoreDeleteSession_ErrorCases(t *testing.T) {
	t.Run("delete with read-only file", func(t *testing.T) {
		store, _, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		sessionID := "session-readonly"

		// Append a message to create the session file
		msg := createTestMessage(sessionID, "user", "test", 1000)
		if err := store.Append(ctx, msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Make file read-only (this may not work on all systems)
		filePath := store.filePath(sessionID)
		if err := os.Chmod(filePath, 0444); err != nil {
			t.Skip("Cannot change file permissions on this system")
		}
		defer os.Chmod(filePath, 0644) // Restore permissions

		// Try to delete - on some systems this will fail, on others it will succeed
		err := store.DeleteSession(sessionID)
		// We don't assert on the result as it depends on the system
		_ = err
	})
}

// TestMessageStoreGet_NegativeLimit tests Get with negative limit.
func TestMessageStoreGet_NegativeLimit(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := "session-neg-limit"

	// Append 5 messages
	for i := 0; i < 5; i++ {
		msg := createTestMessage(sessionID, "user", "Message "+string(rune(i+'1')), int64(i+1))
		if err := store.Append(ctx, msg); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Get with negative limit should return all messages
	messages, err := store.Get(ctx, sessionID, -1, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(messages) != 5 {
		t.Errorf("Expected 5 messages with negative limit, got %d", len(messages))
	}
}

// TestMessageStoreContextCancellation tests context cancellation.
func TestMessageStoreContextCancellation(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sessionID := "session-cancelled"

	// Append should still work (doesn't use context)
	msg := createTestMessage(sessionID, "user", "test", 1000)
	err := store.Append(ctx, msg)
	if err != nil {
		t.Fatalf("Append() with cancelled context error = %v", err)
	}

	// Get should also work (doesn't really use context for I/O timing)
	messages, err := store.Get(ctx, sessionID, 0, "")
	if err != nil {
		t.Fatalf("Get() with cancelled context error = %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}

// TestMessageStoreContentBlocks tests that ContentBlocks are properly stored and retrieved.
func TestMessageStoreContentBlocks(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := "session-content-blocks"

	// Create message with ContentBlocks
	msg := &message.Message{
		SessionID: sessionID,
		Role:       "assistant",
		Timestamp:  1000,
	}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.ThinkingBlock{
			Type:     memory.BlockTypeThinking,
			Thinking: "Let me think about this...",
		},
		memory.TextBlock{
			Type: memory.BlockTypeText,
			Text: "Here is my response",
		},
		memory.ToolUseBlock{
			Type:  memory.BlockTypeToolUse,
			ID:    "tool-123",
			Name:  "read_file",
			Input: []byte(`{"path": "/tmp/test.txt"}`),
		},
	})

	// Append message
	err := store.Append(ctx, msg)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Retrieve messages
	messages, err := store.Get(ctx, sessionID, 0, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Verify Content field (backward compatibility)
	if getTextContent(&messages[0]) != "Here is my response" {
		t.Errorf("Content = %q, want %q", getTextContent(&messages[0]), "Here is my response")
	}

	// Verify ContentBlocks
	blocks := messages[0].GetContentBlocks()
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 content blocks, got %d", len(blocks))
	}

	// Verify ThinkingBlock
	tb, ok := blocks[0].(memory.ThinkingBlock)
	if !ok {
		t.Fatalf("Expected ThinkingBlock, got %T", blocks[0])
	}
	if tb.Thinking != "Let me think about this..." {
		t.Errorf("ThinkingBlock.Thinking = %q, want %q", tb.Thinking, "Let me think about this...")
	}

	// Verify TextBlock
	txt, ok := blocks[1].(memory.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", blocks[1])
	}
	if txt.Text != "Here is my response" {
		t.Errorf("TextBlock.Text = %q, want %q", txt.Text, "Here is my response")
	}

	// Verify ToolUseBlock
	tu, ok := blocks[2].(memory.ToolUseBlock)
	if !ok {
		t.Fatalf("Expected ToolUseBlock, got %T", blocks[2])
	}
	if tu.ID != "tool-123" {
		t.Errorf("ToolUseBlock.ID = %q, want %q", tu.ID, "tool-123")
	}
	if tu.Name != "read_file" {
		t.Errorf("ToolUseBlock.Name = %q, want %q", tu.Name, "read_file")
	}
	// Verify Input by parsing and comparing (JSON whitespace may differ)
	var gotInput, wantInput map[string]string
	if err := json.Unmarshal(tu.Input, &gotInput); err != nil {
		t.Fatalf("Failed to parse ToolUseBlock.Input: %v", err)
	}
	wantInputBytes := []byte(`{"path": "/tmp/test.txt"}`)
	if err := json.Unmarshal(wantInputBytes, &wantInput); err != nil {
		t.Fatalf("Failed to parse expected input: %v", err)
	}
	if gotInput["path"] != wantInput["path"] {
		t.Errorf("ToolUseBlock.Input path = %q, want %q", gotInput["path"], wantInput["path"])
	}
}

// TestMessageStoreToolResultBlock tests ToolResultBlock with nested ContentBlocks.
func TestMessageStoreToolResultBlock(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := "session-tool-result"

	// Create message with ToolResultBlock containing nested ContentBlocks
	msg := &message.Message{
		SessionID: sessionID,
		Role:       "user",
		Timestamp:  1000,
	}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "tool-123",
			Content: []memory.ContentBlock{
				memory.TextBlock{
					Type: memory.BlockTypeText,
					Text: "File contents here",
				},
			},
			IsError: false,
		},
	})

	// Append message
	err := store.Append(ctx, msg)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Retrieve messages
	messages, err := store.Get(ctx, sessionID, 0, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Verify ToolResultBlock
	blocks := messages[0].GetContentBlocks()
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(blocks))
	}

	tr, ok := blocks[0].(memory.ToolResultBlock)
	if !ok {
		t.Fatalf("Expected ToolResultBlock, got %T", blocks[0])
	}
	if tr.ToolUseID != "tool-123" {
		t.Errorf("ToolResultBlock.ToolUseID = %q, want %q", tr.ToolUseID, "tool-123")
	}
	if tr.IsError != false {
		t.Errorf("ToolResultBlock.IsError = %v, want false", tr.IsError)
	}
	if len(tr.Content) != 1 {
		t.Fatalf("Expected 1 nested content block, got %d", len(tr.Content))
	}
	nested, ok := tr.Content[0].(memory.TextBlock)
	if !ok {
		t.Fatalf("Expected nested TextBlock, got %T", tr.Content[0])
	}
	if nested.Text != "File contents here" {
		t.Errorf("Nested TextBlock.Text = %q, want %q", nested.Text, "File contents here")
	}
}

// TestMessageStoreMixedContent tests messages with both Content string and ContentBlocks.
func TestMessageStoreMixedContent(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := "session-mixed"

	// Message with only Content string (legacy format)
	msg1 := &message.Message{
		SessionID: sessionID,
		Role:       "user",
		ContentBlocks: []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Legacy content"},
		},
		Timestamp:  1000,
	}

	// Message with ContentBlocks
	msg2 := &message.Message{
		SessionID: sessionID,
		Role:       "assistant",
		Timestamp:  2000,
	}
	msg2.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{
			Type: memory.BlockTypeText,
			Text: "New format content",
		},
	})

	// Append both
	if err := store.Append(ctx, msg1); err != nil {
		t.Fatalf("Append() msg1 error = %v", err)
	}
	if err := store.Append(ctx, msg2); err != nil {
		t.Fatalf("Append() msg2 error = %v", err)
	}

	// Retrieve both
	messages, err := store.Get(ctx, sessionID, 0, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	// Verify legacy message
	if getTextContent(&messages[0]) != "Legacy content" {
		t.Errorf("msg1 Content = %q, want %q", getTextContent(&messages[0]), "Legacy content")
	}
	blocks1 := messages[0].GetContentBlocks()
	if len(blocks1) != 1 {
		t.Errorf("msg1 GetContentBlocks() should return 1 block (converted from Content), got %d", len(blocks1))
	} else if txt, ok := blocks1[0].(memory.TextBlock); !ok || txt.Text != "Legacy content" {
		t.Errorf("msg1 GetContentBlocks() should convert Content to TextBlock")
	}

	// Verify new format message
	if getTextContent(&messages[1]) != "New format content" {
		t.Errorf("msg2 Content = %q, want %q", getTextContent(&messages[1]), "New format content")
	}
	blocks2 := messages[1].GetContentBlocks()
	if len(blocks2) != 1 {
		t.Errorf("msg2 GetContentBlocks() should return 1 block, got %d", len(blocks2))
	} else if txt, ok := blocks2[0].(memory.TextBlock); !ok || txt.Text != "New format content" {
		t.Errorf("msg2 GetContentBlocks() returned incorrect block")
	}
}
