// Package config provides MCP server configuration types.
package config

import (
	"time"

	"github.com/oneliang/aura/mcp/pkg/constants"
)

// Config represents the MCP configuration (compatible with Claude Code mcp.json format).
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig represents a single MCP server configuration.
type ServerConfig struct {
	// stdio fields
	Command string            `json:"command"`       // executable to run
	Args    []string          `json:"args"`          // command-line arguments
	Env     map[string]string `json:"env,omitempty"` // environment variables
	// HTTP fields
	Type    string            `json:"type,omitempty"`    // "stdio" (default) or "http"
	URL     string            `json:"url,omitempty"`     // HTTP/SSE endpoint URL
	Headers map[string]string `json:"headers,omitempty"` // HTTP headers (e.g. Authorization)
	// common
	Disabled bool          `json:"disabled,omitempty"` // skip this server
	Timeout  time.Duration `json:"timeout,omitempty"`  // max wait time for tool call
}

// IsHTTP returns true if this server uses HTTP/SSE transport.
func (s ServerConfig) IsHTTP() bool {
	return s.Type == "http" || s.URL != ""
}

// GetTimeout returns the configured timeout or DefaultTimeout.
func (s ServerConfig) GetTimeout() time.Duration {
	if s.Timeout <= 0 {
		return constants.DefaultTimeout
	}
	return s.Timeout
}

// ServerInfo represents runtime state of an MCP server.
type ServerInfo struct {
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Status    string    `json:"status"` // running, stopped, error, crashed
	ToolCount int       `json:"tool_count"`
	Error     string    `json:"error,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
}
