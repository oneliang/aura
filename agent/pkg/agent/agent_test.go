package agent

import (
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

// TestAgentMeta tests AgentMeta struct.
func TestAgentMeta(t *testing.T) {
	meta := AgentMeta{
		Name:        "test-agent",
		Description: "A test agent",
		LLMModel:    "qwen3:8b",
	}

	if meta.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", meta.Name)
	}
	if meta.Description != "A test agent" {
		t.Errorf("Expected description 'A test agent', got '%s'", meta.Description)
	}
	if meta.LLMModel != "qwen3:8b" {
		t.Errorf("Expected LLMModel 'qwen3:8b', got '%s'", meta.LLMModel)
	}
}

// TestAgentMeta_Validate tests Validate method.
func TestAgentMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    AgentMeta
		wantErr bool
	}{
		{
			name: "valid meta",
			meta: AgentMeta{
				Name:        "test-agent",
				Description: "A test agent",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			meta: AgentMeta{
				Description: "A test agent",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			meta: AgentMeta{
				Name: "test-agent",
			},
			wantErr: true,
		},
		{
			name:    "missing both",
			meta:    AgentMeta{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAgentMeta_GetLLMOverride tests GetLLMOverride method.
func TestAgentMeta_GetLLMOverride(t *testing.T) {
	tests := []struct {
		name         string
		meta         AgentMeta
		wantOverride bool
		wantModel    string
	}{
		{
			name: "with LLM model",
			meta: AgentMeta{
				Name:        "test-agent",
				Description: "A test agent",
				LLMModel:    "qwen3:8b",
			},
			wantOverride: true,
			wantModel:    "qwen3:8b",
		},
		{
			name: "without LLM model",
			meta: AgentMeta{
				Name:        "test-agent",
				Description: "A test agent",
			},
			wantOverride: false,
			wantModel:    "",
		},
		{
			name: "empty LLM model",
			meta: AgentMeta{
				Name:        "test-agent",
				Description: "A test agent",
				LLMModel:    "",
			},
			wantOverride: false,
			wantModel:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := tt.meta.GetLLMOverride()
			if tt.wantOverride {
				if override == nil {
					t.Error("Expected override, got nil")
				} else if override.Model != tt.wantModel {
					t.Errorf("Expected model '%s', got '%s'", tt.wantModel, override.Model)
				}
			} else {
				if override != nil {
					t.Errorf("Expected nil override, got %v", override)
				}
			}
		})
	}
}

// TestAgentMeta_EmbeddedAgentConfig tests that AgentConfig is properly embedded.
func TestAgentMeta_EmbeddedAgentConfig(t *testing.T) {
	meta := AgentMeta{
		Name:        "test-agent",
		Description: "A test agent",
		AgentConfig: config.AgentConfig{
			PlanningMode: "explicit",
			Temperature:  0.7,
		},
	}

	if meta.PlanningMode != "explicit" {
		t.Errorf("Expected PlanningMode 'explicit', got '%s'", meta.PlanningMode)
	}
	if meta.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", meta.Temperature)
	}
}

// TestAgentMeta_DisableTools tests DisableTools field.
func TestAgentMeta_DisableTools(t *testing.T) {
	meta := AgentMeta{
		Name:         "test-agent",
		Description:  "A test agent",
		DisableTools: []string{"bash", "file_write"},
	}

	if len(meta.DisableTools) != 2 {
		t.Errorf("Expected 2 disabled tools, got %d", len(meta.DisableTools))
	}
	if meta.DisableTools[0] != "bash" {
		t.Errorf("Expected first disabled tool 'bash', got '%s'", meta.DisableTools[0])
	}
}

// TestAgent tests Agent struct.
func TestAgent(t *testing.T) {
	ag := &Agent{
		Name:        "test-agent",
		Description: "A test agent for testing",
		FilePath:    "/path/to/AGENT.md",
		Content:     "full content",
		Body:        "agent body",
		Meta: AgentMeta{
			Name:        "test-agent",
			Description: "A test agent for testing",
		},
	}

	if ag.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", ag.Name)
	}
	if ag.Description != "A test agent for testing" {
		t.Errorf("Expected description 'A test agent for testing', got '%s'", ag.Description)
	}
	if ag.FilePath != "/path/to/AGENT.md" {
		t.Errorf("Expected FilePath '/path/to/AGENT.md', got '%s'", ag.FilePath)
	}
	if ag.Content != "full content" {
		t.Errorf("Expected Content 'full content', got '%s'", ag.Content)
	}
	if ag.Body != "agent body" {
		t.Errorf("Expected Body 'agent body', got '%s'", ag.Body)
	}
	if ag.Meta.Name != "test-agent" {
		t.Errorf("Expected Meta.Name 'test-agent', got '%s'", ag.Meta.Name)
	}
}

// TestAgent_EmptyMeta tests Agent with empty Meta.
func TestAgent_EmptyMeta(t *testing.T) {
	ag := &Agent{
		Name:        "test-agent",
		Description: "A test agent",
	}

	if ag.Meta.Name != "" {
		t.Errorf("Expected empty Meta.Name, got '%s'", ag.Meta.Name)
	}
}
