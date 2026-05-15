package feishu

import (
	"context"
	"errors"
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/model"
)

// TestAPIError tests APIError type.
func TestAPIError(t *testing.T) {
	err := &APIError{
		Code:    10001,
		Message: "test error",
		Op:      "TestOp",
	}

	expectedMsg := "feishu API error: TestOp failed: code=10001, msg=test error"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestAPIError_Is tests APIError.Is method.
func TestAPIError_Is(t *testing.T) {
	err1 := &APIError{Code: 10001, Message: "error1"}
	err2 := &APIError{Code: 10001, Message: "error2"}
	err3 := &APIError{Code: 10002, Message: "error3"}

	// Same code should match
	if !err1.Is(err2) {
		t.Error("APIError.Is() should return true for same code")
	}

	// Different code should not match
	if err1.Is(err3) {
		t.Error("APIError.Is() should return false for different code")
	}

	// Different type should not match
	if err1.Is(errors.New("different error")) {
		t.Error("APIError.Is() should return false for different error type")
	}
}

// TestNewAPIError tests newAPIError function.
func TestNewAPIError(t *testing.T) {
	err := newAPIError("TestOp", 10001, "test message")

	if err == nil {
		t.Fatal("newAPIError() returned nil")
	}
	if err.Op != "TestOp" {
		t.Errorf("Op = %q, want %q", err.Op, "TestOp")
	}
	if err.Code != 10001 {
		t.Errorf("Code = %d, want %d", err.Code, 10001)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
}

// TestGetSenderID tests getSenderID helper.
// Note: The implementation doesn't handle nil events, so we test with valid events only.
func TestGetSenderID(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	// Test with empty event (event exists but has no data)
	event := &larkim.P2MessageReceiveV1{}
	id := adapter.getSenderID(event)
	if id != "" {
		t.Errorf("getSenderID(empty event) = %q, want empty string", id)
	}

	// Test with valid sender ID
	openID := "ou_test123"
	event = &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					OpenId: &openID,
				},
			},
		},
	}
	id = adapter.getSenderID(event)
	if id != openID {
		t.Errorf("getSenderID() = %q, want %q", id, openID)
	}
}

// TestGetChatID tests getChatID helper.
func TestGetChatID(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	// Test with empty event
	event := &larkim.P2MessageReceiveV1{}
	id := adapter.getChatID(event)
	if id != "" {
		t.Errorf("getChatID(empty event) = %q, want empty string", id)
	}

	// Test with valid chat ID
	chatID := "oc_test123"
	event = &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Message: &larkim.EventMessage{
				ChatId: &chatID,
			},
		},
	}
	id = adapter.getChatID(event)
	if id != chatID {
		t.Errorf("getChatID() = %q, want %q", id, chatID)
	}
}

// TestGetMessageID tests getMessageID helper.
func TestGetMessageID(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	// Test with empty event
	event := &larkim.P2MessageReceiveV1{}
	id := adapter.getMessageID(event)
	if id != "" {
		t.Errorf("getMessageID(empty event) = %q, want empty string", id)
	}

	// Test with valid message ID
	messageID := "om_test123"
	event = &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Message: &larkim.EventMessage{
				MessageId: &messageID,
			},
		},
	}
	id = adapter.getMessageID(event)
	if id != messageID {
		t.Errorf("getMessageID() = %q, want %q", id, messageID)
	}
}

// TestGetMessageText tests getMessageText helper.
func TestGetMessageText(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	tests := []struct {
		name     string
		event    *larkim.P2MessageReceiveV1
		expected string
	}{
		{
			name:     "empty event",
			event:    &larkim.P2MessageReceiveV1{},
			expected: "",
		},
		{
			name: "text message",
			event: &larkim.P2MessageReceiveV1{
				Event: &larkim.P2MessageReceiveV1Data{
					Message: &larkim.EventMessage{
						Content: strPtr(`{"text":"hello world"}`),
					},
				},
			},
			expected: "hello world",
		},
		{
			name: "non-text message",
			event: &larkim.P2MessageReceiveV1{
				Event: &larkim.P2MessageReceiveV1Data{
					Message: &larkim.EventMessage{
						Content: strPtr(`{"image":"image_key"}`),
					},
				},
			},
			expected: `{"image":"image_key"}`,
		},
		{
			name: "invalid JSON",
			event: &larkim.P2MessageReceiveV1{
				Event: &larkim.P2MessageReceiveV1Data{
					Message: &larkim.EventMessage{
						Content: strPtr(`invalid json`),
					},
				},
			},
			expected: "invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.getMessageText(tt.event)
			if result != tt.expected {
				t.Errorf("getMessageText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestAdapter_GetUserStore tests GetUserStore method.
func TestAdapter_GetUserStore(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	// Before initialization, userStore is nil
	store := adapter.GetUserStore()
	if store != nil {
		t.Error("GetUserStore() should return nil before initialization")
	}
}

// TestUserInfo tests UserInfo struct.
func TestUserInfo(t *testing.T) {
	userInfo := &UserInfo{
		SessionID: "session-123",
		OpenID:    "ou_test",
		UserID:    "user_test",
		UnionID:   "on_test",
		IsGroup:   false,
		ChatID:    "oc_test",
	}

	if userInfo.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", userInfo.SessionID, "session-123")
	}
	if userInfo.OpenID != "ou_test" {
		t.Errorf("OpenID = %q, want %q", userInfo.OpenID, "ou_test")
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// TestMockResourceManager_WithCustomFunctions tests custom function callbacks.
func TestMockResourceManager_WithCustomFunctions(t *testing.T) {
	mock := &MockResourceManager{}
	ctx := context.Background()

	// Test with custom getRuntimeFunc
	mock.getRuntimeFunc = func(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
		return nil, errors.New("custom error")
	}

	_, err := mock.GetRuntime(ctx, "session-123")
	if err == nil || err.Error() != "custom error" {
		t.Errorf("GetRuntime() should return custom error, got %v", err)
	}

	// Test with custom processMessageFunc
	mock.processMessageFunc = func(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error) {
		return nil, errors.New("process error")
	}

	_, err = mock.ProcessMessage(ctx, "session-123", "hello")
	if err == nil || err.Error() != "process error" {
		t.Errorf("ProcessMessage() should return custom error, got %v", err)
	}

	// Test with custom createSessionFunc
	mock.createSessionFunc = func(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error) {
		return nil, errors.New("create error")
	}

	_, err = mock.CreateSession(ctx, "test", nil, "")
	if err == nil || err.Error() != "create error" {
		t.Errorf("CreateSession() should return custom error, got %v", err)
	}
}
