// Package runtime provides the unified runtime for Aura.
package runtime

import (
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
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

	// Enable sub-agent delegation (default: true)
	// When false, LLM cannot delegate tasks to other agents (single-agent mode).
	EnableSubAgent bool

	// Auto-approve all tool executions (default: false)
	// When true, all permissions default to "allow" - no confirmation required.
	// Useful for SDK usage without interactive environment.
	AutoApprove bool

	// Message source for persistence (cli/tui/api/feishu/etc.)
	MessageSource string

	// Permission inheritance for sub-agent
	// PermissionMode: how sub-agent inherits permissions from parent
	PermissionMode string
	// PermissionLevel: target control level for downgrade mode
	PermissionLevel string

	// ===== 共享事件通道（多 runtime 场景）=====

	// SharedEventCh is an externally provided event channel for multi-runtime scenarios.
	// When provided, events are sent to this shared channel (shared mode).
	// When nil, runtime creates its own local channel (independent mode).
	SharedEventCh chan events.Event

	// RuntimeID identifies the source runtime for events.
	// Used in shared mode to route events and responses.
	// Default: "main" for primary runtime, custom for temp/sub-agent runtimes.
	RuntimeID string
}

// DefaultRuntimeConfig returns a default runtime configuration.
func DefaultRuntimeConfig() *RuntimeConfig {
	cfg := config.DefaultConfig()
	return &RuntimeConfig{
		Config:         cfg,
		EnableSubAgent: cfg.Agent.EnableSubAgent, // Read from embedded config (default: true)
		AutoApprove:    cfg.Agent.AutoApprove,    // Read from embedded config (default: false)
	}
}

// FromConfig creates a runtime config from the main app config.
func FromConfig(cfg *config.Config) *RuntimeConfig {
	return &RuntimeConfig{
		Config:         cfg,
		EnableSubAgent: cfg.Agent.EnableSubAgent,
		AutoApprove:    cfg.Agent.AutoApprove,
	}
}
