// Package loader provides skill loading functionality from directories.
package loader

import (
	"fmt"

	sharedloader "github.com/oneliang/aura/shared/pkg/loader"
	sharedmanager "github.com/oneliang/aura/shared/pkg/manager"
	"github.com/oneliang/aura/skill/pkg/skill"
)

// Loader loads skills from configured directories.
type Loader struct {
	baseDirs []string
	skills   []skill.Skill
}

// NewLoader creates a new skill loader.
func NewLoader(baseDirs []string) *Loader {
	return &Loader{
		baseDirs: baseDirs,
	}
}

// Load loads all skills from configured directories.
// Each directory can contain multiple skill subdirectories, each with a SKILL.md file.
func (l *Loader) Load() ([]skill.Skill, error) {
	results, err := sharedloader.LoadFromDirectories(
		l.baseDirs,
		sharedloader.FileSpec{FileName: "SKILL.md"},
		parseSkill,
	)
	if err != nil {
		return nil, err
	}

	l.skills = make([]skill.Skill, len(results))
	for i, r := range results {
		l.skills[i] = r.Item
	}
	return l.skills, nil
}

// parseSkill implements sharedloader.ParseFunc[skill.Skill].
func parseSkill(content, filePath string) (skill.Skill, string, error) {
	var meta struct {
		Name            string `yaml:"name"`
		Description     string `yaml:"description"`
		PermissionLevel string `yaml:"permission_level,omitempty"`
	}

	body, err := sharedloader.UnmarshalYAMLFrontmatter(content, &meta)
	if err != nil {
		return skill.Skill{}, "", err
	}

	if meta.Name == "" {
		return skill.Skill{}, "", fmt.Errorf("skill name is required")
	}
	if meta.Description == "" {
		return skill.Skill{}, "", fmt.Errorf("skill description is required")
	}

	// Parse permission level, default to "execute" for safety
	permLevel := parseSkillPermissionLevel(meta.PermissionLevel)

	return skill.Skill{
		Name:            meta.Name,
		Description:     meta.Description,
		FilePath:        filePath,
		Content:         content,
		Body:            body,
		PermissionLevel: permLevel,
	}, body, nil
}

// parseSkillPermissionLevel parses a string to SkillPermissionLevel.
// Defaults to "execute" for safety if not specified or invalid.
func parseSkillPermissionLevel(s string) skill.SkillPermissionLevel {
	switch s {
	case "read", "readonly":
		return skill.SkillPermissionReadOnly
	case "write":
		return skill.SkillPermissionWrite
	case "execute", "exec":
		return skill.SkillPermissionExecute
	case "admin":
		return skill.SkillPermissionAdmin
	default:
		// Default to execute (requires confirmation) for safety
		return skill.SkillPermissionExecute
	}
}

// GetSkills returns the loaded skills.
func (l *Loader) GetSkills() []skill.Skill {
	return l.skills
}

// GetItems returns skills as shared manager Item slice.
func (l *Loader) GetItems() []sharedmanager.Item {
	if l == nil {
		return nil
	}
	skills := l.GetSkills()
	if skills == nil {
		return nil
	}
	items := make([]sharedmanager.Item, len(skills))
	for i := range skills {
		items[i] = &skills[i]
	}
	return items
}
