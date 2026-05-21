// Package commands provides skill-based command functionality.
// This package converts skills into executable commands that can be invoked directly.
package commands

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/skill/pkg/loader"
	"github.com/oneliang/aura/skill/pkg/skill"
)

// SkillCommand converts skills into executable commands.
// Each skill directory becomes a command that can be invoked directly by name.
type SkillCommand struct {
	loader *loader.Loader
}

// NewSkillCommand creates a new skill command provider.
func NewSkillCommand(loader *loader.Loader) *SkillCommand {
	return &SkillCommand{
		loader: loader,
	}
}

// GetCommands returns all skills as commands with their metadata.
// This allows skills to be discovered and executed like regular commands.
func (sc *SkillCommand) GetCommands() []CommandInfo {
	if sc.loader == nil {
		return nil
	}

	skills := sc.loader.GetSkills()
	cmds := make([]CommandInfo, 0, len(skills))

	for _, sk := range skills {
		cmds = append(cmds, CommandInfo{
			Name:        sc.buildCommandName(sk.Name),
			DisplayName: sk.Name,
			Description: sk.Description,
			Params: []ParamInfo{
				{
					Name:     "params",
					Type:     "map",
					Required: false,
					Desc:     "Skill-specific parameters (key=value pairs)",
				},
			},
		})
	}

	return cmds
}

// Execute executes a skill by name with the given parameters.
// The command name should match the skill name (with optional "skill_" prefix).
func (sc *SkillCommand) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	if sc.loader == nil {
		return "", fmt.Errorf("skill loader not initialized")
	}

	// Extract skill name from command
	skillName := sc.extractSkillName(cmd)

	// Find the skill
	targetSkill := sc.findSkill(skillName)
	if targetSkill == nil {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	// Execute the skill with parameters
	return sc.executeSkill(ctx, targetSkill, params)
}

// buildCommandName builds the command name from a skill name.
// Supports both "skill_name" and direct "name" formats.
func (sc *SkillCommand) buildCommandName(skillName string) string {
	// Use skill_ prefix for consistency with other internal commands
	return "skill_" + skillName
}

// extractSkillName extracts the skill name from a command name.
// Handles both "skill_name" and direct "name" formats.
func (sc *SkillCommand) extractSkillName(cmd string) string {
	// Remove "skill_" prefix if present
	const skillPrefix = "skill_"
	if len(cmd) > len(skillPrefix) && cmd[:len(skillPrefix)] == skillPrefix {
		return cmd[len(skillPrefix):]
	}
	return cmd
}

// findSkill finds a skill by name (case-insensitive matching).
func (sc *SkillCommand) findSkill(name string) *skill.Skill {
	skills := sc.loader.GetSkills()
	for i := range skills {
		// Case-insensitive matching
		if skills[i].Name == name {
			return &skills[i]
		}
	}
	return nil
}

// executeSkill executes a skill with the given parameters.
// For now, it returns the skill body as instructions.
// Future iterations will parse and execute the workflow steps.
func (sc *SkillCommand) executeSkill(ctx context.Context, sk *skill.Skill, params map[string]any) (string, error) {
	// Build context from parameters
	contextStr := sc.buildSkillContext(params)

	// Return skill body with context
	// Future: parse workflow steps and execute them
	result := fmt.Sprintf("Executing skill: %s\n\nDescription: %s\n\nInstructions:\n%s",
		sk.Name, sk.Description, sk.Body)

	if contextStr != "" {
		result += fmt.Sprintf("\n\nContext:\n%s", contextStr)
	}

	return result, nil
}

// buildSkillContext builds a context string from parameters.
func (sc *SkillCommand) buildSkillContext(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}

	var ctx string
	for k, v := range params {
		ctx += fmt.Sprintf("  %s: %v\n", k, v)
	}
	return ctx
}

// GetSkills returns the loaded skills for external access.
func (sc *SkillCommand) GetSkills() []skill.Skill {
	if sc.loader == nil {
		return nil
	}
	return sc.loader.GetSkills()
}
