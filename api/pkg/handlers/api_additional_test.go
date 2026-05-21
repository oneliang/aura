// Package handlers provides additional tests for the handlers package.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/trigger"
)

// MockSessionService implements SessionService interface for testing.
type MockSessionService struct {
	listSessionsFunc  func(userID string) ([]*model.Session, error)
	createSessionFunc func(userID string, name string, subs []model.Subscription, prompt string) (*model.Session, error)
	getSessionFunc    func(id string, userID string) (*model.Session, error)
	updateSessionFunc func(id string, userID string, prompt *string, role *string) error
	deleteSessionFunc func(id string, userID string) error
	getMessagesFunc   func(ctx context.Context, sessionID string, limit int, userID string) ([]model.Message, error)
	sendMessageFunc   func(ctx context.Context, sessionID, content string) error
	addSubFunc        func(sessionID string, sub model.Subscription) error
	removeSubFunc     func(sessionID, subID, trigger string) error
	appendMessageFunc func(ctx context.Context, sessionID, role, content, source string) error
}

func (m *MockSessionService) ListSessions(userID string) ([]*model.Session, error) {
	if m.listSessionsFunc != nil {
		return m.listSessionsFunc(userID)
	}
	return []*model.Session{}, nil
}

func (m *MockSessionService) CreateSession(userID string, name string, subs []model.Subscription, prompt string) (*model.Session, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(userID, name, subs, prompt)
	}
	return &model.Session{ID: "test-id", Name: name}, nil
}

func (m *MockSessionService) GetSession(id string, userID string) (*model.Session, error) {
	if m.getSessionFunc != nil {
		return m.getSessionFunc(id, userID)
	}
	if id == "not-found" {
		return nil, http.ErrMissingFile
	}
	return &model.Session{ID: id, Name: "test"}, nil
}

func (m *MockSessionService) UpdateSession(id string, userID string, prompt *string, role *string) error {
	if m.updateSessionFunc != nil {
		return m.updateSessionFunc(id, userID, prompt, role)
	}
	return nil
}

func (m *MockSessionService) DeleteSession(id string, userID string) error {
	if m.deleteSessionFunc != nil {
		return m.deleteSessionFunc(id, userID)
	}
	return nil
}

func (m *MockSessionService) GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]model.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, sessionID, limit, userID)
	}
	return []model.Message{}, nil
}

func (m *MockSessionService) SendMessage(ctx context.Context, sessionID, content string) error {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, sessionID, content)
	}
	return nil
}

func (m *MockSessionService) AddSubscription(sessionID string, sub model.Subscription) error {
	if m.addSubFunc != nil {
		return m.addSubFunc(sessionID, sub)
	}
	return nil
}

func (m *MockSessionService) RemoveSubscription(sessionID, subID, trigger string) error {
	if m.removeSubFunc != nil {
		return m.removeSubFunc(sessionID, subID, trigger)
	}
	return nil
}

func (m *MockSessionService) AppendMessage(ctx context.Context, sessionID, role, content, source string) error {
	if m.appendMessageFunc != nil {
		return m.appendMessageFunc(ctx, sessionID, role, content, source)
	}
	return nil
}

// TestSessionHandler_HandleListSessions tests HandleListSessions method.
func TestSessionHandler_HandleListSessions(t *testing.T) {
	service := &MockSessionService{
		listSessionsFunc: func(userID string) ([]*model.Session, error) {
			return []*model.Session{{ID: "1", Name: "Test Session"}}, nil
		},
	}

	handler := NewSessionHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	handler.HandleListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Test Session") {
		t.Error("Response should contain session name")
	}
}

// TestSessionHandler_HandleListSessions_MethodNotAllowed tests HandleListSessions with wrong method.
func TestSessionHandler_HandleListSessions_MethodNotAllowed(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()

	handler.HandleListSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestSessionHandler_HandleListSessions_Error tests HandleListSessions with error.
func TestSessionHandler_HandleListSessions_Error(t *testing.T) {
	service := &MockSessionService{
		listSessionsFunc: func(userID string) ([]*model.Session, error) {
			return nil, http.ErrMissingFile
		},
	}

	handler := NewSessionHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	handler.HandleListSessions(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestSessionHandler_HandleCreateSession tests HandleCreateSession method.
func TestSessionHandler_HandleCreateSession(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{"name": "New Session", "system_prompt": "test prompt"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", body)
	w := httptest.NewRecorder()

	handler.HandleCreateSession(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "New Session") {
		t.Error("Response should contain session name")
	}
}

// TestSessionHandler_HandleCreateSession_InvalidJSON tests HandleCreateSession with invalid JSON.
func TestSessionHandler_HandleCreateSession_InvalidJSON(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", body)
	w := httptest.NewRecorder()

	handler.HandleCreateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestSessionHandler_HandleCreateSession_MissingName tests HandleCreateSession without name.
func TestSessionHandler_HandleCreateSession_MissingName(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", body)
	w := httptest.NewRecorder()

	handler.HandleCreateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestSessionHandler_HandleGetSession tests HandleGetSession method.
func TestSessionHandler_HandleGetSession(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test-id", nil)
	w := httptest.NewRecorder()

	handler.HandleGetSession(w, req, "test-id")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestSessionHandler_HandleGetSession_NotFound tests HandleGetSession with non-existent session.
func TestSessionHandler_HandleGetSession_NotFound(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-found", nil)
	w := httptest.NewRecorder()

	handler.HandleGetSession(w, req, "not-found")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestSessionHandler_HandleDeleteSession tests HandleDeleteSession method.
func TestSessionHandler_HandleDeleteSession(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/test-id", nil)
	w := httptest.NewRecorder()

	handler.HandleDeleteSession(w, req, "test-id")

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

// TestSessionHandler_HandleGetSessionMessages tests HandleGetSessionMessages method.
func TestSessionHandler_HandleGetSessionMessages(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test-id/messages", nil)
	w := httptest.NewRecorder()

	handler.HandleGetSessionMessages(w, req, "test-id")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSendMessage tests HandleSendMessage method.
func TestSessionHandler_HandleSendMessage(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{"content": "Hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/test-id/message", body)
	w := httptest.NewRecorder()

	handler.HandleSendMessage(w, req, "test-id", func(ctx context.Context, sid, content string) error {
		return nil
	})

	// Should return accepted immediately
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSendMessage_EmptyContent tests HandleSendMessage without content.
func TestSessionHandler_HandleSendMessage_EmptyContent(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/test-id/message", body)
	w := httptest.NewRecorder()

	handler.HandleSendMessage(w, req, "test-id", func(ctx context.Context, sid, content string) error {
		return nil
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSessionSubscriptions_Add tests adding subscription.
func TestSessionHandler_HandleSessionSubscriptions_Add(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{"trigger": "keyword", "source": "feishu"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/test-id/subscriptions", body)
	w := httptest.NewRecorder()

	handler.HandleSessionSubscriptions(w, req, "test-id")

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSessionSubscriptions_Add_EmptyTrigger tests adding subscription without trigger.
func TestSessionHandler_HandleSessionSubscriptions_Add_EmptyTrigger(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{"source": "feishu"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/test-id/subscriptions", body)
	w := httptest.NewRecorder()

	handler.HandleSessionSubscriptions(w, req, "test-id")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSessionSubscriptions_Remove tests removing subscription.
func TestSessionHandler_HandleSessionSubscriptions_Remove(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{"id": "sub-1"}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/test-id/subscriptions", body)
	w := httptest.NewRecorder()

	handler.HandleSessionSubscriptions(w, req, "test-id")

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

// TestSessionHandler_HandleSessionSubscriptions_Remove_Empty tests removing subscription without id/trigger.
func TestSessionHandler_HandleSessionSubscriptions_Remove_Empty(t *testing.T) {
	service := &MockSessionService{}
	handler := NewSessionHandler(service)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/test-id/subscriptions", body)
	w := httptest.NewRecorder()

	handler.HandleSessionSubscriptions(w, req, "test-id")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestSkillsServiceWrapper tests skills service wrapper.
func TestSkillsServiceWrapper(t *testing.T) {
	// Test with nil command provider
	wrapper := NewSkillsServiceWrapper(nil)
	result, err := wrapper.ListSkills()

	if err != nil {
		t.Errorf("ListSkills() returned error: %v", err)
	}
	if result != "Skills not available" {
		t.Errorf("Expected 'Skills not available', got %q", result)
	}
}

// TestSkillsHandler tests skills handler.
func TestSkillsHandler(t *testing.T) {
	service := &MockSkillsService{
		listSkillsFunc: func() (string, error) {
			return "skill1, skill2", nil
		},
	}

	handler := NewSkillsHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	w := httptest.NewRecorder()

	handler.HandleListSkills(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "skill1") {
		t.Error("Response should contain skill1")
	}
}

// TestSkillsHandler_MethodNotAllowed tests skills handler with wrong method.
func TestSkillsHandler_MethodNotAllowed(t *testing.T) {
	service := &MockSkillsService{}
	handler := NewSkillsHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/skills", nil)
	w := httptest.NewRecorder()

	handler.HandleListSkills(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestSkillsHandler_Error tests skills handler with error.
func TestSkillsHandler_Error(t *testing.T) {
	service := &MockSkillsService{
		listSkillsFunc: func() (string, error) {
			return "", http.ErrMissingFile
		},
	}

	handler := NewSkillsHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	w := httptest.NewRecorder()

	handler.HandleListSkills(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// MockSkillsService implements SkillsService interface for testing.
type MockSkillsService struct {
	listSkillsFunc func() (string, error)
}

func (m *MockSkillsService) ListSkills() (string, error) {
	if m.listSkillsFunc != nil {
		return m.listSkillsFunc()
	}
	return "", nil
}

// MockWebhookService implements WebhookService interface for testing.
type MockWebhookService struct {
	processEventFunc func(ctx context.Context, event trigger.Event) error
}

func (m *MockWebhookService) ProcessEvent(ctx context.Context, event trigger.Event) error {
	if m.processEventFunc != nil {
		return m.processEventFunc(ctx, event)
	}
	return nil
}

// TestWebhookHandler_HandleFeishuWebhook tests HandleFeishuWebhook method.
func TestWebhookHandler_HandleFeishuWebhook(t *testing.T) {
	service := &MockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	handler := NewWebhookHandler(service)

	// Valid Feishu webhook payload
	body := strings.NewReader(`{"msg_type":"text","content":{"text":"Hello"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu", body)
	w := httptest.NewRecorder()

	handler.HandleFeishuWebhook(w, req)

	// Should return success or bad request depending on payload validation
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleFeishuWebhook_MethodNotAllowed tests with wrong method.
func TestWebhookHandler_HandleFeishuWebhook_MethodNotAllowed(t *testing.T) {
	service := &MockWebhookService{}
	handler := NewWebhookHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/webhooks/feishu", nil)
	w := httptest.NewRecorder()

	handler.HandleFeishuWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleCronTrigger tests HandleCronTrigger method.
func TestWebhookHandler_HandleCronTrigger(t *testing.T) {
	service := &MockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	handler := NewWebhookHandler(service)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/cron", body)
	w := httptest.NewRecorder()

	handler.HandleCronTrigger(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleGenericWebhook tests HandleGenericWebhook method.
func TestWebhookHandler_HandleGenericWebhook(t *testing.T) {
	service := &MockWebhookService{
		processEventFunc: func(ctx context.Context, event trigger.Event) error {
			return nil
		},
	}

	handler := NewWebhookHandler(service)

	body := strings.NewReader(`{"event": "test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/slack", body)
	w := httptest.NewRecorder()

	handler.HandleGenericWebhook(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200 or 400, got %d", w.Code)
	}
}

// TestWebhookHandler_HandleGenericWebhook_EmptySource tests with empty source.
func TestWebhookHandler_HandleGenericWebhook_EmptySource(t *testing.T) {
	service := &MockWebhookService{}
	handler := NewWebhookHandler(service)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/", body)
	w := httptest.NewRecorder()

	handler.HandleGenericWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestWriteOK_WithData tests WriteOK with only data.
func TestWriteOK_WithData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteOK(w, map[string]string{"key": "value"}, "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestWriteOK_WithMessage tests WriteOK with only message.
func TestWriteOK_WithMessage(t *testing.T) {
	w := httptest.NewRecorder()

	WriteOK(w, nil, "success")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
