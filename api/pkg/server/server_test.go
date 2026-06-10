package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/oneliang/aura/api/pkg/handlers"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/session/pkg/subscription"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/user"
)

// setupTestServer creates a real server with temporary storage for integration testing.
func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "api-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionManager, err := manager.NewSessionManager(store, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a mock LLM client (nil is okay for tests that don't process messages)
	var llmClient llm.Client = nil

	// Create logger
	log := logger.New(logger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})

	// Create user manager for authentication
	usersCfg := &config.UsersConfig{
		Default: "test-user",
		Definitions: []config.UserConfig{
			{
				ID:       "test-user",
				APIToken: "test-token-123",
			},
		},
	}
	userManager, err := user.NewManagerWithBaseDir(usersCfg, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create user manager: %v", err)
	}

	// Create a minimal config to prevent nil pointer dereference in getOrCreateRuntime
	testConfig := &config.Config{
		Intent: config.IntentConfig{
			Enabled:             false,
			ConfidenceThreshold: 0.7,
		},
	}

	server := &Server{
		manager:       sessionManager,
		store:         store,
		port:          "8080",
		config:        testConfig,
		llmClient:     llmClient,
		logger:        log,
		userManager:   userManager,
		runtimes:      make(map[string]*sessionRuntime),
		orchestrators: make(map[string]*SessionOrchestrator),
		sseProcessing: make(map[string]bool),
	}

	// Initialize handlers
	server.sessionHandler = handlers.NewSessionHandler(server)
	server.webhookHandler = handlers.NewWebhookHandler(server)

	// Create subscription store and scheduler for subscription handler
	subStore, err := subscription.NewStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create subscription store: %v", err)
	}

	triggerFunc := func(ctx context.Context, sub *subscription.Subscription) error {
		return nil // No-op for tests
	}

	server.subscriptionStore = subStore
	server.subscriptionScheduler = subscription.NewScheduler(subStore, triggerFunc, log)

	// Create subscription service wrapper for subscription handler
	subWrapper := &subscriptionServiceWrapper{scheduler: server.subscriptionScheduler}
	server.subscriptionHandler = handlers.NewSubscriptionHandler(subWrapper)

	cleanup := func() {
		sessionManager.Close()
		os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func TestHandleHealth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestHandleSessions_List(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
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

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response format: {status: "ok", data: {sessions: [...]}}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response")
	}

	sessions, ok := data["sessions"].([]interface{})
	if !ok {
		t.Fatalf("Expected sessions array in data")
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestHandleSessions_Create(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name: "valid session",
			body: map[string]interface{}{
				"name": "New Session",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "session with subscriptions",
			body: map[string]interface{}{
				"name": "Subscribed Session",
				"subscriptions": []map[string]interface{}{
					{"id": "sub1", "source": "feishu", "trigger": "告警", "active": true},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing name",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleSessions(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandleSessionByID_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+session.ID, nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response format: {status: "ok", data: {session: {...}}}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response")
	}

	savedSession, ok := data["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected session field in data")
	}
	if savedSession["name"] != "Test Session" {
		t.Errorf("Expected name 'Test Session', got %v", savedSession["name"])
	}
}

func TestHandleSessionByID_GetNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/non-existent", nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleSessionByID_Delete(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("To Delete", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID, nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify session is deleted
	_, err = server.manager.GetSession(session.ID, "")
	if err == nil {
		t.Error("Expected session to be deleted")
	}
}

func TestHandleSessionMessages_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("Message Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+session.ID+"/messages", nil)
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response format: {status: "ok", data: {messages: [...]}}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response")
	}

	messages, ok := data["messages"].([]interface{})
	if !ok {
		t.Fatalf("Expected messages array in data")
	}
	// Should be empty since no messages added
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

func TestSendMessage(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("Message Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name: "valid message",
			body: map[string]interface{}{
				"content": "Hello",
				"source":  "api",
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "missing content",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty content",
			body: map[string]interface{}{
				"content": "",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/message", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleSessionByID(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandleFeishuWebhook(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "valid feishu message",
			body:       []byte(`{"text": "Hello from Feishu", "msg_type": "text"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       []byte(`{invalid}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong method",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.name == "wrong method" {
				method = http.MethodGet
			} else {
				method = http.MethodPost
			}

			req := httptest.NewRequest(method, "/api/webhooks/feishu", bytes.NewReader(tt.body))
			w := httptest.NewRecorder()

			server.handleWebhooks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandleCronTrigger(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "valid cron task",
			body:       []byte(`{"task": "daily_report", "content": "Generate report"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "task without content",
			body:       []byte(`{"task": "cleanup"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       []byte(`{invalid}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong method",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			if tt.name == "wrong method" {
				method = http.MethodGet
			} else {
				method = http.MethodPost
			}

			req := httptest.NewRequest(method, "/api/cron", bytes.NewReader(tt.body))
			w := httptest.NewRecorder()

			server.handleCronTrigger(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandleGenericWebhook(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "valid slack webhook",
			path:       "/api/webhooks/slack",
			body:       []byte(`{"content": "Hello from Slack"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid email webhook",
			path:       "/api/webhooks/email",
			body:       []byte(`{"content": "Email content"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing source",
			path:       "/api/webhooks/",
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			path:       "/api/webhooks/test",
			body:       []byte(`{invalid}`),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body))
			w := httptest.NewRecorder()

			server.handleWebhooks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestUpdateSession(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("Original Name", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:   "update name",
			method: http.MethodPut,
			body: map[string]string{
				"name": "Updated Name",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "update role",
			method: http.MethodPatch,
			body: map[string]string{
				"role": "helper",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "update system prompt",
			method: http.MethodPut,
			body: map[string]string{
				"system_prompt": "You are a helpful assistant",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "role and prompt conflict",
			method: http.MethodPut,
			body: map[string]string{
				"role":          "helper",
				"system_prompt": "Custom prompt",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(tt.method, "/api/sessions/"+session.ID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleSessionByID(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandleSessionSubscriptions(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	session, err := server.manager.CreateSession("Test Session", nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Run("add subscription", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"trigger": "alert",
			"source":  "feishu",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Response format: {status: "created", data: {subscription: {...}}}
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response")
		}

		sub, ok := data["subscription"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected subscription field in data")
		}
		if sub["trigger"] != "alert" {
			t.Errorf("Expected trigger 'alert', got %v", sub["trigger"])
		}
	})

	t.Run("add subscription with default source", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"trigger": "keyword",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("add subscription missing trigger", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"source": "feishu",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("delete subscription by trigger", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"trigger": "alert",
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("delete subscription by id", func(t *testing.T) {
		// First add a subscription
		body, _ := json.Marshal(map[string]interface{}{
			"trigger": "to-delete",
			"source":  "test",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.handleSessionByID(w, req)

		// Get the subscription ID
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Response format: {status: "created", data: {subscription: {...}}}
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response")
		}

		sub, ok := data["subscription"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected subscription field in data")
		}
		subID := sub["id"].(string)

		// Delete by ID
		body, _ = json.Marshal(map[string]interface{}{
			"id": subID,
		})

		req = httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("delete subscription not found", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"trigger": "non-existent",
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("delete missing parameters", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{})

		req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+session.ID+"/subscriptions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSessionByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleSessionSubscriptions_SessionNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"trigger": "alert",
		"source":  "feishu",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/non-existent/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSessionByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
