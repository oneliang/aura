// Package cli provides the CLI application entry point.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oneliang/aura/cli/pkg/commands"
	"github.com/oneliang/aura/cli/pkg/tui"
	cmds "github.com/oneliang/aura/commands/pkg"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/user"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/oneliang/aura/shared/pkg/version"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	useTUI      bool
	useCLI      bool
	noTools     bool
	autoConfirm bool
)

// rootCmd is the main command for aura.
var rootCmd = &cobra.Command{
	Use:   "aura [message]",
	Short: "Aura - Your personal AI assistant",
	Long: `Aura is a personal AI assistant that learns from you and helps with various tasks.

It supports multiple modes:
- Interactive CLI: Run 'aura' without arguments
- Single query: Run 'aura "your message"'
- TUI mode: Run 'aura --tui' for full terminal interface
- Chat-only: Run 'aura --no-tools' to disable tools`,
	Args: cobra.ArbitraryArgs,
	Run:  runAgent,
}

// Execute runs the root command.
func Execute() {
	if getCommandContext() == nil {
		setCommandContext(defaultCommandContext())
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.aura/config.yaml)")
	rootCmd.Flags().BoolVar(&useTUI, "tui", false, "force use interactive TUI (default)")
	rootCmd.Flags().BoolVar(&useCLI, "cli", false, "use CLI mode instead of TUI")
	rootCmd.Flags().BoolVar(&noTools, "no-tools", false, "disable tool usage")
	rootCmd.Flags().BoolVar(&autoConfirm, "auto-confirm", false, "automatically confirm sensitive tool operations (use with caution)")

	rootCmd.AddCommand(commands.SessionCmd)
	rootCmd.AddCommand(commands.ServeCmd)
	rootCmd.AddCommand(commands.KnowledgeCmd)
	rootCmd.AddCommand(commands.ProfileCmd)
	rootCmd.AddCommand(commands.SkillsCmd)
	rootCmd.AddCommand(commands.OrchestratorCmd)
	rootCmd.AddCommand(commands.UserCmd)
	rootCmd.AddCommand(commands.HabitCmd)
	rootCmd.AddCommand(commands.MCPCmd)
	rootCmd.AddCommand(commands.InitCmd)

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	ctx := getCommandContext()
	if ctx == nil {
		ctx = defaultCommandContext()
		setCommandContext(ctx)
	}

	var err error
	ctx.Config, err = ctx.ConfigLoader.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if ctx.PermissionMgr == nil {
		permCfg := ConvertPermissionsConfig(ctx.Config.Permissions)
		ctx.PermissionMgr, err = ctx.PermissionManagerFactory.NewManager(permCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating permission manager: %v\n", err)
			os.Exit(1)
		}
	}

	ctx.Logger = createLogger(ctx.Config)
}

// runAgent is the main entry point for the unified agent.
func runAgent(cmd *cobra.Command, args []string) {
	startTime := time.Now()
	logger.RegistryDefault().Debug("[DIAG] runAgent: starting")

	runCtx, cancel := context.WithCancel(context.Background())

	setupSignalHandler(runCtx, cancel)
	logger.RegistryDefault().Debug("[DIAG] runAgent: signal handler setup done", "elapsed", time.Since(startTime))

	initConfig()
	ctx := getCommandContext()
	logger.RegistryDefault().Debug("[DIAG] runAgent: config init done", "elapsed", time.Since(startTime))

	// Get default user from config (empty = legacy single-user mode)
	ctx.UserID = user.GetDefaultUserID()

	// Resolve session data directory for task persistence
	if dataDir, err := getDataDir(); err == nil {
		ctx.DataDir = dataDir
	}

	// Check directory trust
	checkDirectoryTrust(ctx, runCtx, useTUI)
	logger.RegistryDefault().Debug("[DIAG] runAgent: directory trust check done", "elapsed", time.Since(startTime))

	// Load profile and initialize session
	prof := loadProfile(ctx.UserID)
	sessionMgr, currentSessionID := initSessionServiceWrapper(ctx.Config)
	logger.RegistryDefault().Debug("[DIAG] runAgent: session service init done", "elapsed", time.Since(startTime))

	// Create command provider (creates SkillLoader internally)
	cmdProvider := createCommandProvider(ctx, prof, sessionMgr)
	logger.RegistryDefault().Debug("[DIAG] runAgent: command provider created", "elapsed", time.Since(startTime))

	// Determine mode BEFORE createRuntime (so correct channel mode is used)
	if len(args) > 0 && !cmd.Flags().Changed("tui") {
		useCLI = true
	}

	// Create and initialize runtime
	rt, mcpManager, sharedEventCh := createRuntime(ctx, currentSessionID, cmdProvider, sessionMgr)
	logger.RegistryDefault().Debug("[DIAG] runAgent: runtime created", "elapsed", time.Since(startTime))
	if rt == nil {
		os.Exit(1)
	}

	// Defer order matters: cancel() first (stop goroutines), then rt.Shutdown()
	// signal handler goroutine waits on ctx.Done(), so cancel must come before Shutdown
	// defer executes in reverse order (LIFO), so rt.Shutdown() is registered first, cancel() second
	// execution order: cancel() → rt.Shutdown()
	defer rt.Shutdown()
	defer cancel()
	// Close sharedEventCh after runtime shutdown (LIFO: registered first, executes last)
	if sharedEventCh != nil {
		defer func() { close(sharedEventCh) }()
	}

	// Inject MCP list function into command provider
	if mcpManager != nil {
		cmdProvider.SetMCPListFunc(mcpListServersAdapter(mcpManager))
	}

	// Setup event-driven command handlers
	setupCommandProviderHandlers(cmdProvider, rt.GetMemory(), rt, ctx, runCtx, currentSessionID, sessionMgr)
	ctx.CommandProvider = cmdProvider
	logger.RegistryDefault().Debug("[DIAG] runAgent: command handlers setup done", "elapsed", time.Since(startTime))

	// Run appropriate mode
	if len(args) > 0 {
		runSingleMessage(runCtx, rt, args, createCLIEventHandler())
		return
	}

	// Interactive mode
	sl := tui.LoadSessionLearner()
	logger.RegistryDefault().Debug("[DIAG] runAgent: session learner loaded", "elapsed", time.Since(startTime))
	if !useCLI {
		runTUIMode(runCtx, rt, sl, sessionMgr, currentSessionID, ctx, prof.DisplayName(), mcpManager, sharedEventCh)
		return
	}

	// CLI interactive mode
	runInteractiveWithRuntime(runCtx, rt, sl, sessionMgr, mcpManager)
}

// loadProfile loads the user profile for the current user.
func loadProfile(userID string) *profile.Profile {
	var profilePath string

	if userID != "" {
		// Load user-specific profile from users config
		usersCfg, err := user.LoadConfig()
		if err == nil {
			for _, u := range usersCfg.Definitions {
				if u.ID == userID && u.ProfilePath != "" {
					profilePath = u.ProfilePath
					break
				}
			}
		}
	}

	// Fallback to default profile path
	if profilePath == "" {
		profilePath = ffp.MustAuraHomePath(constants.DefaultProfileFile)
	}

	prof, _ := profile.Load(profilePath)
	return prof
}

// initSessionServiceWrapper initializes session manager and returns current session ID.
func initSessionServiceWrapper(cfg *config.Config) (*sdk.SessionManager, string) {
	userID := getCommandContext().UserID
	dataDir, err := getDataDir()
	if err != nil {
		return nil, ""
	}
	mgr, id, err := initSessionManager(dataDir, cfg, userID)
	if err != nil {
		return nil, ""
	}
	return mgr, id
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		ctx := getCommandContext()
		fmt.Println("Current configuration:")
		fmt.Printf("  LLM Provider: %s\n", ctx.Config.LLM.Provider)
		fmt.Printf("  LLM Base URL: %s\n", ctx.Config.LLM.BaseURL)
		fmt.Printf("  LLM Model: %s\n", ctx.Config.LLM.Model)
		fmt.Printf("  Log Level: %s\n", ctx.Config.Log.Level)
		fmt.Printf("  Enabled Tools: %v\n", ctx.Config.Tools.Enabled)
	},
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available tools:")
		fmt.Println("\nFile System:")
		fmt.Println("  file_read   - Read file contents")
		fmt.Println("  file_write  - Write to files (requires confirmation)")
		fmt.Println("  file_search - Search in files")
		fmt.Println("  file_list   - List directory contents")
		fmt.Println("\nSearch:")
		fmt.Println("  glob        - File pattern matching (e.g., '**/*.go')")
		fmt.Println("  grep        - Regex content search (requires ripgrep for best performance)")
		fmt.Println("\nSystem:")
		fmt.Println("  bash        - Execute shell commands (requires confirmation)")
		fmt.Println("\nRemote (SSH):")
		fmt.Println("  ssh_exec    - Execute commands on remote servers (requires confirmation)")
		fmt.Println("\nWeb:")
		fmt.Println("  web_fetch   - Fetch content from URLs")
		fmt.Println("  web_search  - Search the web")
		fmt.Println("\nUtility:")
		fmt.Println("  datetime    - Get date and time information")
		fmt.Println("  calculator  - Perform calculations")
		fmt.Println("  text        - Text manipulation")
		fmt.Println("  ask_user_question - Proactively ask user questions for clarification")
		fmt.Println("\nKnowledge:")
		fmt.Println("  knowledge_search - Search personal knowledge base")
		fmt.Println("  knowledge_import - Import documents to knowledge base")
		fmt.Println("\nCode Navigation:")
		fmt.Println("  code_navigate    - LSP-based code navigation (requires gopls)")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Aura %s\n", version.FullVersion())
	},
}

// runInteractiveWithRuntime runs the CLI interactive loop using agent runtime.
func runInteractiveWithRuntime(ctx context.Context, rt *sdk.Runtime, sl *tui.SessionLearner, sessionMgr *sdk.SessionManager, mcpManager *sdk.MCPManager) {
	cmdCtx := getCommandContext()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Aura - Your Personal AI Assistant")
	fmt.Println("Type your message and press Enter. Type /exit to quit.")
	if sessionMgr != nil {
		items, _ := sessionMgr.ListSessions()
		if len(items) > 0 {
			current := items[0]
			for _, item := range items[1:] {
				if item.Updated > current.Updated {
					current = item
				}
			}
			fmt.Printf("Current session: %s\n", current.Name)
		}
	}
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch input {
		case commands.CmdExit, commands.CmdQuit, commands.CmdQuickExit:
			fmt.Println("\nGoodbye!")
			return
		case commands.CmdClear:
			fmt.Print("\033[2J\033[H")
			continue
		case commands.CmdHelp:
			printHelpWithSessions()
			continue
		case commands.CmdSessions:
			if sessionMgr != nil {
				items, err := sessionMgr.ListSessions()
				if err != nil {
					fmt.Printf("Error listing sessions: %v\n", err)
				} else {
					fmt.Println(renderSessionListFromSDK(items))
				}
			} else {
				fmt.Println("Session manager not available")
			}
			continue
		case commands.CmdSkills:
			cmdCtx := getCommandContext()
			if cmdCtx != nil && cmdCtx.CommandProvider != nil {
				result, err := cmdCtx.CommandProvider.Execute(ctx, cmds.CmdNameSkills, nil)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println(result)
				}
			} else {
				fmt.Println("Skills not available")
			}
			continue
		case commands.CmdTools:
			if err := execCmd(cmdCtx, cmds.CmdNameTools); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdConfig:
			if err := execCmd(cmdCtx, cmds.CmdNameConfigShow); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdKnowledge:
			if err := execCmd(cmdCtx, cmds.CmdNameKnowledgeSearch); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdProfile:
			if err := execCmd(cmdCtx, cmds.CmdNameProfileShow); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdMemory:
			if err := execCmd(cmdCtx, cmds.CmdNameMemory); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdModel:
			if err := execCmd(cmdCtx, cmds.CmdNameModel); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdCompact:
			if err := execCmd(cmdCtx, cmds.CmdNameCompact); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdRole:
			if err := execCmd(cmdCtx, cmds.CmdNameRole); err != nil {
				fmt.Println(err)
			}
			continue
		case commands.CmdMcp:
			fmt.Println(renderMCPList(mcpManager))
			continue
		case commands.CmdVersion:
			fmt.Printf("Aura %s\n", version.FullVersion())
			continue
		}

		if sl != nil {
			sl.Observe(input)
		}

		// Start runtime event stream
		if err := rt.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		logger.RegistryDefault().Debug("[CLI_EVENT] rt.Start completed", "error", err)

		// Send user input event
		requestID := fmt.Sprintf("cli_%d", time.Now().UnixNano())
		userEvent := events.NewEvent(events.EventTypeUserInput, input, requestID)
		logger.RegistryDefault().Debug("[CLI_EVENT] sending EventTypeUserInput", "requestID", requestID, "input", input)
		if err := rt.SendEvent(ctx, userEvent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			rt.Stop(ctx)
			return
		}
		logger.RegistryDefault().Debug("[CLI_EVENT] SendEvent completed", "error", err)

		// Process events from stream
		logger.RegistryDefault().Debug("[CLI_EVENT] starting event loop, waiting for rt.Events()")
		eventCount := 0
		eventsLoop:
		for {
			select {
			case <-ctx.Done():
				// Context cancelled (Ctrl+C or /exit)
				logger.RegistryDefault().Debug("[CLI_EVENT] ctx.Done received, breaking loop")
				break eventsLoop
			case ev, ok := <-rt.Events():
				if !ok {
					// Channel closed
					logger.RegistryDefault().Debug("[CLI_EVENT] rt.Events channel closed, breaking loop")
					break eventsLoop
				}
				eventCount++
				logger.RegistryDefault().Debug("[CLI_EVENT] received event", "count", eventCount, "type", ev.Type(), "content_len", len(ev.Content()))
				switch ev.Type() {
				case sdk.EventTypeThinkingStart:
					fmt.Printf("\033[90m%s\033[0m\n", ev.Content())
				case sdk.EventTypeThinkingChunk:
					// Accumulate thinking content (display in gray, no newline for streaming)
					fmt.Printf("\033[90m%s\033[0m", ev.Content())
				case sdk.EventTypeThinkingEnd:
					// Complete thinking - print newline if any content was displayed
					fmt.Println()
				case sdk.EventTypeResponseChunk:
					// Ignore chunks in CLI mode - wait for complete response
				case sdk.EventTypeResponse:
					fmt.Printf("Aura: %s\n\n", ev.Content())
				case sdk.EventTypeAction:
					fmt.Printf("\033[33m%s\033[0m\n", ev.Content())
				case sdk.EventTypeResult:
					fmt.Printf("\033[36m%s\033[0m\n", ev.Content())
				case sdk.EventTypeTaskCreate, sdk.EventTypeTaskUpdate, sdk.EventTypeTaskList:
					fmt.Print(formatTaskEvent(ev))
				case sdk.EventTypeStep:
					stepNum := ""
					if s, ok := ev.Extra()["step"]; ok {
						stepNum = fmt.Sprintf(" %v", s)
					}
					fmt.Printf("\033[90m[%s%s]\033[0m\n", i18n.T("cli.step_label"), stepNum)
				case sdk.EventTypePlanCreated:
					fmt.Printf("\033[36m[%s] %s (%d steps)\033[0m\n", i18n.T("cli.plan_label"), ev.Extra()["goal"], ev.Extra()["total_steps"])
				case sdk.EventTypePlanStep:
					fmt.Printf("\033[36m[%s %d/%d] %s\033[0m\n", i18n.T("cli.plan_step_label"), ev.Extra()["step_num"], ev.Extra()["total_steps"], ev.Extra()["step_desc"])
				case sdk.EventTypePlanComplete:
					fmt.Printf("\033[32m[%s] %s\033[0m\n", i18n.T("cli.plan_complete_label"), ev.Content())
				case sdk.EventTypeError:
					fmt.Fprintf(os.Stderr, "\033[31mError: %s\033[0m\n", ev.Content())
				case sdk.EventTypeDone:
					// Done event signals completion
					break eventsLoop
				}
			}
		}
		rt.Stop(ctx)
		// Print newline after response complete
		fmt.Println()
	}
}

// execCmd executes a command via the CommandProvider and returns the result or error.
func execCmd(cmdCtx *CommandContext, cmdName string) error {
	if cmdCtx == nil || cmdCtx.CommandProvider == nil {
		return fmt.Errorf("command provider not available")
	}
	result, err := cmdCtx.CommandProvider.Execute(context.Background(), cmdName, nil)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

// renderMCPList returns a formatted string of MCP servers.
func renderMCPList(mgr *sdk.MCPManager) string {
	if mgr == nil {
		return "  No MCP servers configured."
	}
	infos := mgr.ListServers()
	if len(infos) == 0 {
		return "  No MCP servers configured."
	}
	var b strings.Builder
	b.WriteString("  MCP Servers:\n")
	for _, info := range infos {
		b.WriteString(fmt.Sprintf("    %-12s %s  %s  tools: %d\n", info.Name, info.Command, info.Status, info.ToolCount))
		if len(info.Args) > 0 && info.Args[0] != "" {
			b.WriteString(fmt.Sprintf("                 %s\n", info.Args[0]))
		}
	}
	return b.String()
}

// printHelpWithSessions prints help message with session commands.
func printHelpWithSessions() {
	fmt.Print(`
Commands:
  /exit, /quit, /q  - Exit the chat
  /clear            - Clear the screen
  /help             - Show this help message
  /sessions         - List all sessions
  /session create   - Create a new session
  /skills           - List loaded skills
  /tools            - List available tools
  /config           - Show configuration
  /knowledge        - Search knowledge base
  /profile          - Show user profile
  /memory           - Show memory usage
  /model            - Show model info
  /compact          - Compact conversation
  /role             - Show/set session role
  /mcp              - List MCP servers
  /version          - Show version
`)
}

// renderSessionList renders a session list for CLI display.
func renderSessionListFromSDK(sessions []sdk.SessionInfo) string {
	if len(sessions) == 0 {
		return "  No sessions found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sessions (%d):\n\n", len(sessions)))

	for i, s := range sessions {
		updatedStr := time.UnixMilli(s.Updated).Format("2006-01-02 15:04")
		sb.WriteString(fmt.Sprintf("  %d. %s  [%s]\n", i+1, s.Name, updatedStr))
	}

	return sb.String()
}
