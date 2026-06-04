// Package sdk provides the unified agent runtime and factories.
// This is the main entry point for all modes (CLI, TUI, API).
//
// This file is the SDK facade layer — it re-exports types and forwards calls
// to internal packages (runtime, factory, session, permissions, etc.).
// New types and functions added here should follow the existing pattern:
// type aliases for re-exported types, thin wrapper functions for factories.
// Direct modifications to internal packages should NOT be mirrored here
// unless they are part of the public API contract.
//
// Example usage (new event stream pattern):
//
//	// Create runtime from config
//	cfg := config.Load(...)
//	runtime, err := sdk.NewRuntime(sdk.FromConfig(cfg))
//	if err != nil { ... }
//
//	// Initialize
//	if err := runtime.Initialize(ctx); err != nil { ... }
//
//	// Start event stream
//	if err := runtime.Start(ctx); err != nil { ... }
//
//	// Get output event stream
//	events := runtime.Events()
//
//	// Send user input as event
//	runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "Hello, Aura!", requestID))
//
//	// Process events
//	for ev := range events {
//	    switch ev.Type() {
//	    case sdk.EventTypeInteractionRequest:
//	        // Handle interaction request
//	        runtime.SendEvent(ctx, sdk.NewEventWithExtra(sdk.EventTypeInteractionResponse, "", map[string]any{"approved": true}, ev.RequestID()))
//	    case sdk.EventTypeDone:
//	        // Processing complete
//	        break
//	    }
//	}
//
//	// Stop
//	runtime.Stop(ctx)
//	defer runtime.Shutdown()
//
package sdk

import (
	"context"
	"fmt"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/factory"
	"github.com/oneliang/aura/core/pkg/intent"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/core/pkg/prompt"
	"github.com/oneliang/aura/core/pkg/runtime"
	sessionService "github.com/oneliang/aura/session/pkg/service"
	sessionStorage "github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/storage/pkg/jsonl"

	tools "github.com/oneliang/aura/tools/pkg"

	mcpconfig "github.com/oneliang/aura/mcp/pkg/config"
	mcploader "github.com/oneliang/aura/mcp/pkg/loader"
	mcpmanager "github.com/oneliang/aura/mcp/pkg/manager"
)

// Runtime is the unified agent runtime.
type Runtime = runtime.AgentRuntime

// RuntimeOption is a runtime configuration option.
type RuntimeOption = runtime.RuntimeOption

// Event represents a runtime event.
type Event = runtime.Event

// EventType represents the type of event.
type EventType = runtime.EventType

// Event types
const (
	EventTypeThinkingStart       = runtime.EventTypeThinkingStart
	EventTypeThinkingChunk       = runtime.EventTypeThinkingChunk
	EventTypeThinkingEnd         = runtime.EventTypeThinkingEnd
	EventTypeAction              = runtime.EventTypeAction
	EventTypeResult              = runtime.EventTypeResult
	EventTypeResponse            = runtime.EventTypeResponse
	EventTypeResponseStart       = runtime.EventTypeResponseStart
	EventTypeResponseChunk       = runtime.EventTypeResponseChunk
	EventTypeResponseEnd         = runtime.EventTypeResponseEnd
	EventTypeThinkingContent     = runtime.EventTypeThinkingContent
	EventTypeError               = runtime.EventTypeError
	EventTypeStep                = runtime.EventTypeStep
	EventTypeToolStart           = runtime.EventTypeToolStart
	EventTypeToolEnd             = runtime.EventTypeToolEnd
	EventTypeConfirmationRequest = runtime.EventTypeConfirmationRequest
	EventTypeCommandMatched      = runtime.EventTypeCommandMatched
	EventTypeCommandResult       = runtime.EventTypeCommandResult
	EventTypeDone                = runtime.EventTypeDone
	EventTypeTaskCreate          = runtime.EventTypeTaskCreate
	EventTypeTaskUpdate          = runtime.EventTypeTaskUpdate
	EventTypeTaskList            = runtime.EventTypeTaskList
	EventTypePlanCreated         = runtime.EventTypePlanCreated
	EventTypePlanReviewStart     = runtime.EventTypePlanReviewStart
	EventTypePlanReviewFiles     = runtime.EventTypePlanReviewFiles
	EventTypePlanStep            = runtime.EventTypePlanStep
	EventTypePlanComplete        = runtime.EventTypePlanComplete
	EventTypePlanModeExit        = runtime.EventTypePlanModeExit
	EventTypeEnterPlanMode       = runtime.EventTypeEnterPlanMode
	EventTypePlanVerifyStart     = runtime.EventTypePlanVerifyStart
	EventTypePlanVerifyResult    = runtime.EventTypePlanVerifyResult
	EventTypePlanVerifyEnd       = runtime.EventTypePlanVerifyEnd
	EventTypeSnapshotCreated     = runtime.EventTypeSnapshotCreated
	EventTypeRollbackOffer       = runtime.EventTypeRollbackOffer
	EventTypeRollbackComplete    = runtime.EventTypeRollbackComplete
	EventTypeMaxStepsExceeded    = runtime.EventTypeMaxStepsExceeded

	// ===== 新架构：统一事件流 =====

	// IN事件类型
	EventTypeUserInput           = runtime.EventTypeUserInput
	EventTypeUserMessage         = runtime.EventTypeUserMessage
	EventTypeInteractionResponse = runtime.EventTypeInteractionResponse
	EventTypeSystemCommand       = runtime.EventTypeSystemCommand

	// OUT事件类型
	EventTypeInteractionRequest  = runtime.EventTypeInteractionRequest
	EventTypeAgentStart          = runtime.EventTypeAgentStart
	EventTypeAgentStop           = runtime.EventTypeAgentStop
)

// InteractionType 交互类型
type InteractionType = runtime.InteractionType

const (
	InteractionTypeToolConfirmation  = runtime.InteractionTypeToolConfirmation
	InteractionTypePlanReview        = runtime.InteractionTypePlanReview
	InteractionTypeAskUserQuestion   = runtime.InteractionTypeAskUserQuestion
	InteractionTypeRollbackConfirm   = runtime.InteractionTypeRollbackConfirm
)

// InteractionRequest 交互请求
type InteractionRequest = runtime.InteractionRequest

// InteractionResponse 交互响应
type InteractionResponse = runtime.InteractionResponse

// RuntimeConfig holds configuration for the Aura runtime.
type RuntimeConfig = runtime.RuntimeConfig

// NewRuntime creates a new agent runtime.
func NewRuntime(cfg *RuntimeConfig, opts ...RuntimeOption) (*Runtime, error) {
	return runtime.New(cfg, opts...)
}

// ===== 新架构：事件创建函数 =====

// NewEvent creates a new event with type, content, and request ID.
func NewEvent(typ EventType, content string, requestID string) Event {
	return events.NewEvent(typ, content, requestID)
}

// NewEventWithExtra creates a new event with extra data.
func NewEventWithExtra(typ EventType, content string, extra map[string]any, requestID string) Event {
	return events.NewEventWithExtra(typ, content, extra, requestID)
}

// NewInteractionEvent creates a new interaction event.
func NewInteractionEvent(typ EventType, interactionType InteractionType, requestID string, extra map[string]any) Event {
	return events.NewInteractionEvent(typ, interactionType, requestID, extra)
}

// FromConfig creates a runtime config from the main app config.
func FromConfig(cfg *config.Config) *RuntimeConfig {
	return runtime.FromConfig(cfg)
}

// DefaultRuntimeConfig returns a default runtime configuration.
func DefaultRuntimeConfig() *RuntimeConfig {
	return runtime.DefaultRuntimeConfig()
}

// WithSessionStore sets the session store for persistence.
func WithSessionStore(store *jsonl.MessageStore) RuntimeOption {
	return runtime.WithSessionStore(store)
}

// WithSessionID sets the session ID.
func WithSessionID(id string) RuntimeOption {
	return runtime.WithSessionID(id)
}

// WithUserID sets the user ID for multi-user isolation.
func WithUserID(id string) RuntimeOption {
	return runtime.WithUserID(id)
}

// WithDataDir sets the session data directory for task persistence.
func WithDataDir(dataDir string) RuntimeOption {
	return runtime.WithDataDir(dataDir)
}

// WithCommands sets the command provider for internal commands.
func WithCommands(cmdProvider commands.Command) RuntimeOption {
	return runtime.WithCommands(cmdProvider)
}

// WithIntentService sets the intent service for natural language command recognition.
func WithIntentService(intentSvc *intent.Service) RuntimeOption {
	return runtime.WithIntentService(intentSvc)
}

// WithLogger sets the logger for the runtime.
func WithLogger(log *logger.Logger) RuntimeOption {
	return runtime.WithLogger(log)
}

// MCP manager exports
type MCPManager = runtime.MCPManager
type ServerInfo = mcpconfig.ServerInfo
type MCPConfig = mcpconfig.Config

// WithMCPManager sets the MCP manager for dynamic tool loading.
func WithMCPManager(mgr *MCPManager) RuntimeOption {
	return runtime.WithMCPManager(mgr)
}

// WithAutoApprove enables auto-approve mode for all tool executions.
// When enabled, all permissions default to "allow" - no confirmation required.
// Useful for SDK usage without interactive environment.
func WithAutoApprove() RuntimeOption {
	return runtime.WithAutoApprove()
}

// NewMCPManager creates an MCP manager from the default config file (~/.aura/mcp.json).
// Returns nil if the config file does not exist or has no servers configured.
func NewMCPManager() *MCPManager {
	mcpLdr := mcploader.NewLoader("")
	if _, err := mcpLdr.Load(); err == nil {
		if len(mcpLdr.GetServers()) > 0 {
			return mcpmanager.NewManager(mcpLdr)
		}
	}
	return nil
}

// AddMCPServer adds a new MCP server, discovers tools, then stops the server.
// Config is persisted; the runtime will start the server on next Initialize.
// Returns discovered tool info for display.
func AddMCPServer(ctx context.Context, name, command string, args []string) ([]tools.Tool, error) {
	cfg := mcpconfig.ServerConfig{
		Command: command,
		Args:    args,
	}
	mgr := NewMCPManager()
	if mgr == nil {
		mcpLdr := mcploader.NewLoader("")
		if _, err := mcpLdr.Load(); err != nil {
			// Non-fatal: load may fail if config file doesn't exist yet
			// Continue with empty config
		}
		mgr = mcpmanager.NewManager(mcpLdr)
	}
	tools, err := mgr.AddServer(ctx, name, cfg)
	if err != nil {
		return nil, err
	}
	// Config is persisted by AddServer. Server process stays alive until this CLI exits.
	// On next runtime Initialize(), the server will be started from the saved config.
	return tools, nil
}

// RemoveMCPServer removes an MCP server by name.
func RemoveMCPServer(ctx context.Context, name string) error {
	mcpLdr := mcploader.NewLoader("")
	if _, err := mcpLdr.Load(); err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}
	mgr := mcpmanager.NewManager(mcpLdr)
	return mgr.RemoveServer(ctx, name)
}

// LoadMCPConfig loads the raw MCP config without starting servers.
// Used by CLI for listing configured servers.
func LoadMCPConfig() (*mcpconfig.Config, error) {
	mcpLdr := mcploader.NewLoader("")
	return mcpLdr.Load()
}

// ListMCPServers returns all configured MCP servers.
func ListMCPServers() []ServerInfo {
	mcpLdr := mcploader.NewLoader("")
	if _, err := mcpLdr.Load(); err != nil {
		return nil
	}
	mgr := mcpmanager.NewManager(mcpLdr)
	return mgr.ListServers()
}

// GetMCPServerStatus returns the runtime status of a specific MCP server.
func GetMCPServerStatus(ctx context.Context, name string) *ServerInfo {
	mcpLdr := mcploader.NewLoader("")
	if _, err := mcpLdr.Load(); err != nil {
		return nil
	}
	mgr := mcpmanager.NewManager(mcpLdr)
	info := mgr.ServerInfoForName(name)
	return &info
}

// Message source.
type MessageSource = memory.MessageSource

const (
	SourceCLI = memory.SourceCLI
	SourceTUI = memory.SourceTUI
	SourceAPI = memory.SourceAPI
)

// Core type aliases.
type (
	Command           = commands.Command
	Summarizer        = memory.Summarizer
	IntentService     = intent.Service
	PermissionManager = permissions.Manager
)

// NewIntentService creates a new intent recognition service.
func NewIntentService(cmdProvider commands.Command, threshold float64) *IntentService {
	return intent.NewService(cmdProvider, threshold)
}

// Session service exports
type SessionService = sessionService.Service

// NewSessionService creates a new session service.
func NewSessionService(store *sessionStorage.JSONLStore) *SessionService {
	return sessionService.NewService(store)
}

// Factory exports
type (
	LLMFactory    = factory.LLMFactory
	EngineFactory = factory.EngineFactory
	ToolRegistry  = factory.ToolRegistry
	PromptBuilder = prompt.PromptBuilder
	RoleLoader    = prompt.RoleLoader
)

// NewLLMFactory creates a new LLM factory.
func NewLLMFactory(cfg *config.LLMConfig) *LLMFactory {
	return factory.NewLLMFactory(cfg)
}

// NewEngineFactory creates a new engine factory.
func NewEngineFactory(llmClient llm.Client, cfg *config.AgentConfig, permMgr *permissions.Manager, opts ...factory.EngineFactoryOption) *EngineFactory {
	return factory.NewEngineFactory(llmClient, cfg, permMgr, opts...)
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(cfg *config.ToolsConfig, permMgr *permissions.Manager) *ToolRegistry {
	return factory.NewToolRegistry(cfg, permMgr)
}

// Permission config types.
type (
	PermissionConfig    = permissions.PermissionConfig
	CommandRestrictions = permissions.CommandRestrictions
	SSHRestrictions     = permissions.SSHRestrictions
)

// NewPermissionConfig creates a permission config from raw values.
func NewPermissionConfig(
	defaultLevel string,
	tools map[string]string,
	shellRestrictions CommandRestrictions,
	sshRestrictions SSHRestrictions,
	trustedDirs []string,
	autoAskTrust bool,
) *PermissionConfig {
	return permissions.NewPermissionConfig(defaultLevel, tools, shellRestrictions, sshRestrictions, trustedDirs, autoAskTrust)
}

// NewPermissionManagerFromConfig creates a permission manager from a PermissionConfig.
func NewPermissionManagerFromConfig(cfg *PermissionConfig) (*PermissionManager, error) {
	return permissions.NewManager(cfg)
}

// NewPermissionManager creates a new permission manager from app config.
func NewPermissionManager(cfg *config.PermissionsConfig) (*PermissionManager, error) {
	permCfg := permissions.NewPermissionConfig(
		cfg.DefaultLevel,
		cfg.Tools,
		permissions.CommandRestrictions{
			AllowedCommands: cfg.ShellRestrictions.AllowedCommands,
			DeniedCommands:  cfg.ShellRestrictions.DeniedCommands,
		},
		permissions.SSHRestrictions{
			AllowedHosts:    cfg.SSHRestrictions.AllowedHosts,
			DeniedHosts:     cfg.SSHRestrictions.DeniedHosts,
			AllowedCommands: cfg.SSHRestrictions.AllowedCommands,
			DeniedCommands:  cfg.SSHRestrictions.DeniedCommands,
		},
		cfg.TrustedDirs,
		cfg.AutoAskTrust,
	)
	return permissions.NewManager(permCfg)
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder(roleLoader *RoleLoader) *PromptBuilder {
	return prompt.NewPromptBuilder(roleLoader)
}

// NewRoleLoader creates a new role loader.
func NewRoleLoader(baseDir string) *RoleLoader {
	return prompt.NewRoleLoader(baseDir)
}
