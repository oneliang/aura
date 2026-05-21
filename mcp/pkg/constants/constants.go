// Package constants provides MCP-specific constants.
package constants

import "time"

const (
	// ToolNamePrefix is the prefix for MCP-discovered tool names.
	ToolNamePrefix = "mcp__"

	// ToolNameSeparator separates server name and tool name in the full tool name.
	// Format: mcp__{server}__{tool}
	ToolNameSeparator = "__"

	// DefaultTimeout is the default timeout for MCP server operations.
	DefaultTimeout = 60 * time.Second

	// DefaultConfigFile is the default MCP config file name.
	DefaultConfigFile = "mcp.json"
)
