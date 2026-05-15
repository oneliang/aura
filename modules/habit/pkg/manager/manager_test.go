package manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/aura/habit/pkg/manager"
	"github.com/oneliang/aura/habit/pkg/model"
)

func setupTestManager(t *testing.T) *manager.Manager {
	t.Helper()
	cfg := manager.DefaultConfig()
	cfg.MinOccurrences = 1
	cfg.ConfThreshold = 0.1
	m, err := manager.New(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return m
}

func TestNew(t *testing.T) {
	cfg := manager.DefaultConfig()
	m, err := manager.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if m == nil {
		t.Fatal("expected manager to be created")
	}
}

func TestRecordAction(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	action := &model.Action{
		ID:        "test-1",
		UserID:    "user_alice",
		SessionID: "session-1",
		ToolsUsed: []string{"grep"},
		Timestamp: time.Now(),
	}

	err := m.RecordAction(ctx, "user_alice", action)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}
}

func TestRecordActionEmptyUser(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	action := &model.Action{
		ID:        "test-2",
		ToolsUsed: []string{"grep"},
		Timestamp: time.Now(),
	}

	// Should silently skip in legacy mode
	err := m.RecordAction(ctx, "", action)
	if err != nil {
		t.Fatalf("RecordAction should skip empty user: %v", err)
	}
}

func TestGetHabits(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	// Record actions first
	for i := 0; i < 3; i++ {
		m.RecordAction(ctx, "user_bob", &model.Action{
			ID:        "bob-" + string(rune('0'+i)),
			UserID:    "user_bob",
			ToolsUsed: []string{"grep", "file_read"},
			Timestamp: time.Now(),
		})
	}

	habits, err := m.GetHabits(ctx, "user_bob")
	if err != nil {
		t.Fatalf("GetHabits failed: %v", err)
	}

	// Should have generated habits from actions
	t.Logf("Found %d habits for user_bob", len(habits))
}

func TestGetHabitsEmptyUser(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	habits, err := m.GetHabits(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetHabits failed for nonexistent user: %v", err)
	}
	if len(habits) != 0 {
		t.Fatalf("expected 0 habits, got %d", len(habits))
	}
}

func TestRefreshHabits(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	// Record actions
	for i := 0; i < 5; i++ {
		m.RecordAction(ctx, "user_carol", &model.Action{
			ID:        "carol-" + string(rune('0'+i)),
			UserID:    "user_carol",
			ToolsUsed: []string{"grep"},
			Timestamp: time.Now(),
		})
	}

	habits, err := m.RefreshHabits(ctx, "user_carol")
	if err != nil {
		t.Fatalf("RefreshHabits failed: %v", err)
	}

	if len(habits) == 0 {
		t.Log("No habits generated after refresh (may need more actions)")
	}
}

func TestGetPreferences(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	// Record actions
	for i := 0; i < 3; i++ {
		m.RecordAction(ctx, "user_dave", &model.Action{
			ID:          "dave-" + string(rune('0'+i)),
			UserID:      "user_dave",
			ToolsUsed:   []string{"grep"},
			OutputStyle: "concise",
			Timestamp:   time.Now(),
		})
	}

	prefs, err := m.GetPreferences(ctx, "user_dave")
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	t.Logf("Found %d preferences for user_dave", len(prefs))
}

func TestDeleteHabit(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()
	userID := "user_eve"

	// Record actions to generate habits
	m.RecordAction(ctx, userID, &model.Action{
		ID:        "eve-1",
		UserID:    userID,
		ToolsUsed: []string{"grep"},
		Timestamp: time.Now(),
	})

	// Get habits
	habits, err := m.GetHabits(ctx, userID)
	if err != nil {
		t.Fatalf("GetHabits failed: %v", err)
	}

	if len(habits) > 0 {
		habitID := habits[0].ID
		err = m.DeleteHabit(ctx, userID, habitID)
		if err != nil {
			t.Fatalf("DeleteHabit failed: %v", err)
		}

		// Verify deletion
		remaining, err := m.GetHabits(ctx, userID)
		if err != nil {
			t.Fatalf("GetHabits after delete failed: %v", err)
		}

		for _, h := range remaining {
			if h.ID == habitID {
				t.Fatal("habit should have been deleted")
			}
		}
	}
}

func TestCleanup(t *testing.T) {
	m := setupTestManager(t)
	ctx := context.Background()

	// Should not error even with no data
	err := m.Cleanup(ctx, "user_cleanup")
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := manager.DefaultConfig()

	if cfg.MinOccurrences != 3 {
		t.Errorf("expected MinOccurrences 3, got %d", cfg.MinOccurrences)
	}
	if cfg.ConfThreshold != 0.3 {
		t.Errorf("expected ConfThreshold 0.3, got %f", cfg.ConfThreshold)
	}
	if cfg.AnalysisLimit != 500 {
		t.Errorf("expected AnalysisLimit 500, got %d", cfg.AnalysisLimit)
	}
}
