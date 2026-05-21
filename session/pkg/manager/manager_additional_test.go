// Package manager provides additional tests for the manager package.
package manager

import (
	"testing"
)

// TestSessionManager_SaveSession tests SaveSession method.
func TestSessionManager_SaveSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create a session first
	session, err := manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Modify and save the session
	session.Name = "Updated Session"
	err = manager.SaveSession(session)
	if err != nil {
		t.Errorf("SaveSession() returned error: %v", err)
	}

	// Verify the session was saved
	saved, err := manager.GetSession(session.ID, "")
	if err != nil {
		t.Fatalf("GetSession() returned error: %v", err)
	}
	if saved.Name != "Updated Session" {
		t.Errorf("Name = %v, want Updated Session", saved.Name)
	}
}

// TestSessionManager_GetRouter tests GetRouter method.
func TestSessionManager_GetRouter(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	router := manager.GetRouter()
	if router == nil {
		t.Error("GetRouter() returned nil")
	}
}
