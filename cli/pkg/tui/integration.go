package tui

import (
	"context"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// SessionLearner handles session learning.
type SessionLearner struct {
	observations []string
}

// LoadSessionLearner creates a new session learner.
func LoadSessionLearner() *SessionLearner {
	return &SessionLearner{
		observations: make([]string, 0),
	}
}

// Observe records an observation.
func (sl *SessionLearner) Observe(input string) {
	sl.observations = append(sl.observations, input)
}

// Learn applies learning to a session.
func (sl *SessionLearner) Learn(sessionID string) {
	// Placeholder for learning logic
}

// NewModelProvider creates a ModelProvider for TUI model sharing.
func NewModelProvider() *ModelProvider {
	return &ModelProvider{}
}

// Run starts the TUI program with event stream integration.
// The runtime should already be initialized. Run will call Start() to begin event streaming.
func Run(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) error {
	startTime := time.Now()
	log.Debug("[DIAG] Run: starting")

	// Start runtime event stream
	if err := rt.Start(ctx); err != nil {
		return err
	}
	log.Debug("[DIAG] Run: rt.Start done", "elapsed", time.Since(startTime))

	// Create TUI model with runtime reference
	m := NewWithRuntime(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)
	log.Debug("[DIAG] Run: NewWithRuntime done", "elapsed", time.Since(startTime))

	// Create orchestrator for event stream integration
	orchestrator := NewOrchestrator(rt, m)

	// Start event stream forwarding in background
	go orchestrator.Run(ctx)

	// Wait for orchestrator to be ready before starting UI
	// This prevents race condition where user input arrives before goroutines are listening
	orchestrator.WaitReady()
	log.Debug("[DIAG] Run: orchestrator ready, starting Bubble Tea", "elapsed", time.Since(startTime))

	// Run Bubble Tea program
	p := tea.NewProgram(m)
	_, err := p.Run()

	// Close TUI output channel first to unblock TUI→Runtime goroutine
	// This must happen BEFORE orchestrator.Stop() which waits for goroutines
	m.closeEventOutCh()

	// Stop runtime to close agentEvents channel, unblocking Runtime→TUI goroutine
	rt.Stop(ctx)

	// Now wait for goroutines to complete (they exit via closed channels)
	orchestrator.Stop()

	// Close remaining channels
	m.Close()

	return err
}

// RunWithConfig starts the TUI program with configuration options.
func RunWithConfig(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) error {
	return Run(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)
}

// Orchestrator integrates Runtime and TUI event streams.
type Orchestrator struct {
	runtime   *sdk.Runtime
	model     *Model
	wg        sync.WaitGroup // WaitGroup for goroutine lifecycle
	ready     chan struct{}  // Signal when goroutines are ready to receive events
}

// NewOrchestrator creates an orchestrator for event stream integration.
func NewOrchestrator(rt *sdk.Runtime, m *Model) *Orchestrator {
	return &Orchestrator{
		runtime: rt,
		model:   m,
		ready:   make(chan struct{}),
	}
}

// Run starts event stream forwarding between Runtime and TUI.
// Signals readiness via ready channel before entering main loop.
func (o *Orchestrator) Run(ctx context.Context) {
	// Get runtime event stream (OUT)
	agentEvents := o.runtime.Events()

	// Forward: Runtime.out → TUI.in
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-agentEvents:
				if !ok {
					return
				}
				o.model.ReceiveEvent(event)
			}
		}
	}()

	// Forward: TUI.out → Runtime.in
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		uiEvents := o.model.Events()
		// Signal that we're ready to receive events
		close(o.ready)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-uiEvents:
				if !ok {
					return
				}
				if err := o.runtime.SendEvent(ctx, event); err != nil {
				}
			}
		}
	}()
}

// WaitReady waits for the orchestrator goroutines to be ready to receive events.
// This ensures no events are lost if user input arrives before goroutines start.
func (o *Orchestrator) WaitReady() {
	<-o.ready
}

// Stop waits for all goroutines to complete before returning.
// Call this before closing channels to prevent panic from sending to closed channel.
func (o *Orchestrator) Stop() {
	o.wg.Wait()
}

// Adapter provides utilities to adapt SDK events to TUI events.
// Deprecated: Use ReceiveEvent() for direct event handling.
type Adapter struct{}

// NewAdapter creates a new adapter.
// Deprecated: Use ReceiveEvent() for direct event handling.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// ConvertSDKEvent converts an SDK event to a TUI chat event.
// Deprecated: Use ReceiveEvent() for direct event handling.
func (a *Adapter) ConvertSDKEvent(ev sdk.Event) ChatEvent {
	switch ev.Type() {
	case sdk.EventTypeThinkingStart:
		return ChatEvent{Type: EventTypeThinkingStart, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeThinkingChunk:
		return ChatEvent{Type: EventTypeThinkingChunk, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeThinkingEnd:
		return ChatEvent{Type: EventTypeThinkingEnd, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeAction:
		return ChatEvent{Type: EventTypeAction, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeResult:
		return ChatEvent{Type: EventTypeResult, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeResponse:
		return ChatEvent{Type: EventTypeResponse, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeResponseStart:
		return ChatEvent{Type: EventTypeResponseStart, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeResponseChunk:
		return ChatEvent{Type: EventTypeResponseChunk, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeResponseEnd:
		return ChatEvent{Type: EventTypeResponseEnd, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeThinkingContent:
		return ChatEvent{Type: EventTypeThinkingContent, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeError:
		return ChatEvent{Type: EventTypeError, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeStep:
		return ChatEvent{Type: EventTypeStep, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeToolStart:
		return ChatEvent{Type: EventTypeToolStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeToolEnd:
		return ChatEvent{Type: EventTypeToolEnd, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeConfirmationRequest:
		return ChatEvent{Type: EventTypeConfirmationRequest, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeInteractionRequest:
		// New event stream interaction request
		return ChatEvent{Type: EventTypeConfirmationRequest, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeCommandMatched:
		return ChatEvent{Type: EventTypeCommandMatched, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeCommandResult:
		return ChatEvent{Type: EventTypeCommandResult, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeTaskCreate:
		return ChatEvent{Type: EventTypeTaskCreate, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeTaskUpdate:
		return ChatEvent{Type: EventTypeTaskUpdate, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeTaskList:
		return ChatEvent{Type: EventTypeTaskList, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanCreated:
		return ChatEvent{Type: EventTypePlanCreated, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanReviewStart:
		return ChatEvent{Type: EventTypePlanReviewStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanReviewFiles:
		return ChatEvent{Type: EventTypePlanReviewFiles, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanStep:
		return ChatEvent{Type: EventTypePlanStep, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanComplete:
		return ChatEvent{Type: EventTypePlanComplete, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanModeExit:
		return ChatEvent{Type: EventTypePlanModeExit, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeEnterPlanMode:
		return ChatEvent{Type: EventTypeEnterPlanMode, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanVerifyStart:
		return ChatEvent{Type: EventTypePlanVerifyStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanVerifyResult:
		return ChatEvent{Type: EventTypePlanVerifyResult, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypePlanVerifyEnd:
		return ChatEvent{Type: EventTypePlanVerifyEnd, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeSnapshotCreated:
		return ChatEvent{Type: EventTypeSnapshotCreated, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeRollbackComplete:
		return ChatEvent{Type: EventTypeRollbackComplete, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
	case sdk.EventTypeAgentStart:
		return ChatEvent{Type: EventTypeAgentStart, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeAgentStop:
		return ChatEvent{Type: EventTypeAgentStop, Content: ev.Content(), RequestID: ev.RequestID()}
	case sdk.EventTypeDone:
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID()}
	default:
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID()}
	}
}