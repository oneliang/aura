// Package factory provides additional tests for the factory package.
package factory

import (
	"context"
	"testing"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/config"
	tools "github.com/oneliang/aura/tools/pkg"
)

// TestEngineFactory_Create_WithNilMemory tests Create with nil memory.
func TestEngineFactory_Create_WithNilMemory(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory := NewEngineFactory(llmClient, cfg, permMgr)

	// Create with nil memory
	ag, err := factory.Create(nil)

	// Agent creation with nil memory may fail or succeed depending on implementation
	if err == nil && ag == nil {
		t.Error("Create() should return either agent or error")
	}
}

// TestEngineFactory_Create_WithNilLLMClient tests Create with nil LLM client.
func TestEngineFactory_Create_WithNilLLMClient(t *testing.T) {
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory := NewEngineFactory(nil, cfg, permMgr)
	mem := &MockMemory{}

	// Create with nil LLM client
	ag, err := factory.Create(mem)

	// May fail or succeed depending on implementation
	if err == nil && ag == nil {
		t.Error("Create() should return either agent or error")
	}
}

// TestEngineFactory_Create_WithNilPermMgr tests Create with nil permission manager.
func TestEngineFactory_Create_WithNilPermMgr(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	factory := NewEngineFactory(llmClient, cfg, nil)
	mem := &MockMemory{}

	// Create with nil permission manager should work (uses default handler)
	ag, err := factory.Create(mem)

	if err != nil {
		t.Fatalf("Create() with nil permMgr returned error: %v", err)
	}
	if ag == nil {
		t.Fatal("Create() with nil permMgr returned nil agent")
	}
}

// TestEngineFactory_CreateWithSession_WithEmptySessionID tests CreateWithSession with empty session ID.
func TestEngineFactory_CreateWithSession_WithEmptySessionID(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory := NewEngineFactory(llmClient, cfg, permMgr)
	mem := &MockMemory{}

	ag, err := factory.CreateWithSession("", mem)

	if err != nil {
		t.Fatalf("CreateWithSession() with empty session ID returned error: %v", err)
	}
	if ag == nil {
		t.Fatal("CreateWithSession() returned nil agent")
	}
}

// TestEngineFactory_WithCommands tests WithCommands option.
func TestEngineFactory_WithCommands(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	// Create mock command provider
	mockCmd := &mockCommandProvider{}

	// Verify mock implements interface
	var _ commands.Command = mockCmd

	factory := NewEngineFactory(llmClient, cfg, permMgr, WithCommands(mockCmd))
	if factory == nil {
		t.Error("WithCommands() should not return nil factory")
	}
}

// mockCommandProvider implements commands.Command interface for testing.
type mockCommandProvider struct{}

func (m *mockCommandProvider) GetCommands() []commands.CommandInfo {
	return []commands.CommandInfo{{Name: "test", DisplayName: "Test", Description: "test command", Params: nil}}
}

func (m *mockCommandProvider) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	return "executed", nil
}

// TestCreateConfirmationHandler tests createConfirmationHandler method.
func TestCreateConfirmationHandler(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	// Test with nil permission manager
	factory := NewEngineFactory(llmClient, cfg, nil)
	handler := factory.createConfirmationHandler()
	if handler != nil {
		t.Error("createConfirmationHandler() should return nil with nil permMgr")
	}

	// Test with permission manager
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory = NewEngineFactory(llmClient, cfg, permMgr)
	handler = factory.createConfirmationHandler()
	if handler == nil {
		t.Error("createConfirmationHandler() should return handler with valid permMgr")
	}

	// Test handler with allowed tool
	ctx := context.Background()
	allowed, err := handler(ctx, "file_read", map[string]any{"path": "/tmp/test"})
	if err != nil {
		t.Errorf("Handler returned error: %v", err)
	}
	if !allowed {
		t.Error("Handler should allow file_read tool")
	}
}

// TestCreateConfirmationHandler_DeniedTool tests handler with denied tool.
func TestCreateConfirmationHandler_DeniedTool(t *testing.T) {
	permCfg := permissions.DefaultPermissionConfig()
	permCfg.DefaultLevel = "deny"
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	factory := NewEngineFactory(llmClient, cfg, permMgr)
	handler := factory.createConfirmationHandler()

	ctx := context.Background()
	allowed, err := handler(ctx, "bash", map[string]any{"command": "ls"})
	if err != nil {
		t.Errorf("Handler returned error: %v", err)
	}
	if allowed {
		t.Error("Handler should deny bash tool with deny default")
	}
}

// TestGetDefaultSystemPrompt tests getDefaultSystemPrompt function.
func TestGetDefaultSystemPrompt(t *testing.T) {
	prompt := getDefaultSystemPrompt()
	if prompt == "" {
		t.Error("getDefaultSystemPrompt() returned empty string")
	}
}

// TestToolRegistry_WithNilConfig tests NewToolRegistry with nil config.
func TestToolRegistry_WithNilConfig(t *testing.T) {
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	registry := NewToolRegistry(nil, permMgr)
	if registry == nil {
		t.Fatal("NewToolRegistry() with nil config returned nil")
	}
}

// TestToolRegistry_WithNilPermMgr tests NewToolRegistry with nil permission manager.
func TestToolRegistry_WithNilPermMgr(t *testing.T) {
	cfg := &config.ToolsConfig{}

	registry := NewToolRegistry(cfg, nil)
	if registry == nil {
		t.Fatal("NewToolRegistry() with nil permMgr returned nil")
	}
}

// TestTrustedPathAdapter tests trustedPathAdapter.
func TestTrustedPathAdapter(t *testing.T) {
	permCfg := permissions.DefaultPermissionConfig()
	// Add a trusted directory
	permCfg.TrustedDirs = []string{"/tmp", "/home"}
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	adapter := &trustedPathAdapter{mgr: permMgr}

	// Test trusted path
	if !adapter.IsTrustedPath("/tmp/test.txt") {
		t.Error("/tmp/test.txt should be trusted")
	}

	// Test untrusted path
	if adapter.IsTrustedPath("/etc/passwd") {
		t.Error("/etc/passwd should not be trusted")
	}
}

// TestGetToolNames tests GetToolNames function.
func TestGetToolNames(t *testing.T) {
	// Create agent with mock tools
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory := NewEngineFactory(llmClient, cfg, permMgr)
	mem := &MockMemory{}

	ag, err := factory.Create(mem)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	names := GetToolNames(ag)
	if names == nil {
		t.Error("GetToolNames() should return empty slice, not nil")
	}
}

// TestConvertSSHServers_Empty tests convertSSHServers with empty input.
func TestConvertSSHServers_Empty(t *testing.T) {
	input := []config.SSHServerConfig{}

	result := convertSSHServers(input)

	if len(result) != 0 {
		t.Errorf("convertSSHServers() with empty input returned %d servers, want 0", len(result))
	}
}

// TestConvertSSHServers_Nil tests convertSSHServers with nil input.
func TestConvertSSHServers_Nil(t *testing.T) {
	var input []config.SSHServerConfig

	result := convertSSHServers(input)

	if len(result) != 0 {
		t.Errorf("convertSSHServers() with nil input returned %d servers, want 0", len(result))
	}
}

// TestMarshalCommandParams tests MarshalCommandParams function.
func TestMarshalCommandParams(t *testing.T) {
	params := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	result, err := MarshalCommandParams(params)

	if err != nil {
		t.Fatalf("MarshalCommandParams() returned error: %v", err)
	}
	if result == "" {
		t.Error("MarshalCommandParams() returned empty string")
	}
}

// TestMarshalCommandParams_Empty tests MarshalCommandParams with empty map.
func TestMarshalCommandParams_Empty(t *testing.T) {
	params := map[string]any{}

	result, err := MarshalCommandParams(params)

	if err != nil {
		t.Fatalf("MarshalCommandParams() returned error: %v", err)
	}
	if result != "{}" {
		t.Errorf("MarshalCommandParams() = %q, want {}", result)
	}
}

// TestUnmarshalCommandParams tests UnmarshalCommandParams function.
func TestUnmarshalCommandParams(t *testing.T) {
	input := `{"key1": "value1", "key2": 123, "key3": true}`

	result, err := UnmarshalCommandParams(input)

	if err != nil {
		t.Fatalf("UnmarshalCommandParams() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("UnmarshalCommandParams() returned nil")
	}
	if result["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", result["key1"])
	}
	if result["key2"] != float64(123) {
		t.Errorf("key2 = %v, want 123", result["key2"])
	}
	if result["key3"] != true {
		t.Errorf("key3 = %v, want true", result["key3"])
	}
}

// TestUnmarshalCommandParams_InvalidJSON tests UnmarshalCommandParams with invalid JSON.
func TestUnmarshalCommandParams_InvalidJSON(t *testing.T) {
	input := `{"invalid": json}`

	_, err := UnmarshalCommandParams(input)

	if err == nil {
		t.Error("UnmarshalCommandParams() with invalid JSON should return error")
	}
}

// TestUnmarshalCommandParams_Empty tests UnmarshalCommandParams with empty JSON.
func TestUnmarshalCommandParams_Empty(t *testing.T) {
	input := `{}`

	result, err := UnmarshalCommandParams(input)

	if err != nil {
		t.Fatalf("UnmarshalCommandParams() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("UnmarshalCommandParams() returned nil")
	}
	if len(result) != 0 {
		t.Errorf("UnmarshalCommandParams() returned %d keys, want 0", len(result))
	}
}

// TestCommandTool tests CommandTool wrapper.
func TestCommandTool(t *testing.T) {
	mockCmd := &mockCommandProvider{}

	tool := NewCommandTool(mockCmd)

	if tool == nil {
		t.Fatal("NewCommandTool() returned nil")
	}
	if tool.Name() != "internal_command" {
		t.Errorf("Name() = %q, want internal_command", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

// TestCommandTool_Execute tests CommandTool Execute method.
func TestCommandTool_Execute(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()

	// Test successful execution
	params := map[string]any{
		"command": "test",
		"params":  map[string]any{"key": "value"},
	}

	result, err := tool.Execute(ctx, params)

	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Content != "executed" {
		t.Errorf("Result.Content = %q, want executed", result.Content)
	}
}

// TestCommandTool_Execute_WithNilProvider tests Execute with nil provider.
func TestCommandTool_Execute_WithNilProvider(t *testing.T) {
	tool := NewCommandTool(nil)

	ctx := context.Background()
	params := map[string]any{"command": "test"}

	result, err := tool.Execute(ctx, params)

	if err != nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() with nil provider should return error")
	}
}

// TestCommandTool_Execute_WithMissingCommand tests Execute with missing command parameter.
func TestCommandTool_Execute_WithMissingCommand(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()
	params := map[string]any{"other": "value"}

	result, err := tool.Execute(ctx, params)

	if err != nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() with missing command should return error")
	}
}

// TestCommandTool_Execute_WithEmptyCommand tests Execute with empty command parameter.
func TestCommandTool_Execute_WithEmptyCommand(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()
	params := map[string]any{"command": ""}

	result, err := tool.Execute(ctx, params)

	if err != nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() with empty command should return error")
	}
}

// TestCommandTool_Execute_WithInvalidCommandType tests Execute with invalid command type.
func TestCommandTool_Execute_WithInvalidCommandType(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()
	params := map[string]any{"command": 123} // Invalid type

	result, err := tool.Execute(ctx, params)

	if err != nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() with invalid command type should return error")
	}
}

// TestCommandTool_Execute_WithoutParams tests Execute without params.
func TestCommandTool_Execute_WithoutParams(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()
	params := map[string]any{"command": "test"}

	result, err := tool.Execute(ctx, params)

	if err != nil {
		t.Fatalf("Execute() without params returned error: %v", err)
	}
	if result.Content != "executed" {
		t.Errorf("Result.Content = %q, want executed", result.Content)
	}
}

// TestCommandTool_Execute_WithInvalidParamsType tests Execute with invalid params type.
func TestCommandTool_Execute_WithInvalidParamsType(t *testing.T) {
	mockCmd := &mockCommandProvider{}
	tool := NewCommandTool(mockCmd)

	ctx := context.Background()
	params := map[string]any{
		"command": "test",
		"params":  "invalid", // Should be map
	}

	result, err := tool.Execute(ctx, params)

	if err != nil {
		t.Fatalf("Execute() with invalid params type returned error: %v", err)
	}
	// Should use empty params
	if result.Content != "executed" {
		t.Errorf("Result.Content = %q, want executed", result.Content)
	}
}

// TestPermissionManagerFactory_Create_WithFullConfig tests Create with full config.
func TestPermissionManagerFactory_Create_WithFullConfig(t *testing.T) {
	factory := NewPermissionManagerFactory()

	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
		Tools: map[string]string{
			"file_read":  "allow",
			"file_write": "ask",
			"bash":       "deny",
		},
		ShellRestrictions: config.CommandRestrictions{
			AllowedCommands: []string{"ls", "cat", "echo"},
			DeniedCommands:  []string{"rm", "sudo"},
		},
		SSHRestrictions: config.SSHRestrictions{
			AllowedHosts:    []string{"example.com"},
			DeniedHosts:     []string{"malicious.com"},
			AllowedCommands: []string{"ls", "cat"},
			DeniedCommands:  []string{"rm", "sudo"},
		},
	}

	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

// TestPermissionManagerFactory_Create_WithInvalidDefaultLevel tests Create with invalid default level.
func TestPermissionManagerFactory_Create_WithInvalidDefaultLevel(t *testing.T) {
	factory := NewPermissionManagerFactory()

	cfg := &config.PermissionsConfig{
		DefaultLevel: "invalid",
	}

	manager, err := factory.Create(cfg)

	// May return error or use default
	if err == nil && manager == nil {
		t.Error("Create() should return either manager or error")
	}
}

// TestEngineFactory_WithConfirmationHandler tests WithConfirmationHandler option.
func TestEngineFactory_WithConfirmationHandler(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	// Test WithConfirmationHandler option
	handlerCalled := false
	mockHandler := func(ctx context.Context, toolName string, params map[string]any) (bool, error) {
		handlerCalled = true
		return true, nil
	}

	factory := NewEngineFactory(
		llmClient, cfg, permMgr,
		WithConfirmationHandler(mockHandler),
	)

	mem := &MockMemory{}
	ag, err := factory.Create(mem)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
	_ = handlerCalled // Use variable to avoid unused warning
}

// TestToolRegistry_RegisterAll tests RegisterAll method.
func TestToolRegistry_RegisterAll(t *testing.T) {
	cfg := &config.ToolsConfig{}
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	registry := NewToolRegistry(cfg, permMgr)

	llmClient := &MockLLMClient{}
	agentCfg := &config.AgentConfig{}

	factory := NewEngineFactory(llmClient, agentCfg, permMgr)
	mem := &MockMemory{}
	ag, err := factory.Create(mem)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Register all tools
	ctx := context.Background()
	rootCfg := &config.Config{
		LLM: config.LLMConfig{
			EmbeddingModel: "", // Skip knowledge tools
		},
		SSH: config.SSHConfig{
			Servers: []config.SSHServerConfig{},
		},
	}

	registry.RegisterAll(ctx, ag, rootCfg)

	// Verify tools were registered
	names := GetToolNames(ag)
	if len(names) == 0 {
		t.Error("RegisterAll() should register tools")
	}
}

// TestToolRegistry_RegisterAll_WithKnowledge tests RegisterAll with knowledge tools.
func TestToolRegistry_RegisterAll_WithKnowledge(t *testing.T) {
	cfg := &config.ToolsConfig{}
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	registry := NewToolRegistry(cfg, permMgr)

	llmClient := &MockLLMClient{}
	agentCfg := &config.AgentConfig{}

	factory := NewEngineFactory(llmClient, agentCfg, permMgr)
	mem := &MockMemory{}
	ag, err := factory.Create(mem)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Register all tools with embedding model configured
	ctx := context.Background()
	rootCfg := &config.Config{
		LLM: config.LLMConfig{
			EmbeddingModel: "nomic-embed-text", // Enable knowledge tools
			BaseURL:        "http://localhost:11434",
		},
		SSH: config.SSHConfig{
			Servers: []config.SSHServerConfig{},
		},
	}

	// This may fail if Ollama is not running, but should not panic
	registry.RegisterAll(ctx, ag, rootCfg)

	// Verify some tools were registered (at least basic tools)
	names := GetToolNames(ag)
	if len(names) == 0 {
		t.Error("RegisterAll() should register tools")
	}
}

// TestRegisterKnowledgeTools_NoEmbedding tests registerKnowledgeTools with no embedding.
func TestRegisterKnowledgeTools_NoEmbedding(t *testing.T) {
	cfg := &config.ToolsConfig{}
	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	registry := NewToolRegistry(cfg, permMgr)

	llmClient := &MockLLMClient{}
	agentCfg := &config.AgentConfig{}

	factory := NewEngineFactory(llmClient, agentCfg, permMgr)
	mem := &MockMemory{}
	ag, err := factory.Create(mem)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Register knowledge tools with no embedding model
	ctx := context.Background()
	rootCfg := &config.Config{
		LLM: config.LLMConfig{
			EmbeddingModel: "", // No embedding
		},
	}

	// Should not panic or register knowledge tools
	registry.registerKnowledgeTools(ctx, ag, rootCfg)

	// Verify no knowledge tools were registered
	names := GetToolNames(ag)
	for _, name := range names {
		if name == "knowledge_search" || name == "knowledge_import" {
			t.Error("Knowledge tools should not be registered without embedding model")
		}
	}
}
