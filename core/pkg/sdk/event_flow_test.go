// Package sdk_test provides integration tests that verify event flow across all consumers.
// These tests use the testrecorder and testconsumers packages to validate that
// CLI, TUI, and Server modes all process events correctly from the same event stream.
package sdk_test

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/core/pkg/sdk/testconsumers"
	"github.com/oneliang/aura/core/pkg/sdk/testrecorder"
	"github.com/oneliang/aura/shared/pkg/events"
)

// TestEventFlow_SimpleChat verifies event flow for a simple chat scenario.
// This test records a simple conversation event stream and validates all consumers.
func TestEventFlow_SimpleChat(t *testing.T) {
	// Simulate recorded events from a simple chat
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart, Content: "Analyzing input..."},
		{Type: events.EventTypeThinkingEnd, Content: ""},
		{Type: events.EventTypeResponse, Content: "Hello! How can I help you today?"},
		{Type: events.EventTypeDone, Content: ""},
	}

	// Test all consumers with the same event stream
	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate CLI output
	t.Run("CLI", func(t *testing.T) {
		if !contains(result.CLI, "Aura:") {
			t.Errorf("CLI output should contain 'Aura:': %s", result.CLI)
		}
		if !contains(result.CLI, "Hello!") {
			t.Errorf("CLI output should contain response content: %s", result.CLI)
		}
		if !contains(result.CLI, "Thinking") {
			t.Errorf("CLI output should contain thinking indicator: %s", result.CLI)
		}
	})

	// Validate TUI events
	t.Run("TUI", func(t *testing.T) {
		if len(result.TUI.GetEvents()) != 4 {
			t.Errorf("Expected 4 TUI events, got %d", len(result.TUI.GetEvents()))
		}
		if !result.TUI.HasEventType(events.EventTypeResponse) {
			t.Error("TUI should have response event")
		}
		if !result.TUI.HasEventType(events.EventTypeThinkingStart) {
			t.Error("TUI should have thinking_start event")
		}
	})

	// Validate Server SSE output
	t.Run("Server", func(t *testing.T) {
		if !contains(result.Server, "event: thinking_start") {
			t.Errorf("Server should have thinking_start: %s", result.Server)
		}
		if !contains(result.Server, "event: response") {
			t.Errorf("Server should have response: %s", result.Server)
		}
		if !contains(result.Server, "data: Hello!") {
			t.Errorf("Server should have response data: %s", result.Server)
		}
	})
}

// TestEventFlow_ToolExecution verifies event flow for tool execution scenario.
func TestEventFlow_ToolExecution(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart, Content: "Need to calculate..."},
		{Type: events.EventTypeThinkingEnd, Content: ""},
		{Type: events.EventTypeToolStart, Content: "calculator", Extra: map[string]any{"tool": "calculator", "params": map[string]any{"expr": "2+2"}}},
		{Type: events.EventTypeToolEnd, Content: "4", Extra: map[string]any{"result": "4"}},
		{Type: events.EventTypeResponse, Content: "The answer is 4."},
		{Type: events.EventTypeDone, Content: ""},
	}

	// Validate event order
	expectedOrder := []events.EventType{
		events.EventTypeThinkingStart,
		events.EventTypeToolStart,
		events.EventTypeToolEnd,
		events.EventTypeResponse,
		events.EventTypeDone,
	}
	err := testrecorder.AssertEventOrder(recordedEvents, expectedOrder)
	if err != nil {
		t.Errorf("Event order validation failed: %v", err)
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate CLI shows tool execution
	t.Run("CLI", func(t *testing.T) {
		if !contains(result.CLI, "Tool starting") {
			t.Errorf("CLI should show tool starting: %s", result.CLI)
		}
		if !contains(result.CLI, "Tool complete") {
			t.Errorf("CLI should show tool complete: %s", result.CLI)
		}
		if !contains(result.CLI, "4") {
			t.Errorf("CLI should show result '4': %s", result.CLI)
		}
	})

	// Validate TUI has tool events
	t.Run("TUI", func(t *testing.T) {
		if !result.TUI.HasEventType(events.EventTypeToolStart) {
			t.Error("TUI should have tool_start event")
		}
		if !result.TUI.HasEventType(events.EventTypeToolEnd) {
			t.Error("TUI should have tool_end event")
		}
	})

	// Validate Server SSE format
	t.Run("Server", func(t *testing.T) {
		if !contains(result.Server, "event: tool_start") {
			t.Errorf("Server should have tool_start: %s", result.Server)
		}
		if !contains(result.Server, "event: tool_end") {
			t.Errorf("Server should have tool_end: %s", result.Server)
		}
	})
}

// TestEventFlow_ErrorHandling verifies error event handling across consumers.
func TestEventFlow_ErrorHandling(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart, Content: ""},
		{Type: events.EventTypeError, Content: "Failed to connect to LLM"},
		{Type: events.EventTypeDone, Content: ""},
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate CLI shows error
	t.Run("CLI", func(t *testing.T) {
		if !contains(result.CLI, "Error:") {
			t.Errorf("CLI should show error: %s", result.CLI)
		}
		if !contains(result.CLI, "Failed to connect") {
			t.Errorf("CLI should show error details: %s", result.CLI)
		}
	})

	// Validate TUI has error event
	t.Run("TUI", func(t *testing.T) {
		if !result.TUI.HasEventType(events.EventTypeError) {
			t.Error("TUI should have error event")
		}
	})

	// Validate Server SSE format
	t.Run("Server", func(t *testing.T) {
		if !contains(result.Server, "event: error") {
			t.Errorf("Server should have error event: %s", result.Server)
		}
	})
}

// TestEventFlow_CommandMatched verifies command matching event flow.
func TestEventFlow_CommandMatched(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeCommandMatched, Content: "session_create", Extra: map[string]any{"command": "session_create", "params": map[string]any{"name": "test"}}},
		{Type: events.EventTypeCommandResult, Content: "Session created successfully"},
		{Type: events.EventTypeDone, Content: ""},
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate CLI shows command
	t.Run("CLI", func(t *testing.T) {
		if !contains(result.CLI, "Command matched") {
			t.Errorf("CLI should show command matched: %s", result.CLI)
		}
		if !contains(result.CLI, "Session created") {
			t.Errorf("CLI should show command result: %s", result.CLI)
		}
	})

	// Validate TUI has command events
	t.Run("TUI", func(t *testing.T) {
		if !result.TUI.HasEventType(events.EventTypeCommandMatched) {
			t.Error("TUI should have command_matched event")
		}
		if !result.TUI.HasEventType(events.EventTypeCommandResult) {
			t.Error("TUI should have command_result event")
		}
	})
}

// TestEventFlow_MultipleSteps verifies multi-step ReAct loop flow.
func TestEventFlow_MultipleSteps(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart, Content: "Step 1"},
		{Type: events.EventTypeToolStart, Content: "file_read", Extra: map[string]any{"tool": "file_read"}},
		{Type: events.EventTypeToolEnd, Content: "File content", Extra: map[string]any{"result": "File content"}},
		{Type: events.EventTypeThinkingStart, Content: "Step 2"},
		{Type: events.EventTypeToolStart, Content: "bash", Extra: map[string]any{"tool": "bash"}},
		{Type: events.EventTypeToolEnd, Content: "Command output", Extra: map[string]any{"result": "Command output"}},
		{Type: events.EventTypeResponse, Content: "Task completed"},
		{Type: events.EventTypeDone, Content: ""},
	}

	// Validate event order
	expectedOrder := []events.EventType{
		events.EventTypeThinkingStart,
		events.EventTypeToolStart,
		events.EventTypeToolEnd,
		events.EventTypeThinkingStart,
		events.EventTypeToolStart,
		events.EventTypeToolEnd,
		events.EventTypeResponse,
		events.EventTypeDone,
	}
	err := testrecorder.AssertEventOrder(recordedEvents, expectedOrder)
	if err != nil {
		t.Errorf("Event order validation failed: %v", err)
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate multiple thinking cycles
	t.Run("TUI", func(t *testing.T) {
		tuiEvents := result.TUI.GetEvents()

		thinkingCount := 0
		for _, ev := range tuiEvents {
			if ev.Type == events.EventTypeThinkingStart {
				thinkingCount++
			}
		}
		if thinkingCount != 2 {
			t.Errorf("Expected 2 thinking_start events, got %d", thinkingCount)
		}

		toolStartCount := 0
		for _, ev := range tuiEvents {
			if ev.Type == events.EventTypeToolStart {
				toolStartCount++
			}
		}
		if toolStartCount != 2 {
			t.Errorf("Expected 2 tool_start events, got %d", toolStartCount)
		}
	})
}

// TestEventFlow_EmptyStream verifies handling of empty event stream.
func TestEventFlow_EmptyStream(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate empty results
	t.Run("CLI", func(t *testing.T) {
		if result.CLI != "" {
			t.Errorf("Expected empty CLI output, got: %s", result.CLI)
		}
	})

	t.Run("TUI", func(t *testing.T) {
		if len(result.TUI.GetEvents()) != 0 {
			t.Errorf("Expected 0 TUI events, got %d", len(result.TUI.GetEvents()))
		}
	})

	t.Run("Server", func(t *testing.T) {
		if result.Server != "" {
			t.Errorf("Expected empty Server output, got: %s", result.Server)
		}
	})
}

// TestEventFlow_PartialStream verifies handling of partial event streams.
func TestEventFlow_PartialStream(t *testing.T) {
	// Stream that ends without done event (simulating interruption)
	recordedEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart, Content: ""},
		{Type: events.EventTypeResponse, Content: "Partial response"},
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Should still process available events
	t.Run("TUI", func(t *testing.T) {
		if len(result.TUI.GetEvents()) != 2 {
			t.Errorf("Expected 2 events, got %d", len(result.TUI.GetEvents()))
		}
	})
}

// TestEventFlow_ExtraData verifies extra data handling in events.
func TestEventFlow_ExtraData(t *testing.T) {
	recordedEvents := []testrecorder.TestEvent{
		{
			Type:    events.EventTypeToolStart,
			Content: "Executing tool",
			Extra: map[string]any{
				"tool":   "file_read",
				"params": map[string]any{"path": "/test/file.txt"},
			},
		},
	}

	result, err := testconsumers.TestAllConsumers(recordedEvents)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate Server includes extra data
	t.Run("Server", func(t *testing.T) {
		if !contains(result.Server, "file_read") {
			t.Errorf("Server should include tool name: %s", result.Server)
		}
	})
}

// Helper function to check substring presence.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
