// Package testconsumers provides mock consumers for testing event stream processing.
// These consumers simulate CLI, TUI, and Server behavior when processing SDK events.
package testconsumers

import (
	"fmt"
	"strings"

	"github.com/oneliang/aura/core/pkg/sdk/testrecorder"
	"github.com/oneliang/aura/shared/pkg/events"
)

// CLIConsumer simulates CLI event consumption behavior.
type CLIConsumer struct {
	output strings.Builder
}

// NewCLIConsumer creates a new CLI consumer.
func NewCLIConsumer() *CLIConsumer {
	return &CLIConsumer{}
}

// Process processes an event and returns CLI-formatted output.
func (c *CLIConsumer) Process(ev testrecorder.TestEvent) string {
	switch ev.Type {
	case events.EventTypeThinkingStart:
		return "🤔 Thinking...\n"
	case events.EventTypeThinkingEnd:
		return ""
	case events.EventTypeAction:
		return fmt.Sprintf("⚡ Action: %s\n", ev.Content)
	case events.EventTypeResult:
		return fmt.Sprintf("📋 Result: %s\n", ev.Content)
	case events.EventTypeResponse:
		return fmt.Sprintf("Aura: %s\n", ev.Content)
	case events.EventTypeResponseChunk:
		return ev.Content
	case events.EventTypeError:
		return fmt.Sprintf("❌ Error: %s\n", ev.Content)
	case events.EventTypeToolStart:
		toolName := ""
		if ev.Extra != nil {
			if name, ok := ev.Extra["tool"].(string); ok {
				toolName = name
			}
		}
		return fmt.Sprintf("🔧 Tool starting: %s\n", toolName)
	case events.EventTypeToolEnd:
		result := ""
		if ev.Extra != nil {
			if r, ok := ev.Extra["result"].(string); ok {
				result = r
			}
		}
		return fmt.Sprintf("✅ Tool complete: %s\n", result)
	case events.EventTypeConfirmationRequest:
		return fmt.Sprintf("⚠️  Confirmation needed: %s\n", ev.Content)
	case events.EventTypeCommandMatched:
		return fmt.Sprintf("📝 Command matched: %s\n", ev.Content)
	case events.EventTypeCommandResult:
		return fmt.Sprintf("📝 Command result: %s\n", ev.Content)
	case events.EventTypeDone:
		return ""
	default:
		return ""
	}
}

// ProcessAll processes all events and returns combined output.
func (c *CLIConsumer) ProcessAll(eventList []testrecorder.TestEvent) string {
	c.output.Reset()
	for _, ev := range eventList {
		c.output.WriteString(c.Process(ev))
	}
	return c.output.String()
}

// GetOutput returns the accumulated output.
func (c *CLIConsumer) GetOutput() string {
	return c.output.String()
}

// TUIConsumer simulates TUI event consumption using the Adapter pattern.
type TUIConsumer struct {
	adapter *TUIAdapter
	events  []TUIChatEvent
	lastErr error
}

// TUIChatEvent represents a TUI chat event.
type TUIChatEvent struct {
	Type      events.EventType
	Content   string
	Extra     map[string]any
	RequestID string
}

// TUIAdapter adapts SDK events to TUI events (mirrors the real TUI adapter).
type TUIAdapter struct{}

// NewTUIAdapter creates a new TUI adapter.
func NewTUIAdapter() *TUIAdapter {
	return &TUIAdapter{}
}

// ConvertSDKEvent converts a test event to TUI chat event.
func (a *TUIAdapter) ConvertSDKEvent(ev testrecorder.TestEvent) TUIChatEvent {
	return TUIChatEvent{
		Type:      ev.Type,
		Content:   ev.Content,
		Extra:     ev.Extra,
		RequestID: ev.RequestID,
	}
}

// NewTUIConsumer creates a new TUI consumer.
func NewTUIConsumer() *TUIConsumer {
	return &TUIConsumer{
		adapter: NewTUIAdapter(),
		events:  make([]TUIChatEvent, 0),
	}
}

// Process processes an event and stores it.
func (t *TUIConsumer) Process(ev testrecorder.TestEvent) error {
	tuiEv := t.adapter.ConvertSDKEvent(ev)

	if tuiEv.Type == "" {
		return fmt.Errorf("invalid event type: %v", ev)
	}

	t.events = append(t.events, tuiEv)
	return nil
}

// ProcessAll processes all events.
func (t *TUIConsumer) ProcessAll(eventList []testrecorder.TestEvent) error {
	t.events = make([]TUIChatEvent, 0)
	for _, ev := range eventList {
		if err := t.Process(ev); err != nil {
			t.lastErr = err
			return err
		}
	}
	return nil
}

// GetEvents returns the processed TUI events.
func (t *TUIConsumer) GetEvents() []TUIChatEvent {
	return t.events
}

// GetLastErr returns the last error.
func (t *TUIConsumer) GetLastErr() error {
	return t.lastErr
}

// HasEventType checks if a specific event type was processed.
func (t *TUIConsumer) HasEventType(typ events.EventType) bool {
	for _, ev := range t.events {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

// ServerConsumer simulates Server SSE event consumption behavior.
type ServerConsumer struct {
	output strings.Builder
}

// NewServerConsumer creates a new Server consumer.
func NewServerConsumer() *ServerConsumer {
	return &ServerConsumer{}
}

// Process processes an event and returns SSE-formatted output.
func (s *ServerConsumer) Process(ev testrecorder.TestEvent) string {
	var sb strings.Builder

	// SSE format: event: <type>\ndata: <content>\n\n
	sb.WriteString(fmt.Sprintf("event: %s\n", ev.Type))

	if ev.Content != "" {
		sb.WriteString(fmt.Sprintf("data: %s\n", ev.Content))
	}

	if len(ev.Extra) > 0 {
		sb.WriteString(fmt.Sprintf("extra: %v\n", ev.Extra))
	}

	sb.WriteString("\n")
	return sb.String()
}

// ProcessAll processes all events and returns combined SSE output.
func (s *ServerConsumer) ProcessAll(eventList []testrecorder.TestEvent) string {
	s.output.Reset()
	for _, ev := range eventList {
		s.output.WriteString(s.Process(ev))
	}
	return s.output.String()
}

// GetOutput returns the accumulated SSE output.
func (s *ServerConsumer) GetOutput() string {
	return s.output.String()
}

// HasSSEFormat checks if output contains valid SSE format.
func (s *ServerConsumer) HasSSEFormat() bool {
	output := s.output.String()
	return strings.Contains(output, "event: ") && strings.Contains(output, "data: ")
}

// ConsumerResult holds the result of multi-consumer testing.
type ConsumerResult struct {
	CLI    string
	TUI    *TUIConsumer
	Server string
}

// TestAllConsumers processes events through all consumers and returns results.
func TestAllConsumers(eventList []testrecorder.TestEvent) (*ConsumerResult, error) {
	cli := NewCLIConsumer()
	cliOutput := cli.ProcessAll(eventList)

	tui := NewTUIConsumer()
	if err := tui.ProcessAll(eventList); err != nil {
		return nil, fmt.Errorf("TUI processing failed: %w", err)
	}

	server := NewServerConsumer()
	serverOutput := server.ProcessAll(eventList)

	return &ConsumerResult{
		CLI:    cliOutput,
		TUI:    tui,
		Server: serverOutput,
	}, nil
}

// ValidateResults validates that all consumers processed events correctly.
func ValidateResults(result *ConsumerResult, expectedEvents int) []error {
	var errs []error

	if len(result.TUI.events) != expectedEvents {
		errs = append(errs, fmt.Errorf("TUI: expected %d events, got %d", expectedEvents, len(result.TUI.events)))
	}

	if expectedEvents > 0 && !strings.Contains(result.Server, "event: ") {
		errs = append(errs, fmt.Errorf("Server: output is not in SSE format"))
	}

	return errs
}
