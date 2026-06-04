package runtime

import (
	"testing"
	"time"
)

func TestEventType_Constants(t *testing.T) {
	// Verify event type constants are defined
	eventTypes := []EventType{
		EventTypeAction,
		EventTypeResult,
		EventTypeResponse,
		EventTypeError,
		EventTypeStep,
		EventTypeToolStart,
		EventTypeToolEnd,
		EventTypeConfirmationRequest,
	}

	for _, et := range eventTypes {
		if et == "" {
			t.Errorf("EventType constant is empty: %v", et)
		}
	}
}

func TestNewEvent(t *testing.T) {
	typ := EventTypeThinkingStart
	content := "test content"

	event := NewEvent(typ, content)

	if event.Type() != typ {
		t.Errorf("NewEvent() typ = %v, want %v", event.Type(), typ)
	}

	if event.Content() != content {
		t.Errorf("NewEvent() content = %q, want %q", event.Content(), content)
	}

	if event.Extra() != nil {
		t.Error("NewEvent() extra should be nil")
	}

	// Timestamp should be set
	if event.Timestamp().IsZero() {
		t.Error("NewEvent() timestamp should be set")
	}

	// Timestamp should be recent (within last second)
	now := time.Now()
	ts := event.Timestamp()
	if ts.Before(now.Add(-time.Second)) || ts.After(now.Add(time.Second)) {
		t.Error("NewEvent() timestamp should be approximately now")
	}
}

func TestNewEventWithExtra(t *testing.T) {
	typ := EventTypeToolStart
	content := "tool execution"
	extra := map[string]any{
		"tool_name": "file_read",
		"file_path": "/tmp/test.txt",
	}

	event := NewEventWithExtra(typ, content, extra)

	if event.Type() != typ {
		t.Errorf("NewEventWithExtra() typ = %v, want %v", event.Type(), typ)
	}

	if event.Content() != content {
		t.Errorf("NewEventWithExtra() content = %q, want %q", event.Content(), content)
	}

	eventExtra := event.Extra()
	if eventExtra == nil {
		t.Fatal("NewEventWithExtra() extra should not be nil")
	}

	if eventExtra["tool_name"] != "file_read" {
		t.Errorf("NewEventWithExtra() tool_name = %q, want %q", eventExtra["tool_name"], "file_read")
	}

	if eventExtra["file_path"] != "/tmp/test.txt" {
		t.Errorf("NewEventWithExtra() file_path = %q, want %q", eventExtra["file_path"], "/tmp/test.txt")
	}
}

func TestNewEventWithExtra_NilExtra(t *testing.T) {
	typ := EventTypeResponse
	content := "response"

	event := NewEventWithExtra(typ, content, nil)

	if event.Extra() != nil {
		t.Error("NewEventWithExtra() with nil extra should result in nil extra")
	}
}

func TestEvent_Type(t *testing.T) {
	typ := EventTypeAction
	event := NewEvent(typ, "content")

	result := event.Type()
	if result != typ {
		t.Errorf("Event.Type() = %v, want %v", result, typ)
	}
}

func TestEvent_Content(t *testing.T) {
	content := "test content"
	event := NewEvent(EventTypeThinkingStart, content)

	result := event.Content()
	if result != content {
		t.Errorf("Event.Content() = %q, want %q", result, content)
	}
}

func TestEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := NewEvent(EventTypeThinkingStart, "content")
	after := time.Now()

	result := event.Timestamp()

	if result.Before(before) {
		t.Error("Event.Timestamp() should not be before creation time")
	}

	if result.After(after) {
		t.Error("Event.Timestamp() should not be after test completion")
	}
}

func TestEvent_Extra(t *testing.T) {
	extra := map[string]any{"key": "value"}
	event := NewEventWithExtra(EventTypeToolStart, "content", extra)

	result := event.Extra()

	if result == nil {
		t.Fatal("Event.Extra() should not return nil")
	}

	if result["key"] != "value" {
		t.Errorf("Event.Extra() key = %q, want %q", result["key"], "value")
	}
}

func TestEvent_Extra_NoExtraProvided(t *testing.T) {
	event := NewEvent(EventTypeThinkingStart, "content")

	result := event.Extra()

	if result != nil {
		t.Error("Event.Extra() should return nil when no extra was provided")
	}
}

func TestConfirmationRequest(t *testing.T) {
	req := ConfirmationRequest{
		ToolName: "file_write",
		Params:   map[string]any{"file": "/tmp/test.txt"},
	}

	if req.ToolName != "file_write" {
		t.Errorf("ConfirmationRequest.ToolName = %q, want %q", req.ToolName, "file_write")
	}

	if req.Params["file"] != "/tmp/test.txt" {
		t.Errorf("ConfirmationRequest.Params file = %q, want %q", req.Params["file"], "/tmp/test.txt")
	}
}
