package runtime

import (
	"context"
	"testing"
	"time"

	agentpkg "github.com/oneliang/aura/agent/pkg/agent"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	tools "github.com/oneliang/aura/tools/pkg"
)

// testToolAdapter adapts testMockTool to tools.Tool interface for filterToolsByNames tests.
type testToolAdapter struct {
	name string
}

func (a *testToolAdapter) Name() string        { return a.name }
func (a *testToolAdapter) Description() string { return "mock tool" }
func (a *testToolAdapter) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "result"}, nil
}

// TestBuildSubAgentConfig tests the buildSubAgentConfig function.
func TestBuildSubAgentConfig(t *testing.T) {
	parentCfg := config.DefaultConfig()
	parentCfg.Agent.PlanningMode = "implicit"
	parentCfg.Agent.Temperature = 0.7

	parent := &AgentRuntime{
		config: &RuntimeConfig{
			Config: parentCfg,
		},
	}

	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Meta: agentpkg.AgentMeta{
			Name:        "test-agent",
			Description: "Test agent",
		},
		Body: "You are a test agent.",
	}

	task := "Complete this test task"

	cfg := buildSubAgentConfig(parent, foundAgent, task)

	if cfg == nil {
		t.Fatal("buildSubAgentConfig() returned nil")
	}

	// Check system prompt includes agent body and task
	if cfg.SystemPrompt == "" {
		t.Error("buildSubAgentConfig() system prompt should not be empty")
	}

	// Check role is set correctly
	if cfg.Role != "sub-agent-test-agent" {
		t.Errorf("buildSubAgentConfig() role = %q, want %q", cfg.Role, "sub-agent-test-agent")
	}

	// Check DisableTools is false by default
	if cfg.DisableTools {
		t.Error("buildSubAgentConfig() DisableTools should be false")
	}
}

// TestBuildSubAgentConfig_ParentLLMInherited verifies that sub-agent inherits parent LLM config
// even when agent has an llm_model set (llm_model is not applied during delegation).
func TestBuildSubAgentConfig_ParentLLMInherited(t *testing.T) {
	parentCfg := config.DefaultConfig()
	parentCfg.LLM.Provider = "openai"
	parentCfg.LLM.BaseURL = "https://api.example.com/v1"
	parentCfg.LLM.Model = "parent-model"
	parentCfg.LLM.APIKey = "parent-key"

	parent := &AgentRuntime{
		config: &RuntimeConfig{
			Config: parentCfg,
		},
	}

	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Meta: agentpkg.AgentMeta{
			Name: "test-agent",
		},
		Body: "Test body",
	}
	// Agent sets llm_model, but it should NOT be applied during delegation
	foundAgent.Meta.LLMModel = "agent-model"

	cfg := buildSubAgentConfig(parent, foundAgent, "test task")

	if cfg == nil {
		t.Fatal("buildSubAgentConfig() returned nil")
	}

	// Check LLM Model is inherited from parent (NOT overridden by agent)
	if cfg.LLM.Model != "parent-model" {
		t.Errorf("buildSubAgentConfig() LLM.Model = %q, want %q (should inherit from parent)", cfg.LLM.Model, "parent-model")
	}
	// Check Provider is inherited from parent
	if cfg.LLM.Provider != "openai" {
		t.Errorf("buildSubAgentConfig() LLM.Provider = %q, want %q", cfg.LLM.Provider, "openai")
	}
	// Check BaseURL is inherited from parent
	if cfg.LLM.BaseURL != "https://api.example.com/v1" {
		t.Errorf("buildSubAgentConfig() LLM.BaseURL = %q, want %q", cfg.LLM.BaseURL, "https://api.example.com/v1")
	}
	// Check APIKey is inherited from parent
	if cfg.LLM.APIKey != "parent-key" {
		t.Errorf("buildSubAgentConfig() LLM.APIKey = %q, want %q", cfg.LLM.APIKey, "parent-key")
	}
}

// TestApplyAgentConfigInheritance tests the applyAgentConfigInheritance function.
func TestApplyAgentConfigInheritance(t *testing.T) {
	cfg := &RuntimeConfig{
		Config: &config.Config{
			Agent: config.AgentConfig{
				PlanningMode: "implicit",
				Temperature:  0.5,
				SummaryTemp:  0.3,
			},
		},
	}

	meta := &agentpkg.AgentMeta{}
	meta.PlanningMode = "explicit"
	meta.Temperature = 0.9
	meta.SummaryTemp = 0.5

	applyAgentConfigInheritance(cfg, meta)

	// Check PlanningMode is overridden
	if cfg.Agent.PlanningMode != "explicit" {
		t.Errorf("applyAgentConfigInheritance() PlanningMode = %q, want %q", cfg.Agent.PlanningMode, "explicit")
	}

	// Check Temperature is overridden
	if cfg.Agent.Temperature != 0.9 {
		t.Errorf("applyAgentConfigInheritance() Temperature = %f, want 0.9", cfg.Agent.Temperature)
	}

	// Check SummaryTemp is overridden
	if cfg.Agent.SummaryTemp != 0.5 {
		t.Errorf("applyAgentConfigInheritance() SummaryTemp = %f, want 0.5", cfg.Agent.SummaryTemp)
	}
}

// TestApplyAgentConfigInheritance_PartialOverride tests partial override.
func TestApplyAgentConfigInheritance_PartialOverride(t *testing.T) {
	cfg := &RuntimeConfig{
		Config: &config.Config{
			Agent: config.AgentConfig{
				PlanningMode: "implicit",
				Temperature:  0.5,
				SummaryTemp:  0.3,
			},
		},
	}

	meta := &agentpkg.AgentMeta{}
	// Other fields use zero values, should not override

	applyAgentConfigInheritance(cfg, meta)

}

// TestFilterDisabledTools tests the filterDisabledTools legacy function.
// Note: The original filterDisabledTools used []interface{}; the new
// filterToolsByNames uses []tools.Tool with map-based lookup.
// This test verifies the original behavior using the new function.
func TestFilterDisabledTools(t *testing.T) {
	// Create mock tools using the new tools.Tool interface
	allTools := []tools.Tool{
		&testToolAdapter{name: "file_read"},
		&testToolAdapter{name: "bash"},
		&testToolAdapter{name: "web_fetch"},
		&testToolAdapter{name: "knowledge_search"},
	}

	disabledTools := []string{"bash", "knowledge_search"}

	filtered := filterToolsByNames(allTools, disabledTools)

	if len(filtered) != 2 {
		t.Errorf("filterToolsByNames() returned %d tools, want 2", len(filtered))
	}

	// Check that the remaining tools are correct
	for _, tool := range filtered {
		name := tool.Name()
		if name == "bash" || name == "knowledge_search" {
			t.Errorf("filterToolsByNames() should have filtered out %s", name)
		}
	}
}

// TestFilterDisabledTools_EmptyDisableList tests with empty disable list.
func TestFilterDisabledTools_EmptyDisableList(t *testing.T) {
	allTools := []tools.Tool{
		&testToolAdapter{name: "file_read"},
		&testToolAdapter{name: "bash"},
	}

	filtered := filterToolsByNames(allTools, nil)

	if len(filtered) != 2 {
		t.Errorf("filterToolsByNames() with nil disable list returned %d tools, want 2", len(filtered))
	}
}

// TestFilterDisabledTools_EmptyTools tests with empty tools list.
func TestFilterDisabledTools_EmptyTools(t *testing.T) {
	filtered := filterToolsByNames(nil, []string{"bash"})

	if len(filtered) != 0 {
		t.Errorf("filterToolsByNames() with nil tools returned %d tools, want 0", len(filtered))
	}
}

// TestTimestamp tests the Timestamp helper function.
func TestTimestampDelegate(t *testing.T) {
	now := time.Now()
	event := NewEvent(EventTypeThinkingStart, "test content")

	ts := Timestamp(event)

	if ts.IsZero() {
		t.Error("Timestamp() returned zero time")
	}

	// The timestamp should be recent (within the last second)
	if ts.Sub(now) > time.Second {
		t.Errorf("Timestamp() = %v, expected recent time", ts)
	}
}

// TestTimestamp_EventWithExtra tests Timestamp with event that has extra data.
func TestTimestampDelegate_EventWithExtra(t *testing.T) {
	event := NewEventWithExtra(EventTypeResponse, "response", map[string]any{
		"key": "value",
	})

	ts := Timestamp(event)

	if ts.IsZero() {
		t.Error("Timestamp() returned zero time for event with extra")
	}
}

// TestFindAgent_NilLoader tests findAgent with nil loader.
func TestFindAgent_NilLoader(t *testing.T) {
	runtime := &AgentRuntime{
		agentLoader: nil,
	}

	_, err := runtime.findAgent("test-agent")

	if err == nil {
		t.Error("findAgent() with nil loader should return error")
	}
}

// TestSetAndGetAgentDelegateFn tests SetAgentDelegateFn and GetAgentDelegateFn.
func TestSetAndGetAgentDelegateFn(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	runtime, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Initially nil
	fn := runtime.GetAgentDelegateFn()
	if fn != nil {
		t.Error("GetAgentDelegateFn() should return nil initially")
	}

	// Set a delegate function
	delegateFn := func(ctx context.Context, agentName, task string) (string, error) {
		return "delegated result", nil
	}

	runtime.SetAgentDelegateFn(delegateFn)

	// Get the delegate function
	fn = runtime.GetAgentDelegateFn()
	if fn == nil {
		t.Fatal("GetAgentDelegateFn() should not return nil after SetAgentDelegateFn")
	}

	// Test the function works
	result, err := fn(context.Background(), "test-agent", "test task")
	if err != nil {
		t.Errorf("Delegate function returned error: %v", err)
	}
	if result != "delegated result" {
		t.Errorf("Delegate function result = %q, want %q", result, "delegated result")
	}
}

// testMockTool is a mock tool for testing delegate package.
type testMockTool struct {
	name string
}

func (m *testMockTool) Name() string {
	return m.name
}

func (m *testMockTool) Description() string {
	return "mock tool"
}

func (m *testMockTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "result"}, nil
}

func (m *testMockTool) RequiresConfirmation() bool {
	return false
}

// TestApplyAgentConfigInheritance_TemperatureZero tests that temperature=0 does NOT override.
func TestApplyAgentConfigInheritance_TemperatureZero(t *testing.T) {
	cfg := &RuntimeConfig{
		Config: &config.Config{
			Agent: config.AgentConfig{
				Temperature: 0.7,
			},
		},
	}
	meta := &agentpkg.AgentMeta{}
	// Temperature defaults to 0 (zero value)

	applyAgentConfigInheritance(cfg, meta)

	if cfg.Agent.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7 (zero should not override)", cfg.Agent.Temperature)
	}
}

// TestApplyAgentConfigInheritance_SummaryTempZero tests that summary_temp=0 does NOT override.
func TestApplyAgentConfigInheritance_SummaryTempZero(t *testing.T) {
	cfg := &RuntimeConfig{
		Config: &config.Config{
			Agent: config.AgentConfig{
				SummaryTemp: 0.5,
			},
		},
	}
	meta := &agentpkg.AgentMeta{}

	applyAgentConfigInheritance(cfg, meta)

	if cfg.Agent.SummaryTemp != 0.5 {
		t.Errorf("SummaryTemp = %f, want 0.5 (zero should not override)", cfg.Agent.SummaryTemp)
	}
}

// TestApplyAgentConfigInheritance_PlanningModeEmpty tests that empty planning_mode does NOT override.
func TestApplyAgentConfigInheritance_PlanningModeEmpty(t *testing.T) {
	cfg := &RuntimeConfig{
		Config: &config.Config{
			Agent: config.AgentConfig{
				PlanningMode: "explicit",
			},
		},
	}
	meta := &agentpkg.AgentMeta{}
	// PlanningMode defaults to "" (empty string)

	applyAgentConfigInheritance(cfg, meta)

	if cfg.Agent.PlanningMode != "explicit" {
		t.Errorf("PlanningMode = %q, want 'explicit' (empty should not override)", cfg.Agent.PlanningMode)
	}
}

// TestBuildSubAgentConfig_SystemPromptFormat verifies exact system prompt format.
func TestBuildSubAgentConfig_SystemPromptFormat(t *testing.T) {
	parent := &AgentRuntime{
		config: &RuntimeConfig{Config: config.DefaultConfig()},
	}
	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Body: "You are a tester.",
		Meta: agentpkg.AgentMeta{Name: "test-agent"},
	}

	cfg := buildSubAgentConfig(parent, foundAgent, "Fix the bug")

	expected := "You are a tester.\n\n## Task\n\nYou have been delegated the following task. Complete it and provide a clear, comprehensive result.\n\nFix the bug"
	if cfg.SystemPrompt != expected {
		t.Errorf("SystemPrompt format mismatch.\nGot:\n%s\nWant:\n%s", cfg.SystemPrompt, expected)
	}
}

// TestFilterDisabledTools_AllDisabled tests filtering when all tools are disabled.
func TestFilterDisabledTools_AllDisabled(t *testing.T) {
	parentTools := []tools.Tool{
		&testToolAdapter{name: "file_read"},
		&testToolAdapter{name: "bash"},
	}
	disabled := []string{"file_read", "bash"}
	filtered := filterToolsByNames(parentTools, disabled)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 tools when all disabled, got %d", len(filtered))
	}
}

// TestFilterDisabledTools_NoneDisabled tests no tools disabled.
func TestFilterDisabledTools_NoneDisabled(t *testing.T) {
	parentTools := []tools.Tool{
		&testToolAdapter{name: "file_read"},
		&testToolAdapter{name: "bash"},
		&testToolAdapter{name: "web_fetch"},
	}
	filtered := filterToolsByNames(parentTools, nil)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(filtered))
	}
}

// TestFilterDisabledTools_NonexistentTool tests disabling a tool that doesn't exist.
func TestFilterDisabledTools_NonexistentTool(t *testing.T) {
	parentTools := []tools.Tool{
		&testToolAdapter{name: "file_read"},
		&testToolAdapter{name: "bash"},
	}
	disabled := []string{"nonexistent_tool"}
	filtered := filterToolsByNames(parentTools, disabled)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 tools (non-existent disable should have no effect), got %d", len(filtered))
	}
}

// TestTruncateStr tests the logger truncateStr function (used by delegation logger).
func TestTruncateStr(t *testing.T) {
	if logger.TruncateStr("short", 100) != "short" {
		t.Error("Short string should not be truncated")
	}
	if got := logger.TruncateStr("this is a long string that exceeds the limit", 20); got != "this is a long strin..." {
		t.Errorf("TruncateStr() = %q, want %q", got, "this is a long strin...")
	}
	if logger.TruncateStr("", 10) != "" {
		t.Error("Empty string should remain empty")
	}
}

// TestBuildSubAgentConfig_PhaseBasedDowngrade tests phase-based permission auto-downgrade.
// Note: This test requires a real engine because AgentRuntime.agent is *engine.Engine (concrete type).
func TestBuildSubAgentConfig_PhaseBasedDowngrade_RequiresEngine(t *testing.T) {
	// This test documents the expected behavior:
	// When parent.agent is nil, permission mode defaults to inherit
	parent := &AgentRuntime{
		config: &RuntimeConfig{Config: config.DefaultConfig()},
		agent:  nil, // No parent engine
	}

	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Meta: agentpkg.AgentMeta{Name: "test-agent"},
	}

	cfg := buildSubAgentConfig(parent, foundAgent, "test task")

	// When parent.agent is nil, should use default inherit
	if cfg.PermissionMode != string(permissions.PermissionInherit) {
		t.Errorf("PermissionMode = %q, want %q (nil parent should use inherit)", cfg.PermissionMode, string(permissions.PermissionInherit))
	}
}

// TestBuildSubAgentConfig_ExplicitReadonly tests explicit readonly mode is respected.
func TestBuildSubAgentConfig_ExplicitReadonly(t *testing.T) {
	parent := &AgentRuntime{
		config: &RuntimeConfig{Config: config.DefaultConfig()},
		agent:  nil,
	}

	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Meta: agentpkg.AgentMeta{
			Name:           "test-agent",
			PermissionMode: string(permissions.PermissionReadonly),
		},
	}

	cfg := buildSubAgentConfig(parent, foundAgent, "test task")

	// Explicit readonly should be respected
	if cfg.PermissionMode != string(permissions.PermissionReadonly) {
		t.Errorf("PermissionMode = %q, want %q", cfg.PermissionMode, string(permissions.PermissionReadonly))
	}
}

// TestBuildSubAgentConfig_IndependentNotDowngradedWhenNil tests independent with nil parent.
func TestBuildSubAgentConfig_IndependentNotDowngradedWhenNil(t *testing.T) {
	parent := &AgentRuntime{
		config: &RuntimeConfig{Config: config.DefaultConfig()},
		agent:  nil,
	}

	foundAgent := &agentpkg.Agent{
		Name: "test-agent",
		Meta: agentpkg.AgentMeta{
			Name:           "test-agent",
			PermissionMode: string(permissions.PermissionIndependent),
		},
	}

	cfg := buildSubAgentConfig(parent, foundAgent, "test task")

	// Independent mode should be respected even with nil parent
	if cfg.PermissionMode != string(permissions.PermissionIndependent) {
		t.Errorf("PermissionMode = %q, want %q", cfg.PermissionMode, string(permissions.PermissionIndependent))
	}
}
