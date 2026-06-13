package tui

import (
	"fmt"
	"strings"
)

// StatusBarWidget renders the status bar at the bottom of the screen.
type StatusBarWidget struct {
	styles UIStyles
	state  *State
	model  *Model
}

// NewStatusBarWidget creates a new StatusBarWidget.
func NewStatusBarWidget(styles UIStyles, state *State, model *Model) *StatusBarWidget {
	return &StatusBarWidget{
		styles: styles,
		state:  state,
		model:  model,
	}
}

// Render renders the status line.
func (w *StatusBarWidget) Render(width int) string {
	// Read confirmState from the live model pointer, not from the stale captured pointer.
	cs := &w.model.confirmState
	if cs.Waiting {
		return w.buildConfirmLine(cs)
	}

	// Build status line
	var status string

	// Show plan mode phase if in plan mode
	if w.state.InPlanMode() {
		phase := w.state.PlanModePhase()
		status = fmt.Sprintf("%s %s", phase.Icon(), phase.String())
		if w.state.Waiting() {
			status += " • " + ProcessingStatusText
		}
	} else if w.state.Waiting() {
		// Show processing/thinking status
		switch w.state.DisplayState() {
		case DisplayThinking:
			status = ThinkingStatusText
		case DisplayProcessing:
			status = ProcessingStatusText
		default:
			status = IdleStatusText
		}
	} else {
		status = IdleStatusText
	}

	// Add token usage if enabled
	if w.state.ShowTokens() && w.state.TokenMax() > 0 {
		status += w.buildTokenUsageDisplay()
	}

	return w.styles.Help.Render(status)
}

// Height returns the number of lines this widget occupies.
func (w *StatusBarWidget) Height() int {
	return 1
}

// buildConfirmLine builds the confirmation line with selection options.
// Dispatches to type-specific builders based on ConfirmationType.
func (w *StatusBarWidget) buildConfirmLine(cs *ConfirmState) string {
	if cs.Request != nil {
		switch cs.Request.Type {
		case ConfirmationPlanReview:
			return w.buildPlanReviewLine(cs)
		case ConfirmationQuestion:
			// Don't render in status bar if popup is showing (popup handles rendering)
			if w.model.questionPopup != nil && w.model.questionPopup.IsShowing() {
				return ""
			}
			return w.buildQuestionLine(cs)
		default:
			return w.buildSensitiveToolLine(cs)
		}
	}
	return w.buildSensitiveToolLine(cs)
}

// buildPlanReviewLine builds the plan review confirmation line.
// Plan content is already rendered in the chat area via PlanWidget.
func (w *StatusBarWidget) buildPlanReviewLine(cs *ConfirmState) string {
	var builder strings.Builder

	yesStyle := w.styles.CommandItem
	noStyle := w.styles.CommandItem
	if cs.Selected == 0 {
		yesStyle = w.styles.CommandItemSelected
	} else {
		noStyle = w.styles.CommandItemSelected
	}

	builder.WriteString(yesStyle.Render(" [Approve] "))
	builder.WriteString(noStyle.Render(" [Reject] "))

	if cs.Request != nil && cs.Request.PlanGoal != "" {
		builder.WriteString(w.styles.Help.Render(fmt.Sprintf("  Plan: %s  |  Enter confirm  Esc reject", cs.Request.PlanGoal)))
	} else {
		builder.WriteString(w.styles.Help.Render("  Enter confirm  Esc reject"))
	}

	return builder.String()
}

// buildSensitiveToolLine builds the sensitive tool confirmation line.
// Layout: [Yes] [No] Message  Enter confirm  Esc cancel
func (w *StatusBarWidget) buildSensitiveToolLine(cs *ConfirmState) string {
	var builder strings.Builder

	// Yes/No options first (left side)
	yesStyle := w.styles.CommandItem
	noStyle := w.styles.CommandItem
	if cs.Selected == 0 {
		yesStyle = w.styles.CommandItemSelected
	} else {
		noStyle = w.styles.CommandItemSelected
	}

	builder.WriteString(yesStyle.Render(" " + ConfirmYesLabel + " "))
	builder.WriteString(noStyle.Render(" " + ConfirmNoLabel + " "))

	// Message after buttons
	if cs.Request != nil {
		builder.WriteString(" ")
		builder.WriteString(w.styles.Help.Render(cs.Request.Message))
	}

	// Hint text
	builder.WriteString(w.styles.Help.Render("  " + ConfirmEnterHint))

	return builder.String()
}

// buildQuestionLine builds the question confirmation line.
// Layout depends on question type:
// - text: Input field with prompt
// - choice: Options to select
// - multi_choice: Multiple selectable options
func (w *StatusBarWidget) buildQuestionLine(cs *ConfirmState) string {
	var builder strings.Builder

	if cs.Request == nil {
		return ""
	}

	switch cs.Request.QuestionType {
	case QuestionTypeText:
		// Text input: show prompt with current input
		builder.WriteString(w.styles.Help.Render(cs.Request.Question + " "))
		builder.WriteString(w.styles.Command.Render(cs.TextInput))
		if cs.Request.DefaultAnswer != "" {
			builder.WriteString(w.styles.Help.Render(" [default: " + cs.Request.DefaultAnswer + "]"))
		}
		builder.WriteString(w.styles.Help.Render("  Enter submit  Esc cancel"))

	case QuestionTypeChoice:
		// Single choice: show options with selection highlight
		for i, opt := range cs.Request.Options {
			style := w.styles.CommandItem
			if cs.Selected == i {
				style = w.styles.CommandItemSelected
			}
			builder.WriteString(style.Render(" [" + opt.Label + "] "))
		}
		builder.WriteString(w.styles.Help.Render("  " + cs.Request.Question + "  Enter select  Esc cancel"))

	case QuestionTypeMultiChoice:
		// Multi-choice: show options with checkboxes
		for i, opt := range cs.Request.Options {
			style := w.styles.CommandItem
			selected := false
			for _, idx := range cs.SelectedOptions {
				if idx == i {
					selected = true
					break
				}
			}
			if selected {
				style = w.styles.CommandItemSelected
			}
			builder.WriteString(style.Render(" [" + opt.Label + "] "))
		}
		builder.WriteString(w.styles.Help.Render("  " + cs.Request.Question + "  Space toggle  Enter done  Esc cancel"))

	default:
		// Fallback: generic question display
		builder.WriteString(w.styles.Help.Render(cs.Request.Question))
		builder.WriteString(w.styles.Help.Render("  Enter confirm  Esc cancel"))
	}

	return builder.String()
}

// buildTokenUsageDisplay builds the token usage display for the status line.
// Uses cached value for efficiency - call updateTokenUsage() to refresh.
func (w *StatusBarWidget) buildTokenUsageDisplay() string {
	if w.model == nil {
		return ""
	}
	if w.state.TokenUsage() == w.model.cachedTokenUsage && w.model.cachedTokenDisplay != "" {
		return w.model.cachedTokenDisplay
	}
	// Recalculate if cache is stale
	percent := float64(w.state.TokenUsage()) / float64(w.state.TokenMax()) * 100
	return fmt.Sprintf("  |  Tokens: %d/%d (%.0f%%)", w.state.TokenUsage(), w.state.TokenMax(), percent)
}

// updateTokenUsage updates the token usage from message content.
// Kept on Model for state mutation; called via model reference.
func (m *Model) updateTokenUsage() {
	if !m.state.ShowTokens() {
		return
	}
	// Simple token estimation: ~4 characters per token
	totalChars := 0
	for _, msg := range m.messages.GetMessages() {
		totalChars += len(msg.Content)
	}
	usage := totalChars / 2
	m.state.SetTokenUsage(usage)

	// Update cached display
	m.cachedTokenUsage = usage
	m.cachedTokenDisplay = fmt.Sprintf("  |  Tokens: %d/%d (%.0f%%)", usage, m.state.TokenMax(), float64(usage)/float64(m.state.TokenMax())*100)
}
