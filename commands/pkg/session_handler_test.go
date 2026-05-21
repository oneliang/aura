// Package commands provides tests for SessionHandler.
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
)

// TestNewSessionHandler tests the NewSessionHandler function.
func TestNewSessionHandler(t *testing.T) {
	var sessionMgr *manager.SessionManager

	handler := NewSessionHandler(sessionMgr, "")

	if handler == nil {
		t.Fatal("NewSessionHandler() returned nil")
	}
	if handler.sessionMgr != sessionMgr {
		t.Error("sessionMgr not set correctly")
	}
}

// TestSessionHandler_ExecuteCommand_UnknownCommand tests unknown command handling.
func TestSessionHandler_ExecuteCommand_UnknownCommand(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	handler := &SessionHandler{
		sessionMgr: mgr,
	}
	ctx := context.Background()

	_, err := handler.ExecuteCommand(ctx, "unknown_cmd", nil)
	if err == nil {
		t.Error("ExecuteCommand() for unknown command should return error")
	}
}

// TestSessionHandler_listSessions tests the listSessions method.
func TestSessionHandler_listSessions(t *testing.T) {
	t.Run("with sessions", func(t *testing.T) {
		// Create in-memory store for testing
		tempDir := t.TempDir()
		store, err := storage.NewJSONLStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		// Create sessions
		session1 := &model.Session{
			ID:        "s1",
			Name:      "Session 1",
			CreatedAt: 1000,
		}
		if err := store.SaveSession(session1); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		mgr, err := manager.NewSessionManager(store, nil)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.listSessions(ctx)
		if err != nil {
			t.Fatalf("listSessions() error = %v", err)
		}
		if result == "" {
			t.Fatal("listSessions() returned empty result")
		}
		if !strings.Contains(result, "Session 1") {
			t.Error("Result should contain 'Session 1'")
		}
	})

	t.Run("empty sessions", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := storage.NewJSONLStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		mgr, err := manager.NewSessionManager(store, nil)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.listSessions(ctx)
		if err != nil {
			t.Fatalf("listSessions() error = %v", err)
		}
		if !strings.Contains(result, "No sessions found") {
			t.Error("Result should contain 'No sessions found'")
		}
	})
}

// TestSessionHandler_createSession tests the createSession method.
func TestSessionHandler_createSession(t *testing.T) {
	t.Run("create without role", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.createSession(ctx, "Test Session", "")
		if err != nil {
			t.Fatalf("createSession() error = %v", err)
		}
		if result == "" {
			t.Fatal("createSession() returned empty result")
		}
		if !strings.Contains(result, "Created session") {
			t.Error("Result should contain 'Created session'")
		}
	})

	t.Run("create with role", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.createSession(ctx, "Test Session", "helper")
		if err != nil {
			t.Fatalf("createSession() error = %v", err)
		}
		if result == "" {
			t.Fatal("createSession() returned empty result")
		}
	})
}

// TestSessionHandler_deleteSession tests the deleteSession method.
func TestSessionHandler_deleteSession(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		// Create a session first
		session, _ := mgr.CreateSession("To Delete", nil, "", "")

		handler := &SessionHandler{sessionMgr: mgr}

		result, err := handler.deleteSession(session.ID)
		if err != nil {
			t.Fatalf("deleteSession() error = %v", err)
		}
		if result == "" {
			t.Fatal("deleteSession() returned empty result")
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		handler := &SessionHandler{sessionMgr: mgr}

		// Non-existent session should error
		result, err := handler.deleteSession("non-existent")
		if err == nil {
			t.Fatal("deleteSession() should return error for non-existent session")
		}
		if result != "" {
			t.Error("deleteSession() should return empty result on error")
		}
	})
}

// TestSessionHandler_showSession tests the showSession method.
func TestSessionHandler_showSession(t *testing.T) {
	t.Run("show non-existent", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SessionHandler{sessionMgr: mgr}

		_, err := handler.showSession("non-existent")
		if err == nil {
			t.Error("showSession() should return error for non-existent session")
		}
	})

	t.Run("show existing", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		session, _ := mgr.CreateSession("Show Test", nil, "", "")

		handler := &SessionHandler{sessionMgr: mgr}

		result, err := handler.showSession(session.ID)
		if err != nil {
			t.Fatalf("showSession() error = %v", err)
		}
		if result == "" {
			t.Fatal("showSession() returned empty result")
		}
		if !strings.Contains(result, "Show Test") {
			t.Error("Result should contain session name")
		}
	})
}

// TestSessionHandler_updateSession tests the updateSession method.
func TestSessionHandler_updateSession(t *testing.T) {
	t.Run("update existing", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		session, _ := mgr.CreateSession("Update Test", nil, "", "")

		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.updateSession(ctx, session.ID, "coder")
		if err != nil {
			t.Fatalf("updateSession() error = %v", err)
		}
		if result == "" {
			t.Fatal("updateSession() returned empty result")
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		handler := &SessionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.updateSession(ctx, "non-existent", "coder")
		if err == nil {
			t.Error("updateSession() should return error for non-existent session")
		}
	})
}

// TestSessionHandler_ExecuteCommand_List tests list command.
func TestSessionHandler_ExecuteCommand_List(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	handler := &SessionHandler{sessionMgr: mgr}
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "list", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
}

// TestSessionHandler_ExecuteCommand_Create tests create command.
func TestSessionHandler_ExecuteCommand_Create(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	handler := &SessionHandler{sessionMgr: mgr}
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "create", map[string]any{"name": "Test Session"})
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
	if !strings.Contains(result, "Created session") {
		t.Error("Result should contain 'Created session'")
	}
}
