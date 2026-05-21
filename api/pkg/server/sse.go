// Package server provides SSE handling functionality.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
)

const (
	sseEventMessage = "message"
	sseEventDone    = "done"
)

// sseRequest represents the parsed SSE request body.
type sseRequest struct {
	Content string `json:"content"`
	Source  string `json:"source,omitempty"`
}

// setupSSEHeaders sets the required SSE headers on the response writer.
func setupSSEHeaders(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return flusher
}

// parseSSERequest parses the SSE request body from the HTTP request.
func (s *Server) parseSSERequest(r *http.Request) (*sseRequest, error) {
	var req sseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid request body")
	}

	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	return &req, nil
}

// createSSEEventHandler creates an event handler that streams events to the SSE client.
func (s *Server) createSSEEventHandler(w http.ResponseWriter, flusher http.Flusher, done chan struct{}, responseContent *strings.Builder) func(ev sdk.Event) {
	var disconnected bool
	return func(ev sdk.Event) {
		s.logger.Debug().Str("module", "server").Str("event_type", string(ev.Type())).Msg("Event handler received event")
		select {
		case <-done:
			s.logger.Debug().Str("module", "server").Msg("Event handler: done channel closed, dropping event")
			return
		default:
			if !disconnected {
				if !s.processSSEEvent(w, flusher, ev, responseContent, done) {
					disconnected = true
					s.logger.Debug().Str("module", "server").Msg("SSE client disconnected, dropping future events")
				}
			}
		}
	}
}

// processSSEEvent processes a single SSE event and sends it to the client.
// Returns false if the client has disconnected.
func (s *Server) processSSEEvent(w http.ResponseWriter, flusher http.Flusher, ev sdk.Event, responseContent *strings.Builder, done chan struct{}) bool {
	switch ev.Type() {
	case sdk.EventTypeResponse:
		content := ev.Content()
		responseContent.WriteString(content)
		s.logger.Debug().Str("module", "server").Str("event_type", "response").Msg("Sending SSE response event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponse), map[string]interface{}{
			"role":    "assistant",
			"content": content,
		})
	case sdk.EventTypeThinkingStart:
		s.logger.Debug().Str("module", "server").Str("event_type", "thinking_start").Msg("Sending SSE thinking start event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingStart), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeThinkingEnd:
		s.logger.Debug().Str("module", "server").Str("event_type", "thinking_end").Msg("Sending SSE thinking end event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingEnd), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeThinkingChunk:
		s.logger.Debug().Str("module", "server").Str("event_type", "thinking_chunk").Msg("Sending SSE thinking chunk event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingChunk), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResponseStart:
		s.logger.Debug().Str("module", "server").Str("event_type", "response_start").Msg("Sending SSE response start event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseStart), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResponseEnd:
		s.logger.Debug().Str("module", "server").Str("event_type", "response_end").Msg("Sending SSE response end event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseEnd), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeToolStart:
		extra := ev.Extra()
		s.logger.Debug().Str("module", "server").Str("event_type", "tool_start").Str("tool", fmt.Sprintf("%v", extra["tool"])).Str("execution_id", fmt.Sprintf("%v", extra["execution_id"])).Msg("Sending SSE tool start event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeToolStart), map[string]interface{}{
			"tool":         extra["tool"],
			"params":       extra["params"],
			"execution_id": extra["execution_id"],
		})
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
		s.logger.Debug().Str("module", "server").Str("event_type", "tool_end").Str("tool", fmt.Sprintf("%v", extra["tool"])).Str("execution_id", fmt.Sprintf("%v", extra["execution_id"])).Int("duration_ms", durationMs).Msg("Sending SSE tool end event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeToolEnd), map[string]interface{}{
			"tool":         extra["tool"],
			"result":       extra["result"],
			"execution_id": extra["execution_id"],
			"duration_ms":  durationMs,
		})
	case sdk.EventTypeError:
		s.logger.Debug().Str("module", "server").Str("event_type", "error").Msg("Sending SSE error event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeError), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeTaskCreate:
		s.logger.Debug().Str("module", "server").Str("event_type", "task_create").Interface("task_id", ev.Extra()["task_id"]).Msg("Sending SSE task create event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskCreate), map[string]interface{}{
			"task_id": ev.Extra()["task_id"],
			"content": ev.Content(),
		})
	case sdk.EventTypeTaskUpdate:
		s.logger.Debug().Str("module", "server").Str("event_type", "task_update").Msg("Sending SSE task update event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskUpdate), map[string]interface{}{
			"task_id": ev.Extra()["task_id"],
			"status":  ev.Extra()["status"],
			"notes":   ev.Extra()["notes"],
		})
	case sdk.EventTypeTaskList:
		s.logger.Debug().Str("module", "server").Str("event_type", "task_list").Msg("Sending SSE task list event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskList), map[string]interface{}{
			"tasks": ev.Extra()["tasks"],
		})
	case sdk.EventTypeResponseChunk:
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseChunk), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeAction:
		s.logger.Debug().Str("module", "server").Str("event_type", "action").Msg("Sending SSE action event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeAction), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResult:
		s.logger.Debug().Str("module", "server").Str("event_type", "result").Msg("Sending SSE result event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResult), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeCommandMatched:
		extra := ev.Extra()
		s.logger.Debug().Str("module", "server").Str("event_type", "command_matched").Str("command", fmt.Sprintf("%v", extra["command"])).Msg("Sending SSE command matched event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeCommandMatched), map[string]interface{}{
			"content": ev.Content(),
			"command": extra["command"],
		})
	case sdk.EventTypeCommandResult:
		s.logger.Debug().Str("module", "server").Str("event_type", "command_result").Msg("Sending SSE command result event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeCommandResult), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeConfirmationRequest:
		extra := ev.Extra()
		s.logger.Debug().Str("module", "server").Str("event_type", "confirmation_request").Str("tool", fmt.Sprintf("%v", extra["toolName"])).Msg("Sending SSE confirmation request event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeConfirmationRequest), map[string]interface{}{
			"content":  ev.Content(),
			"toolName": extra["toolName"],
		})
	case sdk.EventTypeStep:
		s.logger.Debug().Str("module", "server").Str("event_type", "step").Msg("Sending SSE step event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeStep), map[string]interface{}{
			"content": ev.Content(),
			"step":    ev.Extra()["step"],
		})
	case sdk.EventTypePlanCreated:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_created").Msg("Sending SSE plan created event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanCreated), map[string]interface{}{
			"content":     ev.Content(),
			"total_steps": ev.Extra()["total_steps"],
			"goal":        ev.Extra()["goal"],
			"steps":       ev.Extra()["steps"],
		})
	case sdk.EventTypePlanReviewStart:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_review_start").Msg("Sending SSE plan review start event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanReviewStart), map[string]interface{}{
			"content":     ev.Content(),
			"total_steps": ev.Extra()["total_steps"],
			"goal":        ev.Extra()["goal"],
		})
	case sdk.EventTypePlanReviewFiles:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_review_files").Msg("Sending SSE plan review files event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanReviewFiles), map[string]interface{}{
			"files": ev.Extra()["files"],
		})
	case sdk.EventTypePlanStep:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_step").Msg("Sending SSE plan step event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanStep), map[string]interface{}{
			"content":     ev.Content(),
			"step_num":    ev.Extra()["step_num"],
			"total_steps": ev.Extra()["total_steps"],
			"step_desc":   ev.Extra()["step_desc"],
		})
	case sdk.EventTypePlanModeExit:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_mode_exit").Msg("Sending SSE plan mode exit event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanModeExit), map[string]interface{}{
			"plan_file":    ev.Extra()["plan_file"],
			"total_steps":  ev.Extra()["total_steps"],
			"goal":         ev.Extra()["goal"],
		})
	case sdk.EventTypePlanComplete:
		s.logger.Debug().Str("module", "server").Str("event_type", "plan_complete").Msg("Sending SSE plan complete event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanComplete), map[string]interface{}{
			"content": ev.Content(),
			"plan":    ev.Extra()["plan"],
		})
	case sdk.EventTypeDone:
		s.logger.Debug().Str("module", "server").Str("event_type", "done").Msg("Sending SSE done event")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeDone), map[string]interface{}{
			"status": "complete",
		})
	}
	return true
}

// streamSSEEvents consumes the events channel and processes them.
func (s *Server) streamSSEEvents(w http.ResponseWriter, flusher http.Flusher, events <-chan sdk.Event, sessionID string) {
	eventCount := 0
	for ev := range events {
		eventCount++
		s.logger.Debug().Str("module", "server").Str("session_id", sessionID).Str("event_type", string(ev.Type())).Msg("Consumed event")
	}
	s.logger.Debug().Str("module", "server").Str("session_id", sessionID).Int("event_count", eventCount).Msg("Events channel closed")
}
