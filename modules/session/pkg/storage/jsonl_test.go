package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// extractTextFromBlocks extracts text content from ContentBlocks.
func extractTextFromBlocks(blocks []sharedmemory.ContentBlock) string {
	for _, block := range blocks {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// makeTestMessage creates a test message with text content.
func makeTestMessage(sessionID, role, content string) *model.Message {
	return &model.Message{
		SessionID: sessionID,
		Role:      role,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
	}
}

// setupTestStore creates a temporary JSONLStore for testing.
func setupTestStore(t *testing.T) (*JSONLStore, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "jsonl-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestNewJSONLStore(t *testing.T) {
	t.Run("creates data directory if not exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "jsonl-new-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		dataDir := filepath.Join(tmpDir, "new-dir")
		store, err := NewJSONLStore(dataDir)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer os.RemoveAll(dataDir)

		if store == nil {
			t.Error("Expected store to be created")
		}

		// Verify directory was created
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			t.Error("Expected data directory to be created")
		}
	})

	t.Run("loads existing index", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		// Save a session
		session := &model.Session{
			ID:   "test-session",
			Name: "Test",
		}
		if err := store.SaveSession(session); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Create new store instance (should load existing index)
		store2, err := NewJSONLStore(store.dataDir)
		if err != nil {
			t.Fatalf("Failed to create second store: %v", err)
		}

		// Verify session is loaded
		loaded, err := store2.GetSession("test-session", "")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}
		if loaded.Name != "Test" {
			t.Errorf("Expected name 'Test', got %v", loaded.Name)
		}
	})
}

func TestJSONLStore_SaveAndGetSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	tests := []struct {
		name    string
		session *model.Session
		wantErr bool
	}{
		{
			name: "basic session",
			session: &model.Session{
				ID:   "session-1",
				Name: "Test Session 1",
			},
			wantErr: false,
		},
		{
			name: "session with subscriptions",
			session: &model.Session{
				ID:   "session-2",
				Name: "Test Session 2",
				Subscriptions: []model.Subscription{
					{ID: "sub1", Source: "feishu", Trigger: "告警", Active: true},
				},
			},
			wantErr: false,
		},
		{
			name: "session with system prompt",
			session: &model.Session{
				ID:           "session-3",
				Name:         "Test Session 3",
				SystemPrompt: "You are a helpful assistant",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveSession(tt.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got, err := store.GetSession(tt.session.ID, "")
				if err != nil {
					t.Errorf("GetSession() error = %v", err)
					return
				}
				if got.ID != tt.session.ID {
					t.Errorf("ID = %v, want %v", got.ID, tt.session.ID)
				}
				if got.Name != tt.session.Name {
					t.Errorf("Name = %v, want %v", got.Name, tt.session.Name)
				}
			}
		})
	}
}

func TestJSONLStore_ListSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Save multiple sessions
	sessions := []*model.Session{
		{ID: "session-1", Name: "Session 1"},
		{ID: "session-2", Name: "Session 2"},
		{ID: "session-3", Name: "Session 3"},
	}

	for _, s := range sessions {
		if err := store.SaveSession(s); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}
	}

	// List sessions
	got, err := store.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(got) != len(sessions) {
		t.Errorf("Expected %d sessions, got %d", len(sessions), len(got))
	}

	// Verify all sessions are present
	sessionIDs := make(map[string]bool)
	for _, s := range got {
		sessionIDs[s.ID] = true
	}

	for _, s := range sessions {
		if !sessionIDs[s.ID] {
			t.Errorf("Missing session %s", s.ID)
		}
	}
}

func TestJSONLStore_DeleteSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create session with messages
	session := &model.Session{
		ID:   "session-to-delete",
		Name: "To Delete",
	}
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Add a message
	msg := makeTestMessage(session.ID, "user", "Test message")
	if err := store.AppendMessage(context.Background(), msg); err != nil {
		t.Fatalf("Failed to append message: %v", err)
	}

	// Delete session
	ctx := context.Background()
	if err := store.DeleteSession(ctx, session.ID, ""); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted from index
	_, err := store.GetSession(session.ID, "")
	if err == nil {
		t.Error("Expected error getting deleted session")
	}

	// Verify messages file is deleted
	sessionFile := store.getSessionFilePath(session.ID)
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Error("Expected session file to be deleted")
	}
}

func TestJSONLStore_AppendAndGetMessages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sessionID := "test-session"

	// Save session
	if err := store.SaveSession(&model.Session{ID: sessionID, Name: "Test"}); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Append messages
	messages := []*model.Message{
		makeTestMessage(sessionID, "user", "Hello"),
		makeTestMessage(sessionID, "assistant", "Hi there!"),
		makeTestMessage(sessionID, "user", "How are you?"),
	}

	ctx := context.Background()
	for _, msg := range messages {
		if err := store.AppendMessage(ctx, msg); err != nil {
			t.Fatalf("AppendMessage() error = %v", err)
		}
	}

	// Get messages
	got, err := store.GetMessages(ctx, sessionID, 0, "") // 0 = no limit
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(got) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(got))
	}

	// Verify message content
	for i, msg := range messages {
		gotContent := extractTextFromBlocks(got[i].ContentBlocks)
		msgContent := extractTextFromBlocks(msg.ContentBlocks)
		if gotContent != msgContent {
			t.Errorf("Message %d: Content = %v, want %v", i, gotContent, msgContent)
		}
		if got[i].Role != msg.Role {
			t.Errorf("Message %d: Role = %v, want %v", i, got[i].Role, msg.Role)
		}
	}
}

func TestJSONLStore_GetMessages_WithLimit(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sessionID := "test-session-limit"

	// Save session
	if err := store.SaveSession(&model.Session{ID: sessionID, Name: "Test"}); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Append 10 messages
	for i := 0; i < 10; i++ {
		msg := makeTestMessage(sessionID, "user", string(rune('A' + i)))
		if err := store.AppendMessage(context.Background(), msg); err != nil {
			t.Fatalf("AppendMessage() error = %v", err)
		}
	}

	// Get last 3 messages
	got, err := store.GetMessages(context.Background(), sessionID, 3, "")
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(got) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(got))
	}

	// Verify last messages (H, I, J)
	expected := []string{"H", "I", "J"}
	for i, exp := range expected {
		gotContent := extractTextFromBlocks(got[i].ContentBlocks)
		if gotContent != exp {
			t.Errorf("Message %d: Content = %v, want %v", i, gotContent, exp)
		}
	}
}

func TestJSONLStore_GetMessages_EmptySession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sessionID := "empty-session"

	// Save session without messages
	if err := store.SaveSession(&model.Session{ID: sessionID, Name: "Empty"}); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Get messages from empty session
	got, err := store.GetMessages(context.Background(), sessionID, 10, "")
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(got))
	}
}

func TestJSONLStore_GetMessages_NonExistentSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Get messages from non-existent session
	got, err := store.GetMessages(context.Background(), "non-existent", 10, "")
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(got))
	}
}

func TestJSONLStore_SessionIndex_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jsonl-index-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store and save session
	store1, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	session := &model.Session{
		ID:   "persist-test",
		Name: "Persistence Test",
	}
	if err := store1.SaveSession(session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create new store instance
	store2, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	// Verify session persists across instances
	loaded, err := store2.GetSession("persist-test", "")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if loaded.Name != "Persistence Test" {
		t.Errorf("Expected name 'Persistence Test', got %v", loaded.Name)
	}

	// Verify index file exists
	indexFile := filepath.Join(tmpDir, "index.json")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		t.Error("Expected index.json to exist")
	}
}
