package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestProcessingWidget_StartAndActive(t *testing.T) {
	w := NewProcessingWidget()

	rendered, cmd := w.Start("file_read")

	if !w.IsActive() {
		t.Error("ProcessingWidget should be active after Start()")
	}
	if rendered == "" {
		t.Error("Start() should return non-empty rendered string")
	}
	if cmd == nil {
		t.Error("Start() should return a tick command")
	}
}

func TestProcessingWidget_UpdateTool(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	rendered := w.UpdateTool("bash")

	if w.activeTools["bash"] == "" && len(w.activeTools) != 1 {
		t.Errorf("UpdateTool() should set bash as active tool, got %v", w.activeTools)
	}
	if rendered == "" {
		t.Error("UpdateTool() should return rendered string")
	}
}

func TestProcessingWidget_Stop(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	result := w.Stop()

	if w.IsActive() {
		t.Error("ProcessingWidget should not be active after Stop()")
	}
	if result != "" {
		t.Errorf("Stop() should return empty string, got %q", result)
	}
}

func TestProcessingWidget_Reset(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")
	w.Reset()

	if w.IsActive() {
		t.Error("ProcessingWidget should be idle after Reset()")
	}
	if len(w.activeTools) != 0 {
		t.Errorf("Reset() should clear activeTools, got %v", w.activeTools)
	}
}

func TestProcessingWidget_Rendered(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	rendered := w.Rendered()

	if rendered == "" {
		t.Error("Rendered() should return non-empty string when active")
	}
}

func TestProcessingWidget_AddRemoveTool(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	// Add a second tool
	w.AddTool("bash", "")
	if len(w.activeTools) != 2 {
		t.Errorf("AddTool() should have 2 active tools, got %d", len(w.activeTools))
	}
	if !w.IsActive() {
		t.Error("Widget should be active with multiple tools")
	}

	// Remove one tool
	w.RemoveTool("file_read")
	if len(w.activeTools) != 1 {
		t.Errorf("RemoveTool() should have 1 active tool, got %d", len(w.activeTools))
	}
	if _, exists := w.activeTools["bash"]; !exists {
		t.Error("RemoveTool() should keep the remaining tool")
	}

	// Remove last tool — should become inactive
	w.RemoveTool("bash")
	if w.IsActive() {
		t.Error("Widget should be inactive after removing last tool")
	}
}

func TestProcessingWidget_AddTool_Idle(t *testing.T) {
	w := NewProcessingWidget()
	// Don't start — stay idle
	w.AddTool("file_read", "")
	if w.IsActive() {
		t.Error("AddTool() on idle widget should not make it active")
	}
}

func TestProcessingWidget_RemoveTool_NotFound(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	w.RemoveTool("nonexistent")
	if len(w.activeTools) != 1 {
		t.Error("RemoveTool() of nonexistent tool should not affect active tools")
	}
}

func TestProcessingWidget_TickAdvance(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")
	initialFrame := w.frame

	// Process a tick
	rendered, cmd := w.Update(processingTickMsg{})

	if rendered == "" {
		t.Error("Tick should return rendered string")
	}
	if cmd == nil {
		t.Error("Tick should schedule next tick")
	}
	if w.frame == initialFrame {
		t.Error("Tick should advance frame counter")
	}
}

func TestProcessingWidget_IdleUpdate(t *testing.T) {
	w := NewProcessingWidget()
	// Don't start, stay idle

	rendered, cmd := w.Update(processingTickMsg{})

	if cmd != nil {
		t.Error("Idle widget should not schedule next tick")
	}
	_ = rendered
}

func TestProcessingWidget_UpdateNonTick(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	rendered, _ := w.Update(tea.WindowSizeMsg{})

	// Should return current rendered string without advancing frame
	if rendered == "" {
		t.Error("Update with non-tick msg should return rendered string")
	}
}

func TestGetSpinner(t *testing.T) {
	// Test that old getSpinner is no longer needed - ProcessingWidget provides better animation
	w := NewProcessingWidget()
	w.Start("")

	// ProcessingWidget should render with braille spinner frames
	rendered := w.Rendered()
	if rendered == "" {
		t.Error("ProcessingWidget should render spinner when active")
	}
}

func TestStatusBarWidget_BuildTokenUsageDisplay(t *testing.T) {
	state := NewState()
	state.SetTokenUsage(2000)
	state.SetTokenMax(8000)

	model := &Model{
		state: state,
	}
	w := NewStatusBarWidget(UIStyles{}, state, model)

	display := w.buildTokenUsageDisplay()

	if display == "" {
		t.Error("buildTokenUsageDisplay() returned empty string")
	}

	// Test with zero max tokens
	state.SetTokenMax(0)
	// Should not panic with division by zero due to Model check
}

func TestStatusBarWidget_BuildConfirmLine(t *testing.T) {
	cs := &ConfirmState{
		Waiting:  true,
		Selected: 0,
		Request: &ConfirmationRequest{
			Message: "Execute command?",
		},
	}

	model := &Model{confirmState: *cs}
	w := NewStatusBarWidget(UIStyles{}, &State{}, model)

	line := w.buildConfirmLine(&model.confirmState)

	if line == "" {
		t.Error("buildConfirmLine() returned empty string")
	}
}

func TestStatusBarWidget_BuildConfirmLine_NoRequest(t *testing.T) {
	cs := &ConfirmState{
		Waiting:  true,
		Selected: 0,
		Request:  nil,
	}

	model := &Model{confirmState: *cs}
	w := NewStatusBarWidget(UIStyles{}, &State{}, model)

	line := w.buildConfirmLine(&model.confirmState)

	// Should still return Yes/No options
	if line == "" {
		t.Error("buildConfirmLine() should return options even with nil request")
	}
}

func TestUpdateTokenUsage(t *testing.T) {
	m := Model{
		state:    NewState(),
		messages: NewMessageStore(100, "Test"),
	}

	m.state.SetShowTokens(true)
	m.messages.Add(MessageTypeUser, "Hello World", nil, nil, nil, UIStyles{})
	m.messages.Add(MessageTypeAssistant, "Hi there", nil, nil, nil, UIStyles{})

	// Update token usage
	m.updateTokenUsage()

	// Token usage should be > 0 (roughly chars / 2)
	usage := m.state.TokenUsage()
	if usage <= 0 {
		t.Errorf("updateTokenUsage() set usage to %d, expected > 0", usage)
	}
}

func TestUpdateTokenUsage_Disabled(t *testing.T) {
	m := Model{
		state:    NewState(),
		messages: NewMessageStore(100, "Test"),
	}

	m.state.SetShowTokens(false)
	m.messages.Add(MessageTypeUser, "Hello", nil, nil, nil, UIStyles{})

	// Update token usage (should not update when disabled)
	m.updateTokenUsage()

	// Token usage should still be 0
	if m.state.TokenUsage() != 0 {
		t.Errorf("updateTokenUsage() should not update when ShowTokens is false, got %d", m.state.TokenUsage())
	}
}

func TestStatusBarWidget_Render_Confirmation(t *testing.T) {
	model := &Model{
		confirmState: ConfirmState{
			Waiting:  true,
			Selected: 0,
			Request: &ConfirmationRequest{
				Message: "Confirm?",
			},
		},
	}
	w := NewStatusBarWidget(UIStyles{}, &State{}, model)

	status := w.Render(80)

	if status == "" {
		t.Error("StatusBarWidget.Render() returned empty string for confirmation")
	}
}

func TestStatusBarWidget_Render_Waiting(t *testing.T) {
	state := NewState()
	state.SetWaiting(true)

	model := &Model{}
	w := NewStatusBarWidget(UIStyles{}, state, model)

	status := w.Render(80)

	if status == "" {
		t.Error("StatusBarWidget.Render() returned empty string for waiting state")
	}
}

func TestStatusBarWidget_Render_Default(t *testing.T) {
	model := &Model{}
	w := NewStatusBarWidget(UIStyles{}, &State{}, model)

	status := w.Render(80)

	if status == "" {
		t.Error("StatusBarWidget.Render() returned empty string for default state")
	}
}

func TestFilterCommands(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)
	RegisterCommand(&Command{Name: "/help", Description: "Show help"})
	RegisterCommand(&Command{Name: "/history", Description: "Show history"})
	RegisterCommand(&Command{Name: "/exit", Description: "Exit"})

	allCmds := GetAvailableCommands()

	filtered := filterCommands("/h", allCmds)

	// Should return commands starting with /h
	for _, cmd := range filtered {
		if cmd.Name != "/help" && cmd.Name != "/history" {
			t.Errorf("filterCommands() returned unexpected command %s", cmd.Name)
		}
	}
}

func TestFilterCommands_EmptyFilter(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)
	RegisterCommand(&Command{Name: "/help", Description: "Show help"})
	RegisterCommand(&Command{Name: "/exit", Description: "Exit"})

	allCmds := GetAvailableCommands()

	filtered := filterCommands("", allCmds)

	// Should return all commands when filter is empty
	if len(filtered) < 2 {
		t.Errorf("filterCommands() returned %d commands, expected at least 2", len(filtered))
	}
}

func TestFilterCommands_NoMatch(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)
	RegisterCommand(&Command{Name: "/help", Description: "Show help"})

	allCmds := GetAvailableCommands()

	filtered := filterCommands("/nonexistent", allCmds)

	if len(filtered) != 0 {
		t.Errorf("filterCommands() returned %d commands for no match, expected 0", len(filtered))
	}
}

func TestCommandPopup_Render(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)
	RegisterCommand(&Command{Name: "/help", Description: "Show help"})
	RegisterCommand(&Command{Name: "/exit", Description: "Exit"})

	popup := &CommandPopupWidget{styles: UIStyles{}}
	popup.UpdateFilter("", GetAvailableCommands())

	rendered := popup.Render()

	if rendered == "" {
		t.Error("CommandPopupWidget.Render() returned empty string")
	}
}

func TestCommandPopup_NoCommands(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)

	popup := &CommandPopupWidget{styles: UIStyles{}}
	popup.UpdateFilter("/nonexistent", GetAvailableCommands())

	rendered := popup.Render()

	// Should show "No matching commands" message
	if rendered == "" {
		t.Error("CommandPopupWidget.Render() should return message for no commands")
	}
}

func TestCommandPopup_Navigation(t *testing.T) {
	// Reset command registry
	commandRegistry = make(map[string]*Command)
	for i := 0; i < 5; i++ {
		RegisterCommand(&Command{
			Name:        "/cmd" + string(rune('0'+i)),
			Description: "Command " + string(rune('0'+i)),
		})
	}

	popup := &CommandPopupWidget{styles: UIStyles{}}
	popup.UpdateFilter("", GetAvailableCommands())

	// Initial selection
	if popup.selected != 0 {
		t.Errorf("Initial selected should be 0, got %d", popup.selected)
	}

	// Move down
	popup.Down()
	if popup.selected != 1 {
		t.Errorf("After Down, selected should be 1, got %d", popup.selected)
	}

	// Move up
	popup.Up()
	if popup.selected != 0 {
		t.Errorf("After Up, selected should be 0, got %d", popup.selected)
	}

	// Wrap around
	popup.Up()
	if popup.selected != 4 {
		t.Errorf("After Up from 0, should wrap to 4, got %d", popup.selected)
	}

	// Has selection
	if !popup.HasSelection() {
		t.Error("HasSelection() should return true")
	}

	// Selected name
	name := popup.SelectedName()
	if name != "/cmd4" {
		t.Errorf("SelectedName() should be /cmd4, got %s", name)
	}
}

func TestCommandPopup_Height(t *testing.T) {
	popup := &CommandPopupWidget{styles: UIStyles{}}

	// Empty commands
	h := popup.Height()
	if h < 3 {
		t.Errorf("Height for empty popup should be >= 3, got %d", h)
	}
}

// keyPress creates a tea.KeyPressMsg for testing.
func keyPress(text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: text, Mod: mod}
}

func TestHandleConfirmKey_UppercaseYN(t *testing.T) {
	// Test that uppercase Y and N work the same as lowercase in confirmation
	responseCh := make(chan bool, 1)

	m := Model{
		state: NewState(),
		confirmState: ConfirmState{
			Waiting: true,
			Request: &ConfirmationRequest{
				Message:    "Test confirm?",
				ResponseCh: responseCh,
			},
		},
	}

	// Test uppercase Y
	model, cmd := m.handleConfirmKey(keyPress("Y", 0))
	updated := model.(Model)
	if updated.confirmState.Waiting {
		t.Error("confirmState.Waiting should be false after Y")
	}
	select {
	case confirmed := <-responseCh:
		if !confirmed {
			t.Error("Expected confirmed=true for Y")
		}
	default:
		t.Error("Expected response on channel for Y")
	}
	_ = cmd

	// Reset for N test
	m.confirmState.Waiting = true
	m.confirmState.Request.ResponseCh = make(chan bool, 1)

	// Test uppercase N
	model2, _ := m.handleConfirmKey(keyPress("N", 0))
	updated2 := model2.(Model)
	if updated2.confirmState.Waiting {
		t.Error("confirmState.Waiting should be false after N")
	}
	select {
	case confirmed := <-m.confirmState.Request.ResponseCh:
		if confirmed {
			t.Error("Expected confirmed=false for N")
		}
	default:
		t.Error("Expected response on channel for N")
	}
}

// Test ProcessingWidget timing-dependent animation
func TestProcessingWidget_TimingAnimation(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	// Process multiple ticks to verify animation advances
	frames := make(map[int]int)
	for i := 0; i < 5; i++ {
		_, _ = w.Update(processingTickMsg{})
		frames[w.frame]++
		time.Sleep(10 * time.Millisecond)
	}

	// Should have advanced through multiple frames
	if len(frames) < 2 {
		t.Errorf("ProcessingWidget should advance frames over multiple ticks, saw %d unique frames", len(frames))
	}
}
