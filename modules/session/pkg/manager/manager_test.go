package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/utils"
)

// setupTestManager creates a temporary SessionManager for testing.
func setupTestManager(t *testing.T) (*SessionManager, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager, err := NewSessionManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	cleanup := func() {
		manager.Close()
	}

	return manager, cleanup
}

func TestNewSessionManager(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager, err := NewSessionManager(store, nil)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Error("Expected manager to be created")
	}

	if manager.router == nil {
		t.Error("Expected router to be initialized")
	}
}

func TestSessionManager_CreateSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	tests := []struct {
		name          string
		sessionName   string
		subscriptions []model.Subscription
		role          string
		wantErr       bool
	}{
		{
			name:        "basic session",
			sessionName: "Test Session",
			wantErr:     false,
		},
		{
			name:        "session with empty name",
			sessionName: "",
			wantErr:     false,
		},
		{
			name:        "session with subscriptions",
			sessionName: "Subscribed Session",
			subscriptions: []model.Subscription{
				{ID: "sub1", Source: "feishu", Trigger: "告警", Active: true},
			},
			wantErr: false,
		},
		{
			name:        "session with role",
			sessionName: "Role Session",
			role:        "nonexistent", // Will use default prompt since file doesn't exist
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := manager.CreateSession(tt.sessionName, tt.subscriptions, tt.role, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if session.ID == "" {
					t.Error("Expected session ID to be generated")
				}
				if session.Name != tt.sessionName {
					t.Errorf("Name = %v, want %v", session.Name, tt.sessionName)
				}
				if session.CreatedAt == 0 {
					t.Error("Expected CreatedAt to be set")
				}
				if session.UpdatedAt == 0 {
					t.Error("Expected UpdatedAt to be set")
				}
			}
		})
	}
}

func TestSessionManager_GetSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create a session
	created, err := manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the session
	got, err := manager.GetSession(created.ID, "")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %v, want %v", got.ID, created.ID)
	}
	if got.Name != created.Name {
		t.Errorf("Name = %v, want %v", got.Name, created.Name)
	}

	// Get non-existent session
	_, err = manager.GetSession("non-existent", "")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestSessionManager_ListSessions(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create multiple sessions
	expectedCount := 3
	for i := 0; i < expectedCount; i++ {
		_, err := manager.CreateSession(string(rune('A'+i)), nil, "", "")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// List sessions
	sessions, err := manager.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != expectedCount {
		t.Errorf("Expected %d sessions, got %d", expectedCount, len(sessions))
	}
}

func TestSessionManager_DeleteSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create a session
	session, err := manager.CreateSession("To Delete", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete the session
	err = manager.DeleteSession(session.ID, "")
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted
	_, err = manager.GetSession(session.ID, "")
	if err == nil {
		t.Error("Expected error getting deleted session")
	}
}

func TestSessionManager_DeleteSession_WithRunningAgent(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Note: SessionManager no longer manages agent lifecycle directly
	// Agent/runtime lifecycle is handled at a higher level (SDK or server layer)
	// This test verifies the basic delete flow works
	session, err := manager.CreateSession("With Agent", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete should work
	err = manager.DeleteSession(session.ID, "")
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted
	_, err = manager.GetSession(session.ID, "")
	if err == nil {
		t.Error("Expected error getting deleted session")
	}
}

func TestSessionManager_RouteEvent(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create sessions with different subscriptions
	_, err := manager.CreateSession("Feishu Session", []model.Subscription{
		{ID: "sub1", Source: "feishu", Trigger: "", Active: true},
	}, "", "")
	if err != nil {
		t.Fatalf("Failed to create feishu session: %v", err)
	}

	_, err = manager.CreateSession("Monitor Session", []model.Subscription{
		{ID: "sub2", Source: "", Trigger: "监控", Active: true},
	}, "", "")
	if err != nil {
		t.Fatalf("Failed to create monitor session: %v", err)
	}

	tests := []struct {
		name    string
		source  string
		content string
		wantID  string
	}{
		{
			name:    "match feishu source",
			source:  "feishu",
			content: "any message",
			wantID:  "Feishu Session",
		},
		{
			name:    "match trigger keyword",
			source:  "api",
			content: "系统监控告警",
			wantID:  "Monitor Session",
		},
		{
			name:    "no match",
			source:  "email",
			content: "random message",
			wantID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, err := manager.RouteEvent(tt.source, tt.content, "")
			if err != nil {
				t.Fatalf("RouteEvent() error = %v", err)
			}

			// Get the matched session to verify name
			if sessionID != "" {
				session, err := manager.GetSession(sessionID, "")
				if err != nil {
					t.Fatalf("Failed to get matched session: %v", err)
				}
				if session.Name != tt.wantID {
					t.Errorf("Matched session name = %v, want %v", session.Name, tt.wantID)
				}
			} else if tt.wantID != "" {
				t.Errorf("Expected to match %s, got empty session ID", tt.wantID)
			}
		})
	}
}

func TestSessionManager_Close(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create sessions
	_, err := manager.CreateSession("Session 1", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}
	_, err = manager.CreateSession("Session 2", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	// Close manager
	manager.Close()

	// Verify sessions still exist in storage
	sessions, err := manager.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions after close, got %d", len(sessions))
	}
}

func TestRandString(t *testing.T) {
	// Test that RandString generates expected length
	s := utils.MustRandString(10)
	if len(s) != 10 {
		t.Errorf("Expected length 10, got %d", len(s))
	}

	// Test that consecutive calls generate different values
	s1 := utils.MustRandString(10)
	s2 := utils.MustRandString(10)

	// They should likely be different (but not guaranteed)
	// This test is more about ensuring the function works
	if len(s1) != 10 || len(s2) != 10 {
		t.Error("Expected both strings to have length 10")
	}
}

func TestLoadRolePrompt(t *testing.T) {
	t.Run("empty role returns empty", func(t *testing.T) {
		result := loadRolePrompt("")
		if result != "" {
			t.Errorf("Expected empty string for empty role, got %q", result)
		}
	})

	t.Run("nonexistent role returns empty", func(t *testing.T) {
		result := loadRolePrompt("nonexistent_role_12345")
		if result != "" {
			t.Errorf("Expected empty string for nonexistent role, got %q", result)
		}
	})

	t.Run("existing role returns content", func(t *testing.T) {
		// Create a temporary roles directory
		tmpDir := t.TempDir()
		rolesDir := filepath.Join(tmpDir, ".aura", "roles")
		if err := os.MkdirAll(rolesDir, 0755); err != nil {
			t.Fatalf("Failed to create roles dir: %v", err)
		}

		// Create a test role file
		roleFile := filepath.Join(rolesDir, "test_role.md")
		testContent := "You are a test role."
		if err := os.WriteFile(roleFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to write role file: %v", err)
		}

		// Temporarily change home dir for testing
		origHome := os.Getenv("HOME")
		t.Setenv("HOME", tmpDir)
		defer t.Setenv("HOME", origHome)

		result := loadRolePrompt("test_role")
		if result != testContent {
			t.Errorf("Expected %q, got %q", testContent, result)
		}
	})

	t.Run("empty file returns empty", func(t *testing.T) {
		// Create a temporary roles directory
		tmpDir := t.TempDir()
		rolesDir := filepath.Join(tmpDir, ".aura", "roles")
		if err := os.MkdirAll(rolesDir, 0755); err != nil {
			t.Fatalf("Failed to create roles dir: %v", err)
		}

		// Create an empty role file
		roleFile := filepath.Join(rolesDir, "empty_role.md")
		if err := os.WriteFile(roleFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write role file: %v", err)
		}

		// Temporarily change home dir for testing
		origHome := os.Getenv("HOME")
		t.Setenv("HOME", tmpDir)
		defer t.Setenv("HOME", origHome)

		result := loadRolePrompt("empty_role")
		if result != "" {
			t.Errorf("Expected empty string for empty file, got %q", result)
		}
	})
}

func TestSessionManager_UpdateSession(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create a session
	session, err := manager.CreateSession("Update Test", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Run("update with custom prompt", func(t *testing.T) {
		newPrompt := "You are a custom assistant."
		err := manager.UpdateSession(session.ID, &newPrompt, nil, "")
		if err != nil {
			t.Fatalf("UpdateSession() error = %v", err)
		}

		// Verify the session was updated
		updated, err := manager.GetSession(session.ID, "")
		if err != nil {
			t.Fatalf("GetSession() error = %v", err)
		}
		if updated.SystemPrompt != newPrompt {
			t.Errorf("SystemPrompt = %v, want %v", updated.SystemPrompt, newPrompt)
		}
	})

	t.Run("update with role", func(t *testing.T) {
		// Create a temporary role file
		tmpDir := t.TempDir()
		rolesDir := filepath.Join(tmpDir, ".aura", "roles")
		if err := os.MkdirAll(rolesDir, 0755); err != nil {
			t.Fatalf("Failed to create roles dir: %v", err)
		}

		roleContent := "You are a test role."
		roleFile := filepath.Join(rolesDir, "update_test_role.md")
		if err := os.WriteFile(roleFile, []byte(roleContent), 0644); err != nil {
			t.Fatalf("Failed to write role file: %v", err)
		}

		// Temporarily change home dir for testing
		origHome := os.Getenv("HOME")
		t.Setenv("HOME", tmpDir)
		defer t.Setenv("HOME", origHome)

		roleName := "update_test_role"
		err := manager.UpdateSession(session.ID, nil, &roleName, "")
		if err != nil {
			t.Fatalf("UpdateSession() error = %v", err)
		}

		// Verify the session was updated
		updated, err := manager.GetSession(session.ID, "")
		if err != nil {
			t.Fatalf("GetSession() error = %v", err)
		}
		if updated.SystemPrompt != roleContent {
			t.Errorf("SystemPrompt = %v, want %v", updated.SystemPrompt, roleContent)
		}
	})

	t.Run("update non-existent session", func(t *testing.T) {
		prompt := "test prompt"
		err := manager.UpdateSession("non-existent", &prompt, nil, "")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})
}
