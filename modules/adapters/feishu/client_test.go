package feishu

import (
	"context"
	"testing"

	"github.com/oneliang/aura/shared/pkg/logger"
)

func TestClient_SendMessage(t *testing.T) {
	log := logger.Default()
	client := NewClient("test_app_id", "test_secret", log)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Test with invalid credentials - should fail
	err := client.SendMessage(context.Background(), "ou_test", "open_id", map[string]string{"text": "test"})
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}

func TestClient_SendMessage_EmptyContent(t *testing.T) {
	log := logger.Default()
	client := NewClient("test_app_id", "test_secret", log)

	// Test with empty content map
	err := client.SendMessage(context.Background(), "ou_test", "open_id", map[string]string{})
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}

func TestClient_SendReplyMessage(t *testing.T) {
	log := logger.Default()
	client := NewClient("test_app_id", "test_secret", log)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Test with invalid credentials - should fail
	err := client.SendReplyMessage(context.Background(), "chat_test", "msg_123", "text", map[string]string{"text": "reply"})
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}

func TestClient_SendTextMessage_EmptyText(t *testing.T) {
	log := logger.Default()
	client := NewClient("test_app_id", "test_secret", log)

	// Test with empty text
	err := client.SendTextMessage(context.Background(), "ou_test", "open_id", "")
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}
