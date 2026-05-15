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
