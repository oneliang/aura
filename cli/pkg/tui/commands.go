package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	climds "github.com/oneliang/aura/cli/pkg/commands"
	commands "github.com/oneliang/aura/commands/pkg"
	initpkg "github.com/oneliang/aura/commands/pkg/init"
	"github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/version"
)

func init() {
	// Register all built-in commands
	RegisterCommand(&Command{
		Name:        climds.CmdHelp,
		Description: "Show available commands",
		Handler:     cmdHelp,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdExit,
		Description: "Exit the application",
		Handler:     cmdExit,
		Aliases:     []string{climds.CmdQuit, climds.CmdQuickExit},
	})

	RegisterCommand(&Command{
		Name:        climds.CmdClear,
		Description: "Clear conversation history",
		Handler:     cmdClear,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdCompact,
		Description: "Compact conversation history with summary",
		Handler:     cmdCompact,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdTools,
		Description: "List available tools",
		Handler:     cmdTools,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdSkills,
		Description: "List loaded skills",
		Handler:     cmdSkills,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdStatus,
		Description: "Show current execution status",
		Handler:     cmdStatus,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdHistory,
		Description: "Show tool execution history",
		Handler:     cmdHistory,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdMemory,
		Description: "Show memory usage",
		Handler:     cmdMemory,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdModel,
		Description: "Show current model info",
		Handler:     cmdModel,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdProfile,
		Description: "Show user profile",
		Handler:     cmdProfile,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdConfig,
		Description: "Show configuration",
		Handler:     cmdConfig,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdRole,
		Description: "Set role: /role [name]",
		Handler:     cmdRole,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdKnowledge,
		Description: "Knowledge management",
		Handler:     cmdKnowledge,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdSessions,
		Description: "List and switch sessions",
		Handler:     cmdSessions,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdSession,
		Description: "Session commands: show|delete|export|create",
		Handler:     cmdSession,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdSubscription,
		Description: "Subscription commands: show|add|delete",
		Handler:     cmdSubscription,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdMcp,
		Description: "List MCP servers",
		Handler:     cmdMcp,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdVersion,
		Description: "Show version",
		Handler:     cmdVersion,
	})

	RegisterCommand(&Command{
		Name:        climds.CmdInit,
		Description: "Initialize AURA.md with codebase documentation",
		Handler:     cmdInit,
	})
}

// Command handlers

func cmdExit(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func cmdClear(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	// Execute clear command through CommandProvider (clears SessionMemory)
	if m.commandProvider != nil {
		result, err := m.commandProvider.Execute(m.ctx, commands.CmdNameClear, nil)
		if err != nil {
			m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Error: %v", err)))
			return m, m.scrollToBottom()
		}

		// Clear UI state using shared function (consistent with intent-based command_clear)
		m.clearUIState()

		// Show result after clearing (user sees the confirmation)
		m.messages.AddRaw(m.styles.Help.Render(result))
	}

	return m, m.scrollToBottom()
}

func cmdHelp(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showHelp())
	return m, m.scrollToBottom()
}

func cmdTools(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showTools())
	return m, m.scrollToBottom()
}

func cmdSkills(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showSkills())
	return m, m.scrollToBottom()
}

func cmdStatus(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showStatus())
	return m, m.scrollToBottom()
}

func cmdHistory(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showHistory())
	return m, m.scrollToBottom()
}

func cmdMemory(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showMemory())
	return m, m.scrollToBottom()
}

func cmdCompact(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showCompact())
	return m, m.scrollToBottom()
}

func cmdSessions(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	// Sessions uses popup, no text output
	m.handleSessions()
	return m, nil
}

func cmdSession(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) > 1 {
		switch parts[1] {
		case "create":
			return m.handleSessionCreate(input)
		case "show":
			m.messages.AddRaw(m.showSessionShow())
			return m, m.scrollToBottom()
		case "delete":
			m.messages.AddRaw(m.showSessionDelete())
			return m, m.scrollToBottom()
		case "export":
			m.messages.AddRaw(m.showSessionExport())
			return m, m.scrollToBottom()
		}
	}
	m.messages.AddRaw(m.styles.Help.Render("  Usage: /session [show|delete|export|create]"))
	return m, m.scrollToBottom()
}

func cmdRole(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showRole())
	return m, m.scrollToBottom()
}

func cmdKnowledge(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showKnowledge())
	return m, m.scrollToBottom()
}

func cmdModel(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showModel())
	return m, m.scrollToBottom()
}

func cmdProfile(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showProfile())
	return m, m.scrollToBottom()
}

func cmdConfig(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.showConfig())
	return m, m.scrollToBottom()
}

func cmdSubscription(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) > 1 {
		switch parts[1] {
		case "show":
			// Subscription show uses popup
			return m.handleSubscriptionShow()
		case "add":
			m.handleSubscriptionAdd()
			return m, nil
		case "delete":
			m.messages.AddRaw(m.handleSubscriptionDelete())
			return m, m.scrollToBottom()
		}
	}
	m.messages.AddRaw(m.styles.Help.Render("  Usage: /subscription [show|add|delete]"))
	return m, m.scrollToBottom()
}

// handleCommand handles slash commands using the registry.
func (m Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	cmd := strings.Fields(input)
	if len(cmd) == 0 {
		return m, m.input.EnableAndFocus()
	}

	cmdName := strings.ToLower(strings.TrimSpace(cmd[0]))

	// Check for /session create with arguments first
	if strings.HasPrefix(cmdName, "/session") && len(cmd) > 1 && cmd[1] == "create" {
		model, teaCmd := m.handleSessionCreate(input)
		if mm, ok := model.(Model); ok {
			if teaCmd == nil {
				return model, mm.input.EnableAndFocus()
			}
			return model, tea.Sequence(teaCmd, mm.input.EnableAndFocus())
		}
		return model, teaCmd
	}

	// Look up command in registry
	if handler := GetCommand(cmdName); handler != nil && handler.Handler != nil {
		model, teaCmd := handler.Handler(m.ctx, m, input) // Pass ctx from Model
		// Re-enable input after command execution, unless command is waiting for async result
		if mm, ok := model.(Model); ok {
			// If command is async (waiting), let the Msg handler manage input state
			if mm.state.Waiting() {
				if teaCmd == nil {
					return model, nil
				}
				return model, teaCmd
			}
			// Sync command: re-enable input immediately
			if teaCmd == nil {
				return model, mm.input.EnableAndFocus()
			}
			return model, tea.Sequence(teaCmd, mm.input.EnableAndFocus())
		}
		return model, teaCmd
	}

	// Try to execute as a skill command
	if model, teaCmd := m.tryExecuteSkill(cmdName); teaCmd != nil {
		if mm, ok := model.(Model); ok {
			return model, tea.Sequence(teaCmd, mm.input.EnableAndFocus())
		}
		return model, teaCmd
	}

	errMsg := m.styles.Error.Render("  Unknown command: "+cmdName) + m.styles.Help.Render(" Try /help")
	m.messages.AddRaw(errMsg)
	return m, tea.Sequence(m.scrollToBottom(), m.input.EnableAndFocus())
}

// showHelp shows help text generated from command registry.
// Returns the rendered content for adding to the viewport.
func (m Model) showHelp() string {
	var b strings.Builder

	// Keyboard shortcuts
	b.WriteString(m.styles.Command.Render("Keyboard shortcuts:\n"))
	for _, kb := range GetAllBindings() {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", formatKeyDisplay(kb.Keys[0]), kb.HelpText))
	}

	b.WriteString("\n")

	b.WriteString(m.styles.Command.Render("Available commands:\n"))

	// Get unique commands and render
	cmds := GetAllCommands()
	for _, cmd := range cmds {
		b.WriteString(fmt.Sprintf("  %-24s - %s\n", cmd.Name, cmd.Description))
		if len(cmd.Aliases) > 0 {
			for _, alias := range cmd.Aliases {
				b.WriteString(fmt.Sprintf("  %-24s - Alias for %s\n", alias, cmd.Name))
			}
		}
	}

	b.WriteString("\nType / to see command suggestions with skill commands.\n")
	b.WriteString("Press Enter to send message")

	return m.messages.AddRaw(m.styles.Help.Render(b.String()))
}

// formatKeyDisplay formats a key string for display (e.g. "ctrl+l" → "Ctrl+L").
func formatKeyDisplay(key string) string {
	parts := strings.Split(key, "+")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "+")
}

// getFilteredCommands returns commands matching the current filter.
func (m Model) getFilteredCommands() []CommandInfo {
	var filtered []CommandInfo

	prefix := strings.TrimPrefix(m.state.CommandFilter(), "/")

	// Get built-in commands from registry
	for _, cmd := range GetAllCommands() {
		name := strings.TrimPrefix(cmd.Name, "/")
		if prefix == "" || strings.HasPrefix(name, prefix) {
			filtered = append(filtered, CommandInfo{
				Name:        cmd.Name,
				Description: cmd.Description,
			})
		}
	}

	// Add skills as commands
	skills := m.getSkillsForCommandCompletion(prefix)
	filtered = append(filtered, skills...)

	return filtered
}

// getSkillDirectories returns the configured skill directories.
func (m Model) getSkillDirectories() []string {
	return m.config.SkillDirectories
}

// getSkillsForCommandCompletion loads skills and returns them as CommandItems for command completion.
// Returns nil if skills are disabled, loading fails, or skipSkillLoading is set (for testing).
func (m Model) getSkillsForCommandCompletion(prefix string) []CommandInfo {
	// Skip skill loading in test mode
	if m.skipSkillLoading {
		return nil
	}

	// Check cache first
	if m.cachedSkills != nil && time.Since(m.cachedSkillsTime) < SkillCacheTTL {
		var items []CommandInfo
		for _, sk := range m.cachedSkills {
			if prefix == "" || strings.HasPrefix(sk.Name, prefix) {
				items = append(items, sk)
			}
		}
		return items
	}

	// Load skills via SDK
	skills, err := sdk.LoadSkills(m.getSkillDirectories())
	if err != nil || len(skills) == 0 {
		return nil
	}

	m.cachedSkills = make([]CommandInfo, 0, len(skills))
	m.cachedSkillDefs = make([]sdk.SkillInfo, 0, len(skills))
	for _, sk := range skills {
		item := CommandInfo{
			Name:        "/" + sk.Name,
			Description: sk.Description,
		}
		m.cachedSkills = append(m.cachedSkills, item)
		m.cachedSkillDefs = append(m.cachedSkillDefs, sk)
	}
	m.cachedSkillsTime = time.Now()

	var items []CommandInfo
	for _, sk := range m.cachedSkills {
		if prefix == "" || strings.HasPrefix(sk.Name, prefix) {
			items = append(items, sk)
		}
	}
	return items
}

// tryExecuteSkill checks if the input matches a skill name and executes it.
func (m Model) tryExecuteSkill(input string) (tea.Model, tea.Cmd) {
	// Remove leading "/" for skill matching
	skillName := strings.TrimPrefix(input, "/")

	// Skip skill loading in test mode
	if m.skipSkillLoading {
		return m, nil
	}

	// Check cache first - load skills if cache is empty or expired
	if m.cachedSkills == nil || m.cachedSkillDefs == nil || time.Since(m.cachedSkillsTime) >= SkillCacheTTL {
		skills, err := sdk.LoadSkills(m.getSkillDirectories())
		if err != nil || len(skills) == 0 {
			return m, nil
		}

		m.cachedSkills = make([]CommandInfo, 0, len(skills))
		m.cachedSkillDefs = make([]sdk.SkillInfo, 0, len(skills))
		for _, sk := range skills {
			m.cachedSkills = append(m.cachedSkills, CommandInfo{
				Name:        "/" + sk.Name,
				Description: sk.Description,
			})
			m.cachedSkillDefs = append(m.cachedSkillDefs, sk)
		}
		m.cachedSkillsTime = time.Now()
	}

	// Find matching skill from cached definitions (which have Body field)
	for i, sk := range m.cachedSkillDefs {
		// Compare name without leading "/"
		name := strings.TrimPrefix(sk.Name, "/")
		if name == skillName {
			// Execute skill through runFn
			m.messages.Add(MessageTypeSystem, "Executing skill: "+sk.Name, nil, renderMessage, m.renderer, m.styles)
			m.state.SetWaiting(true)
			m.state.SetStartTime(time.Now())
			m.input.SetDisabled(true)
			return m, tea.Sequence(
				m.sendMessage(m.cachedSkillDefs[i].Body),
			)
		}
	}

	return m, nil
}

// cmdMcp lists MCP servers.
func cmdMcp(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	if m.mcpManager == nil {
		m.messages.AddRaw(m.styles.Help.Render("  No MCP servers configured."))
		return m, m.scrollToBottom()
	}

	infos := m.mcpManager.ListServers()
	if len(infos) == 0 {
		m.messages.AddRaw(m.styles.Help.Render("  No MCP servers configured."))
		return m, m.scrollToBottom()
	}

	var b strings.Builder
	b.WriteString("\n  MCP Servers:\n")
	for _, info := range infos {
		b.WriteString(fmt.Sprintf("    %-12s %s  %s  tools: %d\n", info.Name, info.Command, info.Status, info.ToolCount))
		if len(info.Args) > 0 && info.Args[0] != "" {
			b.WriteString(fmt.Sprintf("                 %s\n", info.Args[0]))
		}
	}
	m.messages.AddRaw(m.styles.Help.Render(b.String()))
	return m, m.scrollToBottom()
}

// cmdVersion shows version.
func cmdVersion(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	m.messages.AddRaw(m.styles.Help.Render("  Aura " + version.FullVersion()))
	return m, m.scrollToBottom()
}

// cmdInit initializes AURA.md with codebase documentation.
// Uses sendMessageWithConfig to integrate with TUI event system for proper UI feedback.
func cmdInit(ctx context.Context, m Model, input string) (tea.Model, tea.Cmd) {
	// Get config from context
	cfg := config.GetConfig(ctx)
	if cfg == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Error: config not available in context"))
		return m, m.scrollToBottom()
	}

	// Parse for "force" argument
	parts := strings.Fields(input)
	force := len(parts) > 1 && parts[1] == "force"

	cwd, _ := os.Getwd()
	auraMdPath := filepath.Join(cwd, "AURA.md")

	// Check if AURA.md already exists
	if !force {
		if _, err := os.Stat(auraMdPath); err == nil {
			m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  AURA.md already exists at: %s\n  Use /init force to regenerate.", auraMdPath)))
			return m, m.scrollToBottom()
		}
	}

	// Build init-specific runtime config
	initCfg := sdk.FromConfig(cfg)
	initCfg.SystemPrompt = initpkg.InitSystemPrompt
	initCfg.EnableSubAgent = false
	initCfg.SessionID = "" // No persistence
	// Force implicit planning mode - init should not trigger plan review
	initCfg.Agent.PlanningMode = string(engine.ModeImplicit)

	// Build init prompt - LLM will explore codebase using tools
	prompt := initpkg.BuildInitPrompt(cwd)

	// Add user message to show what's happening
	m.messages.Add(MessageTypeUser, "/init", nil, renderMessage, m.renderer, m.styles)

	// Set init state for content capture
	m.initPending = true
	m.initAuraMdPath = auraMdPath
	m.initContent = ""

	// Set UI state (same as handleSubmit)
	m.state.SetWaiting(true)
	m.state.SetStartTime(time.Now())
	m.state.SetDisplayState(DisplayThinking)
	m.input.SetDisabled(true)

	// Reset widgets for clean state
	m.thinking.Reset()
	m.processing.Reset()
	m.plan.Reset()

	// Set scroll state for proper viewport behavior
	m.autoScroll = true
	m.manualScroll = false
	m.manualScrollOffset = 0

	// Start thinking indicator
	_, thinkingCmd := m.thinking.StartAndRender()

	// Use sendMessageWithConfig to send through event system
	return m, tea.Batch(
		m.sendMessageWithConfig(prompt, initCfg),
		thinkingCmd,
		m.scrollToBottom(),
		m.eventLoop(),
	)
}
