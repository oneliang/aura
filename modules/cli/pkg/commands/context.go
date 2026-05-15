// Package commands provides CLI commands.
package commands

import (
	"github.com/oneliang/aura/cli/pkg/common"
	"github.com/oneliang/aura/commands/pkg"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/user"
)

// InitUserContext loads the current user from users.yaml and sets UserID in context.
// Safe to call multiple times - returns immediately if already initialized.
func InitUserContext(ctx *CommandContext) {
	if ctx.UserID != "" {
		return // Already initialized
	}
	ctx.UserID = user.GetDefaultUserID()
}

// CommandContext holds all dependencies for CLI commands.
type CommandContext struct {
	// Runtime dependencies (populated at runtime)
	Config          *config.Config
	Logger          *logger.Logger
	PermissionMgr   *sdk.PermissionManager
	CommandProvider commands.Command // Unified command provider

	// Factory interfaces (can be mocked for testing)
	HomeDirProvider          HomeDirProvider
	KnowledgeStoreFactory    KnowledgeStoreFactory
	PermissionManagerFactory PermissionManagerFactory
	ConfigLoader             ConfigLoader

	// TUI confirmation channel
	TUIConfirmCh chan TUIConfirmationRequest

	// Current user ID (empty for legacy single-user mode)
	UserID string
}

// TUIConfirmationRequest represents a request for user confirmation in TUI mode.
type TUIConfirmationRequest struct {
	ToolName   string
	Params     map[string]any
	ResponseCh chan bool
}

// globalCmdCtx is the global command context.
var globalCmdCtx *CommandContext

// GetCommandContext returns the global command context.
func GetCommandContext() *CommandContext {
	return globalCmdCtx
}

// SetCommandContext sets the global command context (primarily for testing).
func SetCommandContext(ctx *CommandContext) {
	globalCmdCtx = ctx
}

// DefaultCommandContext creates a new CommandContext with default implementations.
func DefaultCommandContext() *CommandContext {
	return &CommandContext{
		ConfigLoader:             &common.DefaultConfigLoader{},
		HomeDirProvider:          &common.DefaultHomeDirProvider{},
		KnowledgeStoreFactory:    &common.DefaultKnowledgeStoreFactory{},
		PermissionManagerFactory: &common.DefaultPermissionManagerFactory{},
	}
}

// init initializes the global command context with defaults.
func init() {
	globalCmdCtx = DefaultCommandContext()
}
