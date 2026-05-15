package model_test

import (
	"testing"
	"time"

	"github.com/oneliang/aura/habit/pkg/model"
)

func TestHabitConstants(t *testing.T) {
	if model.CategoryToolUsage != "tool_usage" {
		t.Errorf("expected CategoryToolUsage to be 'tool_usage', got %s", model.CategoryToolUsage)
	}
	if model.CategoryCommand != "command" {
		t.Errorf("expected CategoryCommand to be 'command', got %s", model.CategoryCommand)
	}
	if model.CategoryStyle != "style" {
		t.Errorf("expected CategoryStyle to be 'style', got %s", model.CategoryStyle)
	}
	if model.CategoryPreference != "preference" {
		t.Errorf("expected CategoryPreference to be 'preference', got %s", model.CategoryPreference)
	}
	if model.CategoryWorkflow != "workflow" {
		t.Errorf("expected CategoryWorkflow to be 'workflow', got %s", model.CategoryWorkflow)
	}
}

func TestTrendConstants(t *testing.T) {
	if model.TrendIncreasing != "increasing" {
		t.Errorf("expected TrendIncreasing to be 'increasing', got %s", model.TrendIncreasing)
	}
	if model.TrendStable != "stable" {
		t.Errorf("expected TrendStable to be 'stable', got %s", model.TrendStable)
	}
	if model.TrendDecreasing != "decreasing" {
		t.Errorf("expected TrendDecreasing to be 'decreasing', got %s", model.TrendDecreasing)
	}
}

func TestSourceConstants(t *testing.T) {
	if model.SourceExplicit != "explicit" {
		t.Errorf("expected SourceExplicit to be 'explicit', got %s", model.SourceExplicit)
	}
	if model.SourceImplicit != "implicit" {
		t.Errorf("expected SourceImplicit to be 'implicit', got %s", model.SourceImplicit)
	}
}

func TestHabitCreation(t *testing.T) {
	habit := &model.Habit{
		ID:       "test-habit-1",
		UserID:   "user_alice",
		Name:     "Uses grep frequently",
		Category: model.CategoryToolUsage,
		Pattern: model.Pattern{
			ToolUsage: []string{"grep"},
			Keywords:  []string{"search", "find"},
		},
		Frequency: model.Frequency{
			TotalCount: 10,
			Trend:      model.TrendIncreasing,
		},
		Confidence: 0.8,
		LastSeen:   time.Now(),
	}

	if habit.ID != "test-habit-1" {
		t.Errorf("expected ID 'test-habit-1', got %s", habit.ID)
	}
	if habit.UserID != "user_alice" {
		t.Errorf("expected UserID 'user_alice', got %s", habit.UserID)
	}
	if habit.Confidence != 0.8 {
		t.Errorf("expected Confidence 0.8, got %f", habit.Confidence)
	}
}

func TestActionCreation(t *testing.T) {
	action := &model.Action{
		ID:        "test-action-1",
		UserID:    "user_alice",
		SessionID: "session-1",
		Input:     "Search for TODO comments",
		ToolsUsed: []string{"grep", "file_read"},
		Duration:  time.Second * 2,
	}

	if len(action.ToolsUsed) != 2 {
		t.Errorf("expected 2 tools, got %d", len(action.ToolsUsed))
	}
	if action.Input != "Search for TODO comments" {
		t.Errorf("expected input 'Search for TODO comments', got %s", action.Input)
	}
}

func TestPreferenceCreation(t *testing.T) {
	pref := &model.Preference{
		ID:       "test-pref-1",
		UserID:   "user_alice",
		Category: model.CategoryStyle,
		Name:     "output_style",
		Value:    "concise",
		Source:   model.SourceImplicit,
	}

	if pref.Value != "concise" {
		t.Errorf("expected value 'concise', got %s", pref.Value)
	}
	if pref.Source != model.SourceImplicit {
		t.Errorf("expected source 'implicit', got %s", pref.Source)
	}
}
