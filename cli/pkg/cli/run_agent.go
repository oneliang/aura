// Package cli provides runAgent function decomposition.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/oneliang/aura/cli/pkg/tui"
	cmds "github.com/oneliang/aura/commands/pkg"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	skillmanager "github.com/oneliang/aura/skill/pkg/manager"
)

// readUserConfirmation prompts the user for confirmation via stdin.
// Returns true if confirmed (y/yes), false otherwise.
func readUserConfirmation(prompt string) bool {
	fmt.Printf("%s", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// setupSignalHandler sets up signal handling for graceful shutdown.
func setupSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nGoodbye!")
		cancel()
	}()
}

// checkDirectoryTrust checks and asks trust for the current directory.
func checkDirectoryTrust(ctx *CommandContext, runCtx context.Context, useTUI bool) {
	if ctx.Config.Permissions.AutoAskTrust && !useTUI {
		needAsk, currentDir, err := ctx.PermissionMgr.CheckAndAskTrust()
		if err == nil && needAsk {
			fmt.Printf("Aura has not been configured to trust the current directory:\n")
			fmt.Printf("  %s\n", currentDir)

			if readUserConfirmation("Do you want to trust this directory? (y/n): ") {
				if err := ctx.PermissionMgr.AddTrustedDir(currentDir); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to add trusted directory: %v\n", err)
				} else {
					ctx.Config.Permissions.TrustedDirs = append(ctx.Config.Permissions.TrustedDirs, currentDir)
					savePath := cfgFile
					if savePath == "" {
						savePath = ffp.MustAuraHomePath(constants.DefaultConfigFile)
					}
					if err := ctx.Config.Save(cfgFile); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
						fmt.Printf("Directory %q added to trusted directories (temporary for this session).\n", currentDir)
					} else {
						fmt.Printf("Directory %q has been added to trusted directories.\n", currentDir)
						fmt.Printf("Configuration saved to: %s\n", savePath)
					}
				}
			} else {
				fmt.Printf("Directory not trusted. File access may be restricted.\n\n")
			}
		}
	}
}

// createCommandProvider creates the command provider with dependencies.
func createCommandProvider(
	ctx *CommandContext,
	prof *profile.Profile,
	sessionMgr *sdk.SessionManager,
) *cmds.CommandProvider {
	var underlyingMgr *manager.SessionManager
	if sessionMgr != nil {
		underlyingMgr = sessionMgr.GetSessionMgr()
	}

	// Create skill loader and manager if skills are enabled
	var skillLoader *skillloader.Loader
	var skillMgr *skillmanager.SkillManager
	if ctx.Config.Skills.Enabled && len(ctx.Config.Skills.Directories) > 0 {
		skillLoader = skillloader.NewLoader(ctx.Config.Skills.Directories)
		skillMgr = skillmanager.NewSkillManager(skillLoader, ctx.Config.Skills.Directories)
		if _, err := skillLoader.Load(); err != nil {
			logger.RegistryDefault().Warn().Err(err).Msg("Failed to load skills for CommandProvider")
			skillLoader = nil
			skillMgr = nil
		}
	}

	return cmds.NewCommandProvider(cmds.CommandProviderDeps{
		SessionMgr:        underlyingMgr,
		Profile:           prof,
		Config:            ctx.Config,
		UserID:            ctx.UserID,
		SkillLoader:       skillLoader,
		SkillManager:      skillMgr,
		AgentDelegateFunc: nil,
	})
}

// mcpListServersAdapter adapts sdk.MCPManager.ListServers to cmds.ListServersFunc.
func mcpListServersAdapter(mcpMgr *sdk.MCPManager) func() []cmds.MCPInfo {
	return func() []cmds.MCPInfo {
		servers := mcpMgr.ListServers()
		result := make([]cmds.MCPInfo, len(servers))
		for i, s := range servers {
			args := ""
			if len(s.Args) > 0 {
				args = s.Args[0]
			}
			result[i] = cmds.MCPInfo{
				Name:      s.Name,
				Command:   s.Command,
				Args:      args,
				Status:    s.Status,
				ToolCount: s.ToolCount,
				Error:     s.Error,
				LastSeen:  s.LastSeen,
			}
		}
		return result
	}
}

// createCLIEventHandler creates an event handler for CLI mode.
func createCLIEventHandler() func(sdk.Event) {
	return func(ev sdk.Event) {
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
			fmt.Printf("\033[32m%s\033[0m\n", i18n.T("cli.assistant_response", ev.Content()))
		case sdk.EventTypeAction:
			fmt.Printf("\033[33m%s\033[0m\n", ev.Content())
		case sdk.EventTypeResult:
			fmt.Printf("\033[36m%s\033[0m\n", ev.Content())
		case sdk.EventTypeError:
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", i18n.T("cli.error", ev.Content()))
		case sdk.EventTypeToolStart:
			fmt.Printf("\033[35m%s\033[0m\n", i18n.T("cli.tool_using", ev.Content()))
		case sdk.EventTypeToolEnd:
			fmt.Printf("\033[35m%s\033[0m\n", i18n.T("cli.tool_completed"))
		case sdk.EventTypeTaskCreate, sdk.EventTypeTaskUpdate, sdk.EventTypeTaskList:
			fmt.Print(formatTaskEvent(ev))
		case sdk.EventTypeStep:
			fmt.Printf("\033[90m[%s] %s\033[0m\n", i18n.T("cli.step_label"), ev.Content())
		case sdk.EventTypePlanCreated:
			fmt.Printf("\033[36m[%s] %s (%d steps)\033[0m\n", i18n.T("cli.plan_label"), ev.Extra()["goal"], ev.Extra()["total_steps"])
		case sdk.EventTypePlanReviewStart:
			fmt.Printf("\033[36m[%s] %s\033[0m\n", i18n.T("cli.review_label"), ev.Content())
		case sdk.EventTypePlanReviewFiles:
			files := ev.Extra()["files"]
			if files != nil {
				fmt.Printf("\033[36m[%s] Files: %v\033[0m\n", i18n.T("cli.review_files_label"), files)
			}
		case sdk.EventTypePlanStep:
			fmt.Printf("\033[36m[%s %d/%d] %s\033[0m\n", i18n.T("cli.plan_step_label"), ev.Extra()["step_num"], ev.Extra()["total_steps"], ev.Extra()["step_desc"])
		case sdk.EventTypePlanModeExit:
			fmt.Printf("\033[32m[%s] %s\033[0m\n", i18n.T("cli.exit_plan_mode_label"), i18n.T("cli.starting_execution"))
		case sdk.EventTypePlanComplete:
			fmt.Printf("\033[32m[%s] %s\033[0m\n", i18n.T("cli.plan_complete_label"), ev.Content())
		}
	}
}

// createCLIConfirmHandler creates a confirmation handler for CLI mode.
func createCLIConfirmHandler(runCtx context.Context) func(sdk.ConfirmationRequest) {
	return func(req sdk.ConfirmationRequest) {
		cmdCtx := getCommandContext()
		if cmdCtx == nil || cmdCtx.PermissionMgr == nil {
			req.ResponseCh <- true
			return
		}

		allowed, requiresConfirm, reason := cmdCtx.PermissionMgr.CheckPermission(runCtx, req.ToolName, req.Params)
		if !allowed {
			fmt.Fprintf(os.Stderr, "\033[31mPermission denied: %s\033[0m\n", reason)
			req.ResponseCh <- false
			return
		}

		if requiresConfirm && !autoConfirm {
			// Ask user for confirmation in CLI mode
			fmt.Printf("\n\033[33m%s\033[0m\n", i18n.T("cli.sensitive_operation"))
			fmt.Printf("  Tool: %s\n", req.ToolName)
			if params, ok := req.Params["command"]; ok {
				fmt.Printf("  Command: %v\n", params)
			}
			if params, ok := req.Params["path"]; ok {
				fmt.Printf("  Path: %v\n", params)
			}

			if readUserConfirmation("Do you want to proceed? (y/n): ") {
				req.ResponseCh <- true
			} else {
				req.ResponseCh <- false
			}
			return
		}

		req.ResponseCh <- true
	}
}

// setupCommandProviderHandlers registers command handlers on the event bus.
// This is the event-driven approach - handlers are registered on the bus,
// and commands send events that get processed by these handlers.
func setupCommandProviderHandlers(
	cmdProvider *cmds.CommandProvider,
	memoryProvider cmds.MemoryProvider,
	taskProvider cmds.TaskProvider,
	cmdCtx *CommandContext,
	runCtx context.Context,
	currentSessionID string,
	sessionMgr *sdk.SessionManager,
) {
	bus := cmdProvider.GetEventBus()

	// Use shared handler registration for common commands
	cmds.RegisterDefaultCommandHandlers(bus, memoryProvider, taskProvider, cmdCtx.Config)

	// Override session role handler with logic that queries actual session role
	bus.RegisterCommandHandler(events.CommandTypeSessionRole, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		if sessionMgr != nil && currentSessionID != "" {
			session, err := sessionMgr.GetSession(currentSessionID)
			if err != nil {
				return events.CommandResponse{Success: false, Result: fmt.Sprintf("Failed to get session info: %v", err)}
			}
			if session != nil && session.Role != "" {
				return events.CommandResponse{Success: true, Result: fmt.Sprintf("Current role: %s", session.Role)}
			}
			return events.CommandResponse{Success: true, Result: "No role set. Use /role to set one."}
		}
		return events.CommandResponse{Success: true, Result: "No role set. Use /role to set one."}
	})
}

// runSingleMessage runs a single message query and returns.
func runSingleMessage(runCtx context.Context, rt *sdk.Runtime, args []string, eventHandler func(sdk.Event)) {
	events, err := rt.Process(runCtx, strings.Join(args, " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	for ev := range events {
		eventHandler(ev)
	}
	// Print newline after streaming complete
	fmt.Println()
}

// runTUIMode starts the TUI mode.
func runTUIMode(
	runCtx context.Context,
	rt *sdk.Runtime,
	sl *tui.SessionLearner,
	sessionMgr *sdk.SessionManager,
	currentSessionID string,
	ctx *CommandContext,
	profName string,
	mcpManager *sdk.MCPManager,
) {
	fn := tuiRunFunc(rt, sl, ctx)

	modelProvider := tui.NewModelProvider()

	summarizer := rt.GetSummarizer()

	cfg := tui.Config{
		Mode:             "aura",
		UserName:         profName,
		Tools:            rt.GetToolNames(),
		ShowTokens:       ctx.Config.Debug.ShowTokens,
		TokenMax:         ctx.Config.Memory.MaxTokens,
		DebugMode:        ctx.Config.TUI.DebugMode,
		SessionID:        currentSessionID,
		SkillDirectories: ctx.Config.Skills.Directories,
		EnableReview:     ctx.Config.Agent.Plan.EnableReview,
	}
	if err := tui.RunWithConfig(runCtx, fn, cfg, sessionMgr, summarizer, modelProvider, ctx.CommandProvider, mcpManager); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}
}

// createRuntime creates and initializes the agent runtime.
func createRuntime(ctx *CommandContext, sessionID string, cmdProvider *cmds.CommandProvider, sessionMgr *sdk.SessionManager) (*sdk.Runtime, *sdk.MCPManager) {
	runtimeConfig := sdk.FromConfig(ctx.Config)
	if sessionID != "" {
		runtimeConfig.SessionID = sessionID
	}

	if useCLI {
		runtimeConfig.MessageSource = string(sdk.SourceCLI)
	} else {
		runtimeConfig.MessageSource = string(sdk.SourceTUI)
	}

	// Get MessageStore from session manager for persistence
	var sessionStore *sdk.MessageStore
	if sessionMgr != nil {
		sessionStore = sessionMgr.GetStore().MessageStore()
	}

	// Create IntentService for natural language command recognition
	var intentSvc *sdk.IntentService
	if cmdProvider != nil && ctx.Config.Intent.Enabled {
		intentSvc = sdk.NewIntentService(cmdProvider, ctx.Config.Intent.ConfidenceThreshold)
	}

	// Build runtime options
	opts := []sdk.RuntimeOption{
		sdk.WithConfirmationHandler(createCLIConfirmHandler(context.Background())),
		sdk.WithCommands(cmdProvider),
		sdk.WithIntentService(intentSvc),
		sdk.WithSessionStore(sessionStore),
		sdk.WithSessionID(sessionID),
		sdk.WithUserID(ctx.UserID),
		sdk.WithDataDir(ctx.DataDir),
	}

	// TUI mode: inject unified logger and set mode
	if !useCLI {
		opts = append(opts, sdk.WithLogger(tui.GetLogger()), sdk.WithMode(sdk.RuntimeModeTUI))
	}

	// Create MCP manager if config exists
	mcpManager := sdk.NewMCPManager()
	if mcpManager != nil {
		opts = append(opts, sdk.WithMCPManager(mcpManager))
	}

	rt, err := sdk.NewRuntime(runtimeConfig, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create runtime: %v\n", err)
		return nil, nil
	}

	if err := rt.Initialize(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize runtime: %v\n", err)
		return nil, nil
	}

	return rt, mcpManager
}

// createLogger creates a logger from config.
func createLogger(cfg *config.Config) *logger.Logger {
	return logger.NewNamed(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
		Module: "cli",
	})
}

// tuiRunFunc adapts an agent runtime into a tui.RunFunc.
func tuiRunFunc(rt *sdk.Runtime, sl *tui.SessionLearner, ctx *CommandContext) tui.RunFunc {
	return func(runtimeCtx context.Context, input string) (<-chan tui.ChatEvent, error) {
		if sl != nil {
			sl.Observe(input)
		}

		out := make(chan tui.ChatEvent, 100)

		// Set up TUI confirmation handler
		rt.SetConfirmationHandler(func(req sdk.ConfirmationRequest) {
			event := tui.ChatEvent{
				Type:       tui.EventTypeConfirmationRequest,
				Content:    fmt.Sprintf("%s %s", i18n.T("cli.sensitive_operation"), req.ToolName),
				Extra:      map[string]any{"toolName": req.ToolName, "params": req.Params},
				ResponseCh: req.ResponseCh,
			}
			out <- event
		})

		// Run Process() in background goroutine so out channel events
		// (like confirmation requests) flow immediately.
		go func() {
			defer close(out)

			events, err := rt.Process(runtimeCtx, input)
			if err != nil {
				out <- tui.ChatEvent{Type: tui.EventTypeError, Content: err.Error()}
				return
			}

			for ev := range events {
				var chatEvent tui.ChatEvent
				switch ev.Type() {
				case sdk.EventTypeThinkingStart:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeThinkingStart, Content: ev.Content()}
				case sdk.EventTypeThinkingChunk:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeThinkingChunk, Content: ev.Content()}
				case sdk.EventTypeThinkingEnd:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeThinkingEnd, Content: ev.Content()}
				case sdk.EventTypeThinkingContent:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeThinkingContent, Content: ev.Content()}
				case sdk.EventTypeResponseStart:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResponseStart, Content: ev.Content()}
				case sdk.EventTypeResponseChunk:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResponseChunk, Content: ev.Content()}
				case sdk.EventTypeResponseEnd:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResponseEnd, Content: ev.Content()}
				case sdk.EventTypeAction:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeAction, Content: ev.Content()}
				case sdk.EventTypeResult:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResult, Content: ev.Content()}
				case sdk.EventTypeResponse:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResponse, Content: ev.Content()}
				case sdk.EventTypeError:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeError, Content: ev.Content()}
				case sdk.EventTypeStep:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeStep, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeToolStart:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeToolStart, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeToolEnd:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeToolEnd, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeCommandMatched:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeCommandMatched, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeCommandResult:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeCommandResult, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeTaskCreate:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeTaskCreate, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeTaskUpdate:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeTaskUpdate, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypeTaskList:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeTaskList, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanCreated:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanCreated, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanReviewStart:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanReviewStart, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanReviewFiles:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanReviewFiles, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanStep:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanStep, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanModeExit:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanModeExit, Content: ev.Content(), Extra: ev.Extra()}
				case sdk.EventTypePlanComplete:
					chatEvent = tui.ChatEvent{Type: tui.EventTypePlanComplete, Content: ev.Content(), Extra: ev.Extra()}
				default:
					chatEvent = tui.ChatEvent{Type: tui.EventTypeResponse, Content: ev.Content()}
				}
				out <- chatEvent
			}
			// Explicitly send Done event before closing channel
			out <- tui.ChatEvent{Type: tui.EventTypeDone}
		}()

		return out, nil
	}
}
