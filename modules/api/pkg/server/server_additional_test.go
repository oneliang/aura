// Package server provides additional tests for the server package.
package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneliang/aura/api/pkg/handlers"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/subscription"
	"github.com/oneliang/aura/session/pkg/trigger"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// TestParseSessionPath tests parseSessionPath function.
func TestParseSessionPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		wantID string
		wantOp string
		wantOK bool
	}{
		{
			name:   "session only",
			path:   "/api/sessions/session-123",
			wantID: "session-123",
			wantOp: "get",
			wantOK: true,
		},
		{
			name:   "session with messages",
			path:   "/api/sessions/session-123/messages",
			wantID: "session-123",
			wantOp: "messages",
			wantOK: true,
		},
		{
			name:   "session with message",
			path:   "/api/sessions/session-123/message",
			wantID: "session-123",
			wantOp: "message",
			wantOK: true,
		},
		{
			name:   "session with subscriptions",
			path:   "/api/sessions/session-123/subscriptions",
			wantID: "session-123",
			wantOp: "subscriptions",
			wantOK: true,
		},
		{
			name:   "empty path",
			path:   "/api/sessions/",
			wantID: "",
			wantOp: "",
			wantOK: false,
		},
		{
			name:   "no webhook path",
			path:   "/api/sessions",
			wantID: "",
			wantOp: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, op, ok := parseSessionPath(tt.path)
			if ok != tt.wantOK {
				t.Errorf("parseSessionPath() ok = %v, want %v", ok, tt.wantOK)
			}
			if sessionID != tt.wantID {
				t.Errorf("parseSessionPath() sessionID = %q, want %q", sessionID, tt.wantID)
			}
			if op != tt.wantOp {
				t.Errorf("parseSessionPath() op = %q, want %q", op, tt.wantOp)
			}
		})
	}
}

// TestExtractWebhookSource tests extractWebhookSource function.
func TestExtractWebhookSource(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "feishu webhook",
			path: "/api/webhooks/feishu",
			want: "feishu",
		},
		{
			name: "slack webhook",
			path: "/api/webhooks/slack",
			want: "slack",
		},
		{
			name: "webhook with trailing path",
			path: "/api/webhooks/slack/extra",
			want: "slack",
		},
		{
			name: "empty webhook",
			path: "/api/webhooks/",
			want: "",
		},
		{
			name: "no webhook path",
			path: "/api/webhooks",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := extractWebhookSource(tt.path)
			if source != tt.want {
				t.Errorf("extractWebhookSource() = %q, want %q", source, tt.want)
			}
		})
	}
}

// TestServer_handleSkills tests handleSkills method with nil handler.
func TestServer_handleSkills(t *testing.T) {
	// When skillsHandler is nil, calling handleSkills will panic
	// This test documents that behavior - handler must be set before routing
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Verify handler is nil by default
	if server.skillsHandler != nil {
		t.Fatal("Expected skillsHandler to be nil by default")
	}

	// Note: We don't actually call handleSkills here because it would panic
	// The test verifies that the server requires explicit handler setup
	t.Log("handleSkills requires skillsHandler to be set via SetSkillsHandler or direct assignment")
}

// TestServer_handleSkills_WithHandler tests handleSkills with handler.
func TestServer_handleSkills_WithHandler(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mockService := &mockSkillsService{
		listSkillsFunc: func() (string, error) {
			return "skill1, skill2", nil
		},
	}

	sh := handlers.NewSkillsHandler(mockService)
	server.skillsHandler = sh

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	w := httptest.NewRecorder()

	server.handleSkills(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestServer_handleSkills_MethodNotAllowed tests handleSkills with wrong method.
func TestServer_handleSkills_MethodNotAllowed(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mockService := &mockSkillsService{}
	sh := handlers.NewSkillsHandler(mockService)
	server.skillsHandler = sh

	req := httptest.NewRequest(http.MethodPost, "/api/skills", nil)
	w := httptest.NewRecorder()

	server.handleSkills(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestServer_handleWebhooks tests handleWebhooks method.
func TestServer_handleWebhooks(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	server.webhookHandler = nil

	// Test with empty source
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/", nil)
	w := httptest.NewRecorder()

	server.handleWebhooks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestServer_handleWebhooks_Feishu tests feishu webhook.
func TestServer_handleWebhooks_Feishu(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mockWebhookService := &mockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	server.webhookHandler = handlers.NewWebhookHandler(mockWebhookService)

	body := strings.NewReader(`{"msg_type":"text","content":{"text":"Hello"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu", body)
	w := httptest.NewRecorder()

	server.handleWebhooks(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestServer_handleWebhooks_Generic tests generic webhook.
func TestServer_handleWebhooks_Generic(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mockWebhookService := &mockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	server.webhookHandler = handlers.NewWebhookHandler(mockWebhookService)

	body := strings.NewReader(`{"event": "test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/slack", body)
	w := httptest.NewRecorder()

	server.handleWebhooks(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestServer_handleCronTrigger tests handleCronTrigger method.
func TestServer_handleCronTrigger(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mockWebhookService := &mockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	server.webhookHandler = handlers.NewWebhookHandler(mockWebhookService)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/cron", body)
	w := httptest.NewRecorder()

	server.handleCronTrigger(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestServer_handleHealth tests handleHealth method.
func TestServer_handleHealth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "healthy") {
		t.Error("Response should contain 'healthy'")
	}
}

// TestServer_handleSessions_Get tests handleSessions GET method.
func TestServer_handleSessions_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := server.manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	server.handleSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestServer_handleSessions_Post tests handleSessions POST method.
func TestServer_handleSessions_Post(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := strings.NewReader(`{"name": "Test Session"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", body)
	w := httptest.NewRecorder()

	server.handleSessions(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

// TestServer_handleSessions_MethodNotAllowed tests handleSessions with wrong method.
func TestServer_handleSessions_MethodNotAllowed(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions", nil)
	w := httptest.NewRecorder()

	server.handleSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestServer_handleSubscriptions_Get tests handleSubscriptions GET method.
func TestServer_handleSubscriptions_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session to have valid subscription target
	_, err := server.manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	w := httptest.NewRecorder()

	server.handleSubscriptions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestServer_handleSubscriptions_Post tests handleSubscriptions POST method.
func TestServer_handleSubscriptions_Post(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a valid session first
	session, err := server.manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Use 6-field cron expression (with seconds field)
	body := strings.NewReader(`{"session_id":"` + session.ID + `","event_type":"daily","cron_expr":"0 0 9 * * *"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", body)
	w := httptest.NewRecorder()

	server.handleSubscriptions(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 201 or 400, got %d", w.Code)
	}
}

// TestServer_handleSessionByID_Messages tests handleSessionByID with messages operation.
func TestServer_handleSessionByID_Messages(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test-id/messages", nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	// May return 404 for non-existent session or 200 for empty messages
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", w.Code)
	}
}

// TestServer_handleSessionByID_InvalidPath tests handleSessionByID with invalid path.
func TestServer_handleSessionByID_InvalidPath(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/", nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestServer_SetSkillsHandler tests SetSkillsHandler method.
func TestServer_SetSkillsHandler(t *testing.T) {
	mockService := &mockSkillsService{}
	sh := handlers.NewSkillsHandler(mockService)

	server, cleanup := setupTestServer(t)
	defer cleanup()

	server.SetSkillsHandler(sh)

	if server.skillsHandler != sh {
		t.Error("SetSkillsHandler() did not set handler")
	}
}

// TestServer_handleSessionByID_Update tests handleSessionByID with update operation.
func TestServer_handleSessionByID_Update(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session first
	session, _ := server.manager.CreateSession("Test", nil, "", "")

	body := strings.NewReader(`{"name": "Updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/sessions/"+session.ID, body)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestServer_handleSessionByID_Delete tests handleSessionByID with delete operation.
func TestServer_handleSessionByID_Delete(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session first
	session, _ := server.manager.CreateSession("To Delete", nil, "", "")

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID, nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

// TestServer_handleSessionByID_Post tests handleSessionByID with post (send message).
func TestServer_handleSessionByID_Post(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session first
	session, _ := server.manager.CreateSession("Test", nil, "", "")

	body := strings.NewReader(`{"content": "Hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID, body)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}
}

// TestServer_handleSessionByID_MethodNotAllowed tests handleSessionByID with wrong method.
func TestServer_handleSessionByID_MethodNotAllowed(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/test-id/messages", nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestSubscriptionServiceWrapper tests subscriptionServiceWrapper.
func TestSubscriptionServiceWrapper(t *testing.T) {
	subStore, err := subscription.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create subscription store: %v", err)
	}

	triggerFunc := func(ctx context.Context, sub *subscription.Subscription) error {
		return nil
	}

	// Create logger for scheduler (cannot be nil)
	log := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})

	scheduler := subscription.NewScheduler(subStore, triggerFunc, log)
	wrapper := &subscriptionServiceWrapper{scheduler: scheduler}

	// Test ListSubscriptions
	subs := wrapper.ListSubscriptions()
	if subs == nil {
		t.Error("ListSubscriptions() should return empty slice, not nil")
	}

	// Test CreateSubscription - use 6-field cron expression (seconds minutes hours day month weekday)
	sub, err := wrapper.CreateSubscription("session-1", "daily", "0 0 9 * * *", nil)
	if err != nil {
		t.Errorf("CreateSubscription() returned error: %v", err)
	}
	if sub == nil {
		t.Error("CreateSubscription() returned nil")
	}

	// Test TriggerSubscription
	if sub != nil {
		err = wrapper.TriggerSubscription(sub.ID)
		if err != nil {
			t.Errorf("TriggerSubscription() returned error: %v", err)
		}

		// Test DeleteSubscription
		err = wrapper.DeleteSubscription(sub.ID)
		if err != nil {
			t.Errorf("DeleteSubscription() returned error: %v", err)
		}
	}
}

// mockWebhookService implements handlers.WebhookService for testing.
type mockWebhookService struct {
	processEventFunc func(ctx context.Context, event trigger.Event) error
}

func (m *mockWebhookService) ProcessEvent(ctx context.Context, event trigger.Event) error {
	if m.processEventFunc != nil {
		return m.processEventFunc(ctx, event)
	}
	return nil
}

// mockSkillsService implements handlers.SkillsService for testing.
type mockSkillsService struct {
	listSkillsFunc func() (string, error)
}

func (m *mockSkillsService) ListSkills() (string, error) {
	if m.listSkillsFunc != nil {
		return m.listSkillsFunc()
	}
	return "", nil
}

// mockFlusher implements http.Flusher for testing.
type mockFlusher struct{}

func (m *mockFlusher) Flush() {}

// TestProcessSSEEvent_PlanEvents tests that plan and step events are correctly sent as SSE.
func TestProcessSSEEvent_PlanEvents(t *testing.T) {
	tests := []struct {
		name     string
		event    sdk.Event
		wantType string
	}{
		{
			name: "step event",
			event: events.NewEventWithExtra(
				sdk.EventTypeStep,
				"Phase: Exploration",
				map[string]any{"step": 1},
				"test-1",
			),
			wantType: string(sdk.EventTypeStep),
		},
		{
			name: "plan_created event",
			event: events.NewEventWithExtra(
				sdk.EventTypePlanCreated,
				"Plan: analyze and implement",
				map[string]any{
					"total_steps": 3,
					"goal":        "Analyze and implement feature",
					"steps":       []string{"Step 1", "Step 2", "Step 3"},
				},
				"test-1",
			),
			wantType: string(sdk.EventTypePlanCreated),
		},
		{
			name: "plan_step event",
			event: events.NewEventWithExtra(
				sdk.EventTypePlanStep,
				"Step 1/3: Analyze codebase",
				map[string]any{
					"step_num":    1,
					"total_steps": 3,
					"step_desc":   "Analyze codebase",
				},
				"test-1",
			),
			wantType: string(sdk.EventTypePlanStep),
		},
		{
			name: "plan_complete event",
			event: events.NewEventWithExtra(
				sdk.EventTypePlanComplete,
				"Plan execution completed",
				map[string]any{
					"plan": "Plan summary...",
				},
				"test-1",
			),
			wantType: string(sdk.EventTypePlanComplete),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, cleanup := setupTestServer(t)
			defer cleanup()

			w := httptest.NewRecorder()
			w.Body.Reset()
			flusher := &mockFlusher{}

			result := server.processSSEEvent(w, flusher, tt.event, &strings.Builder{}, make(chan struct{}))

			if !result {
				t.Error("processSSEEvent() returned false, want true")
			}

			body := w.Body.String()
			if !strings.Contains(body, "event: "+tt.wantType) {
				t.Errorf("Expected SSE event type %q in body, got:\n%s", tt.wantType, body)
			}
			if !strings.Contains(body, "data:") {
				t.Errorf("Expected SSE data field in body, got:\n%s", body)
			}
		})
	}
}
