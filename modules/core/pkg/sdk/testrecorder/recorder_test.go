// Package testrecorder_test provides integration tests for the event recording system.
// These tests demonstrate how to record and replay events for testing CLI, TUI, and Server consumers.
package testrecorder_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/core/pkg/sdk/testrecorder"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
)

// TestRecorder_BasicRecording tests basic event recording functionality.
func TestRecorder_BasicRecording(t *testing.T) {
	recorder := testrecorder.NewEventRecorder(
		"basic_test",
		"Basic recording test",
		"Hello",
	)

	// Create mock events channel
	eventCh := make(chan sdk.Event, 5)
	eventCh <- &mockEvent{typ: "thinking_start", content: "Thinking..."}
	eventCh <- &mockEvent{typ: "response", content: "Hello! How can I help?"}
	eventCh <- &mockEvent{typ: "done", content: ""}
	close(eventCh)

	err := recorder.RecordFromChannel(eventCh)
	if err != nil {
		t.Fatalf("RecordFromChannel failed: %v", err)
	}

	events := recorder.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	metadata := recorder.GetMetadata()
	if metadata.EventCount != 3 {
		t.Errorf("Expected metadata event count 3, got %d", metadata.EventCount)
	}
}

// TestRecorder_SaveAndLoad tests saving and loading recorded events.
func TestRecorder_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test_events.json")

	recorder := testrecorder.NewEventRecorder(
		"save_load_test",
		"Save and load test",
		"Test input",
	)

	// Add mock events
	eventCh := make(chan sdk.Event, 3)
	eventCh <- &mockEvent{typ: "thinking_start", content: ""}
	eventCh <- &mockEvent{typ: "response", content: "Response"}
	eventCh <- &mockEvent{typ: "done", content: ""}
	close(eventCh)

	_ = recorder.RecordFromChannel(eventCh)

	// Save to file
	err := recorder.SaveToFile(testPath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("Saved file does not exist")
	}

	// Load from file
	loaded, err := testrecorder.LoadFromFile(testPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	loadedEvents := loaded.GetEvents()
	if len(loadedEvents) != 3 {
		t.Errorf("Expected 3 loaded events, got %d", len(loadedEvents))
	}

	loadedMetadata := loaded.GetMetadata()
	if loadedMetadata.Name != "save_load_test" {
		t.Errorf("Expected metadata name 'save_load_test', got %q", loadedMetadata.Name)
	}
}

// TestRecorder_ToChannel tests converting recorded events back to channel.
func TestRecorder_ToChannel(t *testing.T) {
	recorder := testrecorder.NewEventRecorder("channel_test", "Channel test", "Input")

	// Add events
	eventCh := make(chan sdk.Event, 2)
	eventCh <- &mockEvent{typ: "response", content: "Hello"}
	eventCh <- &mockEvent{typ: "done", content: ""}
	close(eventCh)
	_ = recorder.RecordFromChannel(eventCh)

	// Convert to channel
	replayCh := recorder.ToChannel()

	count := 0
	for range replayCh {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 events from channel, got %d", count)
	}
}

// TestRecorder_ToSDKChannel tests converting to SDK event channel.
func TestRecorder_ToSDKChannel(t *testing.T) {
	recorder := testrecorder.NewEventRecorder("sdk_channel_test", "SDK channel test", "Input")

	eventCh := make(chan sdk.Event, 2)
	eventCh <- &mockEvent{typ: "thinking_start", content: "Thinking"}
	eventCh <- &mockEvent{typ: "response", content: "Response"}
	close(eventCh)
	_ = recorder.RecordFromChannel(eventCh)

	// Convert to SDK channel
	sdkCh := recorder.ToSDKChannel()

	count := 0
	for ev := range sdkCh {
		count++
		_ = ev
	}

	if count != 2 {
		t.Errorf("Expected 2 SDK events, got %d", count)
	}
}

// TestRecorder_ToTestEvents tests conversion to TestEvent format.
func TestRecorder_ToTestEvents(t *testing.T) {
	recorder := testrecorder.NewEventRecorder("test_events", "Test events", "Input")

	eventCh := make(chan sdk.Event, 2)
	eventCh <- &mockEvent{typ: "response", content: "Hello", extra: map[string]any{"key": "value"}}
	eventCh <- &mockEvent{typ: "done", content: ""}
	close(eventCh)
	_ = recorder.RecordFromChannel(eventCh)

	testEvents := recorder.ToTestEvents()

	if len(testEvents) != 2 {
		t.Errorf("Expected 2 test events, got %d", len(testEvents))
	}

	if testEvents[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %q", testEvents[0].Content)
	}

	if testEvents[0].Extra["key"] != "value" {
		t.Errorf("Expected extra key='value', got %v", testEvents[0].Extra["key"])
	}
}

// TestRecorder_FilterAndCount tests filtering and counting utilities.
func TestRecorder_FilterAndCount(t *testing.T) {
	events := []testrecorder.TestEvent{
		{Type: "thinking_start"},
		{Type: "response"},
		{Type: "done"},
		{Type: "response"},
	}

	// Test FilterByType
	responses := testrecorder.FilterByType(events, "response")
	if len(responses) != 2 {
		t.Errorf("Expected 2 response events, got %d", len(responses))
	}

	// Test CountByType
	count := testrecorder.CountByType(events, "done")
	if count != 1 {
		t.Errorf("Expected 1 done event, got %d", count)
	}
}

// TestRecorder_AssertEventOrder tests event order assertion.
func TestRecorder_AssertEventOrder(t *testing.T) {
	testEvents := []testrecorder.TestEvent{
		{Type: events.EventTypeThinkingStart},
		{Type: events.EventTypeToolStart},
		{Type: events.EventTypeToolEnd},
		{Type: events.EventTypeResponse},
		{Type: events.EventTypeDone},
	}

	// Valid order
	expected := []events.EventType{
		events.EventTypeThinkingStart,
		events.EventTypeToolStart,
		events.EventTypeToolEnd,
		events.EventTypeResponse,
		events.EventTypeDone,
	}
	err := testrecorder.AssertEventOrder(testEvents, expected)
	if err != nil {
		t.Errorf("Expected valid order, got error: %v", err)
	}

	// Partial order (should still pass)
	partial := []events.EventType{
		events.EventTypeThinkingStart,
		events.EventTypeResponse,
		events.EventTypeDone,
	}
	err = testrecorder.AssertEventOrder(testEvents, partial)
	if err != nil {
		t.Errorf("Expected partial order to pass, got error: %v", err)
	}

	// Invalid order (should fail)
	invalid := []events.EventType{
		events.EventTypeDone,
		events.EventTypeThinkingStart,
	}
	err = testrecorder.AssertEventOrder(testEvents, invalid)
	if err == nil {
		t.Error("Expected error for invalid order, got nil")
	}
}

// TestRecorder_IntegrationWithRuntime tests recording from actual runtime.
// This test requires a working LLM setup (e.g., Ollama).
func TestRecorder_IntegrationWithRuntime(t *testing.T) {
	t.Skip("Requires LLM setup - run manually when LLM is available")

	// Create minimal config
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "qwen3:8b",
		},
		Memory: config.MemoryConfig{
			MaxTokens: 4000,
		},
	}

	runtimeCfg := sdk.FromConfig(cfg)
	rt, err := sdk.NewRuntime(runtimeCfg)
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	recorder := testrecorder.NewEventRecorder(
		"integration_test",
		"Integration test with real runtime",
		"Say hello",
	)

	ctx := context.Background()
	events, err := rt.Process(ctx, "Say hello")
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	err = recorder.RecordFromChannelWithTimeout(events, 30*time.Second)
	if err != nil {
		t.Fatalf("Recording failed: %v", err)
	}

	// Save for later use
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "integration_events.json")
	err = recorder.SaveToFile(testPath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	t.Logf("Recorded %d events to %s", recorder.GetMetadata().EventCount, testPath)
}

// mockEvent is a mock implementation of sdk.Event for testing.
type mockEvent struct {
	typ     events.EventType
	content string
	extra   map[string]any
}

func (m *mockEvent) Type() events.EventType { return m.typ }
func (m *mockEvent) Content() string        { return m.content }
func (m *mockEvent) Extra() map[string]any  { return m.extra }
func (m *mockEvent) Timestamp() time.Time   { return time.Now() }
func (m *mockEvent) RequestID() string      { return "test-request-id" }
