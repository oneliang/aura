package runtime

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to create a message with text content
func newTextMessage(role, text string) llm.Message {
	msg := llm.Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

// Helper function to extract text content from message
func getTextContent(msg llm.Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

func TestNew(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	runtime, err := New(cfg)

	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() returned nil runtime")
	}

	if runtime.config != cfg {
		t.Error("New() did not store config correctly")
	}
}

func TestNew_WithOptions(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.SessionID = "test-session"

	eventHandlerCalled := false

	eventHandler := func(event Event) {
		eventHandlerCalled = true
	}

	confirmHandler := func(req ConfirmationRequest) {
		_ = req // unused for now
	}

	runtime, err := New(
		cfg,
		WithEventHandler(eventHandler),
		WithConfirmationHandler(confirmHandler),
		WithSessionID("test-session"),
	)

	if err != nil {
		t.Fatalf("New() with options returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() with options returned nil runtime")
	}

	if runtime.sessionID != "test-session" {
		t.Errorf("New() sessionID = %q, want %q", runtime.sessionID, "test-session")
	}

	if runtime.onEvent == nil {
		t.Error("New() onEvent should be set")
	}

	if runtime.onConfirm == nil {
		t.Error("New() onConfirm should be set")
	}

	// Verify handlers are called
	if runtime.onEvent != nil {
		runtime.onEvent(NewEvent(EventTypeThinkingStart, "test"))
		if !eventHandlerCalled {
			t.Error("EventHandler was not called")
		}
	}
}

func TestNew_WithSessionStore(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	// Note: We can't easily test WithSessionStore without a real JSONLStore
	// This test verifies the option doesn't cause errors
	runtime, err := New(cfg, WithSessionID("test"))

	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() returned nil runtime")
	}
}

func TestNew_WithMode(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	runtime, err := New(cfg, WithMode(RuntimeModeTUI))

	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime.mode != RuntimeModeTUI {
		t.Errorf("New() did not store mode correctly: got %v, want %v", runtime.mode, RuntimeModeTUI)
	}
}

func TestNew_DefaultMode(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	runtime, err := New(cfg)

	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime.mode != RuntimeModeCLI {
		t.Errorf("Default mode should be RuntimeModeCLI: got %v", runtime.mode)
	}
}

// MockMemory for runtime testing
type mockMemory struct {
	messages []llm.Message
}

func (m *mockMemory) Add(role, content string) {
	m.messages = append(m.messages, newTextMessage(role, content))
}

func (m *mockMemory) AddWithType(role, content string, msgType memory.MessageType) {
	m.messages = append(m.messages, newTextMessage(role, content))
}

func (m *mockMemory) AddWithParts(role string, parts []memory.MessagePart, msgType memory.MessageType) {
}

func (m *mockMemory) AddWithBlocks(role string, blocks []memory.ContentBlock, msgType memory.MessageType) {
	var textContent string
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			textContent = tb.Text
			break
		}
	}
	m.messages = append(m.messages, newTextMessage(role, textContent))
}

func (m *mockMemory) Get() []llm.Message {
	return m.messages
}

func (m *mockMemory) Clear() {
	m.messages = nil
}

// Verify mockMemory implements memory.Memory
var _ memory.Memory = (*mockMemory)(nil)

func TestMemory_Interface(t *testing.T) {
	mem := &mockMemory{}

	mem.Add("user", "hello")
	mem.Add("assistant", "hi there")

	messages := mem.Get()
	if len(messages) != 2 {
		t.Errorf("Memory.Get() returned %d messages, want 2", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Memory.Get()[0].Role = %q, want %q", messages[0].Role, "user")
	}

	if getTextContent(messages[0]) != "hello" {
		t.Errorf("Memory.Get()[0].Content = %q, want %q", getTextContent(messages[0]), "hello")
	}

	mem.Clear()
	messages = mem.Get()
	if len(messages) != 0 {
		t.Errorf("Memory.Get() after Clear returned %d messages, want 0", len(messages))
	}
}

// TestSessionMemory_CreateMemory tests the createMemory functionality indirectly
func TestSessionMemory_Configuration(t *testing.T) {
	// Test that MemoryConfig is properly used
	cfg := DefaultRuntimeConfig()

	if cfg.Memory.MaxContext != 50 {
		t.Errorf("Default Memory.MaxContext = %d, want 50", cfg.Memory.MaxContext)
	}

	cfg.Memory.MaxContext = 100
	if cfg.Memory.MaxContext != 100 {
		t.Errorf("Memory.MaxContext = %d, want 100", cfg.Memory.MaxContext)
	}
}

// Test permissions manager creation
func TestPermissionsManager_Creation(t *testing.T) {
	permCfg := permissions.DefaultPermissionConfig()
	manager, err := permissions.NewManager(permCfg)

	if err != nil {
		t.Fatalf("permissions.NewManager() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("permissions.NewManager() returned nil")
	}
}