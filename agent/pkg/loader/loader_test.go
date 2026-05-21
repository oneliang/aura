// Package loader provides agent loading functionality from directories.
package loader

import (
	"os"
	"path/filepath"
	"testing"

	sharedfilepath "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

func TestLoader_Load_NoDirectories(t *testing.T) {
	loader := NewLoader([]string{})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Load() returned %d agents, want 0", len(agents))
	}
}

func TestLoader_Load_NonExistentDirectory(t *testing.T) {
	loader := NewLoader([]string{"/nonexistent/path/agents"})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Load() returned %d agents, want 0", len(agents))
	}
}

func TestLoader_Load_ValidAgent(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "test-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	agentContent := `---
name: test-agent
description: A test agent for unit testing
llm_model: qwen3:8b
---

## Role

You are a test agent.

## Guidelines

1. Test everything
`
	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Load() returned %d agents, want 1", len(agents))
	}

	a := agents[0]
	if a.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", a.Name, "test-agent")
	}
	if a.Description != "A test agent for unit testing" {
		t.Errorf("Description = %q, want %q", a.Description, "A test agent for unit testing")
	}
	if a.Meta.LLMModel != "qwen3:8b" {
		t.Errorf("LLMModel = %v, want qwen3:8b", a.Meta.LLMModel)
	}
}

func TestLoader_Load_MultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first agent
	agentDir1 := filepath.Join(tmpDir, "agent-one")
	if err := os.MkdirAll(agentDir1, 0755); err != nil {
		t.Fatalf("Failed to create agent dir 1: %v", err)
	}
	agentContent1 := `---
name: agent-one
description: First test agent
---

## Role

First agent body.
`
	if err := os.WriteFile(filepath.Join(agentDir1, "AGENT.md"), []byte(agentContent1), 0644); err != nil {
		t.Fatalf("Failed to write agent 1: %v", err)
	}

	// Create second agent
	agentDir2 := filepath.Join(tmpDir, "agent-two")
	if err := os.MkdirAll(agentDir2, 0755); err != nil {
		t.Fatalf("Failed to create agent dir 2: %v", err)
	}
	agentContent2 := `---
name: agent-two
description: Second test agent
---

## Role

Second agent body.
`
	if err := os.WriteFile(filepath.Join(agentDir2, "AGENT.md"), []byte(agentContent2), 0644); err != nil {
		t.Fatalf("Failed to write agent 2: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("Load() returned %d agents, want 2", len(agents))
	}
}

func TestLoader_Load_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "invalid-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	// Invalid: no YAML frontmatter
	agentContent := `This is not a valid agent file.
No YAML frontmatter here.`

	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid format, got nil")
	}
}

func TestLoader_Load_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "missing-name")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	agentContent := `---
description: No name provided
---

## Role

Test agent.
`
	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() expected error for missing name, got nil")
	}
}

func TestLoader_Load_MissingDescription(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "missing-desc")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	agentContent := `---
name: no-description
---

## Role

Test agent.
`
	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	_, err := loader.Load()
	if err == nil {
		t.Fatal("Load() expected error for missing description, got nil")
	}
}

func TestLoader_GetAgents(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "test-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	agentContent := `---
name: test-agent
description: A test agent
---

## Role

Test.
`
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	_, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agents := loader.GetAgents()
	if len(agents) != 1 {
		t.Errorf("GetAgents() returned %d agents, want 1", len(agents))
	}
}

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"no tilde", "/home/user", false},
		{"tilde alone", "~", false},
		{"tilde with path", "~/agents", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sharedfilepath.ExpandTilde(tt.input)
			if result == "" && tt.input != "" {
				t.Errorf("ExpandTilde(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestLoader_Load_FullAgentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "full-config-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	agentContent := `---
name: full-config-agent
description: Test agent with full AgentConfig fields
llm_model: qwen-coder-plus
planning_mode: explicit
temperature: 0.3
summary_temp: 0.2
disable_tools:
  - bash
  - ssh_exec
---

## Role

Test agent with full configuration.
`
	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Load() returned %d agents, want 1", len(agents))
	}

	a := agents[0]

	// Verify basic fields
	if a.Name != "full-config-agent" {
		t.Errorf("Name = %q, want %q", a.Name, "full-config-agent")
	}
	if a.Meta.LLMModel != "qwen-coder-plus" {
		t.Errorf("LLMModel = %v, want qwen-coder-plus", a.Meta.LLMModel)
	}
	if a.Meta.PlanningMode != "explicit" {
		t.Errorf("PlanningMode = %q, want explicit", a.Meta.PlanningMode)
	}
	if a.Meta.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want 0.3", a.Meta.Temperature)
	}
	if a.Meta.SummaryTemp != 0.2 {
		t.Errorf("SummaryTemp = %f, want 0.2", a.Meta.SummaryTemp)
	}

	// Verify disable_tools
	if len(a.Meta.DisableTools) != 2 {
		t.Errorf("DisableTools = %d items, want 2", len(a.Meta.DisableTools))
	}
}

func TestLoader_Load_ConfigInheritance(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "inheritance-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	// Only specify name and description
	// All other fields should use zero-values (will inherit from parent at runtime)
	agentContent := `---
name: inheritance-agent
description: Test agent with minimal config to verify inheritance
---

## Role

Test agent for config inheritance verification.
`
	agentFile := filepath.Join(agentDir, "AGENT.md")
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	loader := NewLoader([]string{tmpDir})
	agents, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("Load() returned %d agents, want 1", len(agents))
	}

	a := agents[0]

	// Verify unspecified fields use zero-values (will inherit from parent at runtime)
	if a.Meta.LLMModel != "" {
		t.Errorf("LLMModel should be empty (inherit from parent), got %q", a.Meta.LLMModel)
	}
	if a.Meta.PlanningMode != "" {
		t.Errorf("PlanningMode should be empty (inherit from parent), got %q", a.Meta.PlanningMode)
	}
	if a.Meta.Temperature != 0 {
		t.Errorf("Temperature should be 0 (zero-value), got %f", a.Meta.Temperature)
	}
	if a.Meta.SummaryTemp != 0 {
		t.Errorf("SummaryTemp should be 0 (zero-value), got %f", a.Meta.SummaryTemp)
	}
	if len(a.Meta.DisableTools) != 0 {
		t.Errorf("DisableTools should be empty, got %d items", len(a.Meta.DisableTools))
	}
}
