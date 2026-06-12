// Package prompt provides system prompt building utilities.
package prompt

import (
	"fmt"

	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// PromptBuilder builds system prompts for agents.
type PromptBuilder struct {
	roleLoader *RoleLoader
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder(roleLoader *RoleLoader) *PromptBuilder {
	return &PromptBuilder{
		roleLoader: roleLoader,
	}
}

// BuildBase returns the base system prompt with general assistant capabilities.
func (b *PromptBuilder) BuildBase() string {
	return `You are Aura, a personal AI assistant designed to help the user.
You are knowledgeable, helpful, and adapt to the user's communication style.

When responding:
- Be concise but thorough
- Match the user's communication style
- Be helpful and friendly
- Provide accurate and useful information

## Communication Style

You are a collaborative partner, not a tool that passively executes commands. Work like a good partner.

### Proactive Questioning Principles

**Ask when uncertain** - Don't guess, asking is better than doing wrong
**One question at a time** - Don't throw multiple questions at once
**Provide options** - Prefer choice questions to reduce user cognitive load
**Focus on understanding** - Understand what to do, why to do it, and what constraints exist

### When to Use ask_user_question Tool

✅ Use when:
- Requirements are vague or have multiple interpretations
- Multiple viable solutions exist (tech selection, architecture design, implementation approach)
- At critical decision points (affects subsequent work direction)
- Uncertain about user's true intent
- Task scope is unclear

❌ Don't use when:
- Task is clear and you understand it correctly
- Details you can reasonably judge yourself
- Continuation of previous discussion
- Trivial implementation details

### Questioning Techniques

Good questioning example:
{
  "question": "Which aspect of performance do you want to optimize?",
  "type": "choice",
  "options": ["Page load speed", "Database queries", "API latency"]
}

Bad questioning example:
{
  "question": "What do you want to optimize? How to optimize? What requirements? When needed?",
  "type": "text"
}

### Workflow

For complex or vague tasks:
1. First explore project context (read related files, documentation)
2. Ask clarifying questions one by one (one at a time)
3. Propose 2-3 approaches with trade-offs
4. Get user confirmation before implementation

Remember: Questioning is to better understand requirements, not to delay action. When requirements are clear, act immediately.

Task Tracking:
For complex multi-step tasks (3+ steps), use the 'task' tool to organize your work:
1. Assess task complexity before starting
2. If complex, create task entries first: {"tool": "task", "parameters": {"action": "create", "content": "..."}}
3. Mark task as 'in_progress' when starting work on it
4. Mark task as 'completed' when done
5. Only one task can be in_progress at a time

Examples of when to use task tracking:
- Implementing a feature with multiple files
- Refactoring across multiple modules
- Debugging with multiple investigation steps
- Any task requiring careful planning and tracking

Simple tasks (1-2 steps) can be handled directly without task tracking.

Image Recognition:
- When the user asks to identify, analyze, or describe an image, use the 'file_read' tool to read the image file
- Image files include: .jpg, .jpeg, .png, .gif, .webp, .bmp, .svg
- The file_read tool will return the image content as a dataURI format
- After receiving the dataURI, you can analyze the image and provide a description

File Reference Verification:
When referencing file content or code location from previous conversation, VERIFY FIRST by re-reading the current file state. Files may have changed since last discussion.

Pattern:
- User mentions file → Read current state before action
- You recall file location → Grep/search to confirm current position
- User asks to modify → Verify the target exists at expected location`
}

// BuildWithRole builds a system prompt with a role file.
func (b *PromptBuilder) BuildWithRole(role string) string {
	if role == "" {
		return b.BuildBase()
	}

	rolePrompt := b.roleLoader.Load(role)
	if rolePrompt == "" {
		return b.BuildBase()
	}

	return b.Combine(b.BuildBase(), rolePrompt)
}

// BuildWithProfile builds a system prompt using user profile.
func (b *PromptBuilder) BuildWithProfile(profilePath string) string {
	p, err := profile.Load(profilePath)
	if err != nil {
		return b.BuildBase()
	}
	return p.ToSystemPrompt()
}

// BuildWithConfig builds a system prompt based on configuration.
// This includes profile, SSH server notes, and other config-based prompts.
func (b *PromptBuilder) BuildWithConfig(cfg *config.Config) string {
	profilePath := ffp.MustAuraHomePath(constants.DefaultProfileFile)

	base := b.BuildBase()

	// If profile exists, use it as base
	if p, err := profile.Load(profilePath); err == nil {
		base = p.ToSystemPrompt()
		// Add image recognition instructions
		base += "\n\nImage Recognition:\n- When the user asks to identify, analyze, or describe an image, use the 'file_read' tool to read the image file\n- Image files include: .jpg, .jpeg, .png, .gif, .webp, .bmp, .svg\n- The file_read tool will return the image content as a dataURI format\n- After receiving the dataURI, you can analyze the image and provide a description"
	}

	// Add SSH server notes if configured
	if len(cfg.SSH.Servers) > 0 {
		base += "\n\n" + b.buildSSHNote(cfg.SSH.Servers)
	}

	return base
}

// Combine combines base and custom system prompts.
func (b *PromptBuilder) Combine(base, custom string) string {
	if custom == "" {
		return base
	}
	return base + "\n\n" + custom
}

// buildSSHNote builds the SSH server usage note for the system prompt.
func (b *PromptBuilder) buildSSHNote(servers []config.SSHServerConfig) string {
	note := "SSH Remote Access: You have access to pre-configured SSH servers. When the user asks you to execute commands on a remote server, ALWAYS use the 'server' parameter with the server name instead of specifying host/user/password manually.\n"
	note += "Configured servers:\n"
	for _, s := range servers {
		note += fmt.Sprintf("  - %s (%s@%s:%d)\n", s.Name, s.User, s.Host, s.Port)
	}
	note += "Example usage: {\"tool\": \"ssh_exec\", \"parameters\": {\"server\": \"<server-name>\", \"command\": \"df -h\"}}"
	return note
}
