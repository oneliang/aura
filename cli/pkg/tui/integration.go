package tui

import (
	"context"

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
	// Start runtime event stream
	if err := rt.Start(ctx); err != nil {
		return err
	}

	// Create TUI model with runtime reference
	m := NewWithRuntime(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)

	// Create orchestrator for event stream integration
	orchestrator := NewOrchestrator(rt, m)

	// Start event stream forwarding in background
	go orchestrator.Run(ctx)

	// Run Bubble Tea program
	p := tea.NewProgram(m)
	_, err := p.Run()

	// Close TUI channels first, then stop runtime
	m.Close()
	rt.Stop(ctx)

	return err
}

// RunWithConfig starts the TUI program with configuration options.
func RunWithConfig(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) error {
	return Run(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)
}

// Orchestrator integrates Runtime and TUI event streams.
type Orchestrator struct {
	runtime *sdk.Runtime
	model   *Model
}

// NewOrchestrator creates an orchestrator for event stream integration.
func NewOrchestrator(rt *sdk.Runtime, m *Model) *Orchestrator {
	return &Orchestrator{
		runtime: rt,
		model:   m,
	}
}

// Run starts event stream forwarding between Runtime and TUI.
func (o *Orchestrator) Run(ctx context.Context) {
	// Get runtime event stream (OUT)
	agentEvents := o.runtime.Events()

	// Forward: Runtime.out → TUI.in
	go func() {
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
	go func() {
		uiEvents := o.model.Events()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-uiEvents:
				if !ok {
					return
				}
				if err := o.runtime.SendEvent(ctx, event); err != nil {
					log.Debug().Err(err).Msg("Orchestrator: failed to send event to runtime")
				}
			}
		}
	}()
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
	case sdk.EventTypeRollbackOffer:
		return ChatEvent{Type: EventTypeRollbackOffer, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID()}
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