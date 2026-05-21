// Package manager provides additional tests for wrapper.go.
package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSessionItem tests SessionItem methods.
func TestSessionItem(t *testing.T) {
	item := SessionItem{
		id:      "test-id",
		name:    "Test Name",
		created: 1700000000000,
		updated: 1700000001000,
		subs:    3,
		role:    "test-role",
	}

	if item.ID() != "test-id" {
		t.Errorf("ID() = %v, want test-id", item.ID())
	}
	if item.Name() != "Test Name" {
		t.Errorf("Name() = %v, want Test Name", item.Name())
	}
	if item.Created() != 1700000000000 {
		t.Errorf("Created() = %v, want 1700000000000", item.Created())
	}
	if item.Updated() != 1700000001000 {
		t.Errorf("Updated() = %v, want 1700000001000", item.Updated())
	}
	if item.Subs() != 3 {
		t.Errorf("Subs() = %v, want 3", item.Subs())
	}
	if item.Role() != "test-role" {
		t.Errorf("Role() = %v, want test-role", item.Role())
	}
}

// TestSubscriptionItem tests SubscriptionItem methods.
func TestSubscriptionItem(t *testing.T) {
	item := SubscriptionItem{
		id:      "sub-id",
		trigger: "告警",
		source:  "feishu",
		active:  true,
	}

	if item.ID() != "sub-id" {
		t.Errorf("ID() = %v, want sub-id", item.ID())
	}
	if item.Trigger() != "告警" {
		t.Errorf("Trigger() = %v, want 告警", item.Trigger())
	}
	if item.Source() != "feishu" {
		t.Errorf("Source() = %v, want feishu", item.Source())
	}
	if item.Active() != true {
		t.Errorf("Active() = %v, want true", item.Active())
	}
}

// TestSessionServiceWrapper tests SessionServiceWrapper methods.
func TestSessionServiceWrapper_Basic(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)
	if wrapper == nil {
		t.Fatal("NewSessionServiceWrapper() returned nil")
	}

	// Test GetStore
	store := wrapper.GetStore()
	if store == nil {
		t.Error("GetStore() returned nil")
	}

	// Test GetManager
	mgr := wrapper.GetManager()
	if mgr == nil {
		t.Error("GetManager() returned nil")
	}
}

// TestSessionServiceWrapper_ListSessions tests ListSessions method.
func TestSessionServiceWrapper_ListSessions(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create some sessions
	_, err := wrapper.CreateSession("Session 1", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	_, err = wrapper.CreateSession("Session 2", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// List sessions
	items, err := wrapper.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(items))
	}
}

// TestSessionServiceWrapper_CreateSession tests CreateSession method.
func TestSessionServiceWrapper_CreateSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if item == nil {
		t.Fatal("CreateSession() returned nil")
	}
	if item.Name() != "Test Session" {
		t.Errorf("Name() = %v, want Test Session", item.Name())
	}
}

// TestSessionServiceWrapper_GetSession tests GetSession method.
func TestSessionServiceWrapper_GetSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	created, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Get the session
	item, err := wrapper.GetSession(created.ID(), "")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if item.Name() != "Test Session" {
		t.Errorf("Name() = %v, want Test Session", item.Name())
	}

	// Get non-existent session
	_, err = wrapper.GetSession("non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestSessionServiceWrapper_GetMostRecentlyUpdatedSession tests GetMostRecentlyUpdatedSession method.
func TestSessionServiceWrapper_GetMostRecentlyUpdatedSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Empty list should return nil
	item, err := wrapper.GetMostRecentlyUpdatedSession("")
	if err != nil {
		t.Fatalf("GetMostRecentlyUpdatedSession() error = %v", err)
	}
	if item != nil {
		t.Error("Expected nil for empty list")
	}

	// Create sessions
	_, err = wrapper.CreateSession("Session 1", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	time.Sleep(time.Millisecond * 10)
	_, err = wrapper.CreateSession("Session 2", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Get most recent
	item, err = wrapper.GetMostRecentlyUpdatedSession("")
	if err != nil {
		t.Fatalf("GetMostRecentlyUpdatedSession() error = %v", err)
	}
	if item == nil {
		t.Error("Expected non-nil for non-empty list")
	}
}

// TestSessionServiceWrapper_DeleteSession tests DeleteSession method.
func TestSessionServiceWrapper_DeleteSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("To Delete", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Delete the session
	err = wrapper.DeleteSession(item.ID(), "")
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted
	_, err = wrapper.GetSession(item.ID(), "")
	if err == nil {
		t.Error("Expected error for deleted session")
	}
}

// TestSessionServiceWrapper_UpdateSessionRole tests UpdateSessionRole method.
func TestSessionServiceWrapper_UpdateSessionRole(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Update role
	err = wrapper.UpdateSessionRole(item.ID(), "test-role", "")
	if err != nil {
		t.Errorf("UpdateSessionRole() error = %v", err)
	}
}

// TestSessionServiceWrapper_AddSubscription tests AddSubscription method.
func TestSessionServiceWrapper_AddSubscription(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Add subscription
	err = wrapper.AddSubscription(item.ID(), "告警", "feishu", "")
	if err != nil {
		t.Errorf("AddSubscription() error = %v", err)
	}

	// Verify subscription was added
	subs, err := wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(subs))
	}
}

// TestSessionServiceWrapper_AddSubscription_NonExistentSession tests AddSubscription with non-existent session.
func TestSessionServiceWrapper_AddSubscription_NonExistentSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	err := wrapper.AddSubscription("non-existent", "告警", "feishu", "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestSessionServiceWrapper_RemoveSubscription tests RemoveSubscription method.
func TestSessionServiceWrapper_RemoveSubscription(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session with subscription
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	err = wrapper.AddSubscription(item.ID(), "告警", "feishu", "")
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}

	// Get subscription ID
	subs, err := wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("Expected 1 subscription, got %d", len(subs))
	}

	// Remove subscription
	err = wrapper.RemoveSubscription(item.ID(), subs[0].ID(), "")
	if err != nil {
		t.Errorf("RemoveSubscription() error = %v", err)
	}

	// Verify subscription was removed
	subs, err = wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subs))
	}
}

// TestSessionServiceWrapper_RemoveSubscription_NonExistent tests RemoveSubscription with non-existent subscription.
func TestSessionServiceWrapper_RemoveSubscription_NonExistent(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Remove non-existent subscription
	err = wrapper.RemoveSubscription(item.ID(), "non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent subscription")
	}
}

// TestSessionServiceWrapper_ToggleSubscriptionStatus tests ToggleSubscriptionStatus method.
func TestSessionServiceWrapper_ToggleSubscriptionStatus(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session with subscription
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	err = wrapper.AddSubscription(item.ID(), "告警", "feishu", "")
	if err != nil {
		t.Fatalf("AddSubscription() error = %v", err)
	}

	// Get subscription ID
	subs, err := wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("Expected 1 subscription, got %d", len(subs))
	}

	// Toggle status (should become inactive)
	err = wrapper.ToggleSubscriptionStatus(item.ID(), subs[0].ID(), "")
	if err != nil {
		t.Errorf("ToggleSubscriptionStatus() error = %v", err)
	}

	// Verify status was toggled
	subs, err = wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if subs[0].Active() {
		t.Error("Expected subscription to be inactive after toggle")
	}

	// Toggle again (should become active)
	err = wrapper.ToggleSubscriptionStatus(item.ID(), subs[0].ID(), "")
	if err != nil {
		t.Errorf("ToggleSubscriptionStatus() error = %v", err)
	}

	// Verify status was toggled back
	subs, err = wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if !subs[0].Active() {
		t.Error("Expected subscription to be active after second toggle")
	}
}

// TestSessionServiceWrapper_ToggleSubscriptionStatus_NonExistent tests ToggleSubscriptionStatus with non-existent subscription.
func TestSessionServiceWrapper_ToggleSubscriptionStatus_NonExistent(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Toggle non-existent subscription
	err = wrapper.ToggleSubscriptionStatus(item.ID(), "non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent subscription")
	}
}

// TestSessionServiceWrapper_GetSubscriptions tests GetSubscriptions method.
func TestSessionServiceWrapper_GetSubscriptions(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Get subscriptions (should be empty)
	subs, err := wrapper.GetSubscriptions(item.ID(), "")
	if err != nil {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subs))
	}

	// Get subscriptions for non-existent session
	_, err = wrapper.GetSubscriptions("non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// Test_getRoleFromPrompt tests getRoleFromPrompt function.
func Test_getRoleFromPrompt(t *testing.T) {
	// Empty prompt should return empty string
	result := getRoleFromPrompt("")
	if result != "" {
		t.Errorf("getRoleFromPrompt(\"\") = %q, want empty string", result)
	}

	// Non-empty prompt should return empty string (simplified implementation)
	result = getRoleFromPrompt("You are a helpful assistant")
	if result != "" {
		t.Errorf("getRoleFromPrompt() = %q, want empty string", result)
	}
}

// TestSessionServiceWrapper_ExportSession tests ExportSession method.
func TestSessionServiceWrapper_ExportSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Export session
	tmpFile := filepath.Join(t.TempDir(), "export.md")
	outputPath, err := wrapper.ExportSession(item.ID(), "Test Session", tmpFile, "")
	if err != nil {
		t.Errorf("ExportSession() error = %v", err)
	}
	if outputPath == "" {
		t.Error("ExportSession() returned empty path")
	}

	// Verify file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}
	if !strings.Contains(string(content), "# Session: Test Session") {
		t.Error("Exported file should contain session header")
	}
}

// TestSessionServiceWrapper_ExportSession_NoMessages tests ExportSession with no messages.
func TestSessionServiceWrapper_ExportSession_NoMessages(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	wrapper := NewSessionServiceWrapper(manager, manager.store)

	// Create a session
	item, err := wrapper.CreateSession("Test Session", "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Export session (no messages)
	tmpFile := filepath.Join(t.TempDir(), "export_empty.md")
	outputPath, err := wrapper.ExportSession(item.ID(), "Test Session", tmpFile, "")
	if err != nil {
		t.Errorf("ExportSession() error = %v", err)
	}

	// Verify file was created with header
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}
	if !strings.Contains(string(content), "Exported at:") {
		t.Error("Exported file should contain export timestamp")
	}
}
