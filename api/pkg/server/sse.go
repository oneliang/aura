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
	sseEventMessage  = "message"
	sseEventThinking = "thinking"
	sseEventTool     = "tool"
	sseEventError    = "error"
	sseEventDone     = "done"
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
		s.logger.Debug("Event handler received event", "module", "server", "event_type", string(ev.Type()))
		select {
		case <-done:
			s.logger.Debug("Event handler: done channel closed, dropping event", "module", "server")
			return
		default:
			if !disconnected {
				if !s.processSSEEvent(w, flusher, ev, responseContent, done) {
					disconnected = true
					s.logger.Debug("SSE client disconnected, dropping future events", "module", "server")
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
		s.logger.Debug("Sending SSE response event", "module", "server", "event_type", "response")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponse), map[string]interface{}{
			"role":    "assistant",
			"content": content,
		})
	case sdk.EventTypeThinkingStart:
		s.logger.Debug("Sending SSE thinking start event", "module", "server", "event_type", "thinking_start")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingStart), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeThinkingEnd:
		s.logger.Debug("Sending SSE thinking end event", "module", "server", "event_type", "thinking_end")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingEnd), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeThinkingChunk:
		s.logger.Debug("Sending SSE thinking chunk event", "module", "server", "event_type", "thinking_chunk")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeThinkingChunk), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResponseStart:
		s.logger.Debug("Sending SSE response start event", "module", "server", "event_type", "response_start")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseStart), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResponseEnd:
		s.logger.Debug("Sending SSE response end event", "module", "server", "event_type", "response_end")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseEnd), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeToolStart:
		extra := ev.Extra()
		s.logger.Debug("Sending SSE tool start event", "module", "server", "event_type", "tool_start", "tool", fmt.Sprintf("%v", extra["tool"]), "execution_id", fmt.Sprintf("%v", extra["execution_id"]))
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
		s.logger.Debug("Sending SSE tool end event", "module", "server", "event_type", "tool_end", "tool", fmt.Sprintf("%v", extra["tool"]), "execution_id", fmt.Sprintf("%v", extra["execution_id"]), "duration_ms", durationMs)
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeToolEnd), map[string]interface{}{
			"tool":         extra["tool"],
			"result":       extra["result"],
			"execution_id": extra["execution_id"],
			"duration_ms":  durationMs,
		})
	case sdk.EventTypeError:
		s.logger.Debug("Sending SSE error event", "module", "server", "event_type", "error")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeError), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeTaskCreate:
		s.logger.Debug("Sending SSE task create event", "module", "server", "event_type", "task_create", "task_id", ev.Extra()["task_id"])
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskCreate), map[string]interface{}{
			"task_id": ev.Extra()["task_id"],
			"content": ev.Content(),
		})
	case sdk.EventTypeTaskUpdate:
		s.logger.Debug("Sending SSE task update event", "module", "server", "event_type", "task_update")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskUpdate), map[string]interface{}{
			"task_id": ev.Extra()["task_id"],
			"status":  ev.Extra()["status"],
			"notes":   ev.Extra()["notes"],
		})
	case sdk.EventTypeTaskList:
		s.logger.Debug("Sending SSE task list event", "module", "server", "event_type", "task_list")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeTaskList), map[string]interface{}{
			"tasks": ev.Extra()["tasks"],
		})
	case sdk.EventTypeResponseChunk:
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResponseChunk), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeAction:
		s.logger.Debug("Sending SSE action event", "module", "server", "event_type", "action")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeAction), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeResult:
		s.logger.Debug("Sending SSE result event", "module", "server", "event_type", "result")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeResult), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeCommandMatched:
		extra := ev.Extra()
		s.logger.Debug("Sending SSE command matched event", "module", "server", "event_type", "command_matched", "command", fmt.Sprintf("%v", extra["command"]))
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeCommandMatched), map[string]interface{}{
			"content": ev.Content(),
			"command": extra["command"],
		})
	case sdk.EventTypeCommandResult:
		s.logger.Debug("Sending SSE command result event", "module", "server", "event_type", "command_result")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeCommandResult), map[string]interface{}{
			"content": ev.Content(),
		})
	case sdk.EventTypeConfirmationRequest:
		extra := ev.Extra()
		s.logger.Debug("Sending SSE confirmation request event", "module", "server", "event_type", "confirmation_request", "tool", fmt.Sprintf("%v", extra["toolName"]))
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeConfirmationRequest), map[string]interface{}{
			"content":  ev.Content(),
			"toolName": extra["toolName"],
		})
	case sdk.EventTypeStep:
		s.logger.Debug("Sending SSE step event", "module", "server", "event_type", "step")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypeStep), map[string]interface{}{
			"content": ev.Content(),
			"step":    ev.Extra()["step"],
		})
	case sdk.EventTypePlanCreated:
		s.logger.Debug("Sending SSE plan created event", "module", "server", "event_type", "plan_created")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanCreated), map[string]interface{}{
			"content":     ev.Content(),
			"total_steps": ev.Extra()["total_steps"],
			"goal":        ev.Extra()["goal"],
			"steps":       ev.Extra()["steps"],
		})
	case sdk.EventTypePlanReviewStart:
		s.logger.Debug("Sending SSE plan review start event", "module", "server", "event_type", "plan_review_start")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanReviewStart), map[string]interface{}{
			"content":     ev.Content(),
			"total_steps": ev.Extra()["total_steps"],
			"goal":        ev.Extra()["goal"],
		})
	case sdk.EventTypePlanReviewFiles:
		s.logger.Debug("Sending SSE plan review files event", "module", "server", "event_type", "plan_review_files")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanReviewFiles), map[string]interface{}{
			"files": ev.Extra()["files"],
		})
	case sdk.EventTypePlanStep:
		s.logger.Debug("Sending SSE plan step event", "module", "server", "event_type", "plan_step")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanStep), map[string]interface{}{
			"content":     ev.Content(),
			"step_num":    ev.Extra()["step_num"],
			"total_steps": ev.Extra()["total_steps"],
			"step_desc":   ev.Extra()["step_desc"],
		})
	case sdk.EventTypePlanModeExit:
		s.logger.Debug("Sending SSE plan mode exit event", "module", "server", "event_type", "plan_mode_exit")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanModeExit), map[string]interface{}{
			"plan_file":    ev.Extra()["plan_file"],
			"total_steps":  ev.Extra()["total_steps"],
			"goal":         ev.Extra()["goal"],
		})
	case sdk.EventTypePlanComplete:
		s.logger.Debug("Sending SSE plan complete event", "module", "server", "event_type", "plan_complete")
		return s.sendSSEEvent(w, flusher, string(sdk.EventTypePlanComplete), map[string]interface{}{
			"content": ev.Content(),
			"plan":    ev.Extra()["plan"],
		})
	case sdk.EventTypeDone:
		s.logger.Debug("Sending SSE done event", "module", "server", "event_type", "done")
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
		s.logger.Debug("Consumed event", "module", "server", "session_id", sessionID, "event_type", string(ev.Type()))
	}
	s.logger.Debug("Events channel closed", "module", "server", "session_id", sessionID, "event_count", eventCount)
}