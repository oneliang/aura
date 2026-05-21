// Package commands provides CLI commands for session management.
package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/oneliang/aura/session/pkg/service"
	"github.com/oneliang/aura/shared/pkg/constants"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/spf13/cobra"
)

var (
	sessionTrigger string
	sessionSource  string
	sessionRole    string
	sessionPrompt  string
)

// SessionCmd is the root command for session management.
var SessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
	Long:  `Commands for managing conversation sessions.`,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run:   runSessionList,
}

var sessionCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new session",
	Args:  cobra.MaximumNArgs(1),
	Run:   runSessionCreate,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionShow,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionDelete,
}

var sessionSubscribeCmd = &cobra.Command{
	Use:   "subscribe <id>",
	Short: "Add subscription to a session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionSubscribe,
}

var sessionUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update session configuration",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionUpdate,
}

func init() {
	sessionCreateCmd.Flags().StringVar(&sessionRole, "role", "", "Role name (loads ~/.aura/roles/{role}.md as system prompt)")
	sessionUpdateCmd.Flags().StringVar(&sessionRole, "role", "", "Role name (loads ~/.aura/roles/{role}.md as system prompt)")
	sessionUpdateCmd.Flags().StringVar(&sessionPrompt, "prompt", "", "Custom system prompt (overrides role)")
	sessionSubscribeCmd.Flags().StringVar(&sessionTrigger, "trigger", "", "Trigger keyword")
	sessionSubscribeCmd.Flags().StringVar(&sessionSource, "source", "*", "Trigger source (feishu/email/cron/api/*)")
	SessionCmd.AddCommand(sessionListCmd)
	SessionCmd.AddCommand(sessionCreateCmd)
	SessionCmd.AddCommand(sessionShowCmd)
	SessionCmd.AddCommand(sessionDeleteCmd)
	SessionCmd.AddCommand(sessionSubscribeCmd)
	SessionCmd.AddCommand(sessionUpdateCmd)
}

// getSessionsDir returns the Aura sessions directory.
func getSessionsDir() (string, error) {
	return ffp.MustAuraHomePath(constants.DirSessions), nil
}

func runSessionList(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Use CommandProvider for session listing
	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_sessions", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runSessionCreate(cmd *cobra.Command, args []string) {
	name := "Untitled"
	if len(args) > 0 {
		name = args[0]
	}

	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	params := map[string]any{
		"name": name,
		"role": sessionRole,
	}

	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_session_create", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runSessionShow(cmd *cobra.Command, args []string) {
	sessionID := args[0]

	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Use CommandProvider for session details
	params := map[string]any{"id": sessionID}
	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_session_show", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Additional: Get messages from store (CLI-specific display)
	dataDir, err := getSessionsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	svc, err := service.NewServiceFromDataDir(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize session service: %v\n", err)
		os.Exit(1)
	}

	messages, err := svc.GetMessages(cmd.Context(), sessionID, 20, cmdCtx.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get messages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)

	if len(messages) > 0 {
		fmt.Printf("\n  Recent messages (%d):\n", len(messages))
		for _, msg := range messages {
			timestamp := time.UnixMilli(msg.Timestamp).Format("15:04:05")
			role := msg.Role
			if role == "user" {
				role = "You"
			}
			// Extract text from ContentBlocks
			var textContent string
			for _, block := range msg.ContentBlocks {
				if tb, ok := block.(sharedmemory.TextBlock); ok {
					textContent = tb.Text
					break
				}
			}
			fmt.Printf("    [%s] %s: %s\n", timestamp, role, utils.Truncate(textContent, 60))
		}
	}
}

func runSessionDelete(cmd *cobra.Command, args []string) {
	sessionID := args[0]

	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	params := map[string]any{"id": sessionID}
	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_session_delete", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runSessionSubscribe(cmd *cobra.Command, args []string) {
	sessionID := args[0]

	if sessionTrigger == "" {
		fmt.Fprintf(os.Stderr, "Error: --trigger is required\n")
		os.Exit(1)
	}

	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	params := map[string]any{
		"session_id": sessionID,
		"trigger":    sessionTrigger,
		"source":     sessionSource,
	}

	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_subscription_add", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runSessionUpdate(cmd *cobra.Command, args []string) {
	sessionID := args[0]

	// Validate flags: --role and --prompt are mutually exclusive
	if sessionRole != "" && sessionPrompt != "" {
		fmt.Fprintf(os.Stderr, "Error: --role and --prompt cannot be used together\n")
		os.Exit(1)
	}

	if sessionRole == "" && sessionPrompt == "" {
		fmt.Fprintf(os.Stderr, "Error: either --role or --prompt must be specified\n")
		os.Exit(1)
	}

	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Note: Current SessionHandler only supports role update
	// For prompt update, we need to handle it separately or extend the handler
	params := map[string]any{
		"id":   sessionID,
		"role": sessionRole,
	}

	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_session_update", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}
