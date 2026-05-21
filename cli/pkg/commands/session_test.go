package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/storage"
)

// setupTestSessionManager creates a temporary SessionManager for testing.
func setupTestSessionManager(t *testing.T) (*manager.SessionManager, func()) {
	t.Helper()

	tmpDir := t.TempDir()

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mgr, err := manager.NewSessionManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	cleanup := func() {
		mgr.Close()
	}

	return mgr, cleanup
}

func TestSessionCreateWithRole(t *testing.T) {
	mgr, cleanup := setupTestSessionManager(t)
	defer cleanup()

	// Test creating session without role
	session, err := mgr.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session.SystemPrompt != "" {
		t.Errorf("Expected empty SystemPrompt, got %q", session.SystemPrompt)
	}

	// Test creating session with non-existent role (should use empty prompt)
	session2, err := mgr.CreateSession("Test Session 2", nil, "nonexistent", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session2.SystemPrompt != "" {
		t.Errorf("Expected empty SystemPrompt for non-existent role, got %q", session2.SystemPrompt)
	}
}

func TestSessionUpdate(t *testing.T) {
	mgr, cleanup := setupTestSessionManager(t)
	defer cleanup()

	// Create a session
	session, err := mgr.CreateSession("Update Test", nil, "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Test update with custom prompt
	newPrompt := "You are a helpful assistant for testing."
	err = mgr.UpdateSession(session.ID, &newPrompt, nil, "")
	if err != nil {
		t.Fatalf("UpdateSession() error = %v", err)
	}

	// Verify update
	updated, err := mgr.GetSession(session.ID, "")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if updated.SystemPrompt != newPrompt {
		t.Errorf("SystemPrompt = %q, want %q", updated.SystemPrompt, newPrompt)
	}
}

func TestSessionUpdateWithRole(t *testing.T) {
	mgr, cleanup := setupTestSessionManager(t)
	defer cleanup()

	// Create a temporary role file
	tmpDir := t.TempDir()
	rolesDir := filepath.Join(tmpDir, ".aura", "roles")
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		t.Fatalf("Failed to create roles dir: %v", err)
	}

	roleContent := "You are a test role for unit testing."
	roleFile := filepath.Join(rolesDir, "test_unit_role.md")
	if err := os.WriteFile(roleFile, []byte(roleContent), 0644); err != nil {
		t.Fatalf("Failed to write role file: %v", err)
	}

	// Temporarily change home dir for testing
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer t.Setenv("HOME", origHome)

	// Create a session
	session, err := mgr.CreateSession("Role Test", nil, "", "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Test update with role
	roleName := "test_unit_role"
	err = mgr.UpdateSession(session.ID, nil, &roleName, "")
	if err != nil {
		t.Fatalf("UpdateSession() error = %v", err)
	}

	// Verify update
	updated, err := mgr.GetSession(session.ID, "")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if updated.SystemPrompt != roleContent {
		t.Errorf("SystemPrompt = %q, want %q", updated.SystemPrompt, roleContent)
	}
}
