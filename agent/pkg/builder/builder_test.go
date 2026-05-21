// Package builder provides agent prompt building functionality.
package builder

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/agent/pkg/agent"
)

func TestBuildSystemPromptSection_Empty(t *testing.T) {
	result := BuildSystemPromptSection([]agent.Agent{})
	if result != "" {
		t.Errorf("BuildSystemPromptSection([]) = %q, want empty string", result)
	}
}

func TestBuildSystemPromptSection_SingleAgent(t *testing.T) {
	agents := []agent.Agent{
		{
			Name:        "code-reviewer",
			Description: "Review code for quality issues",
			Body:        "You are a code reviewer...",
		},
	}

	result := BuildSystemPromptSection(agents)

	if !strings.Contains(result, "## Available SubAgents") {
		t.Error("BuildSystemPromptSection() missing header")
	}
	if !strings.Contains(result, "code-reviewer") {
		t.Error("BuildSystemPromptSection() missing agent name")
	}
	if !strings.Contains(result, "Review code for quality issues") {
		t.Error("BuildSystemPromptSection() missing agent description")
	}
	if !strings.Contains(result, "command_agent_code-reviewer") {
		t.Error("BuildSystemPromptSection() missing command_agent_code-reviewer reference")
	}
	if !strings.Contains(result, "\"tool\"") {
		t.Error("BuildSystemPromptSection() missing tool JSON format")
	}
}

func TestBuildSystemPromptSection_MultipleAgents(t *testing.T) {
	agents := []agent.Agent{
		{
			Name:        "code-reviewer",
			Description: "Review code for quality issues",
		},
		{
			Name:        "data-analyst",
			Description: "Analyze data and generate reports",
		},
	}

	result := BuildSystemPromptSection(agents)

	if !strings.Contains(result, "code-reviewer") {
		t.Error("BuildSystemPromptSection() missing first agent")
	}
	if !strings.Contains(result, "data-analyst") {
		t.Error("BuildSystemPromptSection() missing second agent")
	}
}

func TestBuildFullPrompt(t *testing.T) {
	a := agent.Agent{
		Name:        "test-agent",
		Description: "A test agent",
		Body:        "You are a test agent. Follow these steps...",
	}

	result := BuildFullPrompt(a)

	if !strings.Contains(result, "## Delegating to SubAgent: test-agent") {
		t.Error("BuildFullPrompt() missing header")
	}
	if !strings.Contains(result, "Description: A test agent") {
		t.Error("BuildFullPrompt() missing description")
	}
	if !strings.Contains(result, "You are a test agent") {
		t.Error("BuildFullPrompt() missing body")
	}
}

func TestBuildAgentMetadata_Empty(t *testing.T) {
	result := BuildAgentMetadata([]agent.Agent{})
	if result != "" {
		t.Errorf("BuildAgentMetadata([]) = %q, want empty string", result)
	}
}

func TestBuildAgentMetadata_SingleAgent(t *testing.T) {
	agents := []agent.Agent{
		{
			Name:        "code-reviewer",
			Description: "Review code",
		},
	}

	result := BuildAgentMetadata(agents)

	if result != "code-reviewer: Review code" {
		t.Errorf("BuildAgentMetadata() = %q, want %q", result, "code-reviewer: Review code")
	}
}

func TestBuildAgentMetadata_MultipleAgents(t *testing.T) {
	agents := []agent.Agent{
		{
			Name:        "agent-one",
			Description: "First agent",
		},
		{
			Name:        "agent-two",
			Description: "Second agent",
		},
	}

	result := BuildAgentMetadata(agents)

	if !strings.Contains(result, "agent-one: First agent") {
		t.Error("BuildAgentMetadata() missing first agent")
	}
	if !strings.Contains(result, "agent-two: Second agent") {
		t.Error("BuildAgentMetadata() missing second agent")
	}
	if !strings.Contains(result, "; ") {
		t.Error("BuildAgentMetadata() missing separator")
	}
}
