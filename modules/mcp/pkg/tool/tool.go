// Package tool provides MCP tool adapter implementing tools.Tool.
package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/mcp/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
)

// MCPTool wraps an MCP-discovered tool as an Aura tools.Tool.
type MCPTool struct {
	serverName  string
	toolName    string
	fullName    string
	description string
	inputSchema map[string]any
	callFn      func(ctx context.Context, args map[string]any) (string, error)
}

// NewMCPTool creates a new MCP tool wrapper.
func NewMCPTool(serverName, toolName, description string, inputSchema map[string]any,
	callFn func(ctx context.Context, args map[string]any) (string, error)) *MCPTool {
	t := &MCPTool{
		serverName:  serverName,
		toolName:    toolName,
		fullName:    fmt.Sprintf("%s%s%s%s", constants.ToolNamePrefix, serverName, constants.ToolNameSeparator, toolName),
		description: description,
		inputSchema: inputSchema,
		callFn:      callFn,
	}
	// Pre-compute the full description for caching
	t.description = t.buildDescription()
	return t
}

// Name returns the fully qualified tool name (mcp__{server}__{tool}).
func (t *MCPTool) Name() string {
	return t.fullName
}

// Description returns the cached tool description.
func (t *MCPTool) Description() string {
	return t.description
}

// buildDescription constructs the tool description with input schema information.
func (t *MCPTool) buildDescription() string {
	var sb strings.Builder
	sb.WriteString(t.description)
	if t.inputSchema != nil {
		sb.WriteString("\n\nParameters:\n")
		if props, ok := t.inputSchema["properties"].(map[string]any); ok {
			for name, prop := range props {
				if pm, ok := prop.(map[string]any); ok {
					sb.WriteString(fmt.Sprintf("- %s", name))
					if typ, ok := pm["type"].(string); ok {
						sb.WriteString(fmt.Sprintf(" (%s)", typ))
					}
					if desc, ok := pm["description"].(string); ok && desc != "" {
						sb.WriteString(fmt.Sprintf(": %s", desc))
					}
					if reqs, ok := t.inputSchema["required"].([]any); ok {
						for _, r := range reqs {
							if r == name {
								sb.WriteString(" (required)")
								break
							}
						}
					}
					sb.WriteString("\n")
				}
			}
		}
	}
	return sb.String()
}

// Execute calls the underlying MCP tool.
func (t *MCPTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	if t.callFn == nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("MCP tool %s has no call function", t.fullName)}, nil
	}
	output, err := t.callFn(ctx, params)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: output}, nil
}

// PermissionLevel returns the permission level for this tool.
// MCP tools can execute arbitrary operations via external servers, so they require confirmation.
func (t *MCPTool) PermissionLevel() string {
	return "execute"
}

// ServerName returns the MCP server name for this tool.
func (t *MCPTool) ServerName() string {
	return t.serverName
}

// ToolName returns the original MCP tool name.
func (t *MCPTool) ToolName() string {
	return t.toolName
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *MCPTool) InputSchema() map[string]any {
	return t.inputSchema
}
