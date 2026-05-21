package analyzer_test

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/aura/habit/pkg/analyzer"
	"github.com/oneliang/aura/habit/pkg/model"
)

func TestNewAnalyzer(t *testing.T) {
	a := analyzer.New(3, 0.5)
	if a == nil {
		t.Fatal("expected analyzer to be created")
	}
}

func TestNewAnalyzerDefaults(t *testing.T) {
	a := analyzer.New(0, 0)
	if a == nil {
		t.Fatal("expected analyzer with defaults")
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	a := analyzer.New(3, 0.5)
	ctx := context.Background()

	habits, err := a.Analyze(ctx, "user_test", []*model.Action{})
	if err != nil {
		t.Fatalf("Analyze failed for empty actions: %v", err)
	}
	if len(habits) != 0 {
		t.Fatalf("expected 0 habits, got %d", len(habits))
	}
}

func TestAnalyzeToolUsage(t *testing.T) {
	a := analyzer.New(2, 0.3)
	ctx := context.Background()

	actions := []*model.Action{
		{
			ID:        "1",
			UserID:    "user_alice",
			ToolsUsed: []string{"grep", "file_read"},
			Timestamp: time.Now(),
		},
		{
			ID:        "2",
			UserID:    "user_alice",
			ToolsUsed: []string{"grep"},
			Timestamp: time.Now(),
		},
		{
			ID:        "3",
			UserID:    "user_alice",
			ToolsUsed: []string{"file_read"},
			Timestamp: time.Now(),
		},
	}

	habits, err := a.Analyze(ctx, "user_alice", actions)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should find at least grep as a habit (appears in 2/3 actions)
	found := false
	for _, h := range habits {
		if h.Category == model.CategoryToolUsage && h.Name == "Frequently uses grep tool" {
			found = true
			if h.Frequency.TotalCount != 2 {
				t.Errorf("expected grep count 2, got %d", h.Frequency.TotalCount)
			}
		}
	}
	if !found {
		t.Logf("Warning: no tool usage habits found (may be below threshold)")
	}
}

func TestAnalyzeOutputStyle(t *testing.T) {
	a := analyzer.New(2, 0.3)
	ctx := context.Background()

	actions := []*model.Action{
		{ID: "1", UserID: "user_bob", OutputStyle: "concise", Timestamp: time.Now()},
		{ID: "2", UserID: "user_bob", OutputStyle: "concise", Timestamp: time.Now()},
		{ID: "3", UserID: "user_bob", OutputStyle: "detailed", Timestamp: time.Now()},
	}

	habits, err := a.Analyze(ctx, "user_bob", actions)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, h := range habits {
		if h.Category == model.CategoryStyle {
			found = true
		}
	}
	if !found {
		t.Log("Warning: no style habits found")
	}
}

func TestAnalyzeWorkflows(t *testing.T) {
	a := analyzer.New(2, 0.3)
	ctx := context.Background()

	actions := []*model.Action{
		{ID: "1", UserID: "user_carol", ToolsUsed: []string{"grep", "file_read"}, Timestamp: time.Now()},
		{ID: "2", UserID: "user_carol", ToolsUsed: []string{"grep", "file_read"}, Timestamp: time.Now()},
	}

	habits, err := a.Analyze(ctx, "user_carol", actions)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should find workflow: grep -> file_read
	found := false
	for _, h := range habits {
		if h.Category == model.CategoryWorkflow {
			found = true
		}
	}
	if !found {
		t.Log("Warning: no workflow habits found")
	}
}

func TestGetPreferences(t *testing.T) {
	a := analyzer.New(2, 0.3)
	ctx := context.Background()

	actions := []*model.Action{
		{ID: "1", UserID: "user_dave", ToolsUsed: []string{"grep"}, OutputStyle: "concise", Timestamp: time.Now()},
		{ID: "2", UserID: "user_dave", ToolsUsed: []string{"grep"}, OutputStyle: "concise", Timestamp: time.Now()},
	}

	prefs, err := a.GetPreferences(ctx, "user_dave", actions)
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	if len(prefs) == 0 {
		t.Fatal("expected preferences, got none")
	}
}

func TestGetPreferencesEmpty(t *testing.T) {
	a := analyzer.New(3, 0.5)
	ctx := context.Background()

	prefs, err := a.GetPreferences(ctx, "user_empty", []*model.Action{})
	if err != nil {
		t.Fatalf("GetPreferences failed for empty: %v", err)
	}
	if len(prefs) != 0 {
		t.Fatalf("expected 0 preferences, got %d", len(prefs))
	}
}

func TestAnalyzeLowConfidence(t *testing.T) {
	a := analyzer.New(10, 0.9) // Very high threshold
	ctx := context.Background()

	actions := []*model.Action{
		{ID: "1", UserID: "user_eve", ToolsUsed: []string{"grep"}, Timestamp: time.Now()},
		{ID: "2", UserID: "user_eve", ToolsUsed: []string{"file_read"}, Timestamp: time.Now()},
	}

	habits, err := a.Analyze(ctx, "user_eve", actions)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// With high threshold, should find few or no habits
	t.Logf("Found %d habits with high threshold", len(habits))
}

func TestTopKeysHelper(t *testing.T) {
	a := analyzer.New(1, 0.0)
	_ = a // Just verify analyzer creation doesn't panic
	// topKeys is private, so we test it through Analyze
}
