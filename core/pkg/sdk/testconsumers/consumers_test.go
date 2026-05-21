package testconsumers_test

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/core/pkg/sdk/testconsumers"
	"github.com/oneliang/aura/core/pkg/sdk/testrecorder"
	"github.com/oneliang/aura/shared/pkg/events"
)

func TestCLIConsumer_Process(t *testing.T) {
	cli := testconsumers.NewCLIConsumer()

	tests := []struct {
		name     string
		event    testrecorder.TestEvent
		expected string
	}{
		{
			name: "thinking_start",
			event: testrecorder.TestEvent{
				Type: events.EventTypeThinkingStart,
			},
			expected: "🤔 Thinking...\n",
		},
		{
			name: "response",
			event: testrecorder.TestEvent{
				Type:    events.EventTypeResponse,
				Content: "Hello!",
			},
			expected: "Aura: Hello!\n",
		},
		{
			name: "error",
			event: testrecorder.TestEvent{
				Type:    events.EventTypeError,
				Content: "Something went wrong",
			},
			expected: "❌ Error: Something went wrong\n",
		},
		{
			name: "tool_start",
			event: testrecorder.TestEvent{
				Type: events.EventTypeToolStart,
				Extra: map[string]any{
					"tool": "calculator",
				},
			},
			expected: "🔧 Tool starting: calculator\n",
		},
		{
			name: "tool_end",
			event: testrecorder.TestEvent{
				Type: events.EventTypeToolEnd,
				Extra: map[string]any{
					"result": "4",
				},
			},
			expected: "✅ Tool complete: 4\n",
		},
		{
			name: "done",
			event: testrecorder.TestEvent{
				Type: events.EventTypeDone,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cli.Process(tt.event)
			if result != tt.expected {
				t.Errorf("Process() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCLIConsumer_ProcessAll(t *testing.T) {
	cli := testconsumers.NewCLIConsumer()

	events := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse, Content: "Hello!"},
		{Type: events.EventTypeDone},
	}

	output := cli.ProcessAll(events)

	if !strings.Contains(output, "Aura: Hello!") {
		t.Errorf("Output should contain response: %s", output)
	}
	if !strings.Contains(output, "Thinking") {
		t.Errorf("Output should contain thinking: %s", output)
	}
}

func TestTUIAdapter_ConvertSDKEvent(t *testing.T) {
	adapter := testconsumers.NewTUIAdapter()

	ev := testrecorder.TestEvent{
		Type:      events.EventTypeResponse,
		Content:   "Hello",
		Extra:     map[string]any{"key": "value"},
		RequestID: "req-123",
	}

	result := adapter.ConvertSDKEvent(ev)

	if result.Type != events.EventTypeResponse {
		t.Errorf("Type = %v, want %v", result.Type, events.EventTypeResponse)
	}
	if result.Content != "Hello" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello")
	}
	if result.Extra["key"] != "value" {
		t.Errorf("Extra[key] = %v, want %v", result.Extra["key"], "value")
	}
	if result.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", result.RequestID, "req-123")
	}
}

func TestTUIConsumer_Process(t *testing.T) {
	tui := testconsumers.NewTUIConsumer()

	ev := testrecorder.TestEvent{
		Type:    events.EventTypeResponse,
		Content: "Hello",
	}

	err := tui.Process(ev)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	events := tui.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestTUIConsumer_ProcessAll(t *testing.T) {
	tui := testconsumers.NewTUIConsumer()

	testEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse, Content: "Hello"},
		{Type: events.EventTypeDone},
	}

	err := tui.ProcessAll(testEvents)
	if err != nil {
		t.Fatalf("ProcessAll() error = %v", err)
	}

	tuiEvents := tui.GetEvents()
	if len(tuiEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(tuiEvents))
	}

	if !tui.HasEventType(events.EventTypeResponse) {
		t.Error("Should have response event")
	}
}

func TestTUIConsumer_ProcessInvalidEvent(t *testing.T) {
	tui := testconsumers.NewTUIConsumer()

	// Event with empty type should fail validation
	ev := testrecorder.TestEvent{
		Type:    "",
		Content: "Invalid",
	}

	err := tui.Process(ev)
	if err == nil {
		t.Error("Expected error for invalid event, got nil")
	}
}

func TestServerConsumer_Process(t *testing.T) {
	server := testconsumers.NewServerConsumer()

	ev := testrecorder.TestEvent{
		Type:    events.EventTypeResponse,
		Content: "Hello",
	}

	result := server.Process(ev)

	if !strings.Contains(result, "event: response") {
		t.Errorf("Should contain SSE event type: %s", result)
	}
	if !strings.Contains(result, "data: Hello") {
		t.Errorf("Should contain SSE data: %s", result)
	}
}

func TestServerConsumer_ProcessAll(t *testing.T) {
	server := testconsumers.NewServerConsumer()

	events := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse, Content: "Hello"},
		{Type: events.EventTypeDone},
	}

	output := server.ProcessAll(events)

	if !strings.Contains(output, "event: thinking_start") {
		t.Errorf("Should contain thinking_start: %s", output)
	}
	if !strings.Contains(output, "event: response") {
		t.Errorf("Should contain response: %s", output)
	}
	if !strings.Contains(output, "data: Hello") {
		t.Errorf("Should contain data: %s", output)
	}
}

func TestServerConsumer_HasSSEFormat(t *testing.T) {
	server := testconsumers.NewServerConsumer()

	events := []testrecorder.TestEvent{
		{Type: events.EventTypeResponse, Content: "Hello"},
	}

	server.ProcessAll(events)

	if !server.HasSSEFormat() {
		t.Error("Output should be in SSE format")
	}
}

func TestTestAllConsumers(t *testing.T) {
	events := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse, Content: "Hello"},
		{Type: events.EventTypeDone},
	}

	result, err := testconsumers.TestAllConsumers(events)
	if err != nil {
		t.Fatalf("TestAllConsumers() error = %v", err)
	}

	// Validate CLI
	if !strings.Contains(result.CLI, "Aura: Hello") {
		t.Errorf("CLI should contain response: %s", result.CLI)
	}

	// Validate TUI
	if len(result.TUI.GetEvents()) != 3 {
		t.Errorf("TUI should have 3 events, got %d", len(result.TUI.GetEvents()))
	}

	// Validate Server
	if !strings.Contains(result.Server, "event: response") {
		t.Errorf("Server should contain response event: %s", result.Server)
	}
}

func TestFilterByType(t *testing.T) {
	testEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse, Content: "Hello"},
		{Type: events.EventTypeToolStart},
		{Type: events.EventTypeResponse, Content: "World"},
		{Type: events.EventTypeDone},
	}

	responses := testrecorder.FilterByType(testEvents, events.EventTypeResponse)
	if len(responses) != 2 {
		t.Errorf("Expected 2 response events, got %d", len(responses))
	}

	tools := testrecorder.FilterByType(testEvents, events.EventTypeToolStart)
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool event, got %d", len(tools))
	}
}

func TestCountByType(t *testing.T) {
	testEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeResponse},
		{Type: events.EventTypeResponse},
		{Type: events.EventTypeDone},
	}

	count := testrecorder.CountByType(testEvents, events.EventTypeResponse)
	if count != 2 {
		t.Errorf("Expected 2 response events, got %d", count)
	}

	doneCount := testrecorder.CountByType(testEvents, events.EventTypeDone)
	if doneCount != 1 {
		t.Errorf("Expected 1 done event, got %d", doneCount)
	}
}
