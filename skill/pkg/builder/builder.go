// Package builder provides skill prompt building functionality.
package builder

import (
	"fmt"
	"strings"

	"github.com/oneliang/aura/skill/pkg/skill"
)

// BuildSystemPromptSection generates a system prompt section for skills.
// This section includes skill names and descriptions for LLM awareness.
// LLM uses skill_activate tool to get detailed instructions when needed.
func BuildSystemPromptSection(skills []skill.Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Skills\n\n")
	sb.WriteString("You have access to specialized skills. When a task matches a skill's purpose:\n")
	sb.WriteString("1. Call the `skill_activate` tool with the exact skill name\n")
	sb.WriteString("2. Follow the returned instructions to complete the task\n\n")

	sb.WriteString("Available skills (call skill_activate to get instructions):\n")
	for _, sk := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", sk.Name, sk.Description))
	}

	return sb.String()
}

// BuildFullPrompt generates the complete skill prompt including body.
// This is used when the LLM decides to use a specific skill.
func BuildFullPrompt(sk skill.Skill) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\n## Using Skill: %s\n\n", sk.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", sk.Description))
	sb.WriteString("Instructions:\n\n")
	sb.WriteString(sk.Body)
	sb.WriteString("\n")

	return sb.String()
}

// BuildSkillMetadata builds a concise skill metadata string for system prompts.
func BuildSkillMetadata(skills []skill.Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, sk := range skills {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", sk.Name, sk.Description))
	}

	return sb.String()
}
