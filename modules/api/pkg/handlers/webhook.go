package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oneliang/aura/session/pkg/trigger"
)

// WebhookService provides webhook processing capabilities.
type WebhookService interface {
	ProcessEvent(ctx context.Context, event trigger.Event) error
}

// WebhookHandler handles webhook-related HTTP requests.
type WebhookHandler struct {
	service WebhookService
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(service WebhookService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

// HandleFeishuWebhook handles POST /api/webhooks/feishu.
func (h *WebhookHandler) HandleFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Feishu webhook payload
	feishuTrigger := trigger.NewFeishuTrigger()
	event, err := feishuTrigger.Parse(body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Invalid Feishu payload: %v", err), http.StatusBadRequest)
		return
	}

	// Route event to session
	if err := h.service.ProcessEvent(r.Context(), *event); err != nil {
		WriteError(w, fmt.Sprintf("Failed to process event: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "success",
	})
}

// HandleCronTrigger handles POST /api/cron.
func (h *WebhookHandler) HandleCronTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse cron payload
	cronTrigger := trigger.NewCronTrigger()
	event, err := cronTrigger.Parse(body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Invalid cron payload: %v", err), http.StatusBadRequest)
		return
	}

	// Route event to session
	if err := h.service.ProcessEvent(r.Context(), *event); err != nil {
		WriteError(w, fmt.Sprintf("Failed to process event: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "success",
	})
}

// HandleGenericWebhook handles POST /api/webhooks/{source}.
func (h *WebhookHandler) HandleGenericWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract source from path: /api/webhooks/{source}
	path := strings.TrimPrefix(r.URL.Path, "/api/webhooks/")
	source := strings.Split(path, "/")[0]

	if source == "" {
		WriteError(w, "Webhook source required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse generic webhook payload
	webhookTrigger := trigger.NewWebhookTrigger(source)
	event, err := webhookTrigger.Parse(body)
	if err != nil {
		WriteError(w, fmt.Sprintf("Invalid webhook payload: %v", err), http.StatusBadRequest)
		return
	}

	// Route event to session
	if err := h.service.ProcessEvent(r.Context(), *event); err != nil {
		WriteError(w, fmt.Sprintf("Failed to process event: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "success",
	})
}
