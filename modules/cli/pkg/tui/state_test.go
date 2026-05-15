package tui

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	state := NewState()

	if state == nil {
		t.Fatal("NewState() returned nil")
	}
}

func TestState_Dimensions(t *testing.T) {
	state := NewState()

	// Test width
	state.SetWidth(80)
	if state.Width() != 80 {
		t.Errorf("Width() = %d, want 80", state.Width())
	}

	// Test height
	state.SetHeight(24)
	if state.Height() != 24 {
		t.Errorf("Height() = %d, want 24", state.Height())
	}
}

func TestState_ProcessingState(t *testing.T) {
	state := NewState()

	// Test waiting
	state.SetWaiting(true)
	if !state.Waiting() {
		t.Error("Waiting() should be true")
	}

	state.SetWaiting(false)
	if state.Waiting() {
		t.Error("Waiting() should be false")
	}

	// Test current tool
	state.SetCurrentTool("file_read")
	if state.CurrentTool() != "file_read" {
		t.Errorf("CurrentTool() = %q, want %q", state.CurrentTool(), "file_read")
	}

	// Test start time
	now := time.Now()
	state.SetStartTime(now)
	if !state.StartTime().Equal(now) {
		t.Errorf("StartTime() = %v, want %v", state.StartTime(), now)
	}

	// Test debug mode
	state.SetDebugMode(true)
	if !state.DebugMode() {
		t.Error("DebugMode() should be true")
	}
}

func TestState_TokenUsage(t *testing.T) {
	state := NewState()

	state.SetTokenUsage(500)
	if state.TokenUsage() != 500 {
		t.Errorf("TokenUsage() = %d, want 500", state.TokenUsage())
	}

	state.SetTokenMax(8000)
	if state.TokenMax() != 8000 {
		t.Errorf("TokenMax() = %d, want 8000", state.TokenMax())
	}

	state.SetShowTokens(true)
	if !state.ShowTokens() {
		t.Error("ShowTokens() should be true")
	}
}

func TestState_CommandCompletion(t *testing.T) {
	state := NewState()

	state.SetShowCommands(true)
	if !state.ShowCommands() {
		t.Error("ShowCommands() should be true")
	}

	state.SetCommandFilter("help")
	if state.CommandFilter() != "help" {
		t.Errorf("CommandFilter() = %q, want %q", state.CommandFilter(), "help")
	}

	state.SetCommandSelected(2)
	if state.CommandSelected() != 2 {
		t.Errorf("CommandSelected() = %d, want 2", state.CommandSelected())
	}
}

func TestState_Concurrent(t *testing.T) {
	state := NewState()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			state.SetWidth(i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = state.Width()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

func TestState_AllGettersSetters(t *testing.T) {
	state := NewState()

	// Test all dimension setters/getters
	state.SetWidth(100)
	state.SetHeight(30)

	if state.Width() != 100 || state.Height() != 30 {
		t.Error("Dimension setters/getters mismatch")
	}

	// Test all processing setters/getters
	state.SetWaiting(true)
	state.SetCurrentTool("test_tool")
	state.SetDebugMode(true)

	if !state.Waiting() ||
		state.CurrentTool() != "test_tool" || !state.DebugMode() {
		t.Error("Processing setters/getters mismatch")
	}

	// Test token setters/getters
	state.SetTokenUsage(1000)
	state.SetTokenMax(4000)
	state.SetShowTokens(true)

	if state.TokenUsage() != 1000 || state.TokenMax() != 4000 || !state.ShowTokens() {
		t.Error("Token setters/getters mismatch")
	}

	// Test command setters/getters
	state.SetShowCommands(true)
	state.SetCommandFilter("/help")
	state.SetCommandSelected(1)

	if !state.ShowCommands() || state.CommandFilter() != "/help" || state.CommandSelected() != 1 {
		t.Error("Command setters/getters mismatch")
	}
}
