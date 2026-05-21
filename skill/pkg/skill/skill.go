// Package skill provides skill definitions for the Aura skill system.
package skill

// SkillPermissionLevel represents the permission level required for a skill.
type SkillPermissionLevel string

const (
	// SkillPermissionReadOnly - Read-only operations (no confirmation required)
	SkillPermissionReadOnly SkillPermissionLevel = "read"
	// SkillPermissionWrite - Write operations (confirmation required)
	SkillPermissionWrite SkillPermissionLevel = "write"
	// SkillPermissionExecute - Command execution (confirmation required)
	SkillPermissionExecute SkillPermissionLevel = "execute"
	// SkillPermissionAdmin - Administrative operations (explicit authorization required)
	SkillPermissionAdmin SkillPermissionLevel = "admin"
)

// Skill represents a skill definition loaded from a SKILL.md file.
// Skills are Markdown-based prompt templates that guide agent behaviors.
type Skill struct {
	// Name is the skill identifier used in system prompts
	Name string
	// Description triggers LLM to use this skill (always included in system prompt)
	Description string
	// FilePath is the path to the SKILL.md file
	FilePath string
	// Content is the complete SKILL.md content (metadata + body)
	Content string
	// Body is the skill body (excluding YAML frontmatter)
	Body string
	// PermissionLevel indicates the permission level required for this skill
	// Default: "execute" if not specified (conservative approach)
	PermissionLevel SkillPermissionLevel
}

// RequiresConfirmation returns true if this skill requires user confirmation.
func (s *Skill) RequiresConfirmation() bool {
	switch s.PermissionLevel {
	case "", SkillPermissionReadOnly:
		return false
	case SkillPermissionWrite, SkillPermissionExecute, SkillPermissionAdmin:
		return true
	default:
		// Default to requiring confirmation for safety
		return true
	}
}

// GetName returns the skill name.
func (s *Skill) GetName() string { return s.Name }

// GetDescription returns the skill description.
func (s *Skill) GetDescription() string { return s.Description }

// GetFilePath returns the path to the SKILL.md file.
func (s *Skill) GetFilePath() string { return s.FilePath }

// GetContent returns the complete SKILL.md content.
func (s *Skill) GetContent() string { return s.Content }

// GetBody returns the skill body (excluding YAML frontmatter).
func (s *Skill) GetBody() string { return s.Body }
