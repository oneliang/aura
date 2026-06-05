package runtime

import (
	"github.com/oneliang/aura/shared/pkg/events"
)

// convertEvent converts an agent event to a runtime event.
func (r *AgentRuntime) convertEvent(ev events.Event) Event {
	requestID := ev.RequestID()
	switch ev.Type() {
	// Engine execution events
	case events.EventTypeThinkingStart:
		return events.NewEvent(events.EventTypeThinkingStart, ev.Content(), requestID)
	case events.EventTypeThinkingChunk:
		return events.NewEvent(events.EventTypeThinkingChunk, ev.Content(), requestID)
	case events.EventTypeThinkingContent:
		return events.NewEvent(events.EventTypeThinkingContent, ev.Content(), requestID)
	case events.EventTypeThinkingEnd:
		return events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
	case events.EventTypeAction:
		return events.NewEvent(events.EventTypeAction, ev.Content(), requestID)
	case events.EventTypeResult:
		return events.NewEvent(events.EventTypeResult, ev.Content(), requestID)
	case events.EventTypeResponse:
		return events.NewEvent(events.EventTypeResponse, ev.Content(), requestID)
	case events.EventTypeResponseStart:
		return events.NewEvent(events.EventTypeResponseStart, ev.Content(), requestID)
	case events.EventTypeResponseChunk:
		return events.NewEvent(events.EventTypeResponseChunk, ev.Content(), requestID)
	case events.EventTypeResponseEnd:
		return events.NewEvent(events.EventTypeResponseEnd, "", requestID)
	case events.EventTypeError:
		return events.NewEvent(events.EventTypeError, ev.Content(), requestID)
	case events.EventTypeStep:
		return events.NewEventWithExtra(events.EventTypeStep, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeToolStart:
		return events.NewEventWithExtra(events.EventTypeToolStart, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeToolEnd:
		return events.NewEventWithExtra(events.EventTypeToolEnd, ev.Content(), ev.Extra(), requestID)

	// Planning events
	case events.EventTypePlanCreated:
		return events.NewEventWithExtra(events.EventTypePlanCreated, ev.Content(), ev.Extra(), requestID)
	case events.EventTypePlanStep:
		return events.NewEventWithExtra(events.EventTypePlanStep, ev.Content(), ev.Extra(), requestID)
	case events.EventTypePlanComplete:
		return events.NewEventWithExtra(events.EventTypePlanComplete, ev.Content(), ev.Extra(), requestID)

	// Memory events
	case events.EventTypeMemoryClearRequest:
		return events.NewEvent(events.EventTypeMemoryClearRequest, ev.Content(), requestID)
	case events.EventTypeMemoryCleared:
		return events.NewEvent(events.EventTypeMemoryCleared, ev.Content(), requestID)
	case events.EventTypeMemoryStatsRequest:
		return events.NewEvent(events.EventTypeMemoryStatsRequest, ev.Content(), requestID)
	case events.EventTypeMemoryStats:
		return events.NewEventWithExtra(events.EventTypeMemoryStats, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeMemoryCompacted:
		return events.NewEvent(events.EventTypeMemoryCompacted, ev.Content(), requestID)

	// Session events
	case events.EventTypeSessionCreated:
		return events.NewEvent(events.EventTypeSessionCreated, ev.Content(), requestID)
	case events.EventTypeSessionSwitched:
		return events.NewEvent(events.EventTypeSessionSwitched, ev.Content(), requestID)
	case events.EventTypeSessionDeleted:
		return events.NewEvent(events.EventTypeSessionDeleted, ev.Content(), requestID)

	// Command events
	case events.EventTypeCommandRequest:
		return events.NewEventWithExtra(events.EventTypeCommandRequest, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeCommandResponse:
		return events.NewEvent(events.EventTypeCommandResponse, ev.Content(), requestID)
	case events.EventTypeCommandMatched:
		return events.NewEventWithExtra(events.EventTypeCommandMatched, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeCommandResult:
		return events.NewEvent(events.EventTypeCommandResult, ev.Content(), requestID)

	// Task tracking events
	case events.EventTypeTaskCreate:
		return events.NewEventWithExtra(events.EventTypeTaskCreate, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeTaskUpdate:
		return events.NewEventWithExtra(events.EventTypeTaskUpdate, ev.Content(), ev.Extra(), requestID)
	case events.EventTypeTaskList:
		return events.NewEventWithExtra(events.EventTypeTaskList, ev.Content(), ev.Extra(), requestID)

	// Confirmation events
	case events.EventTypeConfirmationRequest:
		return events.NewEventWithExtra(events.EventTypeConfirmationRequest, ev.Content(), ev.Extra(), requestID)

	// Done event
	case events.EventTypeDone:
		return events.NewEvent(events.EventTypeDone, "", requestID)

	// Unknown event types are logged but passed through (not dropped)
	default:
		r.logger.Warn("convertEvent: unknown event type, passing through", "module", "runtime", "event_type", string(ev.Type()))
		return ev
	}
}
