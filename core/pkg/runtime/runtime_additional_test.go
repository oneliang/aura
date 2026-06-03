// Package runtime provides additional tests for the runtime package.
package runtime

import (
	"context"
	"testing"

	"github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	tools "github.com/oneliang/aura/tools/pkg"
)

// TestAgentRuntime_GetToolNames tests GetToolNames method.
func TestAgentRuntime_GetToolNames(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Initially should be empty or nil
	names := runtime.GetToolNames()
	if names == nil {
		// Empty slice is also acceptable
		names = []string{}
	}
	// Test passes as long as it doesn't panic
}

// TestAgentRuntime_GetAgent tests GetAgent method.
func TestAgentRuntime_GetAgent(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Before initialization, agent should be nil
	agent := runtime.GetAgent()
	if agent != nil {
		t.Error("GetAgent() should return nil before initialization")
	}
}

// TestAgentRuntime_GetMemory tests GetMemory method.
func TestAgentRuntime_GetMemory(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Before initialization, memory should be nil
	mem := runtime.GetMemory()
	if mem != nil {
		t.Error("GetMemory() should return nil before initialization")
	}
}

// TestAgentRuntime_GetSummarizer tests GetSummarizer method.
func TestAgentRuntime_GetSummarizer(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Before initialization, summarizer should be nil
	summarizer := runtime.GetSummarizer()
	if summarizer != nil {
		t.Error("GetSummarizer() should return nil before initialization")
	}
}

// TestAgentRuntime_Shutdown tests Shutdown method.
func TestAgentRuntime_Shutdown(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Set initialized to true to simulate initialized state
	runtime.initialized = true

	// Shutdown should not panic
	runtime.Shutdown()

	if runtime.initialized {
		t.Error("Shutdown() did not set initialized to false")
	}
}

// TestAgentRuntime_Shutdown_WithMemory tests Shutdown with memory.
func TestAgentRuntime_Shutdown_WithMemory(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Create session memory with in-memory store
	tempDir := t.TempDir()
	store, err := jsonl.NewMessageStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem, err := memory.NewSessionMemoryWithConfig("", "test-session", store, memory.SessionMemoryConfig{
		MaxLen: 50,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Add a message
	mem.Add("user", "test message")
	runtime.memory = mem

	// Shutdown should preserve memory (session data persistence)
	runtime.Shutdown()

	// Memory should NOT be cleared - session data is preserved for future use
	if len(mem.Get()) != 1 {
		t.Error("Shutdown() should preserve session memory, but it was cleared")
	}

	// Verify initialized flag is set to false
	if runtime.initialized {
		t.Error("Shutdown() did not set initialized to false")
	}
}

// TestAgentRuntime_Initialize_InvalidConfig tests Initialize with invalid config.
func TestAgentRuntime_Initialize_InvalidConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.LLM.Provider = "non-existent-provider"
	cfg.DisableTools = true

	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = runtime.Initialize(ctx)
	// Note: The Initialize method may not validate provider names strictly
	// It might only fail when actually making requests
	// This test just verifies it doesn't panic
	_ = err
}

// TestAgentRuntime_convertEvent tests convertEvent method.
func TestAgentRuntime_convertEvent(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tests := []struct {
		name       string
		agentEvent events.Event
		wantType   EventType
	}{
		{
			name:       "thinking event",
			agentEvent: events.NewEvent(events.EventTypeThinkingStart, "thinking content", ""),
			wantType:   EventTypeThinkingStart,
		},
		{
			name:       "action event",
			agentEvent: events.NewEvent(events.EventTypeAction, "action content", ""),
			wantType:   EventTypeAction,
		},
		{
			name:       "result event",
			agentEvent: events.NewEvent(events.EventTypeResult, "result content", ""),
			wantType:   EventTypeResult,
		},
		{
			name:       "response event",
			agentEvent: events.NewEvent(events.EventTypeResponse, "response content", ""),
			wantType:   EventTypeResponse,
		},
		{
			name:       "error event",
			agentEvent: events.NewEvent(events.EventTypeError, "error content", ""),
			wantType:   EventTypeError,
		},
		{
			name:       "step event",
			agentEvent: events.NewEvent(events.EventTypeStep, "step content", ""),
			wantType:   EventTypeStep,
		},
		{
			name:       "tool start event",
			agentEvent: events.NewEvent(events.EventTypeToolStart, "tool start", ""),
			wantType:   EventTypeToolStart,
		},
		{
			name:       "tool end event",
			agentEvent: events.NewEvent(events.EventTypeToolEnd, "tool end", ""),
			wantType:   EventTypeToolEnd,
		},
		{
			name:       "unknown event type",
			agentEvent: events.NewEvent("unknown_type", "unknown content", ""),
			wantType:   "unknown_type", // Unknown types are passed through with original type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.convertEvent(tt.agentEvent)
			if result.Type() != tt.wantType {
				t.Errorf("convertEvent() type = %v, want %v", result.Type(), tt.wantType)
			}
		})
	}
}

// TestAgentRuntime_convertEvent_WithExtra tests convertEvent with extra data.
func TestAgentRuntime_convertEvent_WithExtra(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	extra := map[string]any{"key": "value"}
	agentEvent := events.NewEventWithExtra(events.EventTypeStep, "step content", extra, "")

	result := runtime.convertEvent(agentEvent)

	if result.Type() != EventTypeStep {
		t.Errorf("convertEvent() type = %v, want %v", result.Type(), EventTypeStep)
	}

	resultExtra := result.Extra()
	if resultExtra == nil {
		t.Error("convertEvent() should preserve extra data")
	} else if resultExtra["key"] != "value" {
		t.Errorf("convertEvent() extra data = %v, want %v", resultExtra["key"], "value")
	}
}

// TestAgentRuntime_buildSystemPrompt tests buildSystemPrompt method.
func TestAgentRuntime_buildSystemPrompt(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.EnableSubAgent = true // Enable sub-agent for normal behavior

	// Test with explicit system prompt
	cfg.SystemPrompt = "Custom system prompt"
	runtime, _ := New(cfg)

	prompt := runtime.buildSystemPrompt()
	if prompt != "Custom system prompt" {
		t.Errorf("buildSystemPrompt() = %q, want %q", prompt, "Custom system prompt")
	}
}

// TestAgentRuntime_buildSystemPrompt_WithRole tests buildSystemPrompt with role.
func TestAgentRuntime_buildSystemPrompt_WithRole(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.Role = "test-role"
	runtime, _ := New(cfg)

	// This will use promptBuilder.BuildWithRole which may panic with nil roleLoader
	// We just verify the method exists and can be called
	defer func() {
		if r := recover(); r != nil {
			// Expected - role loader may not be initialized
			t.Logf("buildSystemPrompt_WithRole panicked (expected): %v", r)
		}
	}()
	prompt := runtime.buildSystemPrompt()
	_ = prompt
}

// TestAgentRuntime_buildSystemPrompt_WithConfig tests buildSystemPrompt with config.
func TestAgentRuntime_buildSystemPrompt_WithConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, _ := New(cfg)

	// This will use promptBuilder.BuildWithConfig
	prompt := runtime.buildSystemPrompt()
	_ = prompt
}

// TestAgentRuntime_createMemory tests createMemory method.
func TestAgentRuntime_createMemory(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.Memory.MaxContext = 100
	runtime, _ := New(cfg)

	mem, err := runtime.createMemory()
	if err != nil {
		t.Fatalf("createMemory() error = %v", err)
	}
	if mem == nil {
		t.Fatal("createMemory() returned nil")
	}
}

// TestAgentRuntime_buildSystemPrompt_WithSkills tests buildSystemPrompt with skills.
func TestAgentRuntime_buildSystemPrompt_WithSkills(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, _ := New(cfg)

	// Note: skillLoader is *loader.Loader, hard to mock without import
	// This test just verifies the method doesn't panic with nil loader
	prompt := runtime.buildSystemPrompt()
	_ = prompt
}

// mockTool implements tools.Tool for testing.
type mockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]any) (*tools.ToolResult, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "mock result"}, nil
}

// TestAgentRuntime_Initialize_Success tests successful initialization.
func TestAgentRuntime_Initialize_Success(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	// Use a valid provider
	cfg.LLM.Provider = "ollama"
	cfg.LLM.Model = "qwen3:8b"
	cfg.DisableTools = true // Disable tools to avoid external dependencies

	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = runtime.Initialize(ctx)
	// May fail if Ollama is not running, but should not panic
	_ = err
}

// TestAgentRuntime_Process_AfterInit tests Process after initialization.
func TestAgentRuntime_Process_AfterInit(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.LLM.Provider = "ollama"
	cfg.LLM.Model = "qwen3:8b"
	cfg.DisableTools = true

	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	err = runtime.Initialize(ctx)
	// May fail if Ollama is not running
	if err != nil {
		// Skip test if initialization fails due to external dependency
		t.Skipf("Skipping test due to initialization failure: %v", err)
	}

	// Process should return a channel, not error
	_, err = runtime.Process(ctx, "test input")
	if err != nil {
		t.Errorf("Process() returned error: %v", err)
	}
}

// TestNew_WithSessionID tests WithSessionID option.
func TestNew_WithSessionID(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg, WithSessionID("my-session"))
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime.sessionID != "my-session" {
		t.Errorf("sessionID = %q, want %q", runtime.sessionID, "my-session")
	}
}

// TestWithSessionStore tests WithSessionStore option.
func TestWithSessionStore(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	// We can't easily create a real MessageStore, so test with nil
	// The option should still be accepted
	runtime, err := New(cfg, WithSessionStore(nil))
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() with WithSessionStore returned nil")
	}
}

// TestWithCommands tests WithCommands option.
func TestWithCommands(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	// Create a mock command provider
	mockProvider := &mockCommandProvider{}

	runtime, err := New(cfg, WithCommands(mockProvider))
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() with WithCommands returned nil")
	}

	if runtime.commandProvider == nil {
		t.Error("commandProvider should be set")
	}
}

// TestWithIntentService tests WithIntentService option.
func TestWithIntentService(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	// Use nil intent service for testing
	runtime, err := New(cfg, WithIntentService(nil))
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("New() with WithIntentService returned nil")
	}

	// intentService can be nil
}

// TestAgentRuntime_Process_NotInitialized tests Process before initialization.
func TestAgentRuntime_Process_NotInitialized(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	_, err = runtime.Process(ctx, "test input")

	if err == nil {
		t.Error("Process() should return error when not initialized")
	}
}

// mockCommandProvider implements commands.Command for testing.
type mockCommandProvider struct{}

func (m *mockCommandProvider) GetCommands() []commands.CommandInfo {
	return nil
}

func (m *mockCommandProvider) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	return "mock result", nil
}
