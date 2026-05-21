// Package commands provides tests for SubscriptionHandler.
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
)

// TestNewSubscriptionHandler tests the NewSubscriptionHandler function.
func TestNewSubscriptionHandler(t *testing.T) {
	var sessionMgr *manager.SessionManager

	handler := NewSubscriptionHandler(sessionMgr, "")

	if handler == nil {
		t.Fatal("NewSubscriptionHandler() returned nil")
	}
	if handler.sessionMgr != sessionMgr {
		t.Error("sessionMgr not set correctly")
	}
}

// TestSubscriptionHandler_ExecuteCommand tests the ExecuteCommand method with valid manager.
func TestSubscriptionHandler_ExecuteCommand(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	handler := &SubscriptionHandler{
		sessionMgr: mgr,
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		cmd     string
		params  map[string]any
		wantErr bool
	}{
		{name: "show subscriptions", cmd: "show", params: nil, wantErr: false},
		{name: "unknown command", cmd: "unknown", params: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.ExecuteCommand(ctx, tt.cmd, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSubscriptionHandler_showSubscriptions tests the showSubscriptions method.
func TestSubscriptionHandler_showSubscriptions(t *testing.T) {
	t.Run("empty sessions", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		handler := &SubscriptionHandler{sessionMgr: mgr}

		result, err := handler.showSubscriptions("")
		if err != nil {
			t.Fatalf("showSubscriptions() error = %v", err)
		}
		if result == "" {
			t.Fatal("showSubscriptions() returned empty result")
		}
		if !strings.Contains(result, "No sessions found") {
			t.Error("Result should contain 'No sessions found'")
		}
	})

	t.Run("with sessions", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		// Create a session with subscription
		session := &model.Session{
			ID:   "test-session",
			Name: "Test Session",
			Subscriptions: []model.Subscription{
				{ID: "sub1", Trigger: "keyword", Source: "feishu", Active: true},
			},
		}
		_ = store.SaveSession(session)

		handler := &SubscriptionHandler{sessionMgr: mgr}

		result, err := handler.showSubscriptions("")
		if err != nil {
			t.Fatalf("showSubscriptions() error = %v", err)
		}
		if result == "" {
			t.Fatal("showSubscriptions() returned empty result")
		}
		if !strings.Contains(result, "Test Session") {
			t.Error("Result should contain session name")
		}
		if !strings.Contains(result, "keyword") {
			t.Error("Result should contain trigger")
		}
	})

	t.Run("specific session with subscriptions", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		session := &model.Session{
			ID:   "test-session",
			Name: "Test Session",
			Subscriptions: []model.Subscription{
				{ID: "sub1", Trigger: "keyword", Source: "feishu", Active: true},
			},
		}
		_ = store.SaveSession(session)

		handler := &SubscriptionHandler{sessionMgr: mgr}

		result, err := handler.showSubscriptions("test-session")
		if err != nil {
			t.Fatalf("showSubscriptions() error = %v", err)
		}
		if result == "" {
			t.Fatal("showSubscriptions() returned empty result")
		}
		if !strings.Contains(result, "Test Session") {
			t.Error("Result should contain session name")
		}
	})

	t.Run("specific session not found", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		handler := &SubscriptionHandler{sessionMgr: mgr}

		_, err := handler.showSubscriptions("non-existent")
		if err == nil {
			t.Error("showSubscriptions() should return error for non-existent session")
		}
	})
}

// TestSubscriptionHandler_addSubscription tests the addSubscription method.
func TestSubscriptionHandler_addSubscription(t *testing.T) {
	t.Run("empty session_id", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.addSubscription(ctx, "", "keyword", "feishu")
		if err == nil {
			t.Error("addSubscription() with empty session_id should return error")
		}
	})

	t.Run("empty trigger", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.addSubscription(ctx, "session-1", "", "feishu")
		if err == nil {
			t.Error("addSubscription() with empty trigger should return error")
		}
	})

	t.Run("session not found", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.addSubscription(ctx, "non-existent", "keyword", "feishu")
		if err == nil {
			t.Error("addSubscription() with non-existent session should return error")
		}
	})

	t.Run("successful add", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		// Create a session first
		session := &model.Session{
			ID:   "test-session",
			Name: "Test Session",
		}
		_ = store.SaveSession(session)

		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.addSubscription(ctx, "test-session", "keyword", "feishu")
		if err != nil {
			t.Fatalf("addSubscription() error = %v", err)
		}
		if result == "" {
			t.Fatal("addSubscription() returned empty result")
		}
		if !strings.Contains(result, "keyword") {
			t.Error("Result should contain trigger")
		}
		if !strings.Contains(result, "feishu") {
			t.Error("Result should contain source")
		}
	})
}

// TestSubscriptionHandler_deleteSubscription tests the deleteSubscription method.
func TestSubscriptionHandler_deleteSubscription(t *testing.T) {
	t.Run("empty session_id", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.deleteSubscription(ctx, "", "sub1")
		if err == nil {
			t.Error("deleteSubscription() with empty session_id should return error")
		}
	})

	t.Run("empty subscription_id", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.deleteSubscription(ctx, "session-1", "")
		if err == nil {
			t.Error("deleteSubscription() with empty subscription_id should return error")
		}
	})

	t.Run("session not found", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)
		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.deleteSubscription(ctx, "non-existent", "sub1")
		if err == nil {
			t.Error("deleteSubscription() with non-existent session should return error")
		}
	})

	t.Run("subscription not found", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		// Create a session without subscriptions
		session := &model.Session{
			ID:            "test-session",
			Name:          "Test Session",
			Subscriptions: []model.Subscription{},
		}
		_ = store.SaveSession(session)

		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		_, err := handler.deleteSubscription(ctx, "test-session", "non-existent")
		if err == nil {
			t.Error("deleteSubscription() should return error for non-existent subscription")
		}
	})

	t.Run("successful delete", func(t *testing.T) {
		tempDir := t.TempDir()
		store, _ := storage.NewJSONLStore(tempDir)
		mgr, _ := manager.NewSessionManager(store, nil)

		// Create a session with subscription
		session := &model.Session{
			ID:   "test-session",
			Name: "Test Session",
			Subscriptions: []model.Subscription{
				{ID: "sub1", Trigger: "keyword", Source: "feishu", Active: true},
			},
		}
		_ = store.SaveSession(session)

		handler := &SubscriptionHandler{sessionMgr: mgr}
		ctx := context.Background()

		result, err := handler.deleteSubscription(ctx, "test-session", "sub1")
		if err != nil {
			t.Fatalf("deleteSubscription() error = %v", err)
		}
		if result == "" {
			t.Fatal("deleteSubscription() returned empty result")
		}
		if !strings.Contains(result, "deleted") {
			t.Error("Result should indicate deletion")
		}
	})
}

// TestSubscriptionHandler_ExecuteCommand_ShowAll tests show command without session_id.
func TestSubscriptionHandler_ExecuteCommand_ShowAll(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	handler := &SubscriptionHandler{sessionMgr: mgr}
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "show", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
}

// TestSubscriptionHandler_ExecuteCommand_AddSubscription tests add command.
func TestSubscriptionHandler_ExecuteCommand_AddSubscription(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	// Create a session first
	session := &model.Session{
		ID:   "test-session",
		Name: "Test Session",
	}
	_ = store.SaveSession(session)

	handler := &SubscriptionHandler{sessionMgr: mgr}
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "add", map[string]any{
		"session_id": "test-session",
		"trigger":    "keyword",
		"source":     "feishu",
	})
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
	if !strings.Contains(result, "keyword") {
		t.Error("Result should contain trigger")
	}
}

// TestSubscriptionHandler_ExecuteCommand_DeleteSubscription tests delete command.
func TestSubscriptionHandler_ExecuteCommand_DeleteSubscription(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tempDir)
	mgr, _ := manager.NewSessionManager(store, nil)

	// Create a session with subscription
	session := &model.Session{
		ID:   "test-session",
		Name: "Test Session",
		Subscriptions: []model.Subscription{
			{ID: "sub1", Trigger: "keyword", Source: "feishu", Active: true},
		},
	}
	_ = store.SaveSession(session)

	handler := &SubscriptionHandler{sessionMgr: mgr}
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "delete", map[string]any{
		"session_id":      "test-session",
		"subscription_id": "sub1",
	})
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
	if !strings.Contains(result, "deleted") {
		t.Error("Result should indicate deletion")
	}
}
