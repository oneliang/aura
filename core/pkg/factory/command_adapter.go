// Package factory provides factories for creating core components.
package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
)

// CommandTool wraps a commands.Command as a tools.Tool.
// This allows the LLM to call internal commands through the ReAct loop.
type CommandTool struct {
	provider commands.Command
}

// NewCommandTool creates a new command tool from a commands.Command.
func NewCommandTool(provider commands.Command) *CommandTool {
	return &CommandTool{provider: provider}
}

// Name returns the tool name.
func (t *CommandTool) Name() string {
	return "internal_command"
}

// Description returns the tool description with available commands.
func (t *CommandTool) Description() string {
	cmds := t.provider.GetCommands()
	var sb strings.Builder
	sb.WriteString(i18n.T("internal_command.tool.desc"))
	sb.WriteString("\n\n")
	sb.WriteString(i18n.T("internal_command.tool.available"))
	sb.WriteString(":\n")
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", cmd.GetName(), cmd.GetDescription()))
	}
	return sb.String()
}

// PermissionLevel returns the permission level for this tool.
// Commands can execute arbitrary operations (including bash), so they require confirmation.
func (t *CommandTool) PermissionLevel() string {
	return "execute"
}

// Execute executes an internal command.
// Parameters:
//   - command: The command name (e.g., "command_session_create")
//   - params: Command-specific parameters as a map
func (t *CommandTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	if t.provider == nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "command provider not configured"}, nil
	}

	// Extract command name
	cmdName, ok := params["command"].(string)
	if !ok || cmdName == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "missing 'command' parameter"}, nil
	}

	// Extract command parameters if provided
	cmdParams, ok := params["params"].(map[string]any)
	if !ok {
		cmdParams = make(map[string]any)
	}

	// Execute the command
	result, err := t.provider.Execute(ctx, cmdName, cmdParams)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: result}, nil
}

// Timeout returns a longer timeout for internal commands since some (like agent delegation)
// can take several minutes to complete. Most internal commands finish quickly, but agent
// delegation needs up to 10 minutes.
func (t *CommandTool) Timeout() time.Duration {
	return constants.DefaultAgentDelegationTimeout
}

// MarshalCommandParams converts command parameters to JSON string for LLM consumption.
func MarshalCommandParams(params map[string]any) (string, error) {
	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalCommandParams parses JSON string to command parameters.
func UnmarshalCommandParams(data string) (map[string]any, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(data), &params); err != nil {
		return nil, err
	}
	return params, nil
}
