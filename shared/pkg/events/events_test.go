package events

import (
	"context"
	"testing"
	"time"
)

// TestNewBus tests NewBus function.
func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}
	if bus.subscribers == nil {
		t.Error("NewBus() subscribers map should be initialized")
	}
	if bus.cmdHandlers == nil {
		t.Error("NewBus() cmdHandlers map should be initialized")
	}
}

// TestBus_Subscribe tests Subscribe method.
func TestBus_Subscribe(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe(EventTypeThinkingStart)
	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	// Check that subscription was added
	bus.mu.RLock()
	subs := bus.subscribers[EventTypeThinkingStart]
	bus.mu.RUnlock()

	if len(subs) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subs))
	}
}

// TestBus_Unsubscribe tests Unsubscribe method.
func TestBus_Unsubscribe(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe(EventTypeThinkingStart)
	bus.Unsubscribe(EventTypeThinkingStart, ch)

	// Check that subscription was removed
	bus.mu.RLock()
	subs := bus.subscribers[EventTypeThinkingStart]
	bus.mu.RUnlock()

	if len(subs) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(subs))
	}
}

// TestBus_Publish tests Publish method.
func TestBus_Publish(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe(EventTypeThinkingStart)

	event := NewEvent(EventTypeThinkingStart, "test content", "")
	bus.Publish(event)

	select {
	case received := <-ch:
		if received.Content() != "test content" {
			t.Errorf("Expected content 'test content', got '%s'", received.Content())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive event, but timed out")
	}
}

// TestBus_Publish_MultipleSubscribers tests publishing to multiple subscribers.
func TestBus_Publish_MultipleSubscribers(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe(EventTypeThinkingStart)
	ch2 := bus.Subscribe(EventTypeThinkingStart)

	event := NewEvent(EventTypeThinkingStart, "broadcast", "")
	bus.Publish(event)

	// Both subscribers should receive the event
	select {
	case <-ch1:
	case <-time.After(100 * time.Millisecond):
		t.Error("First subscriber did not receive event")
	}

	select {
	case <-ch2:
	case <-time.After(100 * time.Millisecond):
		t.Error("Second subscriber did not receive event")
	}
}

// TestBus_RegisterCommandHandler tests RegisterCommandHandler method.
func TestBus_RegisterCommandHandler(t *testing.T) {
	bus := NewBus()

	handler := func(ctx context.Context, req *CommandRequest) CommandResponse {
		return CommandResponse{Success: true, Result: "ok"}
	}

	bus.RegisterCommandHandler("test_command", handler)

	bus.mu.RLock()
	_, exists := bus.cmdHandlers["test_command"]
	bus.mu.RUnlock()

	if !exists {
		t.Error("Handler was not registered")
	}
}

// TestBus_UnregisterCommandHandler tests UnregisterCommandHandler method.
func TestBus_UnregisterCommandHandler(t *testing.T) {
	bus := NewBus()

	handler := func(ctx context.Context, req *CommandRequest) CommandResponse {
		return CommandResponse{Success: true}
	}

	bus.RegisterCommandHandler("test_command", handler)
	bus.UnregisterCommandHandler("test_command")

	bus.mu.RLock()
	_, exists := bus.cmdHandlers["test_command"]
	bus.mu.RUnlock()

	if exists {
		t.Error("Handler should have been unregistered")
	}
}

// TestBus_ExecuteCommand tests ExecuteCommand method.
func TestBus_ExecuteCommand(t *testing.T) {
	bus := NewBus()

	handler := func(ctx context.Context, req *CommandRequest) CommandResponse {
		return CommandResponse{Success: true, Result: "executed"}
	}

	bus.RegisterCommandHandler("test_command", handler)

	ctx := context.Background()
	req := &CommandRequest{Command: "test_command"}
	resp := bus.ExecuteCommand(ctx, req)

	if !resp.Success {
		t.Error("Expected success, got failure")
	}
	if resp.Result != "executed" {
		t.Errorf("Expected result 'executed', got '%s'", resp.Result)
	}
}

// TestBus_ExecuteCommand_NoHandler tests ExecuteCommand with no handler.
func TestBus_ExecuteCommand_NoHandler(t *testing.T) {
	bus := NewBus()

	ctx := context.Background()
	req := &CommandRequest{Command: "nonexistent"}
	resp := bus.ExecuteCommand(ctx, req)

	if resp.Success {
		t.Error("Expected failure when no handler is registered")
	}
	if resp.Error != ErrNoHandler {
		t.Errorf("Expected ErrNoHandler, got %v", resp.Error)
	}
}

// TestNewEvent tests NewEvent function.
func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeThinkingStart, "test content", "")

	if event.Type() != EventTypeThinkingStart {
		t.Errorf("Expected type '%s', got '%s'", EventTypeThinkingStart, event.Type())
	}
	if event.Content() != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", event.Content())
	}
	if event.Timestamp().IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// TestNewEventWithExtra tests NewEventWithExtra function.
func TestNewEventWithExtra(t *testing.T) {
	extra := map[string]any{"key": "value"}
	event := NewEventWithExtra(EventTypeAction, "action content", extra, "")

	if event.Type() != EventTypeAction {
		t.Errorf("Expected type '%s', got '%s'", EventTypeAction, event.Type())
	}
	if event.Content() != "action content" {
		t.Errorf("Expected content 'action content', got '%s'", event.Content())
	}
	if event.Extra() == nil {
		t.Fatal("Extra should not be nil")
	}
	if event.Extra()["key"] != "value" {
		t.Errorf("Expected extra['key'] = 'value', got '%v'", event.Extra()["key"])
	}
}

// TestBaseEvent_Type tests Type method.
func TestBaseEvent_Type(t *testing.T) {
	event := NewEvent(EventTypeResponse, "", "")
	if event.Type() != EventTypeResponse {
		t.Errorf("Expected type '%s', got '%s'", EventTypeResponse, event.Type())
	}
}

// TestBaseEvent_Content tests Content method.
func TestBaseEvent_Content(t *testing.T) {
	event := NewEvent(EventTypeResponse, "test", "")
	if event.Content() != "test" {
		t.Errorf("Expected content 'test', got '%s'", event.Content())
	}
}

// TestBaseEvent_Extra tests Extra method.
func TestBaseEvent_Extra(t *testing.T) {
	extra := map[string]any{"foo": "bar"}
	event := NewEventWithExtra(EventTypeResponse, "", extra, "")
	if event.Extra()["foo"] != "bar" {
		t.Errorf("Expected extra['foo'] = 'bar', got '%v'", event.Extra()["foo"])
	}
}

// TestBaseEvent_Timestamp tests Timestamp method.
func TestBaseEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := NewEvent(EventTypeResponse, "", "")
	after := time.Now()
	if !event.Timestamp().After(before) || event.Timestamp().After(after) {
		t.Errorf("Expected timestamp between %v and %v, got %v", before, after, event.Timestamp())
	}
}

// TestCommandError tests CommandError type.
func TestCommandError(t *testing.T) {
	err := &CommandError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", err.Error())
	}
}

// TestEventTypeConstants tests event type constants.
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EventType
		expected string
	}{
		{"Action", EventTypeAction, "action"},
		{"Result", EventTypeResult, "result"},
		{"Response", EventTypeResponse, "response"},
		{"Error", EventTypeError, "error"},
		{"Step", EventTypeStep, "step"},
		{"ToolStart", EventTypeToolStart, "tool_start"},
		{"ToolEnd", EventTypeToolEnd, "tool_end"},
		{"PlanCreated", EventTypePlanCreated, "plan_created"},
		{"PlanStep", EventTypePlanStep, "plan_step"},
		{"PlanComplete", EventTypePlanComplete, "plan_complete"},
		{"Done", EventTypeDone, "done"},
		{"ConfirmationRequest", EventTypeConfirmationRequest, "confirmation_request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.constant)
			}
		})
	}
}

// TestCommandTypeConstants tests command type constants.
func TestCommandTypeConstants(t *testing.T) {
	if CommandTypeMemoryClear != "memory_clear" {
		t.Errorf("Expected 'memory_clear', got '%s'", CommandTypeMemoryClear)
	}
	if CommandTypeMemoryCompact != "memory_compact" {
		t.Errorf("Expected 'memory_compact', got '%s'", CommandTypeMemoryCompact)
	}
	if CommandTypeMemoryStats != "memory_stats" {
		t.Errorf("Expected 'memory_stats', got '%s'", CommandTypeMemoryStats)
	}
}

// TestConfirmationRequest tests ConfirmationRequest struct.
func TestConfirmationRequest(t *testing.T) {
	req := ConfirmationRequest{
		ToolName:   "bash",
		Params:     map[string]any{"cmd": "ls"},
		ResponseCh: make(chan bool, 1),
	}

	if req.ToolName != "bash" {
		t.Errorf("Expected ToolName 'bash', got '%s'", req.ToolName)
	}
	if req.Params["cmd"] != "ls" {
		t.Errorf("Expected Params['cmd'] = 'ls', got '%v'", req.Params["cmd"])
	}
	if req.ResponseCh == nil {
		t.Error("ResponseCh should not be nil")
	}
}

// TestMemoryStats tests MemoryStats struct.
func TestMemoryStats(t *testing.T) {
	stats := MemoryStats{
		MessageCount: 10,
		TokenCount:   1000,
		MaxTokens:    8000,
	}

	if stats.MessageCount != 10 {
		t.Errorf("Expected MessageCount 10, got %d", stats.MessageCount)
	}
	if stats.TokenCount != 1000 {
		t.Errorf("Expected TokenCount 1000, got %d", stats.TokenCount)
	}
	if stats.MaxTokens != 8000 {
		t.Errorf("Expected MaxTokens 8000, got %d", stats.MaxTokens)
	}
}

// TestCommandRequest tests CommandRequest struct.
func TestCommandRequest(t *testing.T) {
	req := CommandRequest{
		Command:    "test_command",
		Params:     map[string]any{"key": "value"},
		ResponseCh: make(chan CommandResponse, 1),
	}

	if req.Command != "test_command" {
		t.Errorf("Expected Command 'test_command', got '%s'", req.Command)
	}
	if req.Params["key"] != "value" {
		t.Errorf("Expected Params['key'] = 'value', got '%v'", req.Params["key"])
	}
	if req.ResponseCh == nil {
		t.Error("ResponseCh should not be nil")
	}
}

// TestCommandResponse tests CommandResponse struct.
func TestCommandResponse(t *testing.T) {
	resp := CommandResponse{
		Success: true,
		Result:  "ok",
		Error:   nil,
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}
	if resp.Result != "ok" {
		t.Errorf("Expected Result 'ok', got '%s'", resp.Result)
	}
}

// TestBus_Publish_FullChannel tests publishing when channel is full.
func TestBus_Publish_FullChannel(t *testing.T) {
	bus := NewBus()

	// Create a channel with buffer size 1
	ch := make(chan Event, 1)
	bus.mu.Lock()
	bus.subscribers[EventTypeThinkingStart] = append(bus.subscribers[EventTypeThinkingStart], ch)
	bus.mu.Unlock()

	// Fill the channel
	ch <- NewEvent(EventTypeThinkingStart, "filler", "")

	// Publish should not block
	event := NewEvent(EventTypeThinkingStart, "test", "")
	done := make(chan bool)
	go func() {
		bus.Publish(event)
		done <- true
	}()

	select {
	case <-done:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("Publish should not block when channel is full")
	}
}
