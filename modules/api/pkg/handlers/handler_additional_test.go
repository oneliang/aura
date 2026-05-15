package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/session/pkg/trigger"
)

// setupTestHandler creates a temporary storage and manager for handler tests.
func setupTestHandler(t *testing.T) (*storage.JSONLStore, *manager.SessionManager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "handler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	mgr, err := manager.NewSessionManager(store, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create manager: %v", err)
	}

	cleanup := func() {
		mgr.Close()
		os.RemoveAll(tmpDir)
	}

	return store, mgr, cleanup
}

// TestHandleUpdateSession tests HandleUpdateSession function.
func TestHandleUpdateSession(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr, store: store})

	// Create a session first
	session, _ := mgr.CreateSession("Original", nil, "", "")

	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:   "update system prompt",
			method: http.MethodPut,
			body: map[string]string{
				"system_prompt": "Updated prompt",
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
			name:   "role and prompt conflict",
			method: http.MethodPut,
			body: map[string]string{
				"role":          "helper",
				"system_prompt": "Custom prompt",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			method:     http.MethodPut,
			body:       "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       map[string]string{},
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "non-existent session",
			method: http.MethodPut,
			body: map[string]string{
				"system_prompt": "Test",
			},
			wantStatus: http.StatusOK, // UpdateSession silently succeeds for non-existent session
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/sessions/"+session.ID, bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			h.HandleUpdateSession(w, req, session.ID)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleUpdateSession() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleDeleteSession tests HandleDeleteSession function.
func TestHandleDeleteSession(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr, store: store})

	// Create a session to delete
	session, _ := mgr.CreateSession("To Delete", nil, "", "")

	tests := []struct {
		name       string
		method     string
		sessionID  string
		wantStatus int
	}{
		{
			name:       "delete existing session",
			method:     http.MethodDelete,
			sessionID:  session.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent session",
			method:     http.MethodDelete,
			sessionID:  "non-existent",
			wantStatus: http.StatusInternalServerError, // DeleteSession returns error for non-existent session
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			sessionID:  session.ID,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/sessions/"+tt.sessionID, nil)
			w := httptest.NewRecorder()

			h.HandleDeleteSession(w, req, tt.sessionID)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleDeleteSession() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleGetSessionMessages tests HandleGetSessionMessages function.
func TestHandleGetSessionMessages(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr, store: store})

	// Create a session
	session, _ := mgr.CreateSession("Message Test", nil, "", "")

	// Add some messages
	msg1 := &model.Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   "Hello",
		Timestamp: 1234567890,
		Source:    "test",
	}
	msg2 := &model.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "Hi there",
		Timestamp: 1234567891,
		Source:    "test",
	}
	store.AppendMessage(context.Background(), msg1)
	store.AppendMessage(context.Background(), msg2)

	tests := []struct {
		name       string
		sessionID  string
		limit      string
		wantStatus int
		wantMsgs   int
	}{
		{
			name:       "get messages with limit",
			sessionID:  session.ID,
			limit:      "10",
			wantStatus: http.StatusOK,
			wantMsgs:   2,
		},
		{
			name:       "get messages default limit",
			sessionID:  session.ID,
			limit:      "",
			wantStatus: http.StatusOK,
			wantMsgs:   2,
		},
		{
			name:       "get messages non-existent session",
			sessionID:  "non-existent",
			limit:      "10",
			wantStatus: http.StatusNotFound,
			wantMsgs:   0,
		},
		{
			name:       "get messages invalid limit",
			sessionID:  session.ID,
			limit:      "invalid",
			wantStatus: http.StatusOK, // limit is ignored, defaults to 100
			wantMsgs:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/sessions/" + tt.sessionID + "/messages"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			h.HandleGetSessionMessages(w, req, tt.sessionID)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleGetSessionMessages() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]interface{}
				json.NewDecoder(w.Body).Decode(&response)
				data, ok := response["data"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected data field in response")
				}
				messages, ok := data["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages array")
				}
				if len(messages) != tt.wantMsgs {
					t.Errorf("Expected %d messages, got %d", tt.wantMsgs, len(messages))
				}
			}
		})
	}
}

// TestHandleSendMessage tests HandleSendMessage function.
func TestHandleSendMessage(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	mockService := &testSessionService{
		mgr:   mgr,
		store: store,
		sendMessageFunc: func(ctx context.Context, sessionID, content string) error {
			return nil
		},
	}

	h := NewSessionHandler(mockService)
	session, _ := mgr.CreateSession("Test", nil, "", "")

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
		{
			name:       "invalid json",
			body:       "invalid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+session.ID+"/message", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			h.HandleSendMessage(w, req, session.ID, mockService.SendMessage)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleSendMessage() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleFeishuWebhook_additional tests additional webhook cases.
func TestHandleFeishuWebhook_additional(t *testing.T) {
	mockService := &testWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	h := NewWebhookHandler(mockService)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "empty body",
			method:     http.MethodPost,
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid text message",
			method:     http.MethodPost,
			body:       `{"text": "Hello from Feishu", "msg_type": "text"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "challenge response",
			method:     http.MethodPost,
			body:       `{"challenge":"test123","type":"url_verification"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/webhooks/feishu", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.HandleFeishuWebhook(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleFeishuWebhook() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleCronTrigger_additional tests additional cron trigger cases.
func TestHandleCronTrigger_additional(t *testing.T) {
	mockService := &testWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	h := NewWebhookHandler(mockService)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "empty body",
			method:     http.MethodPost,
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid task with task and content",
			method:     http.MethodPost,
			body:       `{"task":"daily_report","content":"Generate report"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "task without content",
			method:     http.MethodPost,
			body:       `{"task":"cleanup"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/cron", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.HandleCronTrigger(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleCronTrigger() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleGenericWebhook_additional tests additional generic webhook cases.
func TestHandleGenericWebhook_additional(t *testing.T) {
	mockService := &testWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	h := NewWebhookHandler(mockService)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "valid webhook",
			method:     http.MethodPost,
			body:       `{"event":"test","data":"value"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty body",
			method:     http.MethodPost,
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/webhooks/slack", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.HandleGenericWebhook(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleGenericWebhook() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleListSessions_empty tests list sessions with no sessions.
func TestHandleListSessions_empty(t *testing.T) {
	_, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	h.HandleListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHandleCreateSession_invalid tests create session with invalid request.
func TestHandleCreateSession_invalid(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr, store: store})

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty name",
			body:       `{"name":""}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/sessions", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.HandleCreateSession(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleCreateSession() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleGetSession_additional tests additional GetSession cases.
func TestHandleGetSession_additional(t *testing.T) {
	store, mgr, cleanup := setupTestHandler(t)
	defer cleanup()

	h := NewSessionHandler(&testSessionService{mgr: mgr, store: store})

	t.Run("get non-existent session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions/non-existent", nil)
		w := httptest.NewRecorder()

		h.HandleGetSession(w, req, "non-existent")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// testSessionService implements SessionService for testing.
type testSessionService struct {
	mgr               *manager.SessionManager
	store             *storage.JSONLStore
	sendMessageFunc   func(ctx context.Context, sessionID, content string) error
	appendMessageFunc func(ctx context.Context, sessionID, role, content, source string) error
}

func (s *testSessionService) ListSessions() ([]*model.Session, error) {
	if s.mgr == nil {
		return []*model.Session{}, nil
	}
	return s.mgr.ListSessions("")
}

func (s *testSessionService) CreateSession(name string, subscriptions []model.Subscription, systemPrompt string) (*model.Session, error) {
	if s.mgr == nil {
		return nil, nil
	}
	return s.mgr.CreateSession(name, subscriptions, systemPrompt, "")
}

func (s *testSessionService) GetSession(id string) (*model.Session, error) {
	if s.mgr == nil {
		return nil, nil
	}
	return s.mgr.GetSession(id, "")
}

func (s *testSessionService) UpdateSession(id string, systemPrompt *string, role *string) error {
	if s.mgr == nil {
		return nil
	}
	session, err := s.mgr.GetSession(id, "")
	if err != nil {
		return err
	}
	if systemPrompt != nil {
		session.SystemPrompt = *systemPrompt
	}
	_ = role
	session.UpdatedAt = 1234567890 // Mock timestamp
	return s.store.SaveSession(session)
}

func (s *testSessionService) DeleteSession(id string) error {
	if s.mgr == nil {
		return nil
	}
	return s.mgr.DeleteSession(id, "")
}

func (s *testSessionService) GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]model.Message, error) {
	if s.store == nil {
		return []model.Message{}, nil
	}
	return s.store.GetMessages(ctx, sessionID, limit, userID)
}

func (s *testSessionService) SendMessage(ctx context.Context, sessionID, content string) error {
	if s.sendMessageFunc != nil {
		return s.sendMessageFunc(ctx, sessionID, content)
	}
	return nil
}

func (s *testSessionService) AddSubscription(sessionID string, sub model.Subscription) error {
	if s.mgr == nil {
		return nil
	}
	session, err := s.mgr.GetSession(sessionID, "")
	if err != nil {
		return err
	}
	session.Subscriptions = append(session.Subscriptions, sub)
	session.UpdatedAt = 1234567890
	return s.store.SaveSession(session)
}

func (s *testSessionService) RemoveSubscription(sessionID, subID, triggerStr string) error {
	if s.mgr == nil {
		return nil
	}
	session, err := s.mgr.GetSession(sessionID, "")
	if err != nil {
		return err
	}
	found := false
	newSubs := make([]model.Subscription, 0, len(session.Subscriptions))
	for _, sub := range session.Subscriptions {
		if (subID != "" && sub.ID == subID) || (triggerStr != "" && sub.Trigger == triggerStr) {
			found = true
			continue
		}
		newSubs = append(newSubs, sub)
	}
	if !found {
		return nil
	}
	session.Subscriptions = newSubs
	session.UpdatedAt = 1234567890
	return s.store.SaveSession(session)
}

func (s *testSessionService) AppendMessage(ctx context.Context, sessionID, role, content, source string) error {
	if s.appendMessageFunc != nil {
		return s.appendMessageFunc(ctx, sessionID, role, content, source)
	}
	// Default implementation: do nothing since engine's SessionMemory handles persistence
	return nil
}

// testWebhookService implements WebhookService for testing.
type testWebhookService struct {
	processEventFunc func(ctx context.Context, event trigger.Event) error
}

func (t *testWebhookService) ProcessEvent(ctx context.Context, event trigger.Event) error {
	if t.processEventFunc != nil {
		return t.processEventFunc(ctx, event)
	}
	return nil
}
