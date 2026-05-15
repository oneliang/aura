// Package runtime provides the unified runtime for Aura.
package runtime

import (
	"time"

	"github.com/oneliang/aura/shared/pkg/events"
)

// EventType and Event types are now defined in shared/pkg/events.
// These type aliases are provided for backward compatibility.
type EventType = events.EventType
type Event = events.Event

// Event type constants - delegated to shared/events for consistency.
const (
	EventTypeThinkingStart       = events.EventTypeThinkingStart
	EventTypeThinkingChunk       = events.EventTypeThinkingChunk
	EventTypeThinkingEnd         = events.EventTypeThinkingEnd
	EventTypeAction              = events.EventTypeAction
	EventTypeResult              = events.EventTypeResult
	EventTypeResponse            = events.EventTypeResponse
	EventTypeResponseStart       = events.EventTypeResponseStart
	EventTypeResponseChunk       = events.EventTypeResponseChunk
	EventTypeResponseEnd         = events.EventTypeResponseEnd
	EventTypeThinkingContent     = events.EventTypeThinkingContent
	EventTypeError               = events.EventTypeError
	EventTypeStep                = events.EventTypeStep
	EventTypeToolStart           = events.EventTypeToolStart
	EventTypeToolEnd             = events.EventTypeToolEnd
	EventTypeConfirmationRequest = events.EventTypeConfirmationRequest
	EventTypeCommandMatched      = events.EventTypeCommandMatched
	EventTypeCommandResult       = events.EventTypeCommandResult
	EventTypeDone                = events.EventTypeDone
	EventTypeTaskCreate          = events.EventTypeTaskCreate
	EventTypeTaskUpdate          = events.EventTypeTaskUpdate
	EventTypeTaskList            = events.EventTypeTaskList
	EventTypePlanCreated         = events.EventTypePlanCreated
	EventTypePlanReviewStart     = events.EventTypePlanReviewStart
	EventTypePlanReviewFiles     = events.EventTypePlanReviewFiles
	EventTypePlanStep            = events.EventTypePlanStep
	EventTypePlanComplete        = events.EventTypePlanComplete
	EventTypePlanModeExit        = events.EventTypePlanModeExit
	EventTypeEnterPlanMode       = events.EventTypeEnterPlanMode
	EventTypePlanVerifyStart     = events.EventTypePlanVerifyStart
	EventTypePlanVerifyResult    = events.EventTypePlanVerifyResult
	EventTypePlanVerifyEnd       = events.EventTypePlanVerifyEnd
	EventTypeSnapshotCreated     = events.EventTypeSnapshotCreated
	EventTypeRollbackOffer       = events.EventTypeRollbackOffer
	EventTypeRollbackComplete    = events.EventTypeRollbackComplete
	EventTypeMaxStepsExceeded    = events.EventTypeMaxStepsExceeded
)

// NewEvent creates a new event.
func NewEvent(typ EventType, content string) Event {
	return events.NewEvent(typ, content, "")
}

// NewEventWithRequestID creates a new event with a request ID.
func NewEventWithRequestID(typ EventType, content string, requestID string) Event {
	return events.NewEvent(typ, content, requestID)
}

// NewEventWithExtra creates a new event with extra data.
func NewEventWithExtra(typ EventType, content string, extra map[string]any) Event {
	return events.NewEventWithExtra(typ, content, extra, "")
}

// NewEventWithExtraAndRequestID creates a new event with extra data and request ID.
func NewEventWithExtraAndRequestID(typ EventType, content string, extra map[string]any, requestID string) Event {
	return events.NewEventWithExtra(typ, content, extra, requestID)
}

// ConfirmationRequest represents a request for user confirmation.
type ConfirmationRequest = events.ConfirmationRequest

// EventConfirmationHandler is a callback for handling confirmation requests.
type EventConfirmationHandler func(req ConfirmationRequest)

// EventHandler is a callback for handling events.
type EventHandler func(Event)

// Timestamp returns the timestamp from an event.
// This helper is provided for backward compatibility.
func Timestamp(e Event) time.Time {
	return e.Timestamp()
}
