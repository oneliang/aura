// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import "context"

// Command is the interface for command providers.
// This interface defines the contract for executing internal commands.
type Command interface {
	// GetCommands returns all available commands with their metadata.
	GetCommands() []CommandInfo

	// Execute executes a command by name with the given parameters.
	Execute(ctx context.Context, cmd string, params map[string]any) (string, error)
}

// CommandInfo holds metadata about a command.
type CommandInfo struct {
	Name        string      // Command identifier, e.g., "command_exit"
	DisplayName string      // Display name (i18n translated), e.g., "Exit"
	Description string      // Description (i18n translated)
	Params      []ParamInfo // Parameter descriptions for commands that accept parameters
}

// GetName returns the command name.
func (c CommandInfo) GetName() string {
	return c.Name
}

// GetDescription returns the command description.
func (c CommandInfo) GetDescription() string {
	return c.Description
}

// ParamInfo describes a command parameter.
type ParamInfo struct {
	Name     string // Parameter name
	Type     string // Parameter type (string, int, bool, etc.)
	Required bool   // Whether the parameter is required
	Desc     string // Parameter description
}
