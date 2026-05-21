package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// ThinkingState represents the current state of thinking.
type ThinkingState int

const (
	ThinkingStateIdle ThinkingState = iota
	ThinkingStateActive
	ThinkingStateDone
)

// thinkingTickMsg triggers a ThinkingWidget re-render.
type thinkingTickMsg struct{}

// waveFrames is a flowing Braille dot wave animation.
// Creates a sense of "thinking pulse" — dots flowing left to right.
var waveFrames = []string{
	"⢀⠀", "⡀⠀", "⠄⠀", "⠂⠀", "⠁⠀", "⠈⠀", "⠐⠀", "⠠⠀",
}

// ThinkingWidget manages the thinking indicator independently.
// It does not depend on event ordering - the UI controls its own state.
// Update() is single-threaded (guaranteed by Bubble Tea), so no mutex needed.
type ThinkingWidget struct {
	state     ThinkingState
	startTime time.Time
	endTime   time.Time
	rendered  string
	cleared   bool
	frame     int // animation frame counter
}

// NewThinkingWidget creates a new thinking widget.
func NewThinkingWidget() *ThinkingWidget {
	return &ThinkingWidget{
		state: ThinkingStateIdle,
	}
}

// Start records the thinking start time.
// Called when Engine sends EventTypeThinkingStart event.
// Does not return rendered content - use StartAndRender for that.
func (w *ThinkingWidget) Start() {
	w.state = ThinkingStateActive
	w.startTime = time.Now()
	w.cleared = false
	w.rendered = ""
	w.frame = 0
}

// StartAndRender starts thinking and returns "Thinking..." rendered content.
// Returns a tea.Cmd that triggers the next animation frame.
func (w *ThinkingWidget) StartAndRender() (string, tea.Cmd) {
	w.state = ThinkingStateActive
	w.startTime = time.Now()
	w.cleared = false
	w.frame = 0
	w.rendered = w.renderFrame()
	return w.rendered, w.tickCmd()
}

// tickCmd returns a Cmd that fires after ThinkingWidgetTickInterval to trigger the next frame.
func (w *ThinkingWidget) tickCmd() tea.Cmd {
	return tea.Tick(ThinkingWidgetTickInterval, func(t time.Time) tea.Msg {
		return thinkingTickMsg{}
	})
}

// Update processes a tick message and returns the new rendered text + next Cmd.
// Call this from Model.Update when receiving thinkingTickMsg.
func (w *ThinkingWidget) Update(msg tea.Msg) (string, tea.Cmd) {
	if w.state != ThinkingStateActive {
		return w.rendered, nil
	}

	if _, ok := msg.(thinkingTickMsg); ok {
		w.frame = (w.frame + 1) % len(waveFrames)
		w.rendered = w.renderFrame()
		return w.rendered, w.tickCmd()
	}

	return w.rendered, nil
}

// renderFrame renders the current animation frame with wave pattern and elapsed time.
func (w *ThinkingWidget) renderFrame() string {
	wave := waveFrames[w.frame]
	label := i18n.T("tui.thinking_label")
	elapsed := formatDuration(time.Since(w.startTime))

	return fmt.Sprintf("  %s %s %s", wave, label, elapsed)
}

// Stop completes the thinking and returns the final rendered string.
// The thinking indicator is replaced with duration info.
func (w *ThinkingWidget) Stop() string {
	if w.state != ThinkingStateActive {
		return ""
	}

	w.state = ThinkingStateDone
	w.endTime = time.Now()
	duration := w.endTime.Sub(w.startTime)

	// Replace thinking with thought duration
	w.rendered = fmt.Sprintf("  💭 %s", i18n.T("tui.thought", formatDuration(duration)))
	return w.rendered
}

// Complete stops thinking and returns the thought duration message.
// No ANSI escape codes - just returns the duration message.
// Use this when response arrives to show "Thought for Xs".
func (w *ThinkingWidget) Complete() string {
	if w.state != ThinkingStateActive {
		return ""
	}

	w.state = ThinkingStateDone
	w.endTime = time.Now()
	duration := w.endTime.Sub(w.startTime)

	// Show thought duration without ANSI escape
	w.rendered = fmt.Sprintf("  💭 %s", i18n.T("tui.thought", formatDuration(duration)))
	return w.rendered
}

// Clear marks the thinking as done without displaying anything.
// Used when thinking should be silently completed (e.g., for confirmation dialogs).
func (w *ThinkingWidget) Clear() {
	if w.state != ThinkingStateActive {
		return
	}

	w.cleared = true
	w.state = ThinkingStateDone
	w.endTime = time.Now()
}

// IsActive returns true if thinking is currently active.
func (w *ThinkingWidget) IsActive() bool {
	return w.state == ThinkingStateActive
}

// Duration returns the thinking duration if completed.
func (w *ThinkingWidget) Duration() time.Duration {
	if w.state == ThinkingStateDone {
		return w.endTime.Sub(w.startTime).Round(time.Millisecond)
	}
	return 0
}

// Rendered returns the current rendered string.
func (w *ThinkingWidget) Rendered() string {
	return w.rendered
}

// Reset resets the widget to idle state.
func (w *ThinkingWidget) Reset() {
	w.state = ThinkingStateIdle
	w.rendered = ""
	w.cleared = false
	w.frame = 0
}
