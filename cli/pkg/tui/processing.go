package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// ProcessingState represents the current state of processing.
type ProcessingState int

const (
	ProcessingStateIdle ProcessingState = iota
	ProcessingStateActive
	ProcessingStateDone
)

// processingTickMsg triggers a ProcessingWidget re-render.
type processingTickMsg struct{}

var processingFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProcessingWidget displays an animated spinner with elapsed time and tool info.
type ProcessingWidget struct {
	state       ProcessingState
	startTime   time.Time
	endTime     time.Time
	activeTools map[string]string // toolName → params
	frame       int
	rendered    string
}

// NewProcessingWidget creates a new ProcessingWidget.
func NewProcessingWidget() *ProcessingWidget {
	return &ProcessingWidget{
		state:       ProcessingStateIdle,
		activeTools: make(map[string]string),
	}
}

// Start begins processing animation with an initial tool.
// Returns the initial rendered string and a tick command.
func (w *ProcessingWidget) Start(toolName string) (string, tea.Cmd) {
	w.state = ProcessingStateActive
	w.startTime = time.Now()
	w.activeTools = make(map[string]string)
	w.activeTools[toolName] = ""
	w.frame = 0
	w.rendered = w.renderFrame()
	return w.rendered, w.tickCmd()
}

// AddTool adds another tool to the active tools list.
func (w *ProcessingWidget) AddTool(toolName, params string) string {
	if w.state != ProcessingStateActive {
		return w.rendered
	}
	w.activeTools[toolName] = params
	w.rendered = w.renderFrame()
	return w.rendered
}

// UpdateTool sets the tool name for single-tool display (replaces all active tools).
func (w *ProcessingWidget) UpdateTool(toolName string) string {
	if w.state != ProcessingStateActive {
		return w.rendered
	}
	w.activeTools = map[string]string{toolName: ""}
	w.rendered = w.renderFrame()
	return w.rendered
}

// RemoveTool removes a tool from the active tools list.
func (w *ProcessingWidget) RemoveTool(toolName string) string {
	if w.state != ProcessingStateActive {
		return w.rendered
	}
	delete(w.activeTools, toolName)
	w.rendered = w.renderFrame()
	return w.rendered
}

// Stop completes processing animation and returns final string.
func (w *ProcessingWidget) Stop() string {
	if w.state != ProcessingStateActive {
		return ""
	}
	w.state = ProcessingStateDone
	w.endTime = time.Now()
	clear(w.activeTools)
	w.rendered = ""
	return ""
}

// Update processes a tick message and returns new rendered text + next Cmd.
func (w *ProcessingWidget) Update(msg tea.Msg) (string, tea.Cmd) {
	if w.state != ProcessingStateActive {
		return w.rendered, nil
	}

	if _, ok := msg.(processingTickMsg); ok {
		w.frame = (w.frame + 1) % len(processingFrames)
		w.rendered = w.renderFrame()
		return w.rendered, w.tickCmd()
	}

	return w.rendered, nil
}

// tickCmd returns a Cmd that fires after ProcessingWidgetTickInterval to trigger the next frame.
func (w *ProcessingWidget) tickCmd() tea.Cmd {
	return tea.Tick(ProcessingWidgetTickInterval, func(t time.Time) tea.Msg {
		return processingTickMsg{}
	})
}

// renderFrame renders the current animation frame with elapsed time and tool info.
func (w *ProcessingWidget) renderFrame() string {
	spinner := processingFrames[w.frame]
	label := i18n.T("tui.processing_label")
	elapsed := formatDuration(time.Since(w.startTime))

	if len(w.activeTools) == 0 {
		return fmt.Sprintf("  %s %s  %s", spinner, label, elapsed)
	}

	toolNames := make([]string, 0, len(w.activeTools))
	for name := range w.activeTools {
		toolNames = append(toolNames, name)
	}
	slices.Sort(toolNames)
	toolDisplay := strings.Join(toolNames, ", ")

	if len(w.activeTools) > 1 {
		return fmt.Sprintf("  %s %s (%d)  ⚡ %s  %s", spinner, label, len(w.activeTools), toolDisplay, elapsed)
	}
	return fmt.Sprintf("  %s %s  ⚡ %s  %s", spinner, label, toolDisplay, elapsed)
}

// IsActive returns true if processing animation is running.
func (w *ProcessingWidget) IsActive() bool {
	return w.state == ProcessingStateActive && len(w.activeTools) > 0
}

// Rendered returns the current rendered string.
func (w *ProcessingWidget) Rendered() string {
	return w.rendered
}

// Reset resets the widget to idle state.
func (w *ProcessingWidget) Reset() {
	w.state = ProcessingStateIdle
	w.rendered = ""
	clear(w.activeTools)
	w.frame = 0
}
