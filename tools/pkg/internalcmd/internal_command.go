// Package internalcmd provides a tool for executing internal commands via natural language.
package internalcmd

import (
	"context"
	"time"

	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	tools "github.com/oneliang/aura/tools/pkg"
)

// CommandExecutor is a callback for executing internal commands.
// It allows the TUI layer to inject the actual command execution logic.
type CommandExecutor func(ctx context.Context, cmdName string, params map[string]any) (string, error)

// InternalCommandTool implements tools.Tool interface for internal command execution.
type InternalCommandTool struct {
	executor CommandExecutor
}

// New creates a new internal command tool with the provided executor callback.
func New(executor CommandExecutor) *InternalCommandTool {
	return &InternalCommandTool{executor: executor}
}

// Name returns the tool name.
func (t *InternalCommandTool) Name() string {
	return "internal_command"
}

// Description returns the tool description (i18n supported).
func (t *InternalCommandTool) Description() string {
	return i18n.T("internal_command.tool.desc")
}

// Execute executes the internal command.
// Parameters:
//   - command: The command name (e.g., "command_exit", "command_help")
//   - params: Command-specific parameters
func (t *InternalCommandTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	if t.executor == nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "command executor not configured"}, nil
	}

	cmdName, ok := params["command"].(string)
	if !ok {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "missing 'command' parameter"}, nil
	}

	// Extract params if provided
	cmdParams, ok := params["params"].(map[string]any)
	if !ok {
		cmdParams = make(map[string]any)
	}

	log := logger.RegistryDefault().WithModule("internal_command")
	log.Debug("Execute: dispatching command", "command", cmdName, "params", cmdParams)
	result, err := t.executor(ctx, cmdName, cmdParams)
	log.Debug("Execute: command returned", "command", cmdName, "result_len", len(result), "error", err)

	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}
	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: result,
	}, nil
}

// Timeout returns a longer timeout for internal commands since some (like agent delegation)
// can take several minutes to complete. Most internal commands finish quickly, but agent
// delegation needs up to 10 minutes.
func (t *InternalCommandTool) Timeout() time.Duration {
	return 10 * time.Minute
}
