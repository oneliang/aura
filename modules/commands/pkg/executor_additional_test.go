// Package commands provides additional tests for the executor.go file.
package commands

import (
	"context"
	"os"
	"testing"

	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/skill/pkg/loader"
)

// TestNewCommandProvider_NilDeps tests NewCommandProvider with nil deps.
func TestNewCommandProvider_NilDeps(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	if provider == nil {
		t.Fatal("NewCommandProvider() returned nil")
	}
	if provider.sessionHandler == nil {
		t.Error("sessionHandler should not be nil")
	}
	if provider.profileHandler == nil {
		t.Error("profileHandler should not be nil")
	}
	if provider.configHandler == nil {
		t.Error("configHandler should not be nil")
	}
	if provider.knowledgeHandler == nil {
		t.Error("knowledgeHandler should not be nil")
	}
	if provider.subscriptionHandler == nil {
		t.Error("subscriptionHandler should not be nil")
	}
}

// TestNewCommandProvider_WithConfigPath tests NewCommandProvider with custom config path.
func TestNewCommandProvider_WithConfigPath(t *testing.T) {
	deps := CommandProviderDeps{
		ConfigPath: "/custom/path/config.yaml",
	}
	provider := NewCommandProvider(deps)

	if provider == nil {
		t.Fatal("NewCommandProvider() returned nil")
	}
}

// TestNewCommandProvider_WithSkills tests NewCommandProvider with no skills.
func TestNewCommandProvider_WithSkills(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	if provider == nil {
		t.Fatal("NewCommandProvider() returned nil")
	}
	// skillHandler is nil when no SkillManager provided
	if provider.skillHandler != nil {
		t.Error("skillHandler should be nil when no SkillManager is provided")
	}
}

// TestNewCommandProvider_WithSkillLoader tests NewCommandProvider with skill loader.
func TestNewCommandProvider_WithSkillLoader(t *testing.T) {
	deps := CommandProviderDeps{
		SkillLoader: &loader.Loader{},
	}
	provider := NewCommandProvider(deps)

	if provider == nil {
		t.Fatal("NewCommandProvider() returned nil")
	}
	if provider.skillCommand == nil {
		t.Error("skillCommand should not be nil when SkillLoader is provided")
	}
}

// TestCommandProvider_GetCommands tests GetCommands method.
func TestCommandProvider_GetCommands(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	cmds := provider.GetCommands()
	if len(cmds) == 0 {
		t.Error("GetCommands() should return at least some commands")
	}

	foundSessionCreate := false
	for _, cmd := range cmds {
		if cmd.Name == CmdNameSessionCreate {
			foundSessionCreate = true
			break
		}
	}
	if !foundSessionCreate {
		t.Errorf("GetCommands() should include %s", CmdNameSessionCreate)
	}
}

// TestCommandProvider_GetCommands_WithSkillCommand tests GetCommands with skill commands.
func TestCommandProvider_GetCommands_WithSkillCommand(t *testing.T) {
	deps := CommandProviderDeps{
		SkillLoader: &loader.Loader{},
	}
	provider := NewCommandProvider(deps)

	cmds := provider.GetCommands()
	if len(cmds) == 0 {
		t.Error("GetCommands() should return commands")
	}
}

// TestCommandProvider_Execute_Help tests Execute help command.
func TestCommandProvider_Execute_Help(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameHelp, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameHelp, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return help text", CmdNameHelp)
	}
	if !containsStr(got, "Available commands") {
		t.Errorf("Execute(%s) should contain 'Available commands'", CmdNameHelp)
	}
}

// TestCommandProvider_Execute_Tools tests Execute tools command.
func TestCommandProvider_Execute_Tools(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameTools, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameTools, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return tools list", CmdNameTools)
	}
	if !containsStr(got, "File System") {
		t.Errorf("Execute(%s) should contain 'File System'", CmdNameTools)
	}
	if !containsStr(got, "System") {
		t.Errorf("Execute(%s) should contain 'System'", CmdNameTools)
	}
}

// TestCommandProvider_Execute_UnknownCommand tests Execute with unknown command.
func TestCommandProvider_Execute_UnknownCommand(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	_, err := provider.Execute(ctx, "unknown_command_xyz", nil)
	if err == nil {
		t.Error("Execute(unknown_command_xyz) should return error")
	}
}

// TestCommandProvider_Execute_SessionCommands tests Execute with session commands.
func TestCommandProvider_Execute_SessionCommands(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	deps := CommandProviderDeps{
		SessionMgr: sessionMgr,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Use command_session_list directly
	got, err := provider.Execute(ctx, CmdPrefixSession+"_list", nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdPrefixSession+"_list", err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdPrefixSession+"_list")
	}

	got, err = provider.Execute(ctx, CmdNameSessionCreate, map[string]any{
		"name": "Test Session",
	})
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameSessionCreate, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdNameSessionCreate)
	}
}

// TestCommandProvider_Execute_ProfileCommands tests Execute with profile commands.
func TestCommandProvider_Execute_ProfileCommands(t *testing.T) {
	p := &profile.Profile{
		BasicInfo: profile.BasicInfo{
			Name: "Test User",
		},
	}
	deps := CommandProviderDeps{
		Profile: p,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameProfileShow, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameProfileShow, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdNameProfileShow)
	}
}

// TestCommandProvider_Execute_ConfigCommands tests Execute with config commands.
func TestCommandProvider_Execute_ConfigCommands(t *testing.T) {
	// Create a temporary config file with some content
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"
	configContent := `llm:
  provider: ollama
  model: qwen3:8b
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	deps := CommandProviderDeps{
		ConfigPath: configPath,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Test config show - will work with empty file
	got, err := provider.Execute(ctx, CmdNameConfigShow, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameConfigShow, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdNameConfigShow)
	}
}

// TestCommandProvider_Execute_KnowledgeCommands tests Execute with knowledge commands.
func TestCommandProvider_Execute_KnowledgeCommands(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	deps := CommandProviderDeps{
		Config: cfg,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	_, err := provider.Execute(ctx, CmdNameKnowledgeSearch, map[string]any{})
	if err == nil {
		t.Errorf("Execute(%s) without query should return error", CmdNameKnowledgeSearch)
	}
}

// TestCommandProvider_Execute_SubscriptionCommands tests Execute with subscription commands.
func TestCommandProvider_Execute_SubscriptionCommands(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	deps := CommandProviderDeps{
		SessionMgr: sessionMgr,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameSubscriptionShow, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameSubscriptionShow, err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdNameSubscriptionShow)
	}
}

// TestCommandProvider_Execute_SkillsCommands tests Execute with skills commands.
func TestCommandProvider_Execute_SkillsCommands(t *testing.T) {
	// Skills require SkillLoader; without it, skills commands return "Skills not available"
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameSkills, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameSkills, err)
	}
	if got != "Skills not available" {
		t.Errorf("Execute(%s) should return 'Skills not available' without SkillLoader, got: %s", CmdNameSkills, got)
	}
}

// TestCommandProvider_Execute_SkillsNotAvailable tests Execute skills when not available.
func TestCommandProvider_Execute_SkillsNotAvailable(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	got, err := provider.Execute(ctx, CmdNameSkills, nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdNameSkills, err)
	}
	if !containsStr(got, "Skills not available") {
		t.Errorf("Execute(%s) should indicate skills not available", CmdNameSkills)
	}
}

// TestCommandProvider_Execute_SkillCommandNotFound tests Execute with skill command not found.
func TestCommandProvider_Execute_SkillCommandNotFound(t *testing.T) {
	deps := CommandProviderDeps{
		SkillLoader: &loader.Loader{},
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	_, err := provider.Execute(ctx, "non_existent_skill", nil)
	if err == nil {
		t.Error("Execute(non_existent_skill) should return error")
	}
}

// TestCommandProvider_Execute_SessionCommandRouting tests session command routing.
func TestCommandProvider_Execute_SessionCommandRouting(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	deps := CommandProviderDeps{
		SessionMgr: sessionMgr,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Test that command_session_ routes to list (empty string after prefix becomes "list")
	got, err := provider.Execute(ctx, CmdPrefixSession+"_", nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdPrefixSession+"_", err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdPrefixSession+"_")
	}
}

// TestCommandProvider_Execute_ProfileCommandRouting tests profile command routing.
func TestCommandProvider_Execute_ProfileCommandRouting(t *testing.T) {
	p := &profile.Profile{
		BasicInfo: profile.BasicInfo{
			Name: "Test User",
		},
	}
	deps := CommandProviderDeps{
		Profile: p,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Test that command_profile_ routes to show (empty string after prefix becomes "show")
	got, err := provider.Execute(ctx, CmdPrefixProfile+"_", nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdPrefixProfile+"_", err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdPrefixProfile+"_")
	}
}

// TestCommandProvider_Execute_ConfigCommandRouting tests config command routing.
func TestCommandProvider_Execute_ConfigCommandRouting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"
	configContent := `llm:
  provider: ollama
`
	// Create an empty config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	deps := CommandProviderDeps{
		ConfigPath: configPath,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Test that command_config_ routes to show (empty string after prefix becomes "show")
	got, err := provider.Execute(ctx, CmdPrefixConfig+"_", nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdPrefixConfig+"_", err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdPrefixConfig+"_")
	}
}

// TestCommandProvider_Execute_KnowledgeCommandRouting tests knowledge command routing.
func TestCommandProvider_Execute_KnowledgeCommandRouting(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	deps := CommandProviderDeps{
		Config: cfg,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	_, err := provider.Execute(ctx, CmdPrefixKnowledge+"_", map[string]any{})
	if err == nil {
		t.Errorf("Execute(%s) without query should return error", CmdPrefixKnowledge+"_")
	}

	_, err = provider.Execute(ctx, CmdNameKnowledge, map[string]any{})
	if err == nil {
		t.Errorf("Execute(%s) without query should return error", CmdNameKnowledge)
	}
}

// TestCommandProvider_Execute_SubscriptionCommandRouting tests subscription command routing.
func TestCommandProvider_Execute_SubscriptionCommandRouting(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	deps := CommandProviderDeps{
		SessionMgr: sessionMgr,
	}
	provider := NewCommandProvider(deps)
	ctx := context.Background()

	// Test that command_subscription_ routes to show (empty string after prefix becomes "show")
	got, err := provider.Execute(ctx, CmdPrefixSubscription+"_", nil)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", CmdPrefixSubscription+"_", err)
	}
	if got == "" {
		t.Errorf("Execute(%s) should return output", CmdPrefixSubscription+"_")
	}
}

// TestCommandProvider_SetAgentDelegateFn_LazyCreate tests that SetAgentDelegateFn
// creates the agentHandler when it was not initialized (AgentDelegateFunc was nil).
func TestCommandProvider_SetAgentDelegateFn_LazyCreate(t *testing.T) {
	// Create CommandProvider with nil AgentDelegateFunc (mimics CLI behavior)
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	// Verify delegation returns "not configured" before SetAgentDelegateFn
	result, err := provider.Execute(context.Background(), "command_agent_code-reviewer", map[string]any{
		"task": "review code",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "Agent delegation not configured" {
		t.Fatalf("expected 'Agent delegation not configured', got: %s", result)
	}

	// Now inject the delegate function
	called := false
	provider.SetAgentDelegateFn(func(ctx context.Context, agentName string, task string) (string, error) {
		called = true
		if agentName != "code-reviewer" {
			t.Errorf("expected agent 'code-reviewer', got '%s'", agentName)
		}
		if task != "review code" {
			t.Errorf("expected task 'review code', got '%s'", task)
		}
		return "delegated ok", nil
	})

	// Verify delegation now works using the new command_agent_* format
	result, err = provider.Execute(context.Background(), "command_agent_code-reviewer", map[string]any{
		"task": "review code",
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if !called {
		t.Fatal("delegate function was not called")
	}
	if result != "delegated ok" {
		t.Errorf("expected 'delegated ok', got '%s'", result)
	}
}

// TestCommandProvider_SetAgentDelegateFn_CommandToolIntegration tests the full
// internal_command → command_agent_* delegation path.
func TestCommandProvider_SetAgentDelegateFn_CommandToolIntegration(t *testing.T) {
	// Create CommandProvider
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	// Inject delegate function
	provider.SetAgentDelegateFn(func(ctx context.Context, agentName string, task string) (string, error) {
		return "SubAgent " + agentName + " completed: " + task, nil
	})

	// Simulate the full internal_command path: command_agent_test-agent
	// This mimics what happens when the engine routes through internal_command
	result, err := provider.Execute(context.Background(), "command_agent_test-agent", map[string]any{
		"task": "do something",
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result != "SubAgent test-agent completed: do something" {
		t.Errorf("expected 'SubAgent test-agent completed: do something', got '%s'", result)
	}
}

// TestCommandProvider_SetMCPListFunc_LazyCreate tests that SetMCPListFunc
// creates the mcpHandler when it was not initialized.
func TestCommandProvider_SetMCPListFunc_LazyCreate(t *testing.T) {
	deps := CommandProviderDeps{}
	provider := NewCommandProvider(deps)

	// Inject the list function
	provider.SetMCPListFunc(func() []MCPInfo {
		return []MCPInfo{{Name: "test-server", Command: "test", Status: "running"}}
	})

	// Verify MCP list works
	result, err := provider.Execute(context.Background(), CmdPrefixMcp+"_list", map[string]any{})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if !containsStr(result, "test-server") {
		t.Errorf("expected 'test-server' in result, got: %s", result)
	}
}

// Helper function to check if a string contains a substring.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
