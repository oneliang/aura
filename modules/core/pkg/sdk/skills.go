// Package sdk provides the unified agent runtime and factories.
// This file provides Skill loading — a thin wrapper over the skill loader
// that eliminates cross-layer imports for application code (CLI/TUI).
package sdk

import (
	"fmt"

	skillloader "github.com/oneliang/aura/skill/pkg/loader"
)

// SkillInfo represents a skill for SDK-level consumers.
// Mirrors skill.Skill with all fields exported.
type SkillInfo struct {
	Name                 string
	Description          string
	Body                 string
	FilePath             string
	PermissionLevel      string
	RequiresConfirmation bool
}

// LoadSkills loads all skills from the given directories and returns SkillInfo slices.
// This replaces the duplicated skill loading logic in TUI commands.go
// (getSkillsForCommandCompletion + tryExecuteSkill both do config.Load + loader.Load).
func LoadSkills(directories []string) ([]SkillInfo, error) {
	loader := skillloader.NewLoader(directories)
	skills, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load skills: %w", err)
	}

	result := make([]SkillInfo, len(skills))
	for i, sk := range skills {
		result[i] = SkillInfo{
			Name:                 sk.Name,
			Description:          sk.Description,
			Body:                 sk.Body,
			FilePath:             sk.FilePath,
			PermissionLevel:      string(sk.PermissionLevel),
			RequiresConfirmation: sk.RequiresConfirmation(),
		}
	}
	return result, nil
}
