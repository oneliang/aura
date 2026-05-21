package tui

import (
	"strings"
	"testing"
)

func TestPlanWidget_HandleCreate(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})

	widget.HandleCreate("Test goal", []string{"Step 1", "Step 2", "Step 3"})

	if !widget.visible {
		t.Error("Expected widget to be visible")
	}
	if widget.goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got %q", widget.goal)
	}
	if len(widget.steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(widget.steps))
	}
	for i, step := range widget.steps {
		if step.Description != "Step "+string(rune('1'+i)) {
			t.Errorf("Step %d description = %q, want 'Step %d'", i, step.Description, i+1)
		}
	}
}

func TestPlanWidget_HandleCreate_Empty(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})

	widget.HandleCreate("", []string{})

	if widget.visible {
		t.Error("Expected widget to not be visible with no steps")
	}
	if len(widget.steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(widget.steps))
	}
}

func TestPlanWidget_Reset(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})
	widget.HandleCreate("goal", []string{"A"})

	widget.Reset()

	if widget.visible {
		t.Error("Expected visible to be false after reset")
	}
	if widget.goal != "" {
		t.Error("Expected goal to be empty after reset")
	}
	if widget.steps != nil {
		t.Error("Expected steps to be nil after reset")
	}
}

func TestPlanWidget_Render(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})
	widget.HandleCreate("goal", []string{"First", "Second", "Third"})

	rendered := widget.Render()

	if !strings.Contains(rendered, "1. First") {
		t.Error("Expected first step in rendered output")
	}
	if !strings.Contains(rendered, "2. Second") {
		t.Error("Expected second step in rendered output")
	}
	if !strings.Contains(rendered, "3. Third") {
		t.Error("Expected third step in rendered output")
	}
}

func TestPlanWidget_Render_Empty(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})

	rendered := widget.Render()

	if rendered != "" {
		t.Errorf("Expected empty render, got %q", rendered)
	}
}

func TestPlanWidget_Render_NotVisible(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})
	widget.visible = false
	widget.steps = []PlanStep{{Description: "test"}}

	rendered := widget.Render()

	if rendered != "" {
		t.Errorf("Expected empty render when not visible, got %q", rendered)
	}
}

func TestPlanWidget_RenderStyled(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})
	widget.HandleCreate("goal", []string{"A", "B"})

	rendered := widget.RenderStyled()

	if !strings.Contains(rendered, "A") {
		t.Error("Expected step A in rendered output")
	}
	if !strings.Contains(rendered, "B") {
		t.Error("Expected step B in rendered output")
	}
	if !strings.Contains(rendered, "Plan") {
		t.Error("Expected 'Plan' title in rendered output")
	}
}

func TestPlanWidget_RenderStyled_Empty(t *testing.T) {
	widget := NewPlanWidget(UIStyles{})

	rendered := widget.RenderStyled()

	if rendered != "" {
		t.Errorf("Expected empty render, got %q", rendered)
	}
}
