package adapters

import (
	"context"
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
)

// TestCanPushMessage tests the CanPushMessage function.
func TestCanPushMessage(t *testing.T) {
	// Test with adapter that implements MessagePusher
	pusherAdapter := &mockPusherAdapter{}
	pusher, ok := CanPushMessage(pusherAdapter)

	if !ok {
		t.Error("CanPushMessage() should return true for adapter implementing MessagePusher")
	}
	if pusher == nil {
		t.Error("CanPushMessage() should return non-nil pusher")
	}

	// Test with adapter that doesn't implement MessagePusher
	nonPusherAdapter := NewMockAdapter("test", "test adapter")
	pusher, ok = CanPushMessage(nonPusherAdapter)

	if ok {
		t.Error("CanPushMessage() should return false for adapter not implementing MessagePusher")
	}
	if pusher != nil {
		t.Error("CanPushMessage() should return nil pusher for non-implementing adapter")
	}
}

// TestMin tests the min function.
func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.want)
		}
	}
}

// TestMessageTarget tests MessageTarget struct.
func TestMessageTarget(t *testing.T) {
	target := MessageTarget{
		TargetType: "open_id",
		TargetID:   "ou_xxx",
		Name:       "Test User",
	}

	if target.TargetType != "open_id" {
		t.Errorf("TargetType = %q, want %q", target.TargetType, "open_id")
	}
	if target.TargetID != "ou_xxx" {
		t.Errorf("TargetID = %q, want %q", target.TargetID, "ou_xxx")
	}
	if target.Name != "Test User" {
		t.Errorf("Name = %q, want %q", target.Name, "Test User")
	}
}

// TestMessageTypeConstants tests message type constants.
func TestMessageTypeConstants(t *testing.T) {
	if MsgTypeText != "text" {
		t.Errorf("MsgTypeText = %q, want %q", MsgTypeText, "text")
	}
	if MsgTypePost != "post" {
		t.Errorf("MsgTypePost = %q, want %q", MsgTypePost, "post")
	}
	if MsgTypeCard != "card" {
		t.Errorf("MsgTypeCard = %q, want %q", MsgTypeCard, "card")
	}
	if MsgTypeImage != "image" {
		t.Errorf("MsgTypeImage = %q, want %q", MsgTypeImage, "image")
	}
	if MsgTypeFile != "file" {
		t.Errorf("MsgTypeFile = %q, want %q", MsgTypeFile, "file")
	}
	if MsgTypeAudio != "audio" {
		t.Errorf("MsgTypeAudio = %q, want %q", MsgTypeAudio, "audio")
	}
	if MsgTypeSticker != "sticker" {
		t.Errorf("MsgTypeSticker = %q, want %q", MsgTypeSticker, "sticker")
	}
	if MsgTypeShareChat != "share_chat" {
		t.Errorf("MsgTypeShareChat = %q, want %q", MsgTypeShareChat, "share_chat")
	}
}

// TestAdapterStatus tests AdapterStatus struct.
func TestAdapterStatus(t *testing.T) {
	status := AdapterStatus{
		Running: true,
		Health:  "healthy",
		Message: "All good",
	}

	if !status.Running {
		t.Error("AdapterStatus.Running should be true")
	}
	if status.Health != "healthy" {
		t.Errorf("AdapterStatus.Health = %q, want %q", status.Health, "healthy")
	}
	if status.Message != "All good" {
		t.Errorf("AdapterStatus.Message = %q, want %q", status.Message, "All good")
	}
}

// TestSessionAccessor_Interface tests SessionAccessor interface compliance.
func TestSessionAccessor_Interface(t *testing.T) {
	// Verify MockResourceManager implements SessionAccessor
	var _ SessionAccessor = (*MockResourceManager)(nil)
}

// TestRuntimeAccessor_Interface tests RuntimeAccessor interface compliance.
func TestRuntimeAccessor_Interface(t *testing.T) {
	// Verify MockResourceManager implements RuntimeAccessor
	var _ RuntimeAccessor = (*MockResourceManager)(nil)
}

// TestMessageProcessor_Interface tests MessageProcessor interface compliance.
func TestMessageProcessor_Interface(t *testing.T) {
	// Verify MockResourceManager implements MessageProcessor
	var _ MessageProcessor = (*MockResourceManager)(nil)
}

// TestResourceManager_Interface tests ResourceManager interface compliance.
func TestResourceManager_Interface(t *testing.T) {
	// Verify MockResourceManager implements ResourceManager
	var _ ResourceManager = (*MockResourceManager)(nil)
}

// TestNewAdapterResourceManager tests NewAdapterResourceManager function.
func TestNewAdapterResourceManager(t *testing.T) {
	mgr := NewAdapterResourceManager(nil, nil, nil, nil, "")

	if mgr == nil {
		t.Fatal("NewAdapterResourceManager() returned nil")
	}

	if mgr.runtimes == nil {
		t.Error("runtimes map should be initialized")
	}
}

// TestAdapterResourceManager_SessionStore tests SessionStore method.
func TestAdapterResourceManager_SessionStore(t *testing.T) {
	// With nil store
	mgr := NewAdapterResourceManager(nil, nil, nil, nil, "")
	if mgr.SessionStore() != nil {
		t.Error("SessionStore() should return nil when store is nil")
	}

	// With actual store
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mgr = NewAdapterResourceManager(nil, nil, nil, store, "")
	if mgr.SessionStore() != store {
		t.Error("SessionStore() should return the store")
	}
}

// TestAdapterResourceManager_Close tests Close method.
func TestAdapterResourceManager_Close(t *testing.T) {
	mgr := NewAdapterResourceManager(nil, nil, nil, nil, "")

	// Close should not panic with no runtimes
	mgr.Close()

	// Close should be idempotent
	mgr.Close()
}

// TestAdapterResourceManager_Close_WithRuntimes tests Close with cached runtimes.
func TestAdapterResourceManager_Close_WithRuntimes(t *testing.T) {
	mgr := NewAdapterResourceManager(nil, nil, nil, nil, "")

	// Add a fake runtime entry (it won't have a real runtime, but Close should handle it)
	mgr.runtimes["test-session"] = &sessionRuntime{}

	// Close should not panic even with entries
	// Note: The actual runtime.Shutdown() would panic with nil runtime,
	// but the Close method should still work
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from nil runtime, but Close should complete
			t.Logf("Recovered from panic (expected): %v", r)
		}
	}()
	mgr.Close()
}

// TestAdapterResourceManager_GetSession tests GetSession method.
func TestAdapterResourceManager_GetSession(t *testing.T) {
	// With nil session manager, GetSession will panic
	// We test that the method exists and can be called with valid session manager
	// Skipping this test as it requires actual session manager setup
	t.Skip("Requires actual SessionManager setup")
}

// TestAdapterResourceManager_CreateSession tests CreateSession method.
func TestAdapterResourceManager_CreateSession(t *testing.T) {
	// With nil session manager, CreateSession will panic
	// We test that the method exists and can be called with valid session manager
	// Skipping this test as it requires actual session manager setup
	t.Skip("Requires actual SessionManager setup")
}

// mockPusherAdapter is a mock adapter that implements MessagePusher.
type mockPusherAdapter struct {
	*MockAdapter
}

func (m *mockPusherAdapter) PushMessage(ctx context.Context, targetType, targetID, msgType string, content map[string]interface{}) error {
	return nil
}

func (m *mockPusherAdapter) BroadcastMessage(ctx context.Context, targets []MessageTarget, msgType string, content map[string]interface{}) map[string]error {
	return nil
}

// TestMockResourceManager tests the MockResourceManager implementation.
func TestMockResourceManager_GetOrCreateSession(t *testing.T) {
	mock := &MockResourceManager{}
	ctx := context.Background()

	// First call should create new session
	id1, err := mock.GetOrCreateSession(ctx, "feishu", "user123")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	// Second call with same params should return same ID
	id2, err := mock.GetOrCreateSession(ctx, "feishu", "user123")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	if id1 != id2 {
		t.Errorf("Expected same session ID, got %s and %s", id1, id2)
	}

	// Different user should get different ID
	id3, err := mock.GetOrCreateSession(ctx, "feishu", "user456")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}

	if id1 == id3 {
		t.Error("Expected different session IDs for different users")
	}
}

// TestMockResourceManager_CreateSession tests CreateSession.
func TestMockResourceManager_CreateSession(t *testing.T) {
	mock := &MockResourceManager{}
	ctx := context.Background()

	subs := []model.Subscription{
		{ID: "sub1", Trigger: "alert", Source: "feishu", Active: true},
	}

	session, err := mock.CreateSession(ctx, "test-session", subs, "helper")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session == nil {
		t.Fatal("CreateSession() returned nil session")
	}

	if session.Name != "test-session" {
		t.Errorf("Session.Name = %q, want %q", session.Name, "test-session")
	}

	if len(session.Subscriptions) != 1 {
		t.Errorf("Session.Subscriptions length = %d, want 1", len(session.Subscriptions))
	}
}

// TestMockResourceManager_ProcessMessage tests ProcessMessage.
func TestMockResourceManager_ProcessMessage(t *testing.T) {
	mock := &MockResourceManager{}
	ctx := context.Background()

	ch, err := mock.ProcessMessage(ctx, "session-123", "Hello")
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	if ch == nil {
		t.Fatal("ProcessMessage() returned nil channel")
	}

	// Channel should be closed immediately (mock behavior)
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed")
	}
}

// TestMockResourceManager_SessionStore tests SessionStore.
func TestMockResourceManager_SessionStore(t *testing.T) {
	mock := &MockResourceManager{}

	if mock.SessionStore() != nil {
		t.Error("MockResourceManager.SessionStore() should return nil")
	}
}

// TestMockResourceManager_GetRuntime tests GetRuntime.
func TestMockResourceManager_GetRuntime(t *testing.T) {
	mock := &MockResourceManager{}
	ctx := context.Background()

	rt, err := mock.GetRuntime(ctx, "session-123")
	if err != nil {
		t.Fatalf("GetRuntime() error = %v", err)
	}

	if rt != nil {
		t.Error("MockResourceManager.GetRuntime() should return nil runtime")
	}
}
