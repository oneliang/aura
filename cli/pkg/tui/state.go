package tui

import (
	"time"
)

// DisplayState represents the current UI display state.
// Update() is single-threaded (guaranteed by Bubble Tea), so no mutex needed.
type DisplayState int

const (
	DisplayIdle       DisplayState = iota // Idle, waiting for input
	DisplayWaiting                        // Waiting for LLM response to start
	DisplayThinking                       // LLM is streaming reasoning content
	DisplayProcessing                     // Tool execution
	DisplayConfirm                        // Waiting for user confirmation
)

// String returns a human-readable state name.
func (d DisplayState) String() string {
	switch d {
	case DisplayIdle:
		return "idle"
	case DisplayWaiting:
		return "waiting"
	case DisplayThinking:
		return "thinking"
	case DisplayProcessing:
		return "processing"
	case DisplayConfirm:
		return "confirm"
	default:
		return "unknown"
	}
}

// PlanModePhase represents the current phase in plan mode workflow.
type PlanModePhase int

const (
	PlanModePhaseNone    PlanModePhase = iota // Not in plan mode
	PlanModePhaseExplore                      // Phase 1: Exploration
	PlanModePhaseDesign                       // Phase 2: Design/Planning
	PlanModePhaseReview                       // Phase 3: Review/Approval
	PlanModePhaseExecute                      // Phase 4: Execution
	PlanModePhaseVerify                       // Phase 5: Verification
)

// String returns a human-readable phase name.
func (p PlanModePhase) String() string {
	switch p {
	case PlanModePhaseNone:
		return ""
	case PlanModePhaseExplore:
		return "Explore"
	case PlanModePhaseDesign:
		return "Design"
	case PlanModePhaseReview:
		return "Review"
	case PlanModePhaseExecute:
		return "Execute"
	case PlanModePhaseVerify:
		return "Verify"
	default:
		return ""
	}
}

// Icon returns an icon for the phase.
func (p PlanModePhase) Icon() string {
	switch p {
	case PlanModePhaseExplore:
		return "🔍"
	case PlanModePhaseDesign:
		return "📝"
	case PlanModePhaseReview:
		return "👁"
	case PlanModePhaseExecute:
		return "⚡"
	case PlanModePhaseVerify:
		return "✓"
	default:
		return ""
	}
}

// ToolStatus represents the status of a tool execution.
type ToolStatus int

const (
	ToolRunning ToolStatus = iota
	ToolDone
)

// ToolState tracks the state of a single tool execution.
type ToolState struct {
	Name      string
	Status    ToolStatus
	StartTime time.Time
	EndTime   time.Time
	Params    string
	Result    string
}

// IsRunning returns true if the tool is still running.
func (t *ToolState) IsRunning() bool {
	return t.Status == ToolRunning
}

// Duration returns the execution duration.
func (t *ToolState) Duration() time.Duration {
	if t.Status == ToolDone {
		return t.EndTime.Sub(t.StartTime).Round(time.Millisecond)
	}
	return time.Since(t.StartTime).Round(time.Millisecond)
}

// State represents UI state.
// Update() is single-threaded (guaranteed by Bubble Tea), so no mutex needed.
type State struct {
	// Dimensions
	width  int
	height int

	// Display state machine
	displayState DisplayState

	// Plan mode phase tracking
	planModePhase PlanModePhase

	// Processing state
	waiting   bool
	startTime time.Time
	debugMode bool

	// Concurrent tool execution (supports multiple tools)
	activeTools    map[string]*ToolState
	completedTools []*ToolState // Recently completed tools for display

	// Token usage tracking
	tokenUsage int
	tokenMax   int
	showTokens bool

	// Command completion
	showCommands    bool
	commandFilter   string
	commandSelected int
}

// NewState creates a new State with default values.
func NewState() *State {
	return &State{
		displayState:   DisplayIdle,
		planModePhase:  PlanModePhaseNone,
		activeTools:    make(map[string]*ToolState),
		completedTools: make([]*ToolState, 0),
	}
}

// --- Display State Machine ---

// DisplayState returns the current display state.
func (s *State) DisplayState() DisplayState {
	return s.displayState
}

// SetDisplayState sets the display state.
func (s *State) SetDisplayState(state DisplayState) {
	s.displayState = state
}

// TransitionTo transitions to a new display state.
// Can add validation logic here if needed.
func (s *State) TransitionTo(newState DisplayState) {
	s.displayState = newState
}

// --- Plan Mode Phase ---

// PlanModePhase returns the current plan mode phase.
func (s *State) PlanModePhase() PlanModePhase {
	return s.planModePhase
}

// SetPlanModePhase sets the plan mode phase.
func (s *State) SetPlanModePhase(phase PlanModePhase) {
	s.planModePhase = phase
}

// InPlanMode returns true if we're in any plan mode phase.
func (s *State) InPlanMode() bool {
	return s.planModePhase != PlanModePhaseNone
}

// --- Dimensions ---

// Width returns the terminal width.
func (s *State) Width() int {
	return s.width
}

// SetWidth sets the terminal width.
func (s *State) SetWidth(w int) {
	s.width = w
}

// Height returns the terminal height.
func (s *State) Height() int {
	return s.height
}

// SetHeight sets the terminal height.
func (s *State) SetHeight(h int) {
	s.height = h
}

// --- Processing State ---

// Waiting returns whether the agent is processing.
func (s *State) Waiting() bool {
	return s.waiting
}

// SetWaiting sets the processing state.
func (s *State) SetWaiting(w bool) {
	s.waiting = w
}

// StartTime returns the processing start time.
func (s *State) StartTime() time.Time {
	return s.startTime
}

// SetStartTime sets the processing start time.
func (s *State) SetStartTime(t time.Time) {
	s.startTime = t
}

// DebugMode returns whether debug mode is enabled.
func (s *State) DebugMode() bool {
	return s.debugMode
}

// SetDebugMode sets debug mode.
func (s *State) SetDebugMode(debug bool) {
	s.debugMode = debug
}

// --- Concurrent Tool State ---

// StartTool starts tracking a tool execution.
func (s *State) StartTool(name, params string) {
	s.activeTools[name] = &ToolState{
		Name:      name,
		Status:    ToolRunning,
		StartTime: time.Now(),
		Params:    params,
	}
	// Transition to processing
	s.displayState = DisplayProcessing
}

// EndTool marks a tool as completed.
func (s *State) EndTool(name, result string) {
	if tool, ok := s.activeTools[name]; ok {
		tool.Status = ToolDone
		tool.EndTime = time.Now()
		tool.Result = result
		// Move to completed list for display
		s.completedTools = append(s.completedTools, tool)
		// Remove from active map
		delete(s.activeTools, name)
	}
}

// GetActiveTools returns all currently running tools.
func (s *State) GetActiveTools() []*ToolState {
	tools := make([]*ToolState, 0, len(s.activeTools))
	for _, t := range s.activeTools {
		tools = append(tools, t)
	}
	return tools
}

// GetCompletedTools returns recently completed tools (limited to last MaxCompletedTools).
func (s *State) GetCompletedTools() []*ToolState {
	if len(s.completedTools) > MaxCompletedTools {
		return s.completedTools[len(s.completedTools)-MaxCompletedTools:]
	}
	return s.completedTools
}

// ClearCompletedTools clears the completed tools list.
func (s *State) ClearCompletedTools() {
	s.completedTools = make([]*ToolState, 0)
}

// HasActiveTools returns true if there are running tools.
func (s *State) HasActiveTools() bool {
	return len(s.activeTools) > 0
}

// CurrentTool returns the first active tool name (for backward compatibility).
func (s *State) CurrentTool() string {
	for _, t := range s.activeTools {
		if t.Status == ToolRunning {
			return t.Name
		}
	}
	return ""
}

// SetCurrentTool sets a single tool (for backward compatibility).
// Prefer StartTool/EndTool for concurrent tool support.
func (s *State) SetCurrentTool(tool string) {
	if tool != "" {
		s.StartTool(tool, "")
	} else {
		// Clear all active tools
		s.activeTools = make(map[string]*ToolState)
	}
}

// --- Token Usage ---

// TokenUsage returns the current token usage.
func (s *State) TokenUsage() int {
	return s.tokenUsage
}

// SetTokenUsage sets the token usage.
func (s *State) SetTokenUsage(usage int) {
	s.tokenUsage = usage
}

// TokenMax returns the maximum tokens.
func (s *State) TokenMax() int {
	return s.tokenMax
}

// SetTokenMax sets the maximum tokens.
func (s *State) SetTokenMax(max int) {
	s.tokenMax = max
}

// ShowTokens returns whether token display is enabled.
func (s *State) ShowTokens() bool {
	return s.showTokens
}

// SetShowTokens sets token display.
func (s *State) SetShowTokens(show bool) {
	s.showTokens = show
}

// --- Command Completion ---

// ShowCommands returns whether command list is showing.
func (s *State) ShowCommands() bool {
	return s.showCommands
}

// SetShowCommands sets command list visibility.
func (s *State) SetShowCommands(show bool) {
	s.showCommands = show
}

// CommandFilter returns the command filter string.
func (s *State) CommandFilter() string {
	return s.commandFilter
}

// SetCommandFilter sets the command filter string.
func (s *State) SetCommandFilter(filter string) {
	s.commandFilter = filter
}

// CommandSelected returns the selected command index.
func (s *State) CommandSelected() int {
	return s.commandSelected
}

// SetCommandSelected sets the selected command index.
func (s *State) SetCommandSelected(selected int) {
	s.commandSelected = selected
}

// --- Reset for new interaction ---

// ResetForNewInteraction resets state for a new interaction.
func (s *State) ResetForNewInteraction() {
	s.displayState = DisplayIdle
	s.waiting = false
	s.activeTools = make(map[string]*ToolState)
	s.completedTools = make([]*ToolState, 0)
}
