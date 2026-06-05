// Package system provides system tools.
package system

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/logdir"
	"github.com/oneliang/aura/shared/pkg/logger"
	tools "github.com/oneliang/aura/tools/pkg"
)

// shellLog is the logger for shell tool execution.
var shellLog *logger.Logger

func init() {
	// Create a logger instance for shell tool, writing to shell.log
	logPath, err := logdir.GetLogFile("shell.log")
	if err != nil {
		// Fallback to stderr if log dir unavailable
		shellLog = logger.New(logger.Config{Level: "info", Module: "shell", Output: "stderr"})
		return
	}
	shellLog = logger.New(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "file",
		Path:   logPath,
		Module: "shell",
	})
	// Write a test log entry to verify logger is working
	shellLog.Info("Shell logger initialized", "status", "initialized", "logPath", logPath)
}

// ShellTool executes shell commands.

// ShellTool executes shell commands.
type ShellTool struct {
	allowedCommands []string
	timeout         time.Duration
}

// ShellOption is a configuration option for ShellTool.
type ShellOption func(*ShellTool)

// WithAllowedCommands sets allowed commands.
func WithAllowedCommands(cmds []string) ShellOption {
	return func(t *ShellTool) {
		t.allowedCommands = cmds
	}
}

// WithTimeout sets the command timeout.
func WithTimeout(d time.Duration) ShellOption {
	return func(t *ShellTool) {
		t.timeout = d
	}
}

// NewShellTool creates a new shell tool.
func NewShellTool(opts ...ShellOption) *ShellTool {
	t := &ShellTool{
		timeout: constants.DefaultShellTimeout,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *ShellTool) Name() string {
	return constants.ToolShellExec
}

// Description returns the tool description.
func (t *ShellTool) Description() string {
	return "Execute a shell command. Parameters: command (string, required)"
}

// OutputSchema returns the JSON schema for structured output.
func (t *ShellTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command":   map[string]any{"type": "string"},
			"stdout":    map[string]any{"type": "string"},
			"stderr":    map[string]any{"type": "string"},
			"exit_code": map[string]any{"type": "integer"},
			"cwd":       map[string]any{"type": "string"},
		},
	}
}

// PermissionLevel returns the permission level for this tool.
func (t *ShellTool) PermissionLevel() string {
	return "execute"
}

// RequiresConfirmation returns true because executing shell commands is a sensitive operation.
func (t *ShellTool) RequiresConfirmation() bool {
	return true
}

// Timeout returns the execution timeout for this tool.
func (t *ShellTool) Timeout() time.Duration {
	return t.timeout
}

// Execute runs a shell command.
// Timeout handling: If the context has no deadline and the tool has a timeout configured,
// a timeout context is created. Engine can override via TimeoutProvider interface.
func (t *ShellTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	command, ok := params["command"].(string)
	if !ok {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "command parameter is required"}, nil
	}

	// Check if command is allowed
	if len(t.allowedCommands) > 0 {
		allowed := false
		firstWord := getFirstCommandWord(command)
		for _, c := range t.allowedCommands {
			if firstWord == c {
				allowed = true
				break
			}
		}
		if !allowed {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("command not allowed: %s (first word: %s)", command, firstWord)}, nil
		}
	}

	// Get current working directory for debugging
	cwd, _ := os.Getwd()

	// Create timeout context if context has no deadline and tool has timeout configured
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok && t.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	// Execute command with context
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	stderrStr := stderr.String()
	if stderr.Len() > 0 {
		output += "\nStderr:\n" + stderrStr
	}

	// Log execution result to shell.log
	shellLog.Info("Shell command executed", "command", command, "output", output, "cwd", cwd, "error", err)

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1 // Other error (timeout, etc.)
		}
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("command failed: %v\n%s", err, output),
			Data: map[string]any{
				"command":   command,
				"stdout":    stdout.String(),
				"stderr":    stderrStr,
				"exit_code": exitCode,
				"cwd":       cwd,
			},
		}, nil
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: output,
		Data: map[string]any{
			"command":   command,
			"stdout":    stdout.String(),
			"stderr":    stderrStr,
			"exit_code": exitCode,
			"cwd":       cwd,
		},
	}, nil
}

// getFirstCommandWord extracts the first word of a shell command.
// This is used to safely check allowed commands without being tricked by injection.
// e.g., "ls -la" -> "ls", "ls; rm -rf /" -> "ls", "sudo rm" -> "sudo"
func getFirstCommandWord(command string) string {
	// Trim leading whitespace
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}

	// Find the first space or shell operator
	for i, c := range command {
		if c == ' ' || c == ';' || c == '|' || c == '&' || c == '\n' || c == '\t' {
			return command[:i]
		}
	}
	return command
}

// InfoTool provides system information.
type InfoTool struct{}

// Name returns the tool name.
func (t *InfoTool) Name() string {
	return "system_info"
}

// Description returns the tool description.
func (t *InfoTool) Description() string {
	return "Get system information. No parameters required."
}

// Execute returns system info.
func (t *InfoTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	info := fmt.Sprintf(`System Information:
OS: %s
Architecture: %s
Go Version: %s
`, runtime.GOOS, runtime.GOARCH, runtime.Version())

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: info,
		Data: map[string]any{
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
			"go_version":   runtime.Version(),
		},
	}, nil
}

// OutputSchema returns the JSON schema for structured output.
func (t *InfoTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"os":         map[string]any{"type": "string"},
			"arch":       map[string]any{"type": "string"},
			"go_version": map[string]any{"type": "string"},
		},
	}
}

// AllTools returns all system tools.
func AllTools() []tools.Tool {
	return []tools.Tool{
		NewShellTool(),
		&InfoTool{},
	}
}
