package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

// WaitingState represents the current state of waiting.
type WaitingState int

const (
	WaitingStateIdle WaitingState = iota
	WaitingStateActive
)

// waitingTickMsg triggers a WaitingWidget re-render.
type waitingTickMsg struct{}

// WaitingWidget displays an animated spinner while waiting for LLM response.
// Used in two scenarios:
// 1. After user submits input (waiting for LLM to start responding)
// 2. After tool execution completes (analyzing tool result, waiting for next LLM response)
type WaitingWidget struct {
	state     WaitingState
	startTime time.Time
	text      string // Display text: "Waiting for response..." or "Analyzing tool result..."
	frame     int
	rendered  string
}

// NewWaitingWidget creates a new WaitingWidget.
func NewWaitingWidget() *WaitingWidget {
	return &WaitingWidget{
		state: WaitingStateIdle,
	}
}

// Start begins waiting animation with display text.
// Returns the initial rendered string and a tick command.
func (w *WaitingWidget) Start(text string) (string, tea.Cmd) {
	w.state = WaitingStateActive
	w.startTime = time.Now()
	w.text = text
	w.frame = 0
	w.rendered = w.renderFrame()
	return w.rendered, w.tickCmd()
}

// StartAndRender begins waiting animation and returns rendered string + tick Cmd.
// Convenience method similar to ThinkingWidget.StartAndRender.
func (w *WaitingWidget) StartAndRender(text string) (string, tea.Cmd) {
	return w.Start(text)
}

// Stop completes waiting animation and clears state.
func (w *WaitingWidget) Stop() string {
	if w.state != WaitingStateActive {
		return ""
	}
	w.state = WaitingStateIdle
	w.text = ""
	w.rendered = ""
	return ""
}

// Update processes a tick message and returns new rendered text + next Cmd.
func (w *WaitingWidget) Update(msg tea.Msg) (string, tea.Cmd) {
	if w.state != WaitingStateActive {
		return w.rendered, nil
	}

	if _, ok := msg.(waitingTickMsg); ok {
		w.frame = (w.frame + 1) % len(processingFrames)
		w.rendered = w.renderFrame()
		return w.rendered, w.tickCmd()
	}

	return w.rendered, nil
}

// tickCmd returns a Cmd that fires after ProcessingWidgetTickInterval to trigger the next frame.
func (w *WaitingWidget) tickCmd() tea.Cmd {
	return tea.Tick(ProcessingWidgetTickInterval, func(t time.Time) tea.Msg {
		return waitingTickMsg{}
	})
}

// renderFrame renders the current animation frame with elapsed time.
func (w *WaitingWidget) renderFrame() string {
	spinner := processingFrames[w.frame]
	elapsed := formatDuration(time.Since(w.startTime))
	return fmt.Sprintf("  %s %s  %s", spinner, w.text, elapsed)
}

// IsActive returns true if waiting animation is running.
func (w *WaitingWidget) IsActive() bool {
	return w.state == WaitingStateActive
}

// Rendered returns the current rendered string.
func (w *WaitingWidget) Rendered() string {
	return w.rendered
}

// Reset resets the widget to idle state.
func (w *WaitingWidget) Reset() {
	w.state = WaitingStateIdle
	w.text = ""
	w.rendered = ""
	w.frame = 0
}