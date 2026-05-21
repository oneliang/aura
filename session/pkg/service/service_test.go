package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
)

// TestNewService tests NewService function.
func TestNewService(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	svc := NewService(store)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.store != store {
		t.Error("Store not set correctly")
	}
	if svc.router == nil {
		t.Error("Router should be initialized")
	}
}

// TestNewServiceFromDataDir tests NewServiceFromDataDir function.
func TestNewServiceFromDataDir(t *testing.T) {
	tmpDir := t.TempDir()

	svc, err := NewServiceFromDataDir(tmpDir)
	if err != nil {
		t.Fatalf("NewServiceFromDataDir() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewServiceFromDataDir() returned nil")
	}
	if svc.store == nil {
		t.Error("Store should be initialized")
	}
}

// TestNewServiceFromDataDir_InvalidPath tests with invalid path.
func TestNewServiceFromDataDir_InvalidPath(t *testing.T) {
	// Try with a path that can't be created (e.g., a file path)
	tmpFile := filepath.Join(t.TempDir(), "file")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	_, err := NewServiceFromDataDir(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

// TestService_GetStore tests GetStore method.
func TestService_GetStore(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	result := svc.GetStore()
	if result != store {
		t.Error("GetStore() should return the same store")
	}
}

// TestService_NewServiceWrapper tests NewServiceWrapper method.
func TestService_NewServiceWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	cfg := config.DefaultConfig()
	wrapper := svc.NewServiceWrapper(cfg)
	if wrapper == nil {
		t.Fatal("NewServiceWrapper() returned nil")
	}
}

// TestService_List tests List method.
func TestService_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Initially empty
	sessions, err := svc.List("")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Create a session
	svc.Create("test-session", nil, "test prompt")

	// Now should have 1 session
	sessions, err = svc.List("")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

// TestService_Get tests Get method.
func TestService_Get(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session
	created, _ := svc.Create("test-session", nil, "test prompt")

	// Get the session
	session, err := svc.Get(created.ID, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if session.Name != "test-session" {
		t.Errorf("Expected name 'test-session', got '%s'", session.Name)
	}
}

// TestService_Get_NotFound tests Get with non-existent ID.
func TestService_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	_, err := svc.Get("non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestService_Create tests Create method.
func TestService_Create(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	subs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Source: "feishu", Active: true},
	}

	session, err := svc.Create("my-session", subs, "system prompt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.Name != "my-session" {
		t.Errorf("Expected name 'my-session', got '%s'", session.Name)
	}
	if session.SystemPrompt != "system prompt" {
		t.Errorf("Expected system prompt 'system prompt', got '%s'", session.SystemPrompt)
	}
	if len(session.Subscriptions) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(session.Subscriptions))
	}
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
}

// TestService_Update tests Update method.
func TestService_Update(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session
	created, _ := svc.Create("test-session", nil, "original prompt")

	// Update the session
	newPrompt := "updated prompt"
	err := svc.Update(created.ID, &newPrompt, nil, "")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	session, _ := svc.Get(created.ID, "")
	if session.SystemPrompt != "updated prompt" {
		t.Errorf("Expected updated prompt, got '%s'", session.SystemPrompt)
	}
}

// TestService_Update_Subscriptions tests updating subscriptions.
func TestService_Update_Subscriptions(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session
	created, _ := svc.Create("test-session", nil, "prompt")

	// Update subscriptions
	newSubs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Active: true},
	}
	err := svc.Update(created.ID, nil, &newSubs, "")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	session, _ := svc.Get(created.ID, "")
	if len(session.Subscriptions) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(session.Subscriptions))
	}
}

// TestService_Update_NotFound tests Update with non-existent ID.
func TestService_Update_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	prompt := "test"
	err := svc.Update("non-existent", &prompt, nil, "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestService_Delete tests Delete method.
func TestService_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session
	created, _ := svc.Create("test-session", nil, "prompt")

	// Delete the session
	ctx := context.Background()
	err := svc.Delete(ctx, created.ID, "")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = svc.Get(created.ID, "")
	if err == nil {
		t.Error("Expected error when getting deleted session")
	}
}

// TestService_AddSubscription tests AddSubscription method.
func TestService_AddSubscription(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session
	created, _ := svc.Create("test-session", nil, "prompt")

	// Add subscription
	sub := model.Subscription{
		ID:      "sub1",
		Trigger: "alert",
		Source:  "feishu",
		Active:  true,
	}
	err := svc.AddSubscription(created.ID, sub, "")
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}

	// Verify
	session, _ := svc.Get(created.ID, "")
	if len(session.Subscriptions) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(session.Subscriptions))
	}
}

// TestService_AddSubscription_NotFound tests AddSubscription with non-existent session.
func TestService_AddSubscription_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	sub := model.Subscription{ID: "sub1"}
	err := svc.AddSubscription("non-existent", sub, "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestService_RemoveSubscription tests RemoveSubscription method.
func TestService_RemoveSubscription(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session with subscription
	subs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Active: true},
		{ID: "sub2", Trigger: "notify", Active: true},
	}
	created, _ := svc.Create("test-session", subs, "prompt")

	// Remove subscription
	err := svc.RemoveSubscription(created.ID, "sub1", "")
	if err != nil {
		t.Fatalf("RemoveSubscription() error = %v", err)
	}

	// Verify
	session, _ := svc.Get(created.ID, "")
	if len(session.Subscriptions) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(session.Subscriptions))
	}
	if session.Subscriptions[0].ID != "sub2" {
		t.Errorf("Expected remaining subscription to be sub2, got %s", session.Subscriptions[0].ID)
	}
}

// TestService_RemoveSubscription_NotFound tests RemoveSubscription with non-existent subscription.
func TestService_RemoveSubscription_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session without subscription
	created, _ := svc.Create("test-session", nil, "prompt")

	// Try to remove non-existent subscription
	err := svc.RemoveSubscription(created.ID, "non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent subscription")
	}
}

// TestService_RouteEvent tests RouteEvent method.
func TestService_RouteEvent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create sessions with subscriptions
	subs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Source: "feishu", Active: true},
	}
	created, _ := svc.Create("test-session", subs, "prompt")

	// Route an event that matches
	sessionID, err := svc.RouteEvent("feishu", "alert: something happened", "")
	if err != nil {
		t.Fatalf("RouteEvent() error = %v", err)
	}
	if sessionID != created.ID {
		t.Errorf("Expected session ID '%s', got '%s'", created.ID, sessionID)
	}
}

// TestService_RouteEvent_NoMatch tests RouteEvent with no matching session.
func TestService_RouteEvent_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	// Create a session with non-matching subscription
	subs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Source: "slack", Active: true},
	}
	svc.Create("test-session", subs, "prompt")

	// Route an event from different source
	sessionID, err := svc.RouteEvent("feishu", "alert: something happened", "")
	if err != nil {
		t.Fatalf("RouteEvent() error = %v", err)
	}
	if sessionID != "" {
		t.Errorf("Expected empty session ID, got '%s'", sessionID)
	}
}

// TestService_GetMessages tests GetMessages method.
func TestService_GetMessages(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewJSONLStore(tmpDir)
	svc := NewService(store)

	ctx := context.Background()

	// Create a session
	created, _ := svc.Create("test-session", nil, "prompt")

	// Get messages (should be empty)
	messages, err := svc.GetMessages(ctx, created.ID, 10, "")
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

// TestFilterSubscriptions tests filterSubscriptions helper.
func TestFilterSubscriptions(t *testing.T) {
	subs := []model.Subscription{
		{ID: "sub1", Active: true},
		{ID: "sub2", Active: false},
		{ID: "sub3", Active: true},
	}

	result := filterSubscriptions(subs, func(s model.Subscription) bool {
		return s.Active
	})

	if len(result) != 2 {
		t.Errorf("Expected 2 active subscriptions, got %d", len(result))
	}
}

// TestFilterSubscriptions_Empty tests filterSubscriptions with empty input.
func TestFilterSubscriptions_Empty(t *testing.T) {
	result := filterSubscriptions(nil, func(s model.Subscription) bool {
		return true
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(result))
	}
}
