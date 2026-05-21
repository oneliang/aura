package sdk

import (
	"context"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/core/pkg/runtime"
	"github.com/oneliang/aura/shared/pkg/config"
)

func TestDefaultRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	if cfg == nil {
		t.Fatal("DefaultRuntimeConfig() returned nil")
	}

	// Verify it returns a valid config
	if cfg.Config == nil {
		t.Error("DefaultRuntimeConfig() Config is nil")
	}
}

func TestFromConfig(t *testing.T) {
	srcCfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "qwen3:8b",
		},
	}

	cfg := FromConfig(srcCfg)

	if cfg == nil {
		t.Fatal("FromConfig() returned nil")
	}

	if cfg.Config == nil {
		t.Error("FromConfig() Config is nil")
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("FromConfig() Provider = %q, want %q", cfg.LLM.Provider, "ollama")
	}
}

func TestNewRuntime(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	runtime, err := NewRuntime(cfg)

	if err != nil {
		t.Fatalf("NewRuntime() returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("NewRuntime() returned nil")
	}
}

func TestNewRuntime_WithOptions(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.SessionID = "test-session"

	eventHandler := func(event Event) {
		_ = event // unused for now
	}

	runtime, err := NewRuntime(
		cfg,
		WithEventHandler(eventHandler),
		WithSessionID("test-session"),
	)

	if err != nil {
		t.Fatalf("NewRuntime() with options returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("NewRuntime() with options returned nil")
	}
}

func TestNewRuntime_WithConfirmationHandler(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	confirmHandler := func(req ConfirmationRequest) {
		_ = req // unused for now
	}

	runtime, err := NewRuntime(
		cfg,
		WithConfirmationHandler(confirmHandler),
	)

	if err != nil {
		t.Fatalf("NewRuntime() with confirmation handler returned error: %v", err)
	}

	if runtime == nil {
		t.Fatal("NewRuntime() with confirmation handler returned nil")
	}
}

func TestNewSessionService(t *testing.T) {
	// Note: We can't easily test this without a real JSONLStore
	// This test verifies the function exists and doesn't crash
	_ = NewSessionService(nil)
}

func TestNewLLMFactory(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "ollama",
	}

	factory := NewLLMFactory(cfg)

	if factory == nil {
		t.Fatal("NewLLMFactory() returned nil")
	}
}

func TestNewEngineFactory(t *testing.T) {
	// This requires a real LLM client and permission manager
	// We just verify the function signature works
	llmClient := &mockLLMClient{}
	cfg := &config.AgentConfig{}
	permMgr, _ := createTestPermissionManager()

	factory := NewEngineFactory(llmClient, cfg, permMgr)

	if factory == nil {
		t.Fatal("NewEngineFactory() returned nil")
	}
}

func TestNewToolRegistry(t *testing.T) {
	cfg := &config.ToolsConfig{}
	permMgr, _ := createTestPermissionManager()

	registry := NewToolRegistry(cfg, permMgr)

	if registry == nil {
		t.Fatal("NewToolRegistry() returned nil")
	}
}

func TestNewPermissionManager(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
	}

	manager, err := NewPermissionManager(cfg)

	if err != nil {
		t.Fatalf("NewPermissionManager() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("NewPermissionManager() returned nil")
	}
}

func TestNewPermissionManager_DefaultConfig(t *testing.T) {
	cfg := &config.PermissionsConfig{}

	manager, err := NewPermissionManager(cfg)

	if err != nil {
		t.Fatalf("NewPermissionManager() with empty config returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("NewPermissionManager() returned nil")
	}
}

func TestNewPromptBuilder(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	if builder == nil {
		t.Fatal("NewPromptBuilder() returned nil")
	}
}

func TestNewRoleLoader(t *testing.T) {
	loader := NewRoleLoader("")

	if loader == nil {
		t.Fatal("NewRoleLoader() returned nil")
	}
}

func TestNewRoleLoader_CustomDir(t *testing.T) {
	loader := NewRoleLoader("/tmp/test-roles")

	if loader == nil {
		t.Fatal("NewRoleLoader() returned nil")
	}
}

func TestWithMode(t *testing.T) {
	option := WithMode(runtime.RuntimeModeTUI)

	if option == nil {
		t.Error("WithMode() returned nil")
	}
}

func TestWithEventHandler(t *testing.T) {
	handler := func(event Event) {}
	option := WithEventHandler(handler)

	if option == nil {
		t.Error("WithEventHandler() returned nil")
	}
}

func TestWithConfirmationHandler(t *testing.T) {
	handler := func(req ConfirmationRequest) {}
	option := WithConfirmationHandler(handler)

	if option == nil {
		t.Error("WithConfirmationHandler() returned nil")
	}
}

func TestWithSessionID(t *testing.T) {
	option := WithSessionID("test-id")

	if option == nil {
		t.Error("WithSessionID() returned nil")
	}
}

// Helper types for testing

type mockLLMClient struct{}

func (m *mockLLMClient) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return &llm.Response{}, nil
}

func (m *mockLLMClient) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk)
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{}, nil
}

func createTestPermissionManager() (*permissions.Manager, error) {
	cfg := permissions.DefaultPermissionConfig()
	return permissions.NewManager(cfg)
}
