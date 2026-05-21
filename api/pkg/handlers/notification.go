package handlers

import (
	"context"
	"encoding/json"
	"net/http"
)

// Default message type for notifications.
const defaultMsgType = "text"

// NotificationService provides notification sending capabilities.
type NotificationService interface {
	SendNotification(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error
	SendTaskNotification(ctx context.Context, taskID, status, result string) error
}

// NotificationHandler handles notification-related HTTP requests.
type NotificationHandler struct {
	service NotificationService
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(service NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// handleSendNotification handles POST /api/notifications.
func (h *NotificationHandler) HandleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string                 `json:"session_id"`
		MsgType   string                 `json:"msg_type"`
		Content   map[string]interface{} `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		WriteError(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if req.MsgType == "" {
		req.MsgType = defaultMsgType
	}
	if req.Content == nil {
		WriteError(w, "content is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.service.SendNotification(ctx, req.SessionID, req.MsgType, req.Content); err != nil {
		WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteOK(w, nil, "Notification sent")
}

// handleTaskNotification handles POST /api/notifications/task.
func (h *NotificationHandler) HandleTaskNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
		Result string `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TaskID == "" {
		WriteError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.service.SendTaskNotification(ctx, req.TaskID, req.Status, req.Result); err != nil {
		WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteOK(w, nil, "Task notification sent")
}
