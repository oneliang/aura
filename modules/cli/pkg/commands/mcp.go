// Package commands provides CLI commands for MCP server management.
package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/spf13/cobra"
)

// MCPCmd is the root command for MCP server management.
var MCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  `Commands for adding, removing, and managing MCP (Model Context Protocol) servers.`,
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name> -- <command> [args...]",
	Short: "Add an MCP server",
	Long: `Add an MCP server configuration.

Examples:
  aura mcp add filesystem -- npx -y @modelcontextprotocol/server-filesystem ~/.aura/workspace
  aura mcp add fetch -- uvx mcp-server-fetch
  aura mcp add myserver --command node --args server.js`,
	Args: cobra.MinimumNArgs(1),
	Run:  runMCPAdd,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured MCP servers",
	Run:   runMCPList,
}

var mcpRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Short:   "Remove an MCP server",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	Run:     runMCPRemove,
}

var mcpStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show MCP server status",
	Args:  cobra.MaximumNArgs(1),
	Run:   runMCPStatus,
}

func init() {
	MCPCmd.AddCommand(mcpAddCmd)
	MCPCmd.AddCommand(mcpListCmd)
	MCPCmd.AddCommand(mcpRemoveCmd)
	MCPCmd.AddCommand(mcpStatusCmd)
}

func runMCPAdd(cmd *cobra.Command, args []string) {
	name := args[0]

	// Find the "--" separator
	sepIdx := -1
	for i, a := range args {
		if a == "--" {
			sepIdx = i
			break
		}
	}

	if sepIdx == -1 {
		fmt.Fprintf(os.Stderr, "Error: missing '--' separator. Usage: aura mcp add %s -- <command> [args...]\n", name)
		os.Exit(1)
	}

	if sepIdx+1 >= len(args) {
		fmt.Fprintf(os.Stderr, "Error: no command specified after '--'\n")
		os.Exit(1)
	}

	command := args[sepIdx+1]
	cmdArgs := args[sepIdx+2:]

	ctx := context.Background()
	tools, err := sdk.AddMCPServer(ctx, name, command, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding MCP server '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Added MCP server '%s' (%s)\n", name, command)
	fmt.Printf("Discovered %d tool(s):\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}
}

func runMCPList(cmd *cobra.Command, args []string) {
	cfg, err := sdk.LoadMCPConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading MCP config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.MCPServers) == 0 {
		fmt.Println("No MCP servers configured. Use 'aura mcp add <name> -- <command>' to add one.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND\tARGS\tSTATUS")
	fmt.Fprintln(w, "----\t-------\t----\t------")

	for name, srv := range cfg.MCPServers {
		status := "enabled"
		if srv.Disabled {
			status = "disabled"
		}
		argsStr := strings.Join(srv.Args, " ")
		if argsStr == "" {
			argsStr = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, srv.Command, argsStr, status)
	}
	w.Flush()
}

func runMCPRemove(cmd *cobra.Command, args []string) {
	name := args[0]

	ctx := context.Background()
	if err := sdk.RemoveMCPServer(ctx, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing MCP server '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Removed MCP server '%s'\n", name)
}

func runMCPStatus(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	if len(args) > 0 {
		// Show specific server
		name := args[0]
		info := sdk.GetMCPServerStatus(ctx, name)
		if info == nil {
			fmt.Fprintf(os.Stderr, "Server '%s' not found\n", name)
			os.Exit(1)
		}
		printServerInfo(info)
	} else {
		// Show all
		mgr := sdk.NewMCPManager()
		if mgr == nil {
			fmt.Println("No MCP servers configured.")
			return
		}
		// Ensure cleanup on all exit paths
		defer func() { _ = mgr.StopAll(ctx) }()

		if _, err := mgr.StartAll(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: some servers failed to start: %v\n", err)
		}

		infos := mgr.ListServers()
		if len(infos) == 0 {
			fmt.Println("No MCP servers configured.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCOMMAND\tSTATUS\tTOOLS")
		fmt.Fprintln(w, "----\t-------\t------\t-----")
		for _, info := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", info.Name, info.Command, info.Status, info.ToolCount)
		}
		w.Flush()
	}
}

func printServerInfo(info *sdk.ServerInfo) {
	fmt.Printf("Name:    %s\n", info.Name)
	fmt.Printf("Command: %s\n", info.Command)
	if len(info.Args) > 0 {
		fmt.Printf("Args:    %s\n", strings.Join(info.Args, " "))
	}
	fmt.Printf("Status:  %s\n", info.Status)
	if info.ToolCount > 0 {
		fmt.Printf("Tools:   %d\n", info.ToolCount)
	}
	if info.Error != "" {
		fmt.Printf("Error:   %s\n", info.Error)
	}
}
