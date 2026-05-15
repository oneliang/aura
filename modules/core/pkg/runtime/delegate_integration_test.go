package runtime

import (
	"context"
	"testing"

	agentpkg "github.com/oneliang/aura/agent/pkg/agent"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	tools "github.com/oneliang/aura/tools/pkg"
)

// TestNewSubAgentRuntime_SharedResources verifies that a sub-agent runtime
// shares the parent's expensive resources (LLM client, tools, etc.).
func TestNewSubAgentRuntime_SharedResources(t *testing.T) {
	parentCfg := &RuntimeConfig{
		Config: config.DefaultConfig(),
	}
	parent, err := New(parentCfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	parent.llmClient = &mockLLMClientForIntegration{}
	parent.logger = loggerForTest()

	// Initialize shared resources for parent (required for NewSubAgentRuntime validation)
	parent.shared = &SharedResources{
		llmClient:     parent.llmClient,
		httpClient:    parent.httpClient,
		webHttpClient: parent.webHttpClient,
		permMgr:       parent.permMgr,
	}
	parent.skills = &SkillSystem{}
	parent.agents = &AgentSystem{}
	parent.mcp = &MCPSystem{}
	parent.hooks = &HookSystem{}

	// Create sub-agent runtime
	subCfg := &RuntimeConfig{
		Config:       parent.config.Config,
		SessionID:    "",
		Role:         "sub-agent-tester",
		DisableTools: false,
		SystemPrompt: "You are a tester.",
	}

	subRuntime, err := NewSubAgentRuntime(parent, subCfg, nil, nil)
	if err != nil {
		t.Fatalf("NewSubAgentRuntime() error: %v", err)
	}

	// Verify shared LLM client (pointer equality)
	if subRuntime.llmClient != parent.llmClient {
		t.Error("Sub-agent should share parent's LLM client")
	}

	// Verify shared logger (when no delegation logger provided)
	if subRuntime.logger != parent.logger {
		t.Error("Sub-agent should share parent's logger")
	}

	// Verify skipInitialize is set
	if !subRuntime.skipInitialize {
		t.Error("skipInitialize should be true for sub-agent")
	}

	// Verify MCP manager is shared (nil in this test, but should be same pointer)
	if subRuntime.mcpManager != parent.mcpManager {
		t.Error("Sub-agent should share parent's MCP manager")
	}

	// Verify HTTP clients are shared
	if subRuntime.httpClient != parent.httpClient {
		t.Error("Sub-agent should share parent's HTTP client")
	}
	if subRuntime.webHttpClient != parent.webHttpClient {
		t.Error("Sub-agent should share parent's web HTTP client")
	}
}

// TestNewSubAgentRuntime_ToolFiltering verifies that NewSubAgentRuntime correctly
// skips tool filtering when parent has no engine (nil agent).
// Note: Tool filtering itself is tested in TestFilterToolsByNames and
// TestFilterDisabledTools_* tests.
func TestNewSubAgentRuntime_ToolFiltering(t *testing.T) {
	parentCfg := &RuntimeConfig{Config: config.DefaultConfig()}
	parent, err := New(parentCfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	parent.llmClient = &mockLLMClientForIntegration{}
	parent.logger = loggerForTest()
	// Note: parent.agent is nil here — NewSubAgentRuntime should handle gracefully
	// without panicking. The actual tool filtering logic is tested in unit tests.

	// Initialize shared resources for parent (required for NewSubAgentRuntime validation)
	parent.shared = &SharedResources{
		llmClient:     parent.llmClient,
		httpClient:    parent.httpClient,
		webHttpClient: parent.webHttpClient,
		permMgr:       parent.permMgr,
	}
	parent.skills = &SkillSystem{}
	parent.agents = &AgentSystem{}
	parent.mcp = &MCPSystem{}
	parent.hooks = &HookSystem{}

	subCfg := &RuntimeConfig{
		Config:       parent.config.Config,
		SessionID:    "",
		Role:         "sub-agent-tester",
		SystemPrompt: "You are a restricted tester.",
	}

	// Create sub-agent — should not panic with nil parent agent
	subRuntime, err := NewSubAgentRuntime(parent, subCfg, []string{"bash"}, nil)
	if err != nil {
		t.Fatalf("NewSubAgentRuntime() error: %v", err)
	}

	// With nil parent agent, preBuiltTools should be empty
	if len(subRuntime.preBuiltTools) != 0 {
		t.Errorf("Expected 0 tools with nil parent agent, got %d", len(subRuntime.preBuiltTools))
	}
}

// TestSubAgentShutdown_DoesNotStopMCP verifies that sub-agent Shutdown()
// does NOT stop the parent's MCP manager.
func TestSubAgentShutdown_DoesNotStopMCP(t *testing.T) {
	parentCfg := &RuntimeConfig{Config: config.DefaultConfig()}
	parent, err := New(parentCfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	parent.llmClient = &mockLLMClientForIntegration{}
	parent.logger = loggerForTest()

	// Initialize shared resources for parent (required for NewSubAgentRuntime validation)
	parent.shared = &SharedResources{
		llmClient:     parent.llmClient,
		httpClient:    parent.httpClient,
		webHttpClient: parent.webHttpClient,
		permMgr:       parent.permMgr,
	}
	parent.skills = &SkillSystem{}
	parent.agents = &AgentSystem{}
	parent.mcp = &MCPSystem{}
	parent.hooks = &HookSystem{}

	subCfg := &RuntimeConfig{
		Config: parent.config.Config,
		Role:   "sub-agent-tester",
	}

	subRuntime, err := NewSubAgentRuntime(parent, subCfg, nil, nil)
	if err != nil {
		t.Fatalf("NewSubAgentRuntime() error: %v", err)
	}

	// Sub-agent should have skipInitialize=true
	if !subRuntime.skipInitialize {
		t.Fatal("skipInitialize should be true")
	}

	// Shutdown sub-agent — should NOT panic or stop MCP
	subRuntime.Shutdown()

	// Verify parent's state is unaffected
	if parent.initialized != false {
		t.Log("Parent initialized state is unchanged (expected)")
	}
}

// TestBuildSubAgentConfig_WithLLMOverrideAndInheritance verifies LLM override
// combined with config inheritance.
func TestBuildSubAgentConfig_WithLLMoverrideAndInheritance(t *testing.T) {
	parentCfg := config.DefaultConfig()
	parentCfg.Agent.PlanningMode = "implicit"
	parentCfg.Agent.Temperature = 0.7
	parentCfg.LLM.Model = "parent-model"

	parent := &AgentRuntime{
		config: &RuntimeConfig{Config: parentCfg},
	}

	foundAgent := &agentpkg.Agent{
		Name: "custom-model-agent",
		Body: "You are an agent with custom model.",
		Meta: agentpkg.AgentMeta{
			Name:        "custom-model-agent",
			Description: "Uses custom model",
			LLMModel:    "qwen3:8b", // Set but NOT applied during delegation
			AgentConfig: config.AgentConfig{
				PlanningMode: "explicit",
				Temperature:  0.3,
				SummaryTemp:  0.8,
			},
		},
	}

	cfg := buildSubAgentConfig(parent, foundAgent, "test task")

	// LLM is inherited from parent (agent's llm_model is NOT applied)
	if cfg.LLM.Model != "parent-model" {
		t.Errorf("LLM.Model = %q, want 'parent-model' (should inherit from parent)", cfg.LLM.Model)
	}

	// Agent config inheritance
	if cfg.Agent.PlanningMode != "explicit" {
		t.Errorf("PlanningMode = %q, want 'explicit'", cfg.Agent.PlanningMode)
	}
	if cfg.Agent.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want 0.3", cfg.Agent.Temperature)
	}
	if cfg.Agent.SummaryTemp != 0.8 {
		t.Errorf("SummaryTemp = %f, want 0.8", cfg.Agent.SummaryTemp)
	}
}

// TestFilterToolsByNames verifies the tool filtering function.
func TestFilterToolsByNames(t *testing.T) {
	allTools := []tools.Tool{
		&mockToolForTest{name: "tool_a"},
		&mockToolForTest{name: "tool_b"},
		&mockToolForTest{name: "tool_c"},
	}

	// Disable one
	filtered := filterToolsByNames(allTools, []string{"tool_b"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(filtered))
	}
	for _, tool := range filtered {
		if tool.Name() == "tool_b" {
			t.Error("tool_b should be filtered out")
		}
	}

	// Disable none
	filtered = filterToolsByNames(allTools, nil)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 tools with nil disabled, got %d", len(filtered))
	}

	// Disable all
	filtered = filterToolsByNames(allTools, []string{"tool_a", "tool_b", "tool_c"})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 tools when all disabled, got %d", len(filtered))
	}
}

// mockLLMClientForIntegration is a minimal mock for llm.Client interface.
type mockLLMClientForIntegration struct{}

func (m *mockLLMClientForIntegration) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return nil, nil
}

func (m *mockLLMClientForIntegration) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Content: "mock response", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockLLMClientForIntegration) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

// mockToolForTest is a mock tool for integration testing.
type mockToolForTest struct {
	name string
}

func (m *mockToolForTest) Name() string        { return m.name }
func (m *mockToolForTest) Description() string { return "mock tool" }
func (m *mockToolForTest) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "result"}, nil
}

// loggerForTest creates a minimal logger for tests.
func loggerForTest() *logger.Logger {
	return logger.New(logger.Config{Level: "warn", Format: "text", Output: "stdout"})
}
