// Package testrecorder provides event recording and playback utilities for testing.
// This package enables recording event streams from the core runtime,
// then replaying them to test different consumers (CLI, TUI, Server).
package testrecorder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/events"
)

// RecordedEvent represents a serialized event for storage.
type RecordedEvent struct {
	Type      events.EventType `json:"type"`
	Content   string           `json:"content"`
	Extra     map[string]any   `json:"extra,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
	RequestID string           `json:"request_id"`
}

// EventRecorder records and replays event streams for testing.
type EventRecorder struct {
	mu       sync.Mutex
	events   []RecordedEvent
	metadata RecordingMetadata
}

// RecordingMetadata stores information about a recording session.
type RecordingMetadata struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Input       string    `json:"input"`
	RecordedAt  time.Time `json:"recorded_at"`
	EventCount  int       `json:"event_count"`
}

// NewEventRecorder creates a new event recorder.
func NewEventRecorder(name, description, input string) *EventRecorder {
	return &EventRecorder{
		events: make([]RecordedEvent, 0),
		metadata: RecordingMetadata{
			Name:        name,
			Description: description,
			Input:       input,
			RecordedAt:  time.Now(),
		},
	}
}

// RecordFromChannel records events from an SDK event channel.
// This function blocks until the channel is closed.
func (r *EventRecorder) RecordFromChannel(eventCh <-chan sdk.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ev := range eventCh {
		r.events = append(r.events, RecordedEvent{
			Type:      ev.Type(),
			Content:   ev.Content(),
			Extra:     ev.Extra(),
			Timestamp: ev.Timestamp(),
			RequestID: ev.RequestID(),
		})
	}

	r.metadata.EventCount = len(r.events)
	return nil
}

// RecordFromChannelWithTimeout records events with a timeout.
// Useful for tests that need to handle potentially hanging channels.
func (r *EventRecorder) RecordFromChannelWithTimeout(eventCh <-chan sdk.Event, timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- r.RecordFromChannel(eventCh)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("recording timeout after %v", timeout)
	}
}

// GetEvents returns the recorded events.
func (r *EventRecorder) GetEvents() []RecordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.events
}

// GetMetadata returns the recording metadata.
func (r *EventRecorder) GetMetadata() RecordingMetadata {
	return r.metadata
}

// SaveToFile saves the recorded events to a JSON file.
func (r *EventRecorder) SaveToFile(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data := struct {
		Metadata RecordingMetadata `json:"metadata"`
		Events   []RecordedEvent   `json:"events"`
	}{
		Metadata: r.metadata,
		Events:   r.events,
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode events: %w", err)
	}

	return nil
}

// LoadFromFile loads recorded events from a JSON file.
func LoadFromFile(path string) (*EventRecorder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data struct {
		Metadata RecordingMetadata `json:"metadata"`
		Events   []RecordedEvent   `json:"events"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	return &EventRecorder{
		events:   data.Events,
		metadata: data.Metadata,
	}, nil
}

// ToChannel converts recorded events back to a channel for replay.
func (r *EventRecorder) ToChannel() <-chan RecordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan RecordedEvent, len(r.events))
	for _, ev := range r.events {
		ch <- ev
	}
	close(ch)
	return ch
}

// ToSDKChannel converts recorded events to SDK event channel for replay.
func (r *EventRecorder) ToSDKChannel() <-chan sdk.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan sdk.Event, len(r.events))
	for _, ev := range r.events {
		sdkEv := events.NewEventWithExtra(ev.Type, ev.Content, ev.Extra, ev.RequestID)
		ch <- sdkEv
	}
	close(ch)
	return ch
}

// ToTestEvents converts recorded events to TestEvent format for consumer testing.
func (r *EventRecorder) ToTestEvents() []TestEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	testEvents := make([]TestEvent, len(r.events))
	for i, ev := range r.events {
		testEvents[i] = TestEvent{
			Type:      ev.Type,
			Content:   ev.Content,
			Extra:     ev.Extra,
			RequestID: ev.RequestID,
		}
	}
	return testEvents
}

// TestEvent is a simplified event structure for testing consumers.
type TestEvent struct {
	Type      events.EventType
	Content   string
	Extra     map[string]any
	RequestID string
}

// AssertEventOrder verifies that events occur in the expected order.
func AssertEventOrder(events []TestEvent, expectedTypes []events.EventType) error {
	if len(events) < len(expectedTypes) {
		return fmt.Errorf("not enough events: got %d, need at least %d", len(events), len(expectedTypes))
	}

	eventIdx := 0
	for _, expectedType := range expectedTypes {
		found := false
		for eventIdx < len(events) {
			if events[eventIdx].Type == expectedType {
				found = true
				eventIdx++
				break
			}
			eventIdx++
		}
		if !found {
			return fmt.Errorf("expected event type %q not found", expectedType)
		}
	}

	return nil
}

// FilterByType returns events matching the given type.
func FilterByType(events []TestEvent, typ events.EventType) []TestEvent {
	result := make([]TestEvent, 0)
	for _, ev := range events {
		if ev.Type == typ {
			result = append(result, ev)
		}
	}
	return result
}

// CountByType returns the count of events with the given type.
func CountByType(events []TestEvent, typ events.EventType) int {
	count := 0
	for _, ev := range events {
		if ev.Type == typ {
			count++
		}
	}
	return count
}
