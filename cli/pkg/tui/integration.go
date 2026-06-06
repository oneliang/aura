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
// sharedEventCh: externally provided shared event channel for multi-runtime scenarios.
func Run(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager, sharedEventCh chan sdk.Event) error {
	startTime := time.Now()
	log.Debug("[DIAG] Run: starting")

	// Start runtime event stream
	if err := rt.Start(ctx); err != nil {
		return err
	}
	log.Debug("[DIAG] Run: rt.Start done", "elapsed", time.Since(startTime))

	// Create TUI model with runtime reference and shared event channel
	m := NewWithRuntime(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager, sharedEventCh)
	log.Debug("[DIAG] Run: NewWithRuntime done", "elapsed", time.Since(startTime))

	// Create orchestrator for event stream integration
	// Note: With sharedEventCh, runtime events go directly to shared channel
	// Orchestrator only handles TUI.out → Runtime.in (user input/responses)
	orchestrator := NewOrchestrator(rt, m, sharedEventCh)

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
func RunWithConfig(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager, sharedEventCh chan sdk.Event) error {
	return Run(ctx, rt, config, sessionMgr, summarizer, modelProvider, commandProvider, mcpManager, sharedEventCh)
}

// Orchestrator integrates Runtime and TUI event streams.
type Orchestrator struct {
	runtime        *sdk.Runtime
	model          *Model
	sharedEventCh  chan sdk.Event // Shared event channel for multi-runtime scenarios
	wg             sync.WaitGroup // WaitGroup for goroutine lifecycle
	ready          chan struct{}  // Signal when goroutines are ready to receive events
}

// NewOrchestrator creates an orchestrator for event stream integration.
func NewOrchestrator(rt *sdk.Runtime, m *Model, sharedEventCh chan sdk.Event) *Orchestrator {
	return &Orchestrator{
		runtime:       rt,
		model:         m,
		sharedEventCh: sharedEventCh,
		ready:         make(chan struct{}),
	}
}

// Run starts event stream forwarding between Runtime and TUI.
// Signals readiness via ready channel before entering main loop.
func (o *Orchestrator) Run(ctx context.Context) {
	// 新架构：使用共享通道监听 runtime 事件
	// 共享模式下，runtime.Events() 返回 nil，所以从 sharedEventCh 监听
	if o.sharedEventCh != nil {
		// Forward: SharedEventCh → TUI.eventChan
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			adapter := NewAdapter()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-o.sharedEventCh:
					if !ok {
						return
					}
					// Convert to ChatEvent and send to TUI event channel
					chatEvent := adapter.ConvertSDKEvent(event)
					select {
					case o.model.eventChan <- chatEvent:
					default:
						// Channel full, skip event
					}
				}
			}
		}()
	} else {
		// 旧架构：独立模式，从 runtime.Events() 监听
		agentEvents := o.runtime.Events()
		if agentEvents != nil {
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
		}
	}

	// Forward: TUI.out → Runtime.in (用户输入/交互响应)
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
		return ChatEvent{Type: EventTypeThinkingStart, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeThinkingChunk:
		return ChatEvent{Type: EventTypeThinkingChunk, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeThinkingEnd:
		return ChatEvent{Type: EventTypeThinkingEnd, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeAction:
		return ChatEvent{Type: EventTypeAction, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeResult:
		return ChatEvent{Type: EventTypeResult, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeResponse:
		return ChatEvent{Type: EventTypeResponse, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeResponseStart:
		return ChatEvent{Type: EventTypeResponseStart, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeResponseChunk:
		return ChatEvent{Type: EventTypeResponseChunk, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeResponseEnd:
		return ChatEvent{Type: EventTypeResponseEnd, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeThinkingContent:
		return ChatEvent{Type: EventTypeThinkingContent, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeError:
		return ChatEvent{Type: EventTypeError, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeStep:
		return ChatEvent{Type: EventTypeStep, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeToolStart:
		return ChatEvent{Type: EventTypeToolStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeToolEnd:
		return ChatEvent{Type: EventTypeToolEnd, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeConfirmationRequest:
		return ChatEvent{Type: EventTypeConfirmationRequest, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeInteractionRequest:
		// New event stream interaction request
		return ChatEvent{Type: EventTypeConfirmationRequest, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeCommandMatched:
		return ChatEvent{Type: EventTypeCommandMatched, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeCommandResult:
		return ChatEvent{Type: EventTypeCommandResult, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeTaskCreate:
		return ChatEvent{Type: EventTypeTaskCreate, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeTaskUpdate:
		return ChatEvent{Type: EventTypeTaskUpdate, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeTaskList:
		return ChatEvent{Type: EventTypeTaskList, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanCreated:
		return ChatEvent{Type: EventTypePlanCreated, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanReviewStart:
		return ChatEvent{Type: EventTypePlanReviewStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanReviewFiles:
		return ChatEvent{Type: EventTypePlanReviewFiles, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanStep:
		return ChatEvent{Type: EventTypePlanStep, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanComplete:
		return ChatEvent{Type: EventTypePlanComplete, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanModeExit:
		return ChatEvent{Type: EventTypePlanModeExit, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeEnterPlanMode:
		return ChatEvent{Type: EventTypeEnterPlanMode, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanVerifyStart:
		return ChatEvent{Type: EventTypePlanVerifyStart, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanVerifyResult:
		return ChatEvent{Type: EventTypePlanVerifyResult, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypePlanVerifyEnd:
		return ChatEvent{Type: EventTypePlanVerifyEnd, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeSnapshotCreated:
		return ChatEvent{Type: EventTypeSnapshotCreated, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeRollbackComplete:
		return ChatEvent{Type: EventTypeRollbackComplete, Content: ev.Content(), Extra: ev.Extra(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeAgentStart:
		return ChatEvent{Type: EventTypeAgentStart, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeAgentStop:
		return ChatEvent{Type: EventTypeAgentStop, Content: ev.Content(), RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID()}
	case sdk.EventTypeDone:
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID(), Reason: DoneReasonNormal}
	default:
		return ChatEvent{Type: EventTypeDone, RequestID: ev.RequestID(), RuntimeID: ev.RuntimeID(), Reason: DoneReasonNormal}
	}
}