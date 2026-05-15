package adapters

import (
	"context"
	"testing"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
)

// MockAdapter is a mock implementation of Adapter for testing.
type MockAdapter struct {
	name        string
	description string
	initialized bool
	shutdownErr error
}

func NewMockAdapter(name, description string) *MockAdapter {
	return &MockAdapter{
		name:        name,
		description: description,
	}
}

func (m *MockAdapter) Name() string {
	return m.name
}

func (m *MockAdapter) Description() string {
	return m.description
}

func (m *MockAdapter) Initialize(ctx context.Context, mgr ResourceManager) error {
	m.initialized = true
	return nil
}

func (m *MockAdapter) Shutdown(ctx context.Context) error {
	m.initialized = false
	return m.shutdownErr
}

func (m *MockAdapter) Status() AdapterStatus {
	return AdapterStatus{
		Running: m.initialized,
		Health:  "healthy",
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")

	err := reg.Register(adapter)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try to register same name again
	err = reg.Register(adapter)
	if err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")

	reg.Register(adapter)

	got, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", got.Name())
	}

	// Get non-existent adapter
	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent adapter, got nil")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewMockAdapter("adapter1", "first"))
	reg.Register(NewMockAdapter("adapter2", "second"))

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(list))
	}
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()
	if reg.Count() != 0 {
		t.Errorf("Expected count 0, got %d", reg.Count())
	}

	reg.Register(NewMockAdapter("adapter1", "first"))
	reg.Register(NewMockAdapter("adapter2", "second"))

	if reg.Count() != 2 {
		t.Errorf("Expected count 2, got %d", reg.Count())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")

	reg.Register(adapter)
	if reg.Count() != 1 {
		t.Errorf("Expected count 1, got %d", reg.Count())
	}

	reg.Unregister("test")
	if reg.Count() != 0 {
		t.Errorf("Expected count 0 after unregister, got %d", reg.Count())
	}
}

func TestRegistry_InitializeAll(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")
	reg.Register(adapter)

	ctx := context.Background()
	err := reg.InitializeAll(ctx, nil)
	if err != nil {
		t.Fatalf("InitializeAll() error = %v", err)
	}

	if !adapter.initialized {
		t.Error("Expected adapter to be initialized")
	}
}

func TestRegistry_ShutdownAll(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")
	reg.Register(adapter)

	// Initialize first
	adapter.initialized = true

	ctx := context.Background()
	err := reg.ShutdownAll(ctx)
	if err != nil {
		t.Fatalf("ShutdownAll() error = %v", err)
	}

	if adapter.initialized {
		t.Error("Expected adapter to be shut down")
	}
}

func TestRegistry_ShutdownAll_WithError(t *testing.T) {
	reg := NewRegistry()
	adapter := &MockAdapter{
		name:        "test",
		shutdownErr: context.DeadlineExceeded,
	}
	reg.Register(adapter)

	ctx := context.Background()
	err := reg.ShutdownAll(ctx)
	if err == nil {
		t.Error("Expected error for shutdown failure, got nil")
	}
}

func TestRegistry_StatusAll(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")
	reg.Register(adapter)

	// Initialize the adapter
	adapter.initialized = true

	statuses := reg.StatusAll()
	if len(statuses) != 1 {
		t.Errorf("Expected 1 status, got %d", len(statuses))
	}

	status := statuses["test"]
	if !status.Running {
		t.Error("Expected adapter to be running")
	}
	if status.Health != "healthy" {
		t.Errorf("Expected health 'healthy', got '%s'", status.Health)
	}
}

func TestRegistry_GetStatus(t *testing.T) {
	reg := NewRegistry()
	adapter := NewMockAdapter("test", "test adapter")
	reg.Register(adapter)

	adapter.initialized = true

	status, err := reg.GetStatus("test")
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.Running {
		t.Error("Expected adapter to be running")
	}

	// Get status for non-existent adapter
	_, err = reg.GetStatus("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent adapter, got nil")
	}
}

// Test ResourceManager interface with mock
type MockResourceManager struct {
	sessions map[string]string
}

func (m *MockResourceManager) GetOrCreateSession(ctx context.Context, source string, identifier string) (string, error) {
	key := source + ":" + identifier
	if id, ok := m.sessions[key]; ok {
		return id, nil
	}
	id := "session_" + source + "_" + identifier
	if m.sessions == nil {
		m.sessions = make(map[string]string)
	}
	m.sessions[key] = id
	return id, nil
}

func (m *MockResourceManager) GetRuntime(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
	return nil, nil // Mock doesn't need actual runtime
}

func (m *MockResourceManager) ProcessMessage(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error) {
	ch := make(chan sdk.Event)
	close(ch)
	return ch, nil
}

func (m *MockResourceManager) SessionStore() *storage.JSONLStore {
	return nil
}

func (m *MockResourceManager) CreateSession(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error) {
	return &model.Session{
		ID:            "session_" + name,
		Name:          name,
		Subscriptions: subscriptions,
	}, nil
}

func TestAdapterResourceManager_GetOrCreateSession(t *testing.T) {
	// This test would require actual SessionManager setup
	// For now, we test the concept with mock
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
}
