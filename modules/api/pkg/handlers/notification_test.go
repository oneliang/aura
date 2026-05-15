package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationService is a mock implementation of NotificationService.
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error {
	args := m.Called(ctx, sessionID, msgType, content)
	return args.Error(0)
}

func (m *MockNotificationService) SendTaskNotification(ctx context.Context, taskID, status, result string) error {
	args := m.Called(ctx, taskID, status, result)
	return args.Error(0)
}

func TestNotificationHandler_HandleSendNotification(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  interface{}
		method       string
		serviceError error
		wantStatus   int
		wantContain  string
	}{
		{
			name:   "valid notification",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"msg_type":   "text",
				"content":    map[string]interface{}{"text": "Hello"},
			},
			serviceError: nil,
			wantStatus:   http.StatusOK,
			wantContain:  "Notification sent",
		},
		{
			name:   "missing session_id",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"msg_type": "text",
				"content":  map[string]interface{}{"text": "Hello"},
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "session_id is required",
		},
		{
			name:   "missing content",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"msg_type":   "text",
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "content is required",
		},
		{
			name:         "invalid json",
			method:       http.MethodPost,
			requestBody:  "invalid",
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "Invalid request body",
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"msg_type":   "text",
				"content":    map[string]interface{}{"text": "Hello"},
			},
			serviceError: nil,
			wantStatus:   http.StatusMethodNotAllowed,
			wantContain:  "Method not allowed",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"msg_type":   "text",
				"content":    map[string]interface{}{"text": "Hello"},
			},
			serviceError: assert.AnError,
			wantStatus:   http.StatusInternalServerError,
			wantContain:  assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockNotificationService)

			// Only set up mock expectations for valid requests that would call the service
			if tt.serviceError == nil && tt.method == http.MethodPost {
				if req, ok := tt.requestBody.(map[string]interface{}); ok {
					if sessionID, ok := req["session_id"].(string); ok {
						if _, hasContent := req["content"]; hasContent {
							mockService.On("SendNotification", mock.Anything, sessionID, mock.Anything, mock.Anything).Return(nil)
						}
					}
				}
			} else if tt.serviceError != nil {
				mockService.On("SendNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.serviceError)
			}

			handler := NewNotificationHandler(mockService)

			var body *bytes.Buffer
			if tt.requestBody != nil && tt.method == http.MethodPost {
				jsonBody, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(jsonBody)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/api/notifications", body)
			w := httptest.NewRecorder()

			handler.HandleSendNotification(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantContain)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNotificationHandler_HandleTaskNotification(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  interface{}
		method       string
		serviceError error
		wantStatus   int
		wantContain  string
	}{
		{
			name:   "valid task notification",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"task_id": "task-123",
				"status":  "completed",
				"result":  "Success",
			},
			serviceError: nil,
			wantStatus:   http.StatusOK,
			wantContain:  "Task notification sent",
		},
		{
			name:   "missing task_id",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"status": "completed",
				"result": "Success",
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "task_id is required",
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			requestBody: map[string]interface{}{
				"task_id": "task-123",
				"status":  "completed",
				"result":  "Success",
			},
			serviceError: nil,
			wantStatus:   http.StatusMethodNotAllowed,
			wantContain:  "Method not allowed",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"task_id": "task-123",
				"status":  "completed",
				"result":  "Success",
			},
			serviceError: assert.AnError,
			wantStatus:   http.StatusInternalServerError,
			wantContain:  assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockNotificationService)

			if tt.serviceError == nil && tt.method == http.MethodPost {
				if req, ok := tt.requestBody.(map[string]interface{}); ok {
					if taskID, ok := req["task_id"].(string); ok {
						mockService.On("SendTaskNotification", mock.Anything, taskID, mock.Anything, mock.Anything).Return(nil)
					}
				}
			} else if tt.serviceError != nil {
				mockService.On("SendTaskNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.serviceError)
			}

			handler := NewNotificationHandler(mockService)

			var body *bytes.Buffer
			if tt.requestBody != nil && tt.method == http.MethodPost {
				jsonBody, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(jsonBody)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/api/notifications/task", body)
			w := httptest.NewRecorder()

			handler.HandleTaskNotification(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantContain)
			mockService.AssertExpectations(t)
		})
	}
}
