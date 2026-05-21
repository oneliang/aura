package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oneliang/aura/session/pkg/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSubscriptionService is a mock implementation of SubscriptionService.
type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) ListSubscriptions() []*subscription.Subscription {
	args := m.Called()
	return args.Get(0).([]*subscription.Subscription)
}

func (m *MockSubscriptionService) CreateSubscription(sessionID, eventType, cronExpr string, config map[string]interface{}) (*subscription.Subscription, error) {
	args := m.Called(sessionID, eventType, cronExpr, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) TriggerSubscription(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSubscriptionService) DeleteSubscription(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestSubscriptionHandler_HandleListSubscriptions(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		subscriptions []*subscription.Subscription
		wantStatus    int
	}{
		{
			name:   "list subscriptions",
			method: http.MethodGet,
			subscriptions: []*subscription.Subscription{
				subscription.NewSubscription("", "session-1", "daily_report", "0 9 * * *", nil),
				subscription.NewSubscription("", "session-2", "weekly_report", "0 9 * * 1", nil),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "empty list",
			method:        http.MethodGet,
			subscriptions: []*subscription.Subscription{},
			wantStatus:    http.StatusOK,
		},
		{
			name:          "wrong method",
			method:        http.MethodPost,
			subscriptions: []*subscription.Subscription{},
			wantStatus:    http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSubscriptionService)

			// Only set up mock expectations for valid GET requests
			if tt.method == http.MethodGet {
				mockService.On("ListSubscriptions").Return(tt.subscriptions)
			}

			handler := NewSubscriptionHandler(mockService)

			req := httptest.NewRequest(tt.method, "/api/subscriptions", nil)
			w := httptest.NewRecorder()

			handler.HandleListSubscriptions(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				_ = json.NewDecoder(w.Body).Decode(&resp)
				// Response format: {status: "ok", data: {subscriptions: [...], count: N}}
				data, ok := resp["data"].(map[string]interface{})
				if ok {
					count, _ := data["count"].(float64)
					assert.Equal(t, len(tt.subscriptions), int(count))
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionHandler_HandleCreateSubscription(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  interface{}
		method       string
		serviceError error
		wantStatus   int
		wantContain  string
	}{
		{
			name:   "valid subscription",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"event_type": "daily_report",
				"cron_expr":  "0 9 * * *",
				"config":     map[string]interface{}{"text": "Daily report"},
			},
			serviceError: nil,
			wantStatus:   http.StatusCreated,
			wantContain:  "created",
		},
		{
			name:   "missing session_id",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"event_type": "daily_report",
				"cron_expr":  "0 9 * * *",
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "session_id is required",
		},
		{
			name:   "missing event_type",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"cron_expr":  "0 9 * * *",
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "event_type is required",
		},
		{
			name:   "missing cron_expr",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"event_type": "daily_report",
			},
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "cron_expr is required",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"event_type": "daily_report",
				"cron_expr":  "0 9 * * *",
			},
			serviceError: assert.AnError,
			wantStatus:   http.StatusInternalServerError,
			wantContain:  assert.AnError.Error(),
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			requestBody: map[string]interface{}{
				"session_id": "test-session",
				"event_type": "daily_report",
				"cron_expr":  "0 9 * * *",
			},
			serviceError: nil,
			wantStatus:   http.StatusMethodNotAllowed,
			wantContain:  "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSubscriptionService)

			// Only set up mock expectations for valid requests that would call the service
			if tt.serviceError == nil && tt.method == http.MethodPost {
				if req, ok := tt.requestBody.(map[string]interface{}); ok {
					sessionID, hasSession := req["session_id"].(string)
					eventType, hasEventType := req["event_type"].(string)
					cronExpr, hasCron := req["cron_expr"].(string)
					config, _ := req["config"].(map[string]interface{})
					if hasSession && hasEventType && hasCron {
						sub := subscription.NewSubscription("", sessionID, eventType, cronExpr, config)
						mockService.On("CreateSubscription", sessionID, eventType, cronExpr, config).Return(sub, nil)
					}
				}
			} else if tt.serviceError != nil {
				mockService.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tt.serviceError)
			}

			handler := NewSubscriptionHandler(mockService)

			var body *bytes.Buffer
			if tt.requestBody != nil && tt.method == http.MethodPost {
				jsonBody, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(jsonBody)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/api/subscriptions", body)
			w := httptest.NewRecorder()

			handler.HandleCreateSubscription(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantContain)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionHandler_HandleTriggerSubscription(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		method       string
		serviceError error
		wantStatus   int
		wantContain  string
	}{
		{
			name:         "valid trigger",
			queryParam:   "id=sub-123",
			method:       http.MethodPost,
			serviceError: nil,
			wantStatus:   http.StatusOK,
			wantContain:  "triggered",
		},
		{
			name:         "missing id",
			queryParam:   "",
			method:       http.MethodPost,
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "subscription id is required",
		},
		{
			name:         "service error",
			queryParam:   "id=sub-123",
			method:       http.MethodPost,
			serviceError: assert.AnError,
			wantStatus:   http.StatusInternalServerError,
			wantContain:  assert.AnError.Error(),
		},
		{
			name:         "wrong method",
			queryParam:   "id=sub-123",
			method:       http.MethodGet,
			serviceError: nil,
			wantStatus:   http.StatusMethodNotAllowed,
			wantContain:  "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSubscriptionService)

			if tt.serviceError == nil && tt.method == http.MethodPost && tt.queryParam != "" {
				mockService.On("TriggerSubscription", "sub-123").Return(nil)
			} else if tt.serviceError != nil {
				mockService.On("TriggerSubscription", "sub-123").Return(tt.serviceError)
			}

			handler := NewSubscriptionHandler(mockService)

			url := "/api/subscriptions/trigger"
			if tt.queryParam != "" {
				url += "?" + tt.queryParam
			}
			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			handler.HandleTriggerSubscription(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantContain)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionHandler_HandleDeleteSubscription(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		method       string
		serviceError error
		wantStatus   int
		wantContain  string
	}{
		{
			name:         "valid delete",
			queryParam:   "id=sub-123",
			method:       http.MethodDelete,
			serviceError: nil,
			wantStatus:   http.StatusOK,
			wantContain:  "removed",
		},
		{
			name:         "missing id",
			queryParam:   "",
			method:       http.MethodDelete,
			serviceError: nil,
			wantStatus:   http.StatusBadRequest,
			wantContain:  "subscription id is required",
		},
		{
			name:         "service error",
			queryParam:   "id=sub-123",
			method:       http.MethodDelete,
			serviceError: assert.AnError,
			wantStatus:   http.StatusInternalServerError,
			wantContain:  assert.AnError.Error(),
		},
		{
			name:         "wrong method",
			queryParam:   "id=sub-123",
			method:       http.MethodGet,
			serviceError: nil,
			wantStatus:   http.StatusMethodNotAllowed,
			wantContain:  "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSubscriptionService)

			if tt.serviceError == nil && tt.method == http.MethodDelete && tt.queryParam != "" {
				mockService.On("DeleteSubscription", "sub-123").Return(nil)
			} else if tt.serviceError != nil {
				mockService.On("DeleteSubscription", "sub-123").Return(tt.serviceError)
			}

			handler := NewSubscriptionHandler(mockService)

			url := "/api/subscriptions/delete"
			if tt.queryParam != "" {
				url += "?" + tt.queryParam
			}
			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			handler.HandleDeleteSubscription(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantContain)
			mockService.AssertExpectations(t)
		})
	}
}
