package tui_test

import (
	"testing"
	"time"

	"github.com/oneliang/aura/cli/pkg/tui"
	"github.com/oneliang/aura/shared/pkg/events"
)

// TestAdapter_ConvertSDKEvent tests the Adapter.ConvertSDKEvent method.
// This verifies that SDK events are correctly converted to TUI chat events.
func TestAdapter_ConvertSDKEvent(t *testing.T) {
	adapter := tui.NewAdapter()

	tests := []struct {
		name            string
		sdkEvent        *mockSDKEvent
		expectedType    events.EventType
		expectedContent string
	}{
		{
			name: "thinking_start",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeThinkingStart,
				content: "Analyzing...",
			},
			expectedType:    tui.EventTypeThinkingStart,
			expectedContent: "Analyzing...",
		},
		{
			name: "thinking_end",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeThinkingEnd,
				content: "",
			},
			expectedType:    tui.EventTypeThinkingEnd,
			expectedContent: "",
		},
		{
			name: "action",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeAction,
				content: "Using tool",
			},
			expectedType:    tui.EventTypeAction,
			expectedContent: "Using tool",
		},
		{
			name: "result",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeResult,
				content: "Tool result",
			},
			expectedType:    tui.EventTypeResult,
			expectedContent: "Tool result",
		},
		{
			name: "response",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeResponse,
				content: "Hello!",
			},
			expectedType:    tui.EventTypeResponse,
			expectedContent: "Hello!",
		},
		{
			name: "response_chunk",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeResponseChunk,
				content: "Chunk",
			},
			expectedType:    tui.EventTypeResponseChunk,
			expectedContent: "Chunk",
		},
		{
			name: "error",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeError,
				content: "Something went wrong",
			},
			expectedType:    tui.EventTypeError,
			expectedContent: "Something went wrong",
		},
		{
			name: "step",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeStep,
				content: "Step 1",
				extra:   map[string]any{"step": 1},
			},
			expectedType:    tui.EventTypeStep,
			expectedContent: "Step 1",
		},
		{
			name: "tool_start",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeToolStart,
				content: "Starting tool",
				extra:   map[string]any{"tool": "bash"},
			},
			expectedType:    tui.EventTypeToolStart,
			expectedContent: "Starting tool",
		},
		{
			name: "tool_end",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeToolEnd,
				content: "Done",
				extra:   map[string]any{"result": "success"},
			},
			expectedType:    tui.EventTypeToolEnd,
			expectedContent: "Done",
		},
		{
			name: "confirmation_request",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeConfirmationRequest,
				content: "Confirm?",
				extra:   map[string]any{"tool": "bash"},
			},
			expectedType:    tui.EventTypeConfirmationRequest,
			expectedContent: "Confirm?",
		},
		{
			name: "command_matched",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeCommandMatched,
				content: "session_create",
				extra:   map[string]any{"command": "session_create"},
			},
			expectedType:    tui.EventTypeCommandMatched,
			expectedContent: "session_create",
		},
		{
			name: "command_result",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeCommandResult,
				content: "Session created",
			},
			expectedType:    tui.EventTypeCommandResult,
			expectedContent: "Session created",
		},
		{
			name: "done",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeDone,
				content: "",
			},
			expectedType:    tui.EventTypeDone,
			expectedContent: "",
		},
		{
			name: "task_create",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeTaskCreate,
				content: "Research project",
				extra:   map[string]any{"task_id": 1},
			},
			expectedType:    tui.EventTypeTaskCreate,
			expectedContent: "Research project",
		},
		{
			name: "task_update",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeTaskUpdate,
				content: "Research project",
				extra:   map[string]any{"task_id": 1, "status": "completed"},
			},
			expectedType:    tui.EventTypeTaskUpdate,
			expectedContent: "Research project",
		},
		{
			name: "task_list",
			sdkEvent: &mockSDKEvent{
				typ:     events.EventTypeTaskList,
				content: "",
				extra:   map[string]any{"tasks": []interface{}{}},
			},
			expectedType:    tui.EventTypeTaskList,
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.ConvertSDKEvent(tt.sdkEvent)

			if result.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", result.Type, tt.expectedType)
			}
			if result.Content != tt.expectedContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.expectedContent)
			}
			if result.RequestID != tt.sdkEvent.requestID {
				t.Errorf("RequestID = %q, want %q", result.RequestID, tt.sdkEvent.requestID)
			}
		})
	}
}

// TestAdapter_ConvertSDKEvent_ExtraData tests that extra data is preserved during conversion.
func TestAdapter_ConvertSDKEvent_ExtraData(t *testing.T) {
	adapter := tui.NewAdapter()

	extraData := map[string]any{
		"tool":   "file_read",
		"params": map[string]any{"path": "/test.txt"},
		"result": "success",
	}

	sdkEvent := &mockSDKEvent{
		typ:     events.EventTypeToolEnd,
		content: "Done",
		extra:   extraData,
	}

	result := adapter.ConvertSDKEvent(sdkEvent)

	if result.Extra == nil {
		t.Fatal("Extra data should not be nil")
	}
	if result.Extra["tool"] != "file_read" {
		t.Errorf("Extra[tool] = %v, want %v", result.Extra["tool"], "file_read")
	}
	if result.Extra["result"] != "success" {
		t.Errorf("Extra[result] = %v, want %v", result.Extra["result"], "success")
	}
}

// TestAdapter_ConvertSDKEvent_UnknownEventType tests handling of unknown event types.
func TestAdapter_ConvertSDKEvent_UnknownEventType(t *testing.T) {
	adapter := tui.NewAdapter()

	sdkEvent := &mockSDKEvent{
		typ:     "unknown_event_type",
		content: "Unknown",
	}

	result := adapter.ConvertSDKEvent(sdkEvent)

	// Unknown events should be treated as done to avoid rendering
	if result.Type != events.EventTypeDone {
		t.Errorf("Unknown event type should be converted to EventTypeDone, got %v", result.Type)
	}
}

// TestCreateRunFunc tests the CreateRunFunc integration.
func TestCreateRunFunc(t *testing.T) {
	t.Skip("Requires full runtime setup - integration test")

	// This test would require:
	// 1. Full SDK runtime creation
	// 2. Mock LLM client
	// 3. Event channel mocking
	//
	// For now, the unit tests above cover the core conversion logic.
}

// TestChatEvent_TypeSafety tests ChatEvent type constants.
func TestChatEvent_TypeSafety(t *testing.T) {
	// Verify event type constants exist and are valid
	eventTypes := []events.EventType{
		tui.EventTypeThinkingStart,
		tui.EventTypeThinkingEnd,
		tui.EventTypeAction,
		tui.EventTypeResult,
		tui.EventTypeResponse,
		tui.EventTypeResponseChunk,
		tui.EventTypeError,
		tui.EventTypeStep,
		tui.EventTypeToolStart,
		tui.EventTypeToolEnd,
		tui.EventTypeConfirmationRequest,
		tui.EventTypeCommandMatched,
		tui.EventTypeCommandResult,
		tui.EventTypeDone,
	}

	for _, et := range eventTypes {
		if et == "" {
			t.Errorf("Event type should not be empty: %v", et)
		}
	}
}

// TestChatEvent_WithRequestID tests request ID tracking for event grouping.
func TestChatEvent_WithRequestID(t *testing.T) {
	adapter := tui.NewAdapter()

	sdkEvent := &mockSDKEvent{
		typ:       events.EventTypeResponse,
		content:   "Hello",
		requestID: "req-123",
	}

	result := adapter.ConvertSDKEvent(sdkEvent)

	if result.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", result.RequestID, "req-123")
	}
}

// mockSDKEvent implements sdk.Event for testing.
type mockSDKEvent struct {
	typ       events.EventType
	content   string
	extra     map[string]any
	requestID string
}

func (m *mockSDKEvent) Type() events.EventType { return m.typ }
func (m *mockSDKEvent) Content() string        { return m.content }
func (m *mockSDKEvent) Extra() map[string]any  { return m.extra }
func (m *mockSDKEvent) Timestamp() time.Time   { return time.Now() }
func (m *mockSDKEvent) RequestID() string      { return m.requestID }
func (m *mockSDKEvent) InteractionType() events.InteractionType {
	if m.extra != nil {
		if it, ok := m.extra["interaction_type"].(events.InteractionType); ok {
			return it
		}
	}
	return "" // Return empty string for non-interaction events
}
