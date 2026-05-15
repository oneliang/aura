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

// Run starts the TUI program.
func Run(ctx context.Context, runFn RunFunc, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) error {
	m := New(ctx, runFn, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

// RunWithConfig starts the TUI program with configuration options.
func RunWithConfig(ctx context.Context, runFn RunFunc, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) error {
	return Run(ctx, runFn, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager)
}

// Adapter provides utilities to adapt SDK events to TUI events.
type Adapter struct{}

// NewAdapter creates a new adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// ConvertSDKEvent converts an SDK event to a TUI chat event.
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

	// Plan mode events
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

	case sdk.EventTypeDone:
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID()}

	default:
		// Unknown event types are ignored (not converted to Response)
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID()} // Treat as done to avoid rendering
	}
}

// CreateRunFunc creates a RunFunc from an SDK runtime.
func CreateRunFunc(rt *sdk.Runtime) RunFunc {
	adapter := NewAdapter()

	return func(ctx context.Context, input string) (<-chan ChatEvent, error) {
		events, err := rt.Process(ctx, input)
		if err != nil {
			return nil, err
		}

		out := make(chan ChatEvent, 100)

		go func() {
			defer close(out)
			var lastRequestID string
			for ev := range events {
				lastRequestID = ev.RequestID()
				out <- adapter.ConvertSDKEvent(ev)
			}
			// Send done event with the last request ID
			out <- ChatEvent{Type: EventTypeDone, RequestID: lastRequestID}
		}()

		return out, nil
	}
}
