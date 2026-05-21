// Package init provides types for the init command.
package init

import "context"

// CommandInfo holds metadata about a command.
type CommandInfo struct {
	Name        string
	DisplayName string
	Description string
	Params      []ParamInfo
}

// ParamInfo describes a command parameter.
type ParamInfo struct {
	Name     string
	Type     string
	Required bool
	Desc     string
}

// Command interface for init handler.
type Command interface {
	GetCommands() []CommandInfo
	Execute(ctx context.Context, cmd string, params map[string]any) (string, error)
}