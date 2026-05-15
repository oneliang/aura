// Package agent provides agent definitions for the Aura agent system.
package agent

import (
	"fmt"

	"github.com/oneliang/aura/shared/pkg/config"
)

// AgentMeta represents the YAML frontmatter in AGENT.md
// It embeds config.AgentConfig to inherit all agent tuning parameters
type AgentMeta struct {
	// Name is the agent identifier
	Name string `yaml:"name"`
	// Description triggers LLM to use this agent
	Description string `yaml:"description"`

	// Agent-specific fields (not part of AgentConfig)
	LLMModel     string   `yaml:"llm_model,omitempty"`
	DisableTools []string `yaml:"disable_tools,omitempty"`

	// Permission inheritance for sub-agent (string values, parsed by runtime)
	// PermissionMode: "inherit", "inherit_downgrade", "independent"
	PermissionMode string `yaml:"permission_mode,omitempty"`
	// PermissionLevel: "allow", "ask", "deny" (used when PermissionMode is "inherit_downgrade")
	PermissionLevel string `yaml:"permission_level,omitempty"`

	// UseReviewer enables post-execution validation by a reviewer agent
	// When true, the result is validated after sub-agent execution
	UseReviewer bool `yaml:"use_reviewer,omitempty"`
	// ReviewerAgent specifies the reviewer agent name (defaults to "code-reviewer")
	// Only used when UseReviewer is true
	ReviewerAgent string `yaml:"reviewer_agent,omitempty"`

	// Embed AgentConfig - fields are flattened into YAML
	config.AgentConfig `yaml:",inline"`
}

// Validate validates the AgentMeta.
func (m *AgentMeta) Validate() error {
	// Validate required fields
	if m.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if m.Description == "" {
		return fmt.Errorf("agent description is required")
	}

	// Validate inherited AgentConfig fields
	return m.AgentConfig.Validate()
}

// GetLLMOverride creates an LLMConfig from the agent's LLMModel field.
func (m *AgentMeta) GetLLMOverride() *config.LLMConfig {
	if m.LLMModel == "" {
		return nil
	}
	return &config.LLMConfig{
		Model: m.LLMModel,
	}
}

// Agent represents an agent definition loaded from an AGENT.md file.
// Agents are Markdown-based templates that define SubAgent behaviors.
type Agent struct {
	// Name is the agent identifier used in system prompts
	Name string
	// Description triggers LLM to use this agent (always included in system prompt)
	Description string
	// FilePath is the path to the AGENT.md file
	FilePath string
	// Content is the complete AGENT.md content (metadata + body)
	Content string
	// Body is the agent body (excluding YAML frontmatter)
	Body string
	// Meta is the parsed YAML metadata
	Meta AgentMeta
}

// GetName returns the agent name.
func (a *Agent) GetName() string { return a.Name }

// GetDescription returns the agent description.
func (a *Agent) GetDescription() string { return a.Description }

// GetFilePath returns the path to the AGENT.md file.
func (a *Agent) GetFilePath() string { return a.FilePath }

// GetContent returns the complete AGENT.md content.
func (a *Agent) GetContent() string { return a.Content }

// GetBody returns the agent body (excluding YAML frontmatter).
func (a *Agent) GetBody() string { return a.Body }
