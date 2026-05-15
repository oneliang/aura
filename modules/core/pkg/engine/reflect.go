package engine

import (
	"github.com/oneliang/aura/shared/pkg/events"
)

// checkMaxSteps returns true if the step limit has been exceeded.
func checkMaxSteps(step, maxSteps int) bool {
	return maxSteps > 0 && step > maxSteps
}

// emitMaxStepsExceededEvent emits a max steps exceeded event.
func emitMaxStepsExceededEvent(eventsCh chan<- events.Event, maxSteps int, requestID string) {
	eventsCh <- events.NewEventWithExtra(
		events.EventTypeMaxStepsExceeded,
		"Maximum steps reached. Providing best available answer.",
		map[string]any{"max_steps": maxSteps},
		requestID,
	)
}
