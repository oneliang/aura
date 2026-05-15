// Package runtime provides the unified runtime for Aura.
package runtime

import (
	"github.com/oneliang/aura/shared/pkg/config"
)

// RuntimeConfig holds configuration for the Aura runtime.
// It embeds *config.Config for access to all configuration fields,
// plus runtime-specific fields.
type RuntimeConfig struct {
	*config.Config

	// Session ID (empty for default session)
	SessionID string

	// Role name for system prompt (optional)
	Role string

	// Custom system prompt (optional, overrides role)
	SystemPrompt string

	// Disable tools
	DisableTools bool

	// Message source for persistence (cli/tui/api/feishu/etc.)
	MessageSource string

	// Permission inheritance for sub-agent
	// PermissionMode: how sub-agent inherits permissions from parent
	PermissionMode string
	// PermissionLevel: target control level for downgrade mode
	PermissionLevel string
}

// DefaultRuntimeConfig returns a default runtime configuration.
func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Config: config.DefaultConfig(),
	}
}

// FromConfig creates a runtime config from the main app config.
func FromConfig(cfg *config.Config) *RuntimeConfig {
	return &RuntimeConfig{
		Config: cfg,
	}
}
