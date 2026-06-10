// Package server provides SSE handling functionality.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// SessionOrchestrator bridges HTTP SSE layer and Runtime, similar to TUI's Orchestrator.
// It manages bidirectional event flow between HTTP clients and the Runtime.
type SessionOrchestrator struct {
	runtime   *sdk.Runtime
	logger    *logger.Logger
	adapter   *SSEAdapter
	writers   map[string]*SSEWriter
	writersMu sync.RWMutex
	inputCh   chan events.Event
	doneCh    chan struct{}
	ready     chan struct{} // Closed when goroutines are ready
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	stopped   bool
	stoppedMu sync.RWMutex
}

// SSEWriter represents an active SSE connection.
type SSEWriter struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	closed   bool
	closedMu sync.RWMutex
}

// NewSessionOrchestrator creates a new SessionOrchestrator.
func NewSessionOrchestrator(runtime *sdk.Runtime, log *logger.Logger) *SessionOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionOrchestrator{
		runtime: runtime,
		logger:  log,
		adapter: NewSSEAdapter(),
		writers: make(map[string]*SSEWriter),
		inputCh: make(chan events.Event, 100),
		doneCh:  make(chan struct{}),
		ready:   make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Run starts the orchestrator's goroutines.
// Goroutine 1: Reads events from Runtime and broadcasts to all SSE writers.
// Goroutine 2: Reads from inputCh and sends to Runtime.
func (o *SessionOrchestrator) Run() {
	// Goroutine 1: Runtime → SSE Writers
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.processRuntimeEvents()
	}()

	// Goroutine 2: inputCh → Runtime
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.processInputEvents()
	}()

	// Signal that goroutines are ready
	close(o.ready)

	o.logger.Debug("SessionOrchestrator started", "module", "server")
}

// WaitReady blocks until the orchestrator's goroutines are ready.
func (o *SessionOrchestrator) WaitReady() {
	<-o.ready
}

// processRuntimeEvents reads events from Runtime and broadcasts to all SSE writers.
func (o *SessionOrchestrator) processRuntimeEvents() {
	eventsCh := o.runtime.Events()
	if eventsCh == nil {
		o.logger.Warn("Runtime events channel is nil (shared mode?)", "module", "server")
		return
	}

	for {
		select {
		case <-o.ctx.Done():
			o.logger.Debug("processRuntimeEvents: context done", "module", "server")
			return
		case ev, ok := <-eventsCh:
			if !ok {
				o.logger.Debug("processRuntimeEvents: events channel closed", "module", "server")
				o.closeAllWriters()
				return
			}

			o.logger.Debug("Runtime event received",
				"module", "server",
				"event_type", string(ev.Type()))

			// Convert SDK event to SSE format
			sseEvent := o.adapter.ConvertToSSE(ev)
			if sseEvent == nil {
				o.logger.Debug("Event not recognized, skipping",
					"module", "server",
					"event_type", string(ev.Type()))
				continue
			}

			// Broadcast to all active SSE writers
			o.broadcastSSEEvent(sseEvent)

			// If done event, close all writers but continue listening
			if ev.Type() == sdk.EventTypeDone {
				o.logger.Debug("Done event received, closing all writers", "module", "server")
				o.closeAllWriters()
				// Don't return! Continue listening for next request
				continue
			}
		}
	}
}

// processInputEvents reads from inputCh and sends to Runtime.
func (o *SessionOrchestrator) processInputEvents() {
	for {
		select {
		case <-o.ctx.Done():
			o.logger.Debug("processInputEvents: context done", "module", "server")
			return
		case ev, ok := <-o.inputCh:
			if !ok {
				o.logger.Debug("processInputEvents: input channel closed", "module", "server")
				return
			}

			o.logger.Debug("Sending event to Runtime",
				"module", "server",
				"event_type", string(ev.Type()))

			if err := o.runtime.SendEvent(o.ctx, ev); err != nil {
				o.logger.Error("Failed to send event to Runtime",
					"module", "server",
					"error", err.Error())
			}
		}
	}
}

// broadcastSSEEvent sends an SSE event to all active writers.
func (o *SessionOrchestrator) broadcastSSEEvent(event *SSEEvent) {
	o.writersMu.RLock()
	defer o.writersMu.RUnlock()

	for id, writer := range o.writers {
		if !writer.Write(event) {
			o.logger.Debug("Writer disconnected, will be removed",
				"module", "server",
				"writer_id", id)
		}
	}
}

// closeAllWriters closes all active SSE writers.
func (o *SessionOrchestrator) closeAllWriters() {
	o.writersMu.Lock()
	defer o.writersMu.Unlock()

	for id, writer := range o.writers {
		writer.Close()
		delete(o.writers, id)
	}
}

// HasActiveWriters returns true if there are active SSE writers.
func (o *SessionOrchestrator) HasActiveWriters() bool {
	o.writersMu.RLock()
	defer o.writersMu.RUnlock()
	return len(o.writers) > 0
}

// AttachSSEWriter registers a new SSE writer.
func (o *SessionOrchestrator) AttachSSEWriter(id string, w http.ResponseWriter, flusher http.Flusher) *SSEWriter {
	o.writersMu.Lock()
	defer o.writersMu.Unlock()

	writer := &SSEWriter{
		id:      id,
		writer:  w,
		flusher: flusher,
		done:    make(chan struct{}),
	}
	o.writers[id] = writer

	o.logger.Debug("SSE writer attached",
		"module", "server",
		"writer_id", id)

	return writer
}

// DetachSSEWriter removes an SSE writer.
func (o *SessionOrchestrator) DetachSSEWriter(id string) {
	o.writersMu.Lock()
	defer o.writersMu.Unlock()

	if writer, ok := o.writers[id]; ok {
		writer.Close()
		delete(o.writers, id)
		o.logger.Debug("SSE writer detached",
			"module", "server",
			"writer_id", id)
	}
}

// SendUserInput sends user input to the Runtime.
func (o *SessionOrchestrator) SendUserInput(content string, requestID string) error {
	o.stoppedMu.RLock()
	if o.stopped {
		o.stoppedMu.RUnlock()
		return fmt.Errorf("orchestrator is stopped")
	}
	o.stoppedMu.RUnlock()

	event := events.NewEvent(events.EventTypeUserInput, content, requestID)
	select {
	case o.inputCh <- event:
		return nil
	case <-o.ctx.Done():
		return o.ctx.Err()
	default:
		return fmt.Errorf("input channel full")
	}
}

// HandleInteraction sends an interaction response to the Runtime.
func (o *SessionOrchestrator) HandleInteraction(response events.Event) error {
	o.stoppedMu.RLock()
	if o.stopped {
		o.stoppedMu.RUnlock()
		return fmt.Errorf("orchestrator is stopped")
	}
	o.stoppedMu.RUnlock()

	select {
	case o.inputCh <- response:
		return nil
	case <-o.ctx.Done():
		return o.ctx.Err()
	default:
		return fmt.Errorf("input channel full")
	}
}

// Stop gracefully stops the orchestrator.
func (o *SessionOrchestrator) Stop() {
	o.stoppedMu.Lock()
	if o.stopped {
		o.stoppedMu.Unlock()
		return
	}
	o.stopped = true
	o.stoppedMu.Unlock()

	o.logger.Debug("Stopping SessionOrchestrator", "module", "server")

	// Cancel context to stop goroutines
	o.cancel()

	// Note: we don't close inputCh here because SendUserInput/HandleInteraction
	// could race with close(). Goroutines exit via ctx.Done() instead.

	// Wait for goroutines to finish
	o.wg.Wait()

	// Close all writers
	o.closeAllWriters()

	// Stop runtime - use fresh context since o.ctx is now cancelled
	if o.runtime != nil {
		o.runtime.Stop(context.Background())
	}

	// Close doneCh to signal completion
	close(o.doneCh)

	o.logger.Debug("SessionOrchestrator stopped", "module", "server")
}

// Done returns a channel that's closed when the orchestrator stops.
func (o *SessionOrchestrator) Done() <-chan struct{} {
	return o.doneCh
}

// Write writes an SSE event to the client.
// Returns false if the client has disconnected.
func (w *SSEWriter) Write(event *SSEEvent) bool {
	w.closedMu.RLock()
	if w.closed {
		w.closedMu.RUnlock()
		return false
	}
	w.closedMu.RUnlock()

	formatted := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Name, formatJSON(event.Data))
	_, err := fmt.Fprint(w.writer, formatted)
	if err != nil {
		w.Close()
		return false
	}

	w.flusher.Flush()
	return true
}

// Close closes the SSE writer.
func (w *SSEWriter) Close() {
	w.closedMu.Lock()
	defer w.closedMu.Unlock()

	if !w.closed {
		w.closed = true
		close(w.done)
	}
}

// Done returns a channel that's closed when the writer is closed.
func (w *SSEWriter) Done() <-chan struct{} {
	return w.done
}
