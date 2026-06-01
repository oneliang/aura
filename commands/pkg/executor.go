// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agentloader "github.com/oneliang/aura/agent/pkg/loader"
	agentmanager "github.com/oneliang/aura/agent/pkg/manager"
	"github.com/oneliang/aura/knowledge/pkg"
	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/memory"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	skillmanager "github.com/oneliang/aura/skill/pkg/manager"
)

// CommandProvider is the command provider implementation.
// It provides a unified API for executing commands across different UI layers (TUI, CLI, and adapter scenarios).
type CommandProvider struct {
	sessionHandler      *SessionHandler
	profileHandler      *ProfileHandler
	configHandler       *ConfigExecutor
	knowledgeHandler    *KnowledgeExecutor
	subscriptionHandler *SubscriptionHandler
	skillCommand        *SkillCommand
	skillHandler        *SkillHandler
	agentHandler        *AgentHandler
	agentManager        *agentmanager.AgentManager
	mcpHandler          *MCPHandler

	// Event bus for command execution (event-driven)
	eventBus *events.Bus
}

// Compile-time interface check: ensure *CommandProvider implements Command
var _ Command = (*CommandProvider)(nil)

// CommandProviderDeps holds dependencies for creating a CommandProvider.
type CommandProviderDeps struct {
	SessionMgr        *manager.SessionManager
	Profile           *profile.Profile
	Config            *config.Config
	ConfigPath        string                                                                   // Optional, defaults to ~/.aura/config.yaml
	UserID            string                                                                   // Optional, for user-specific knowledge base
	KnowledgeFactory  knowledge.CollectionFactory                                              // Optional, created from Config if not provided
	SkillLoader       *skillloader.Loader                                                      // Optional, for skill command
	SkillManager      *skillmanager.SkillManager                                               // Optional, for skill management commands
	AgentLoader       *agentloader.Loader                                                      // Optional, for agent loading
	AgentManager      *agentmanager.AgentManager                                               // Optional, for agent management commands
	AgentDelegateFunc func(ctx context.Context, agentName string, task string) (string, error) // Optional, for agent handler

	// Event bus for command communication (optional, created if not provided)
	EventBus *events.Bus
}

// NewCommandProvider creates a new command provider with all handlers initialized.
func NewCommandProvider(deps CommandProviderDeps) *CommandProvider {
	// Initialize session handler
	sessionHandler := NewSessionHandler(deps.SessionMgr, deps.UserID)

	// Initialize profile handler
	profileHandler := NewProfileHandler(deps.Profile)

	// Initialize config handler
	configPath := deps.ConfigPath
	if configPath == "" {
		configPath = ffp.MustAuraHomePath(constants.DefaultConfigFile)
	}
	configHandler := NewConfigExecutor(configPath)

	// Initialize knowledge handler
	knowledgeFactory := deps.KnowledgeFactory
	if knowledgeFactory == nil && deps.Config != nil {
		knowledgeFactory = knowledge.NewDefaultCollectionFactory(deps.Config)
	}
	knowledgeHandler := NewKnowledgeExecutor(knowledgeFactory, deps.UserID)

	// Initialize subscription handler
	subscriptionHandler := NewSubscriptionHandler(deps.SessionMgr, deps.UserID)

	// Initialize skill command
	var skillCommand *SkillCommand
	if deps.SkillLoader != nil {
		skillCommand = NewSkillCommand(deps.SkillLoader)
	}

	// Initialize skill handler for skill management
	var skillHandler *SkillHandler
	if deps.SkillManager != nil {
		skillHandler = NewSkillHandler(deps.SkillManager)
	}

	// Initialize agent handler
	var agentHandler *AgentHandler
	if deps.AgentDelegateFunc != nil {
		agentHandler = NewAgentHandlerWithManager(deps.AgentDelegateFunc, deps.AgentManager)
	}

	// Initialize event bus
	eventBus := deps.EventBus
	if eventBus == nil {
		eventBus = events.NewBus()
	}

	return &CommandProvider{
		sessionHandler:      sessionHandler,
		profileHandler:      profileHandler,
		configHandler:       configHandler,
		knowledgeHandler:    knowledgeHandler,
		subscriptionHandler: subscriptionHandler,
		skillCommand:        skillCommand,
		skillHandler:        skillHandler,
		agentHandler:        agentHandler,
		agentManager:        deps.AgentManager,
		mcpHandler:          NewMCPHandler(nil),
		eventBus:            eventBus,
	}
}

// GetEventBus returns the event bus for registering command handlers.
func (c *CommandProvider) GetEventBus() *events.Bus {
	return c.eventBus
}

// SetAgentDelegateFn sets the agent delegation function callback.
// This allows the runtime to inject the actual delegation logic after creation.
func (c *CommandProvider) SetAgentDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error)) {
	log := logger.RegistryDefault().WithModule("command_provider")
	if c.agentHandler == nil {
		c.agentHandler = NewAgentHandlerWithManager(fn, c.agentManager)
		log.Info().Msg("SetAgentDelegateFn: agentHandler created (was nil)")
		return
	}
	c.agentHandler.SetDelegateFn(fn)
	log.Info().Msg("SetAgentDelegateFn: delegateFn set on existing agentHandler")
}

// SetMCPListFunc sets the MCP server list callback function.
// This allows the runtime to inject the actual MCP list function after creation.
func (c *CommandProvider) SetMCPListFunc(fn func() []MCPInfo) {
	if c.mcpHandler == nil {
		c.mcpHandler = NewMCPHandler(fn)
		return
	}
	c.mcpHandler.SetListServersFunc(fn)
}

// SetCurrentSessionID sets the current session ID for session_show command.
func (c *CommandProvider) SetCurrentSessionID(sessionID string) {
	if c.sessionHandler != nil {
		c.sessionHandler.SetCurrentSessionID(sessionID)
	}
}

// GetCommands returns all available commands.
func (c *CommandProvider) GetCommands() []CommandInfo {
	cmds := GetInternalCommands()
	// Convert concrete type to interface type
	result := make([]CommandInfo, len(cmds))
	copy(result, cmds)

	// Add skill commands if available
	if c.skillCommand != nil {
		result = append(result, c.skillCommand.GetCommands()...)
	}

	return result
}

// Execute executes a command by name with the given parameters.
// The command name should match one of the commands returned by GetCommands().
//
// Command routing:
//   - command_session_* -> SessionHandler
//   - command_profile_* -> ProfileHandler
//   - command_config_* -> ConfigHandler
//   - command_knowledge_* -> KnowledgeHandler
//   - command_subscription_* -> SubscriptionHandler
//   - command_skills -> SkillHandler (list)
//   - command_skill_* -> SkillHandler (CRUD)
//   - command_exit, command_quit, command_clear, command_compact, command_help, command_memory -> handled directly
func (c *CommandProvider) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	log := logger.RegistryDefault().WithModule("command_provider").With().Str("cmd", cmd).Logger()
	log.Debug().Interface("params", params).Bool("agentHandler_nil", c.agentHandler == nil).Msg("Execute: routing command")

	// Route command based on prefix
	switch {
	case strings.HasPrefix(cmd, CmdPrefixSession):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixSession+"_")
		if subCmd == "" || subCmd == "sessions" || subCmd == "session" {
			subCmd = "list"
		}
		return c.sessionHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixProfile):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixProfile+"_")
		if subCmd == "" || subCmd == "profile" {
			subCmd = "show"
		}
		return c.profileHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixConfig):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixConfig+"_")
		if subCmd == "" || subCmd == "config" {
			subCmd = "show"
		}
		return c.configHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixKnowledge):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixKnowledge+"_")
		if subCmd == "" || subCmd == "knowledge" {
			subCmd = "search"
		}
		return c.knowledgeHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixSubscription):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixSubscription+"_")
		if subCmd == "" || subCmd == "subscriptions" {
			subCmd = "show"
		}
		return c.subscriptionHandler.ExecuteCommand(ctx, subCmd, params)

	case cmd == CmdNameSkills:
		if c.skillHandler == nil {
			return i18n.T("command.skills_not_available"), nil
		}
		return c.skillHandler.ExecuteCommand(ctx, "list", params)

	case strings.HasPrefix(cmd, CmdPrefixSkill):
		if c.skillHandler == nil {
			return i18n.T("command.skill_management_not_configured"), nil
		}
		subCmd := strings.TrimPrefix(cmd, CmdPrefixSkill+"_")
		if subCmd == "" || subCmd == "skill" {
			subCmd = "list"
		}
		return c.skillHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixAgent+"_"):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixAgent+"_")
		if subCmd == "" {
			subCmd = "list"
		}
		// Has task parameter → delegation mode
		if task, ok := params["task"].(string); ok && task != "" {
			log.Debug().Str("subCmd", subCmd).Str("task", task).Bool("agentHandler_nil", c.agentHandler == nil).Msg("Execute: agent delegation mode")
			if c.agentHandler == nil {
				return i18n.T("command.agent_delegation_not_configured"), nil
			}
			return c.agentHandler.ExecuteCommand(ctx, "delegate_to_agent", map[string]any{
				"agent": subCmd,
				"task":  task,
			})
		}
		// No task parameter → management commands (list, create, etc.)
		log.Debug().Str("subCmd", subCmd).Bool("agentHandler_nil", c.agentHandler == nil).Msg("Execute: agent management mode")
		if c.agentHandler == nil {
			return i18n.T("command.agent_management_not_configured"), nil
		}
		return c.agentHandler.ExecuteCommand(ctx, subCmd, params)

	case strings.HasPrefix(cmd, CmdPrefixMcp):
		subCmd := strings.TrimPrefix(cmd, CmdPrefixMcp+"_")
		if subCmd == "" || subCmd == "mcp" {
			subCmd = "list"
		}
		return c.mcpHandler.ExecuteCommand(ctx, subCmd, params)

	// Basic commands that don't require handlers
	case cmd == CmdNameExit, cmd == CmdNameQuit, cmd == CmdNameQ:
		return "exit", nil
	case cmd == CmdNameClear:
		return c.executeCommand(ctx, events.CommandTypeMemoryClear, params)
	case cmd == CmdNameCompact:
		return c.executeCommand(ctx, events.CommandTypeMemoryCompact, params)
	case cmd == CmdNameHelp:
		return c.getHelp(), nil
	case cmd == CmdNameMemory:
		return c.executeCommand(ctx, events.CommandTypeMemoryStats, params)
	case cmd == CmdNameInit:
		return c.executeInitCommand(ctx, params)
	case cmd == CmdNameTools:
		return c.getToolsList(), nil
	case cmd == CmdNameStatus:
		return c.executeCommand(ctx, events.CommandTypeEngineStatus, params)
	case cmd == CmdNameHistory:
		return c.executeCommand(ctx, events.CommandTypeToolHistory, params)
	case cmd == CmdNameModel:
		return c.executeCommand(ctx, events.CommandTypeLLMConfig, params)
	case cmd == CmdNameRole:
		return c.executeCommand(ctx, events.CommandTypeSessionRole, params)

	case cmd == CmdNameDelegateToAgent:
		if c.agentHandler == nil {
			return i18n.T("command.agent_delegation_not_configured"), nil
		}
		return c.agentHandler.ExecuteCommand(ctx, "delegate_to_agent", params)

	default:
		// Check if it's a skill command
		if c.skillCommand != nil {
			// Try to execute as skill command (supports both "skill_name" and "name" formats)
			result, err := c.skillCommand.Execute(ctx, cmd, params)
			if err == nil {
				return result, nil
			}
			// If error is "skill not found", fall through to unknown command error
			if !strings.Contains(err.Error(), "skill not found") {
				return "", err
			}
		}
		return "", fmt.Errorf(i18n.T("error.command.unknown"), cmd)
	}
}

// executeCommand executes a command via the event bus.
// This enables event-driven communication with runtime handlers.
func (c *CommandProvider) executeCommand(ctx context.Context, commandType string, params map[string]any) (string, error) {
	req := &events.CommandRequest{
		Command:    commandType,
		Params:     params,
		ResponseCh: make(chan events.CommandResponse, 1),
	}

	resp := c.eventBus.ExecuteCommand(ctx, req)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result, nil
}

// executeInitCommand executes the init command for generating AURA.md.
func (c *CommandProvider) executeInitCommand(ctx context.Context, params map[string]any) (string, error) {
	// Simple init handler for generating AURA.md
	cwd, ok := params["cwd"].(string)
	if !ok || cwd == "" {
		cwd, _ = os.Getwd()
	}
	claudeMdPath := filepath.Join(cwd, "AURA.md")

	force := params["force"] == true

	if !force {
		if _, err := os.Stat(claudeMdPath); err == nil {
			return fmt.Sprintf("AURA.md already exists at: %s\nUse force=true to regenerate.", claudeMdPath), nil
		}
	}

	// Generate basic AURA.md content
	content := c.generateBasicClaudeMd(cwd)

	if err := os.WriteFile(claudeMdPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write AURA.md: %w", err)
	}

	return fmt.Sprintf("Generated AURA.md at: %s\n\nPreview:\n%s", claudeMdPath, truncatePreview(content, 500)), nil
}

// generateBasicClaudeMd generates basic AURA.md content.
func (c *CommandProvider) generateBasicClaudeMd(cwd string) string {
	var content strings.Builder
	content.WriteString("# AURA.md\n\n")
	content.WriteString("This file provides guidance for working with this repository.\n\n")
	content.WriteString("## Build & Development Commands\n\n")
	content.WriteString("Run appropriate build/test commands based on project type.\n\n")
	content.WriteString("## Architecture Overview\n\n")
	content.WriteString("Analyze the project structure to understand architecture.\n\n")
	content.WriteString("## Notes\n\n")
	content.WriteString("- This file was auto-generated by `aura init`\n")
	content.WriteString("- Review and customize for your project\n")
	return content.String()
}

// truncatePreview truncates content for preview display.
func truncatePreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// getHelp returns help text for all commands.
func (c *CommandProvider) getHelp() string {
	cmds := c.GetCommands()
	var sb strings.Builder
	sb.WriteString(i18n.T("command.available_commands"))
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("  %s - %s\n", cmd.DisplayName, cmd.Description))
	}
	return sb.String()
}

// getToolsList returns a list of available tools.
func (c *CommandProvider) getToolsList() string {
	return `Available tools:

File System:
  file_read   - Read file contents
  file_write  - Write to files (requires confirmation)
  file_search - Search in files
  file_list   - List directory contents

System:
  bash        - Execute shell commands (requires confirmation)

Remote (SSH):
  ssh_exec    - Execute commands on remote servers (requires confirmation)

Web:
  web_fetch   - Fetch content from URLs
  web_search  - Search the web

Utility:
  datetime    - Get date and time information
  calculator  - Perform calculations
  text        - Text manipulation

Knowledge:
  knowledge_search - Search personal knowledge base
  knowledge_import - Import documents to knowledge base
`
}

// MemoryProvider provides memory operations for command handlers.
type MemoryProvider interface {
	Clear()
	Get() []memory.Message
	GetTokenCount() int
	MaybeSummarize(ctx context.Context) error
}

// TaskProvider provides task operations for command handlers.
type TaskProvider interface {
	Clear()
}

// RegisterDefaultCommandHandlers registers default handlers for runtime commands.
// This is the standard handler registration used by CLI, Server, and Adapters.
// Pass nil for memory if memory handlers should not be registered.
// Pass nil for taskProvider if task clearing should not be registered.
func RegisterDefaultCommandHandlers(bus *events.Bus, memory MemoryProvider, taskProvider TaskProvider, cfg *config.Config) {
	// Register memory clear handler (also clears tasks)
	bus.RegisterCommandHandler(events.CommandTypeMemoryClear, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		if memory != nil {
			memory.Clear()
		}
		if taskProvider != nil {
			taskProvider.Clear()
		}
		result := i18n.T("command.memory_cleared")
		if taskProvider != nil {
			result += i18n.T("command.tasks_cleared")
		}
		return events.CommandResponse{Success: true, Result: result}
	})

	// Register memory compact handler
	bus.RegisterCommandHandler(events.CommandTypeMemoryCompact, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		if memory != nil {
			if err := memory.MaybeSummarize(ctx); err != nil {
				return events.CommandResponse{Success: false, Result: "", Error: err}
			}
			return events.CommandResponse{Success: true, Result: i18n.T("command.memory_compacted")}
		}
		return events.CommandResponse{Success: false, Result: "compact", Error: nil}
	})

	// Register memory stats handler
	bus.RegisterCommandHandler(events.CommandTypeMemoryStats, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		if memory != nil {
			messages := memory.Get()
			tokens := memory.GetTokenCount()
			result := fmt.Sprintf(i18n.T("command.memory_stats"), len(messages), tokens)
			return events.CommandResponse{Success: true, Result: result}
		}
		return events.CommandResponse{Success: false, Result: i18n.T("command.memory_stats_not_available")}
	})

	// Register engine status handler
	bus.RegisterCommandHandler(events.CommandTypeEngineStatus, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: i18n.T("command.engine_status")}
	})

	// Register tool history handler
	bus.RegisterCommandHandler(events.CommandTypeToolHistory, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: i18n.T("command.tool_history_not_available")}
	})

	// Register LLM config handler
	bus.RegisterCommandHandler(events.CommandTypeLLMConfig, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		if cfg != nil {
			result := fmt.Sprintf(i18n.T("command.model_info"), cfg.LLM.Provider, cfg.LLM.Model)
			return events.CommandResponse{Success: true, Result: result}
		}
		return events.CommandResponse{Success: false, Result: i18n.T("command.model_info_not_available")}
	})

	// Register session role handler
	bus.RegisterCommandHandler(events.CommandTypeSessionRole, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: i18n.T("command.no_role_set")}
	})
}
