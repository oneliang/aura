package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/oneliang/aura/session/pkg/subscription"
)

// SubscriptionService provides subscription management capabilities.
type SubscriptionService interface {
	ListSubscriptions() []*subscription.Subscription
	CreateSubscription(sessionID, eventType, cronExpr string, config map[string]interface{}) (*subscription.Subscription, error)
	TriggerSubscription(id string) error
	DeleteSubscription(id string) error
}

// SubscriptionHandler handles subscription-related HTTP requests.
type SubscriptionHandler struct {
	service SubscriptionService
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(service SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

// handleListSubscriptions handles GET /api/subscriptions.
func (h *SubscriptionHandler) HandleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subs := h.service.ListSubscriptions()

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"subscriptions": subs,
			"count":         len(subs),
		},
	})
}

// handleCreateSubscription handles POST /api/subscriptions.
func (h *SubscriptionHandler) HandleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string                 `json:"session_id"`
		EventType string                 `json:"event_type"`
		CronExpr  string                 `json:"cron_expr"`
		Config    map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		WriteError(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if req.EventType == "" {
		WriteError(w, "event_type is required", http.StatusBadRequest)
		return
	}
	if req.CronExpr == "" {
		WriteError(w, "cron_expr is required", http.StatusBadRequest)
		return
	}

	sub, err := h.service.CreateSubscription(req.SessionID, req.EventType, req.CronExpr, req.Config)
	if err != nil {
		WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusCreated, Response{
		Status: "created",
		Data: map[string]interface{}{
			"subscription": sub,
		},
	})
}

// handleTriggerSubscription handles POST /api/subscriptions/trigger?id={id}.
func (h *SubscriptionHandler) HandleTriggerSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		WriteError(w, "subscription id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.TriggerSubscription(id); err != nil {
		WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteOK(w, nil, "Subscription triggered manually")
}

// handleDeleteSubscription handles DELETE /api/subscriptions/delete?id={id}.
func (h *SubscriptionHandler) HandleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		WriteError(w, "subscription id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteSubscription(id); err != nil {
		WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteOK(w, nil, "Subscription removed")
}
