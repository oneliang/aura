// Package builder provides agent prompt building functionality.
package builder

import (
	"fmt"
	"strings"

	"github.com/oneliang/aura/agent/pkg/agent"
)

// BuildSystemPromptSection generates a system prompt section for agents.
// This section includes agent names and descriptions for LLM triggering.
func BuildSystemPromptSection(agents []agent.Agent) string {
	if len(agents) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Available SubAgents\n\n")
	sb.WriteString("You have access to these SubAgents as tools. ")
	sb.WriteString("When the user's request matches a SubAgent's description, delegate the task immediately.\n\n")

	sb.WriteString("To delegate a task, use the following JSON action format:\n")
	sb.WriteString("Action: {\"tool\": \"command_agent_<name>\", \"parameters\": {\"task\": \"<task description>\"}}\n\n")

	for _, a := range agents {
		sb.WriteString(fmt.Sprintf("- **command_agent_%s**: %s\n", a.Name, a.Description))
	}

	sb.WriteString("\nEach SubAgent has isolated context and specialized capabilities. ")
	sb.WriteString("Use them to offload specific tasks while maintaining focus on the main objective.\n")
	sb.WriteString("\n**IMPORTANT**: Do NOT use regular tools (file_read, file_write, bash, etc.) for tasks that clearly match a SubAgent's description. Always delegate first.\n")

	return sb.String()
}

// BuildFullPrompt generates the complete agent prompt including body.
// This is used when the LLM decides to use a specific agent.
func BuildFullPrompt(a agent.Agent) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\n## Delegating to SubAgent: %s\n\n", a.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", a.Description))
	sb.WriteString("Instructions:\n\n")
	sb.WriteString(a.Body)
	sb.WriteString("\n")

	return sb.String()
}

// BuildAgentMetadata builds a concise agent metadata string for system prompts.
func BuildAgentMetadata(agents []agent.Agent) string {
	if len(agents) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, a := range agents {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", a.Name, a.Description))
	}

	return sb.String()
}
