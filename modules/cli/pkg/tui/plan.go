// Package tui provides the terminal user interface for Aura.
package tui

import (
	"fmt"
	"strings"
)

// PlanStep represents a single step in the plan widget.
type PlanStep struct {
	Description string
}

// PlanWidget displays the current plan as a checklist with completion state.
type PlanWidget struct {
	goal           string
	steps          []PlanStep
	completedSteps map[int]struct{} // completed step indexes (0-based)
	currentStep    int              // current executing step (0-based, -1 = none)
	visible        bool
	styles         UIStyles
}

// NewPlanWidget creates a new plan widget.
func NewPlanWidget(styles UIStyles) *PlanWidget {
	return &PlanWidget{styles: styles}
}

// HandleCreate initializes the widget from a plan_created event.
func (w *PlanWidget) HandleCreate(goal string, steps []string) {
	w.goal = goal
	w.steps = make([]PlanStep, 0, len(steps))
	for _, desc := range steps {
		w.steps = append(w.steps, PlanStep{Description: desc})
	}
	w.completedSteps = make(map[int]struct{})
	w.currentStep = -1
	w.visible = len(steps) > 0
}

// MarkStepCompleted marks a step as completed by 0-based index.
func (w *PlanWidget) MarkStepCompleted(stepIndex int) {
	if w.completedSteps == nil {
		w.completedSteps = make(map[int]struct{})
	}
	if stepIndex >= 0 && stepIndex < len(w.steps) {
		w.completedSteps[stepIndex] = struct{}{}
	}
}

// SetCurrentStep sets the currently executing step (0-based, -1 = none).
func (w *PlanWidget) SetCurrentStep(stepIndex int) {
	w.currentStep = stepIndex
}

// Reset clears the widget for a new interaction.
func (w *PlanWidget) Reset() {
	w.goal = ""
	w.steps = nil
	w.completedSteps = nil
	w.currentStep = -1
	w.visible = false
}

// Render returns the formatted plan list string with checkbox states.
func (w *PlanWidget) Render() string {
	if !w.visible || len(w.steps) == 0 {
		return ""
	}

	var b strings.Builder
	for i, step := range w.steps {
		checkbox := "[ ]"
		if _, ok := w.completedSteps[i]; ok {
			checkbox = "[x]"
		} else if i == w.currentStep {
			checkbox = "[•]" // Current step indicator
		}
		b.WriteString(fmt.Sprintf("  %s %d. %s", checkbox, i+1, step.Description))
		b.WriteByte('\n')
	}
	return b.String()
}

// RenderStyled returns the plan list with visual framing (border + title) and checkbox states.
func (w *PlanWidget) RenderStyled() string {
	if !w.visible || len(w.steps) == 0 {
		return ""
	}

	border := w.styles.PlanWidgetBorder.Render(strings.Repeat("─", TaskWidgetBorderWidth))
	var b strings.Builder
	b.WriteString(border)
	b.WriteByte('\n')

	// Show goal as title
	if w.goal != "" {
		b.WriteString(w.styles.PlanWidgetTitle.Render("  Plan: " + w.goal))
		b.WriteByte('\n')
	} else {
		b.WriteString(w.styles.PlanWidgetTitle.Render("  Plan"))
		b.WriteByte('\n')
	}

	// Show progress count
	completed := len(w.completedSteps)
	total := len(w.steps)
	progress := fmt.Sprintf("  Progress: %d/%d", completed, total)
	b.WriteString(w.styles.Help.Render(progress))
	b.WriteByte('\n')

	for i, step := range w.steps {
		checkbox := "[ ]"
		if _, ok := w.completedSteps[i]; ok {
			checkbox = "[x]"
		} else if i == w.currentStep {
			checkbox = "[•]" // Current step indicator (executing)
		}
		b.WriteString(fmt.Sprintf("  %s %d. %s", checkbox, i+1, step.Description))
		b.WriteByte('\n')
	}
	b.WriteString(border)
	return b.String()
}

// Progress returns the completion progress (completed/total).
func (w *PlanWidget) Progress() (completed, total int) {
	return len(w.completedSteps), len(w.steps)
}

// Goal returns the plan goal.
func (w *PlanWidget) Goal() string {
	return w.goal
}
