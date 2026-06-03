// Package cli provides the CLI application entry point.
package cli

import (
	"github.com/oneliang/aura/cli/pkg/common"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// CommandContext holds all dependencies for CLI commands.
type CommandContext struct {
	// Runtime dependencies (populated at runtime)
	Config          *config.Config
	Logger          *logger.Logger
	PermissionMgr   *sdk.PermissionManager
	CommandProvider sdk.Command // Unified command provider

	// Session data directory for task persistence
	DataDir string

	// Factory interfaces (can be mocked for testing)
	HomeDirProvider          HomeDirProvider
	KnowledgeStoreFactory    KnowledgeStoreFactory
	PermissionManagerFactory PermissionManagerFactory
	ConfigLoader             ConfigLoader

	// Current user ID (empty for legacy single-user mode)
	UserID string
}

// globalCmdCtx is the global command context.
var globalCmdCtx *CommandContext

// getCommandContext returns the global command context.
func getCommandContext() *CommandContext {
	return globalCmdCtx
}

// setCommandContext sets the global command context (primarily for testing).
func setCommandContext(ctx *CommandContext) {
	globalCmdCtx = ctx
}

// defaultCommandContext creates a new CommandContext with default implementations.
func defaultCommandContext() *CommandContext {
	return &CommandContext{
		ConfigLoader:             &common.DefaultConfigLoader{},
		HomeDirProvider:          &common.DefaultHomeDirProvider{},
		KnowledgeStoreFactory:    &common.DefaultKnowledgeStoreFactory{},
		PermissionManagerFactory: &common.DefaultPermissionManagerFactory{},
	}
}

// init initializes the global command context with defaults.
func init() {
	globalCmdCtx = defaultCommandContext()
}
