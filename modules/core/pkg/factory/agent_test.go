package factory

import (
	"context"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/config"
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

// MockMemory implements engine.Memory interface for testing
type MockMemory struct {
	messages []llm.Message
}

func (m *MockMemory) Add(role, content string) {
	m.messages = append(m.messages, newTextMessage(role, content))
}

func (m *MockMemory) Get() []llm.Message {
	return m.messages
}

func (m *MockMemory) Clear() {
	m.messages = nil
}

func (m *MockMemory) AddWithType(role, content string, msgType memory.MessageType) {
	m.messages = append(m.messages, newTextMessage(role, content))
}

func (m *MockMemory) AddWithParts(role string, parts []memory.MessagePart, msgType memory.MessageType) {
}

func (m *MockMemory) AddWithBlocks(role string, blocks []memory.ContentBlock, msgType memory.MessageType) {
	// Extract text from first TextBlock for Content
	var textContent string
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			textContent = tb.Text
			break
		}
	}
	m.messages = append(m.messages, newTextMessage(role, textContent))
}

// MockLLMClient implements llm.Client interface for testing
type MockLLMClient struct{}

func (m *MockLLMClient) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return &llm.Response{Message: newTextMessage("assistant", "mock response")}, nil
}

func (m *MockLLMClient) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Content: "mock chunk"}
	close(ch)
	return ch, nil
}

func (m *MockLLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2, 0.3}}, nil
}

func TestEngineFactory_Create_DefaultConfig(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{} // Empty config, should use defaults

	// Create real permission manager
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

	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
}

func TestEngineFactory_Create_CustomConfig(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{
		PlanningMode: "explicit",
	}

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

	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
}

func TestEngineFactory_Create_WithSystemPrompt(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	customPrompt := "You are a custom assistant."
	factory := NewEngineFactory(
		llmClient, cfg, permMgr,
		WithSystemPrompt(customPrompt),
	)
	mem := &MockMemory{}

	ag, err := factory.Create(mem)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
}

func TestEngineFactory_CreateWithSession(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	factory := NewEngineFactory(llmClient, cfg, permMgr)
	mem := &MockMemory{}

	ag, err := factory.CreateWithSession("test-session", mem)

	if err != nil {
		t.Fatalf("CreateWithSession() returned error: %v", err)
	}

	if ag == nil {
		t.Fatal("CreateWithSession() returned nil agent")
	}
}

func TestEngineFactory_PlanningMode_Auto(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{
		PlanningMode: "auto",
	}

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

	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
}

func TestEngineFactory_InvalidPlanningMode(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{
		PlanningMode: "invalid",
	}

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

	if ag == nil {
		t.Fatal("Create() returned nil agent")
	}
	// Invalid planning mode should default to implicit
}

func TestEngineFactory_WithPermissionManager(t *testing.T) {
	llmClient := &MockLLMClient{}
	cfg := &config.AgentConfig{}

	permCfg := permissions.DefaultPermissionConfig()
	permMgr, err := permissions.NewManager(permCfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}

	// Create factory with nil permMgr initially
	factory := NewEngineFactory(llmClient, cfg, nil, WithPermissionManager(permMgr))

	if factory.permMgr != permMgr {
		t.Error("WithPermissionManager() did not set permission manager")
	}
}