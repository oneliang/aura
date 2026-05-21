// Package events provides unified event definitions for the Aura system.
package events

import (
	"context"
	"sync"
)

// Bus provides an event bus for publishing and subscribing to events.
// It supports both async events (for streaming) and sync request/response pattern.
type Bus struct {
	mu           sync.RWMutex
	subscribers  map[EventType][]chan Event
	cmdHandlers  map[string]CommandHandler
	droppedCount map[EventType]int // Tracks events dropped due to full channels
}

// CommandHandler handles a command request and returns a response.
type CommandHandler func(ctx context.Context, req *CommandRequest) CommandResponse

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscribers:  make(map[EventType][]chan Event),
		cmdHandlers:  make(map[string]CommandHandler),
		droppedCount: make(map[EventType]int),
	}
}

// Subscribe creates a subscription for events of the given type.
// Returns a channel that will receive events of that type.
func (b *Bus) Subscribe(typ EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 16)
	b.subscribers[typ] = append(b.subscribers[typ], ch)
	return ch
}

// Unsubscribe removes a subscription.
func (b *Bus) Unsubscribe(typ EventType, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[typ]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[typ] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

// Publish sends an event to all subscribers of that event type.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs := b.subscribers[event.Type()]
	for _, ch := range subs {
		go func(ch chan Event) {
			defer func() { recover() }()
			select {
			case ch <- event:
			default:
				b.mu.Lock()
				b.droppedCount[event.Type()]++
				b.mu.Unlock()
			}
		}(ch)
	}
}

// Stats returns the number of dropped events per type.
func (b *Bus) Stats() map[EventType]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[EventType]int, len(b.droppedCount))
	for k, v := range b.droppedCount {
		result[k] = v
	}
	return result
}

// RegisterCommandHandler registers a handler for a command type.
// Only one handler per command type is allowed.
func (b *Bus) RegisterCommandHandler(commandType string, handler CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cmdHandlers[commandType] = handler
}

// UnregisterCommandHandler removes a command handler.
func (b *Bus) UnregisterCommandHandler(commandType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.cmdHandlers, commandType)
}

// ExecuteCommand executes a command synchronously using registered handlers.
// Returns the response from the handler, or an error response if no handler is registered.
func (b *Bus) ExecuteCommand(ctx context.Context, req *CommandRequest) CommandResponse {
	b.mu.RLock()
	handler, ok := b.cmdHandlers[req.Command]
	b.mu.RUnlock()

	if !ok {
		return CommandResponse{
			Success: false,
			Result:  "",
			Error:   ErrNoHandler,
		}
	}

	return handler(ctx, req)
}

// ErrNoHandler is returned when no handler is registered for a command.
var ErrNoHandler = &CommandError{Message: "no handler registered for command"}

// CommandError represents a command execution error.
type CommandError struct {
	Message string
	Cause   error
}

func (e *CommandError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *CommandError) Unwrap() error {
	return e.Cause
}
