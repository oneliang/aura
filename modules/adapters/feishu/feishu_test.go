package feishu

import (
	"context"
	"testing"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/logger"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled:   true,
				AppID:     "test_app_id",
				AppSecret: "test_secret",
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing app_id",
			config: &Config{
				Enabled:   true,
				AppSecret: "test_secret",
			},
			wantErr: true,
		},
		{
			name: "missing app_secret",
			config: &Config{
				Enabled: true,
				AppID:   "test_app_id",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("DefaultConfig().Enabled should be false")
	}
	if !cfg.AsyncProcessing {
		t.Error("DefaultConfig().AsyncProcessing should be true")
	}
	if !cfg.AutoReply {
		t.Error("DefaultConfig().AutoReply should be true")
	}
}

func TestAdapter_NameAndDescription(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	if adapter.Name() != "feishu" {
		t.Errorf("Expected name 'feishu', got '%s'", adapter.Name())
	}

	if adapter.Description() == "" {
		t.Error("Expected description to be non-empty")
	}
}

func TestAdapter_Initialize_InvalidConfig(t *testing.T) {
	adapter := NewAdapter(&Config{
		Enabled: true,
		// Missing required fields
	})

	err := adapter.Initialize(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

func TestAdapter_Status(t *testing.T) {
	adapter := NewAdapter(DefaultConfig())

	status := adapter.Status()
	if status.Running {
		t.Error("Expected adapter not running before Initialize")
	}
	if status.Health != "initializing" {
		t.Errorf("Expected health 'initializing', got '%s'", status.Health)
	}
}

func TestClient_NewClient(t *testing.T) {
	log := logger.Default()
	client := NewClient("test_app_id", "test_secret", log)

	if client == nil {
		t.Error("Expected non-nil client")
	}
	if client.appID != "test_app_id" {
		t.Errorf("Expected app_id 'test_app_id', got '%s'", client.appID)
	}
	if client.appSecret != "test_secret" {
		t.Errorf("Expected app_secret 'test_secret', got '%s'", client.appSecret)
	}
}

func TestClient_SendTextMessage_InvalidToken(t *testing.T) {
	log := logger.Default()
	client := NewClient("invalid_app_id", "invalid_secret", log)

	err := client.SendTextMessage(context.Background(), "ou_test", "open_id", "test message")
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}

// MockResourceManager for testing
type MockResourceManager struct {
	sessions           map[string]string
	runtimeErr         error
	getRuntimeFunc     func(ctx context.Context, sessionID string) (*sdk.Runtime, error)
	processMessageFunc func(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error)
	createSessionFunc  func(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error)
}

func (m *MockResourceManager) GetOrCreateSession(ctx context.Context, source string, identifier string) (string, error) {
	if m.sessions == nil {
		m.sessions = make(map[string]string)
	}
	key := source + ":" + identifier
	if id, ok := m.sessions[key]; ok {
		return id, nil
	}
	id := "session_" + source + "_" + identifier
	m.sessions[key] = id
	return id, nil
}

func (m *MockResourceManager) GetRuntime(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
	if m.getRuntimeFunc != nil {
		return m.getRuntimeFunc(ctx, sessionID)
	}
	if m.runtimeErr != nil {
		return nil, m.runtimeErr
	}
	return nil, nil
}

func (m *MockResourceManager) ProcessMessage(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error) {
	if m.processMessageFunc != nil {
		return m.processMessageFunc(ctx, sessionID, content)
	}
	ch := make(chan sdk.Event)
	close(ch)
	return ch, nil
}

func (m *MockResourceManager) SessionStore() *storage.JSONLStore {
	return nil
}

func (m *MockResourceManager) CreateSession(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, name, subscriptions, role)
	}
	return &model.Session{
		ID:            "session_" + name,
		Name:          name,
		Subscriptions: subscriptions,
	}, nil
}

// Test Adapter Shutdown
func TestAdapter_Shutdown(t *testing.T) {
	adapter := NewAdapter(&Config{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	})

	// Initialize with valid config (will create done channel)
	// Note: This may fail due to network/credentials, but will set up the adapter
	adapter.Initialize(context.Background(), &MockResourceManager{})

	// Now shutdown should work (or at least not panic)
	ctx := context.Background()
	adapter.Shutdown(ctx)
	// Don't check for error here - we just want to verify no panic

	status := adapter.Status()
	if status.Running {
		t.Error("Expected adapter not running after Shutdown")
	}
}

// Test Adapter Shutdown with context cancellation
func TestAdapter_Shutdown_ContextCanceled(t *testing.T) {
	adapter := NewAdapter(&Config{
		Enabled:   true,
		AppID:     "test_app_id",
		AppSecret: "test_secret",
	})

	// Initialize with valid config (will create done channel)
	adapter.Initialize(context.Background(), &MockResourceManager{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := adapter.Shutdown(ctx)
	if err != nil && err != context.Canceled {
		t.Errorf("Shutdown() expected context.Canceled or nil, got = %v", err)
	}
}

// Test MessageEventInfo struct
func TestMessageEventInfo(t *testing.T) {
	info := &MessageEventInfo{
		ChatID:    "ch_test",
		MessageID: "msg_test",
		SenderID:  "ou_test",
		IsGroup:   true,
	}

	if info.ChatID != "ch_test" {
		t.Errorf("Expected ChatID 'ch_test', got '%s'", info.ChatID)
	}
	if info.MessageID != "msg_test" {
		t.Errorf("Expected MessageID 'msg_test', got '%s'", info.MessageID)
	}
	if info.SenderID != "ou_test" {
		t.Errorf("Expected SenderID 'ou_test', got '%s'", info.SenderID)
	}
	if !info.IsGroup {
		t.Error("Expected IsGroup to be true")
	}
}

// Test MockResourceManager
func TestMockResourceManager_GetOrCreateSession(t *testing.T) {
	mock := &MockResourceManager{}

	ctx := context.Background()
	id1, err := mock.GetOrCreateSession(ctx, "feishu", "user123")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	id2, err := mock.GetOrCreateSession(ctx, "feishu", "user123")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	if id1 != id2 {
		t.Errorf("Expected same session ID for same user, got %s and %s", id1, id2)
	}

	id3, err := mock.GetOrCreateSession(ctx, "feishu", "user456")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	if id3 == id1 {
		t.Errorf("Expected different session ID for different user")
	}
}

func TestMockResourceManager_GetRuntime(t *testing.T) {
	mock := &MockResourceManager{
		runtimeErr: context.DeadlineExceeded,
	}

	ctx := context.Background()
	rt, err := mock.GetRuntime(ctx, "session_1")
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
	if rt != nil {
		t.Error("Expected nil runtime")
	}
}

func TestMockResourceManager_GetRuntime_WithCustomFunc(t *testing.T) {
	called := false
	mock := &MockResourceManager{
		getRuntimeFunc: func(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
			called = true
			return nil, nil
		},
	}

	ctx := context.Background()
	_, err := mock.GetRuntime(ctx, "session_1")
	if err != nil {
		t.Errorf("GetRuntime() unexpected error = %v", err)
	}
	if !called {
		t.Error("Expected custom function to be called")
	}
}

func TestMockResourceManager_ProcessMessage(t *testing.T) {
	mock := &MockResourceManager{}

	ctx := context.Background()
	ch, err := mock.ProcessMessage(ctx, "session_1", "test content")
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed")
	}
}

func TestMockResourceManager_CreateSession(t *testing.T) {
	mock := &MockResourceManager{}

	ctx := context.Background()
	session, err := mock.CreateSession(ctx, "test-session", nil, "helper")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.ID != "session_test-session" {
		t.Errorf("Expected session ID 'session_test-session', got '%s'", session.ID)
	}
	if session.Name != "test-session" {
		t.Errorf("Expected session name 'test-session', got '%s'", session.Name)
	}
}

func TestMockResourceManager_CreateSession_WithCustomFunc(t *testing.T) {
	called := false
	mock := &MockResourceManager{
		createSessionFunc: func(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error) {
			called = true
			return &model.Session{
				ID:   "custom-id",
				Name: name,
			}, nil
		},
	}

	ctx := context.Background()
	session, err := mock.CreateSession(ctx, "test", nil, "")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if !called {
		t.Error("Expected custom function to be called")
	}
	if session.ID != "custom-id" {
		t.Errorf("Expected session ID 'custom-id', got '%s'", session.ID)
	}
}
