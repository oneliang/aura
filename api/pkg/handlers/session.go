package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/aura/api/pkg/middleware"
	"github.com/oneliang/aura/session/pkg/model"
)

// SessionRepository provides session CRUD operations.
type SessionRepository interface {
	ListSessions(userID string) ([]*model.Session, error)
	CreateSession(userID string, name string, subscriptions []model.Subscription, systemPrompt string) (*model.Session, error)
	GetSession(id string, userID string) (*model.Session, error)
	UpdateSession(id string, userID string, systemPrompt *string, role *string) error
	DeleteSession(id string, userID string) error
}

// MessageRepository provides message management operations.
type MessageRepository interface {
	GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]model.Message, error)
	SendMessage(ctx context.Context, sessionID, content string) error
	AppendMessage(ctx context.Context, sessionID, role, content, source string) error
}

// SubscriptionRepository provides subscription management operations.
type SubscriptionRepository interface {
	AddSubscription(sessionID string, sub model.Subscription) error
	RemoveSubscription(sessionID, subID, trigger string) error
}

// SessionService provides session management capabilities.
// It is a composite interface combining session, message, and subscription operations.
type SessionService interface {
	SessionRepository
	MessageRepository
	SubscriptionRepository
}

// SessionHandler handles session-related HTTP requests.
type SessionHandler struct {
	service SessionService
}

// NewSessionHandler creates a new session handler.
func NewSessionHandler(service SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

// HandleListSessions handles GET /api/sessions.
func (h *SessionHandler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	sessions, err := h.service.ListSessions(userID)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"sessions": sessions,
		},
	})
}

// HandleCreateSession handles POST /api/sessions.
func (h *SessionHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name          string               `json:"name"`
		SystemPrompt  string               `json:"system_prompt,omitempty"`
		Subscriptions []model.Subscription `json:"subscriptions,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		WriteError(w, "Session name is required", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r)
	session, err := h.service.CreateSession(userID, req.Name, req.Subscriptions, req.SystemPrompt)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, Response{
		Status: "created",
		Data: map[string]interface{}{
			"session": session,
		},
	})
}

// HandleGetSession handles GET /api/sessions/{id}.
func (h *SessionHandler) HandleGetSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	session, err := h.service.GetSession(sessionID, userID)
	if err != nil {
		WriteError(w, fmt.Sprintf("Session not found: %s", sessionID), http.StatusNotFound)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"session": session,
		},
	})
}

// HandleUpdateSession handles PUT/PATCH /api/sessions/{id}.
func (h *SessionHandler) HandleUpdateSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name         string `json:"name,omitempty"`
		SystemPrompt string `json:"system_prompt,omitempty"`
		Role         string `json:"role,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate: role and prompt are mutually exclusive
	if req.Role != "" && req.SystemPrompt != "" {
		WriteError(w, "role and system_prompt are mutually exclusive", http.StatusBadRequest)
		return
	}

	var role *string
	var prompt *string
	if req.Role != "" {
		role = &req.Role
	}
	if req.SystemPrompt != "" {
		prompt = &req.SystemPrompt
	}

	userID := middleware.GetUserID(r)
	if err := h.service.UpdateSession(sessionID, userID, prompt, role); err != nil {
		WriteError(w, fmt.Sprintf("Failed to update session: %v", err), http.StatusInternalServerError)
		return
	}

	session, err := h.service.GetSession(sessionID, userID)
	if err != nil {
		WriteError(w, fmt.Sprintf("Session not found: %s", sessionID), http.StatusNotFound)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"session": session,
		},
	})
}

// HandleDeleteSession handles DELETE /api/sessions/{id}.
func (h *SessionHandler) HandleDeleteSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodDelete {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	if err := h.service.DeleteSession(sessionID, userID); err != nil {
		WriteError(w, fmt.Sprintf("Failed to delete session: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetSessionMessages handles GET /api/sessions/{id}/messages.
func (h *SessionHandler) HandleGetSessionMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify session exists and user owns it
	userID := middleware.GetUserID(r)
	_, err := h.service.GetSession(sessionID, userID)
	if err != nil {
		WriteError(w, fmt.Sprintf("Session not found: %s", sessionID), http.StatusNotFound)
		return
	}

	messages, err := h.service.GetMessages(r.Context(), sessionID, 100, userID)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to get messages: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"messages": messages,
		},
	})
}

// HandleSendMessage handles POST /api/sessions/{id}/message.
func (h *SessionHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request, sessionID string, processFn func(ctx context.Context, sid, content string) error) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
		Source  string `json:"source,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		WriteError(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Return acknowledgment immediately (async processing)
	WriteJSON(w, http.StatusAccepted, Response{
		Status:  "accepted",
		Message: "Message received and will be processed",
	})

	// Process message in background
	go func() {
		// Use context.Background() instead of r.Context() because the request context
		// will be cancelled when the response is sent, but we need to continue processing
		if err := processFn(context.Background(), sessionID, req.Content); err != nil {
			// Log error but don't return (async processing)
			return
		}
	}()
}

// HandleSessionSubscriptions handles POST/DELETE /api/sessions/{id}/subscriptions.
func (h *SessionHandler) HandleSessionSubscriptions(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodPost:
		h.addSubscription(w, r, sessionID)
	case http.MethodDelete:
		h.removeSubscription(w, r, sessionID)
	default:
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SessionHandler) addSubscription(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req struct {
		Trigger string `json:"trigger"`
		Source  string `json:"source,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Trigger == "" {
		WriteError(w, "trigger is required", http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		req.Source = "*"
	}

	newSub := model.Subscription{
		ID:      fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Trigger: req.Trigger,
		Source:  req.Source,
		Active:  true,
	}

	if err := h.service.AddSubscription(sessionID, newSub); err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, fmt.Sprintf("Session not found: %s", sessionID), http.StatusNotFound)
			return
		}
		WriteError(w, fmt.Sprintf("Failed to save session: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, Response{
		Status: "created",
		Data: map[string]interface{}{
			"subscription": newSub,
		},
	})
}

func (h *SessionHandler) removeSubscription(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req struct {
		Trigger string `json:"trigger,omitempty"`
		ID      string `json:"id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Trigger == "" && req.ID == "" {
		WriteError(w, "either trigger or id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveSubscription(sessionID, req.ID, req.Trigger); err != nil {
		WriteError(w, "Subscription not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}