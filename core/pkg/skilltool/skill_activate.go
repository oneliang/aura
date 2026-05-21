package skilltool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	skill "github.com/oneliang/aura/skill/pkg/skill"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	tools "github.com/oneliang/aura/tools/pkg"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// SkillActivateTool allows LLM to explicitly request a skill's detailed instructions.
// This replaces the automatic SkillMatcher approach, giving LLM full control over
// when and which skill to activate.
type SkillActivateTool struct {
	loader   *skillloader.Loader
	injector *SkillInjector
}

// NewSkillActivateTool creates a new skill activation tool.
func NewSkillActivateTool(loader *skillloader.Loader, injector *SkillInjector) *SkillActivateTool {
	return &SkillActivateTool{loader: loader, injector: injector}
}

// Name returns the tool name.
func (t *SkillActivateTool) Name() string {
	return "skill_activate"
}

// Description returns the tool description for LLM.
func (t *SkillActivateTool) Description() string {
	return `Activate a skill by name to get detailed instructions.

Use this tool when you need specialized capabilities mentioned in the Skills section.
First call this tool with the skill name, then follow the returned instructions.

Parameters:
- skill_name (string, required): The exact skill name to activate (e.g., "postgres", "pdf", "xlsx")`
}

// InputSchema returns the JSON schema for tool parameters.
func (t *SkillActivateTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{"skill_name"},
		"properties": map[string]any{
			"skill_name": map[string]any{
				"type":        "string",
				"description": "The exact skill name to activate (e.g., 'postgres', 'pdf')",
			},
		},
	}
}

// Execute activates a skill and returns its body.
func (t *SkillActivateTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	skillName, ok := params["skill_name"].(string)
	if !ok || skillName == "" {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  "skill_name parameter is required",
		}, nil
	}

	// Find skill by name (case-insensitive)
	var foundSkill *skill.Skill
	for _, sk := range t.loader.GetSkills() {
		if strings.ToLower(sk.Name) == strings.ToLower(skillName) {
			foundSkill = &sk
			break
		}
	}

	if foundSkill == nil {
		// Build helpful error message with available skills
		available := make([]string, 0)
		for _, sk := range t.loader.GetSkills() {
			available = append(available, sk.Name)
		}
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("Skill '%s' not found. Available skills: %s", skillName, strings.Join(available, ", ")),
		}, nil
	}

	// Use injector to store skill body for cache-aware retrieval
	// This ensures skill body is available via GetInjectedBodies() in buildReActMessages
	var skillBody string
	if t.injector != nil {
		// InjectSkill stores body internally and returns the formatted message
		msg := t.injector.InjectSkill(*foundSkill)
		// Extract text from ContentBlocks
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				skillBody = tb.Text
				break
			}
		}
	} else {
		// Fallback: format skill body directly (injector not available)
		skillDir := filepath.Dir(foundSkill.FilePath)
		bodyWithPaths := strings.ReplaceAll(foundSkill.Body, "{skill_dir}", skillDir)
		skillBody = fmt.Sprintf("## Skill: %s\n\n%s", foundSkill.Name, bodyWithPaths)
	}

	// Return skill body to LLM as ToolResult content
	// Engine will retrieve body from SkillInjector.GetInjectedBodies() for cache-aware path
	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: skillBody,
		Data: map[string]any{
			"skill_name": foundSkill.Name,
			"injected":   true,
		},
	}, nil
}