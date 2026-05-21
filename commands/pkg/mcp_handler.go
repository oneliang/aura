// Package commands provides internal command execution for the agent system.
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MCPInfo represents runtime state of an MCP server for display.
type MCPInfo struct {
	Name      string
	Command   string
	Args      string
	Status    string
	ToolCount int
	Error     string
	LastSeen  time.Time
}

// ListServersFunc is the callback type for listing MCP servers.
type ListServersFunc func() []MCPInfo

// MCPHandler handles MCP-related commands.
type MCPHandler struct {
	listServersFn ListServersFunc
}

// NewMCPHandler creates a new MCP handler.
func NewMCPHandler(listServersFn ListServersFunc) *MCPHandler {
	return &MCPHandler{
		listServersFn: listServersFn,
	}
}

// SetListServersFunc sets the server list callback function.
func (h *MCPHandler) SetListServersFunc(fn ListServersFunc) {
	h.listServersFn = fn
}

// ExecuteCommand executes an MCP command.
func (h *MCPHandler) ExecuteCommand(_ context.Context, cmd string, _ map[string]any) (string, error) {
	switch cmd {
	case "list":
		return h.listServers()
	default:
		return "", fmt.Errorf("unknown MCP command: %s", cmd)
	}
}

func (h *MCPHandler) listServers() (string, error) {
	if h.listServersFn == nil {
		return "MCP not configured", nil
	}

	servers := h.listServersFn()
	if len(servers) == 0 {
		return "No MCP servers configured", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("MCP Servers (%d):\n", len(servers)))
	for _, s := range servers {
		statusIcon := "stopped"
		if s.Status == "running" {
			statusIcon = "running"
		}
		b.WriteString(fmt.Sprintf("  %-12s [%s]  tools: %d\n", s.Name, statusIcon, s.ToolCount))
		if s.Args != "" {
			b.WriteString(fmt.Sprintf("               %s\n", s.Args))
		}
		if s.Status != "running" && s.Error != "" {
			b.WriteString(fmt.Sprintf("               Error: %s\n", s.Error))
		}
	}
	return b.String(), nil
}
