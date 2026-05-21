package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/aura/habit/pkg/model"
	"github.com/oneliang/aura/habit/pkg/storage"
)

func setupTestStorage(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	tmpDir := t.TempDir()
	s, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	return s, tmpDir
}

func TestNew(t *testing.T) {
	s, _ := setupTestStorage(t)
	if s == nil {
		t.Fatal("expected storage to be created")
	}
}

func TestAppendAndGetActions(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_alice"

	action := &model.Action{
		ID:        "action-1",
		UserID:    userID,
		SessionID: "session-1",
		Input:     "Test input",
		ToolsUsed: []string{"grep", "file_read"},
		Timestamp: time.Now(),
	}

	err := s.AppendAction(ctx, userID, action)
	if err != nil {
		t.Fatalf("AppendAction failed: %v", err)
	}

	actions, err := s.GetActions(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetActions failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].ID != "action-1" {
		t.Errorf("expected action ID 'action-1', got %s", actions[0].ID)
	}
}

func TestGetActionsLimit(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_bob"

	// Add 5 actions
	for i := 0; i < 5; i++ {
		action := &model.Action{
			ID:        "action-" + string(rune('0'+i)),
			UserID:    userID,
			SessionID: "session-1",
			Timestamp: time.Now(),
		}
		s.AppendAction(ctx, userID, action)
	}

	// Get only 2 most recent
	actions, err := s.GetActions(ctx, userID, 2)
	if err != nil {
		t.Fatalf("GetActions failed: %v", err)
	}

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
}

func TestGetActionsEmptyUser(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()

	actions, err := s.GetActions(ctx, "nonexistent_user", 10)
	if err != nil {
		t.Fatalf("GetActions failed for empty user: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(actions))
	}
}

func TestSaveAndGetHabits(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_charlie"

	habits := []*model.Habit{
		{
			ID:       "habit-1",
			UserID:   userID,
			Name:     "Uses grep",
			Category: model.CategoryToolUsage,
		},
	}

	err := s.SaveHabits(ctx, userID, habits)
	if err != nil {
		t.Fatalf("SaveHabits failed: %v", err)
	}

	loaded, err := s.GetHabits(ctx, userID)
	if err != nil {
		t.Fatalf("GetHabits failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 habit, got %d", len(loaded))
	}
	if loaded[0].Name != "Uses grep" {
		t.Errorf("expected habit name 'Uses grep', got %s", loaded[0].Name)
	}
}

func TestGetHabitsEmptyUser(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()

	habits, err := s.GetHabits(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetHabits failed for empty user: %v", err)
	}
	if len(habits) != 0 {
		t.Fatalf("expected 0 habits, got %d", len(habits))
	}
}

func TestUserIsolation(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()

	userA := "user_alice"
	userB := "user_bob"

	// User A actions
	s.AppendAction(ctx, userA, &model.Action{
		ID:        "a-1",
		UserID:    userA,
		ToolsUsed: []string{"grep"},
		Timestamp: time.Now(),
	})

	// User B actions
	s.AppendAction(ctx, userB, &model.Action{
		ID:        "b-1",
		UserID:    userB,
		ToolsUsed: []string{"file_read"},
		Timestamp: time.Now(),
	})

	// Verify isolation
	actionsA, _ := s.GetActions(ctx, userA, 10)
	actionsB, _ := s.GetActions(ctx, userB, 10)

	if len(actionsA) != 1 || actionsA[0].ID != "a-1" {
		t.Errorf("User A actions not isolated: got %v", actionsA)
	}
	if len(actionsB) != 1 || actionsB[0].ID != "b-1" {
		t.Errorf("User B actions not isolated: got %v", actionsB)
	}
}

func TestSaveAndGetPreferences(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_dave"

	prefs := []*model.Preference{
		{
			ID:       "pref-1",
			UserID:   userID,
			Category: model.CategoryStyle,
			Name:     "output_style",
			Value:    "concise",
			Source:   model.SourceImplicit,
		},
	}

	err := s.SavePreferences(ctx, userID, prefs)
	if err != nil {
		t.Fatalf("SavePreferences failed: %v", err)
	}

	loaded, err := s.GetPreferences(ctx, userID)
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 preference, got %d", len(loaded))
	}
	if loaded[0].Value != "concise" {
		t.Errorf("expected preference value 'concise', got %s", loaded[0].Value)
	}
}

func TestCleanupOldActions(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_eve"

	// Recent action
	s.AppendAction(ctx, userID, &model.Action{
		ID:        "recent",
		UserID:    userID,
		Timestamp: time.Now(),
	})

	// Old action (31 days ago)
	oldAction := &model.Action{
		ID:        "old",
		UserID:    userID,
		Timestamp: time.Now().Add(-31 * 24 * time.Hour),
	}
	s.AppendAction(ctx, userID, oldAction)

	err := s.CleanupOldActions(ctx, userID, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldActions failed: %v", err)
	}

	actions, _ := s.GetActions(ctx, userID, 10)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action after cleanup, got %d", len(actions))
	}
	if actions[0].ID != "recent" {
		t.Errorf("expected recent action, got %s", actions[0].ID)
	}
}

func TestStoragePathIsolation(t *testing.T) {
	s, tmpDir := setupTestStorage(t)

	actionsPath := s.GetActionsPath("user_test")
	habitsPath := s.GetHabitsPath("user_test")

	expectedActionsPrefix := filepath.Join(tmpDir, "user_test", "habits", "actions.jsonl")
	expectedHabitsPrefix := filepath.Join(tmpDir, "user_test", "habits", "habits.jsonl")

	if actionsPath != expectedActionsPrefix {
		t.Errorf("expected actions path %s, got %s", expectedActionsPrefix, actionsPath)
	}
	if habitsPath != expectedHabitsPrefix {
		t.Errorf("expected habits path %s, got %s", expectedHabitsPrefix, habitsPath)
	}
}

func TestAppendActionNilContext(t *testing.T) {
	s, _ := setupTestStorage(t)
	// Using a cancelled context to verify error handling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	action := &model.Action{
		ID:        "test",
		UserID:    "user_test",
		Timestamp: time.Now(),
	}

	// Should still succeed since we don't use context for I/O cancellation
	err := s.AppendAction(ctx, "user_test", action)
	if err != nil {
		t.Fatalf("AppendAction failed with cancelled context: %v", err)
	}
}

func TestGetActionsEmptyFile(t *testing.T) {
	s, _ := setupTestStorage(t)
	ctx := context.Background()
	userID := "user_empty"

	// Create empty file
	path := s.GetActionsPath(userID)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(""), 0644)

	actions, err := s.GetActions(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetActions failed for empty file: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions from empty file, got %d", len(actions))
	}
}
