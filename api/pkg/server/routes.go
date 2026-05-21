package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/aura/api/pkg/handlers"
)

// handleSessions handles GET /api/sessions and POST /api/sessions.
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.sessionHandler.HandleListSessions(w, r)
	case http.MethodPost:
		s.sessionHandler.HandleCreateSession(w, r)
	default:
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSubscriptions handles GET /api/subscriptions and POST /api/subscriptions.
func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.subscriptionHandler.HandleListSubscriptions(w, r)
	case http.MethodPost:
		s.subscriptionHandler.HandleCreateSubscription(w, r)
	default:
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionByID handles operations on a specific session.
func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID, op, ok := parseSessionPath(r.URL.Path)
	if !ok {
		handlers.WriteError(w, "Invalid session path", http.StatusBadRequest)
		return
	}

	switch op {
	case "messages":
		s.sessionHandler.HandleGetSessionMessages(w, r, sessionID)
	case "input_history":
		s.handleGetInputHistory(w, r, sessionID)
	case "message":
		// Check if client wants SSE streaming
		if r.Header.Get("Accept") == "text/event-stream" {
			s.handleSendMessageSSE(w, r, sessionID)
		} else {
			s.sessionHandler.HandleSendMessage(w, r, sessionID, s.SendMessage)
		}
	case "subscriptions":
		s.sessionHandler.HandleSessionSubscriptions(w, r, sessionID)
	case "get", "":
		// Determine operation from HTTP method for base session path
		switch r.Method {
		case http.MethodGet:
			s.sessionHandler.HandleGetSession(w, r, sessionID)
		case http.MethodPut, http.MethodPatch:
			s.sessionHandler.HandleUpdateSession(w, r, sessionID)
		case http.MethodPost:
			s.sessionHandler.HandleSendMessage(w, r, sessionID, s.SendMessage)
		case http.MethodDelete:
			s.sessionHandler.HandleDeleteSession(w, r, sessionID)
		default:
			handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleHealth handles GET /api/health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	handlers.WriteJSON(w, http.StatusOK, handlers.Response{
		Status: "healthy",
	})
}

// handleSkills handles GET /api/skills.
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.skillsHandler.HandleListSkills(w, r)
}

// handleWebhooks handles webhook requests from various sources.
func (s *Server) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	source := extractWebhookSource(r.URL.Path)
	if source == "" {
		handlers.WriteError(w, "Webhook source required", http.StatusBadRequest)
		return
	}

	if source == "feishu" {
		s.webhookHandler.HandleFeishuWebhook(w, r)
	} else {
		s.webhookHandler.HandleGenericWebhook(w, r)
	}
}

// handleCronTrigger handles POST /api/cron for cron-triggered tasks.
func (s *Server) handleCronTrigger(w http.ResponseWriter, r *http.Request) {
	s.webhookHandler.HandleCronTrigger(w, r)
}

// parseSessionPath extracts session ID and operation from URL path.
// Returns (sessionID, operation, ok).
// Operations: "get", "update", "delete", "messages", "message", "subscriptions".
func parseSessionPath(path string) (sessionID, operation string, ok bool) {
	// Path format: /api/sessions/{id}[/{operation}]
	path = strings.TrimPrefix(path, "/api/sessions/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	sessionID = parts[0]

	if len(parts) == 1 {
		// No operation specified - default to get
		return sessionID, "get", true
	}

	// Determine operation from HTTP method and path
	op := parts[1]
	switch op {
	case "messages":
		return sessionID, "messages", true
	case "input_history":
		return sessionID, "input_history", true
	case "message":
		return sessionID, "message", true
	case "subscriptions":
		return sessionID, "subscriptions", true
	default:
		// For session operations (update/delete), return empty op
		// Caller should determine operation from HTTP method
		return sessionID, "", false
	}
}

// handleIndex serves the main web UI at /.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	// Serve index.html from embedded filesystem
	f, err := webFS.Open("index.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	http.ServeContent(w, r, "index.html", time.Time{}, f)
}

// extractWebhookSource extracts the webhook source from the URL path.
// Path format: /api/webhooks/{source}
func extractWebhookSource(path string) string {
	path = strings.TrimPrefix(path, "/api/webhooks/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return parts[0]
}
