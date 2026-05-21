// Package tools provides tool implementations.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ToolStatus represents the execution status of a tool.
type ToolStatus string

const (
	// ToolStatusSuccess indicates the tool executed successfully.
	ToolStatusSuccess ToolStatus = "success"
	// ToolStatusError indicates the tool encountered a logical error (not a crash).
	ToolStatusError ToolStatus = "error"
	// ToolStatusPartial indicates the tool partially succeeded.
	ToolStatusPartial ToolStatus = "partial"
)

// ToolResult is the structured result of a tool execution.
type ToolResult struct {
	Status  ToolStatus     `json:"status"`
	Content string         `json:"content"`        // Human-readable output
	Data    map[string]any `json:"data,omitempty"` // Structured data for engine
	Error   string         `json:"error,omitempty"`
}

// String returns the string representation of the result.
func (r *ToolResult) String() string {
	if r.Status == ToolStatusSuccess {
		return r.Content
	}
	return fmt.Sprintf("Error: %s", r.Error)
}

// JSON returns the JSON representation of the result.
func (r *ToolResult) JSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// Tool defines the interface for tools.
// This interface is compatible with agent.Tool interface.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
}

// OutputSchemaProvider is an optional interface tools can implement to declare
// the JSON schema of their structured output (ToolResult.Data field).
// The engine can use this to validate tool outputs at runtime or inject schema
// hints into tool descriptions for the LLM.
type OutputSchemaProvider interface {
	OutputSchema() map[string]any
}

// TimeoutProvider is an optional interface tools can implement to declare
// their execution timeout. The engine uses this to create an independent
// timeout context for each tool execution.
type TimeoutProvider interface {
	Timeout() time.Duration
}

// BaseTool provides a base implementation for tools.
type BaseTool struct {
	name        string
	description string
	executor    func(ctx context.Context, params map[string]any) (*ToolResult, error)
}

// NewBaseTool creates a new base tool.
func NewBaseTool(name, description string, executor func(ctx context.Context, params map[string]any) (*ToolResult, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		executor:    executor,
	}
}

func (t *BaseTool) Name() string        { return t.name }
func (t *BaseTool) Description() string { return t.description }
func (t *BaseTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	return t.executor(ctx, params)
}
