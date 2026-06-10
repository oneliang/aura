// Package server provides SSE handling functionality.
package server

import (
	"encoding/json"
	"time"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
)

// SSEAdapter converts SDK events to SSE format.
type SSEAdapter struct{}

// SSEEvent represents a formatted SSE event ready for transmission.
type SSEEvent struct {
	Name string
	Data map[string]interface{}
}

// NewSSEAdapter creates a new SSEAdapter instance.
func NewSSEAdapter() *SSEAdapter {
	return &SSEAdapter{}
}

// ConvertToSSE converts an SDK event to SSE format.
// Returns the SSE event name and data map.
// Returns nil if the event type is not recognized.
func (a *SSEAdapter) ConvertToSSE(ev sdk.Event) *SSEEvent {
	switch ev.Type() {
	case sdk.EventTypeResponse:
		return &SSEEvent{
			Name: string(sdk.EventTypeResponse),
			Data: map[string]interface{}{
				"role":    "assistant",
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeResponseChunk:
		return &SSEEvent{
			Name: string(sdk.EventTypeResponseChunk),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeResponseStart:
		return &SSEEvent{
			Name: string(sdk.EventTypeResponseStart),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeResponseEnd:
		return &SSEEvent{
			Name: string(sdk.EventTypeResponseEnd),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeThinkingStart:
		return &SSEEvent{
			Name: string(sdk.EventTypeThinkingStart),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeThinkingChunk:
		return &SSEEvent{
			Name: string(sdk.EventTypeThinkingChunk),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeThinkingEnd:
		return &SSEEvent{
			Name: string(sdk.EventTypeThinkingEnd),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeToolStart:
		extra := ev.Extra()
		return &SSEEvent{
			Name: string(sdk.EventTypeToolStart),
			Data: map[string]interface{}{
				"tool":         extra["tool"],
				"params":       extra["params"],
				"execution_id": extra["execution_id"],
			},
		}
	case sdk.EventTypeToolEnd:
		extra := ev.Extra()
		// Convert duration to milliseconds for JSON serialization
		durationMs := 0
		switch d := extra["duration"].(type) {
		case time.Duration:
			durationMs = int(d.Milliseconds())
		case int64:
			durationMs = int(time.Duration(d).Milliseconds())
		case float64:
			durationMs = int(time.Duration(int64(d)).Milliseconds())
		case int:
			durationMs = int(time.Duration(d).Milliseconds())
		}
		return &SSEEvent{
			Name: string(sdk.EventTypeToolEnd),
			Data: map[string]interface{}{
				"tool":         extra["tool"],
				"result":       extra["result"],
				"execution_id": extra["execution_id"],
				"duration_ms":  durationMs,
			},
		}
	case sdk.EventTypeError:
		return &SSEEvent{
			Name: string(sdk.EventTypeError),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeTaskCreate:
		return &SSEEvent{
			Name: string(sdk.EventTypeTaskCreate),
			Data: map[string]interface{}{
				"task_id": ev.Extra()["task_id"],
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeTaskUpdate:
		return &SSEEvent{
			Name: string(sdk.EventTypeTaskUpdate),
			Data: map[string]interface{}{
				"task_id": ev.Extra()["task_id"],
				"status":  ev.Extra()["status"],
				"notes":   ev.Extra()["notes"],
			},
		}
	case sdk.EventTypeTaskList:
		return &SSEEvent{
			Name: string(sdk.EventTypeTaskList),
			Data: map[string]interface{}{
				"tasks": ev.Extra()["tasks"],
			},
		}
	case sdk.EventTypeAction:
		return &SSEEvent{
			Name: string(sdk.EventTypeAction),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeResult:
		return &SSEEvent{
			Name: string(sdk.EventTypeResult),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeCommandMatched:
		extra := ev.Extra()
		return &SSEEvent{
			Name: string(sdk.EventTypeCommandMatched),
			Data: map[string]interface{}{
				"content": ev.Content(),
				"command": extra["command"],
			},
		}
	case sdk.EventTypeCommandResult:
		return &SSEEvent{
			Name: string(sdk.EventTypeCommandResult),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeConfirmationRequest:
		extra := ev.Extra()
		return &SSEEvent{
			Name: string(sdk.EventTypeConfirmationRequest),
			Data: map[string]interface{}{
				"content":   ev.Content(),
				"tool_name": extra["tool_name"],
			},
		}
	case sdk.EventTypeStep:
		return &SSEEvent{
			Name: string(sdk.EventTypeStep),
			Data: map[string]interface{}{
				"content": ev.Content(),
				"step":    ev.Extra()["step"],
			},
		}
	case sdk.EventTypePlanCreated:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanCreated),
			Data: map[string]interface{}{
				"content":     ev.Content(),
				"total_steps": ev.Extra()["total_steps"],
				"goal":        ev.Extra()["goal"],
				"steps":       ev.Extra()["steps"],
			},
		}
	case sdk.EventTypePlanReviewStart:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanReviewStart),
			Data: map[string]interface{}{
				"content":     ev.Content(),
				"total_steps": ev.Extra()["total_steps"],
				"goal":        ev.Extra()["goal"],
			},
		}
	case sdk.EventTypePlanReviewFiles:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanReviewFiles),
			Data: map[string]interface{}{
				"files": ev.Extra()["files"],
			},
		}
	case sdk.EventTypePlanStep:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanStep),
			Data: map[string]interface{}{
				"content":     ev.Content(),
				"step_num":    ev.Extra()["step_num"],
				"total_steps": ev.Extra()["total_steps"],
				"step_desc":   ev.Extra()["step_desc"],
			},
		}
	case sdk.EventTypePlanModeExit:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanModeExit),
			Data: map[string]interface{}{
				"plan_file":   ev.Extra()["plan_file"],
				"total_steps": ev.Extra()["total_steps"],
				"goal":        ev.Extra()["goal"],
			},
		}
	case sdk.EventTypePlanComplete:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanComplete),
			Data: map[string]interface{}{
				"content": ev.Content(),
				"plan":    ev.Extra()["plan"],
			},
		}
	case sdk.EventTypeEnterPlanMode:
		return &SSEEvent{
			Name: string(sdk.EventTypeEnterPlanMode),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypePlanVerifyStart:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanVerifyStart),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypePlanVerifyResult:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanVerifyResult),
			Data: map[string]interface{}{
				"content": ev.Content(),
				"result":  ev.Extra()["result"],
			},
		}
	case sdk.EventTypePlanVerifyEnd:
		return &SSEEvent{
			Name: string(sdk.EventTypePlanVerifyEnd),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeSnapshotCreated:
		return &SSEEvent{
			Name: string(sdk.EventTypeSnapshotCreated),
			Data: map[string]interface{}{
				"snapshot_id": ev.Extra()["snapshot_id"],
			},
		}
	case sdk.EventTypeRollbackComplete:
		return &SSEEvent{
			Name: string(sdk.EventTypeRollbackComplete),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeMaxStepsExceeded:
		return &SSEEvent{
			Name: string(sdk.EventTypeMaxStepsExceeded),
			Data: map[string]interface{}{
				"content":   ev.Content(),
				"max_steps": ev.Extra()["max_steps"],
			},
		}
	case sdk.EventTypeAgentStart:
		return &SSEEvent{
			Name: string(sdk.EventTypeAgentStart),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeAgentStop:
		return &SSEEvent{
			Name: string(sdk.EventTypeAgentStop),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeInteractionRequest:
		return &SSEEvent{
			Name: string(sdk.EventTypeInteractionRequest),
			Data: map[string]interface{}{
				"content":          ev.Content(),
				"interaction_type": string(ev.InteractionType()),
				"request_id":       ev.RequestID(),
				"runtime_id":       ev.RuntimeID(),
			},
		}
	case sdk.EventTypeThinkingContent:
		// Deprecated but still used in some contexts
		return &SSEEvent{
			Name: string(sdk.EventTypeThinkingContent),
			Data: map[string]interface{}{
				"content": ev.Content(),
			},
		}
	case sdk.EventTypeDone:
		return &SSEEvent{
			Name: string(sdk.EventTypeDone),
			Data: map[string]interface{}{
				"status": "complete",
			},
		}
	default:
		return nil
	}
}

// formatJSON converts a map to JSON string.
func formatJSON(data map[string]interface{}) string {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(dataBytes)
}
