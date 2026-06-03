// Package runtime provides the unified runtime for the agent system.
package runtime

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/logger"

	agentloader "github.com/oneliang/aura/agent/pkg/loader"
	commands "github.com/oneliang/aura/commands/pkg"
	enginepkg "github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/intent"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/core/pkg/prompt"
	"github.com/oneliang/aura/core/pkg/skilltool"
	"github.com/oneliang/aura/skill/pkg/loader"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	tools "github.com/oneliang/aura/tools/pkg"

	"github.com/oneliang/aura/habit/pkg/manager"

	mcpmanager "github.com/oneliang/aura/mcp/pkg/manager"

	"github.com/oneliang/aura/shared/pkg/hooks"
)

// MCPManager is the exported type alias for the MCP manager.
type MCPManager = mcpmanager.Manager

// RuntimeOption is a function that configures the runtime.
type RuntimeOption func(*AgentRuntime)

// AgentRuntime is the unified runtime for all modes (CLI, TUI, API).
type AgentRuntime struct {
	config        *RuntimeConfig
	llmClient     llm.Client
	httpClient    *http.Client // Shared HTTP client for LLM requests
	webHttpClient *http.Client // Shared HTTP client for web tools
	agent         *enginepkg.Engine
	permMgr       *permissions.Manager
	promptBuilder *prompt.PromptBuilder
	memory        *memory.SessionMemory

	// Command provider for internal commands
	commandProvider commands.Command

	// Intent service for natural language command recognition
	intentService *intent.Service

	// Skill loader
	skillLoader *loader.Loader

	// Skill injection (for skill_activate tool)
	skillInjector *skilltool.SkillInjector

	// Agent loader for LLM-triggered SubAgents
	agentLoader *agentloader.Loader

	// Agent delegation function for LLM-triggered delegation
	agentDelegateFn func(ctx context.Context, agentName string, task string) (string, error)

	// Callbacks
	handlerMu sync.RWMutex // protects onEvent and onConfirm
	onEvent   EventHandler
	onConfirm EventConfirmationHandler

	// Runtime mode
	mode RuntimeMode

	// Session management
	sessionID    string
	userID       string
	sessionStore *jsonl.MessageStore
	dataDir      string // Session data directory for task persistence

	// Habit tracking
	habitManager *manager.Manager

	// MCP server manager for dynamic tool loading
	mcpManager *mcpmanager.Manager

	// Hooks engine for event-driven subprocess integration
	hookEngine *hooks.Engine

	// State
	initialized bool
	toolNames   []string
	toolNamesMu sync.RWMutex

	// Cleanup goroutine control
	cleanupMu     sync.Mutex // Protects cleanupCtx, cleanupCancel, cleanupDone
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupDone   chan struct{}

	// Sub-agent fast path fields
	skipInitialize      bool         // Skip expensive Initialize() steps for sub-agents
	preBuiltTools       []tools.Tool // Pre-created tools shared from parent runtime
	parentToolAllowlist []string     // Tool allowlist inherited from parent agent

	// Logger
	logger *logger.Logger

	// ===== Component fields (new architecture) =====
	// Shared resources (pointer shared with sub-agents)
	shared *SharedResources

	// Optional systems (nil if disabled, shared pointer if enabled)
	skills *SkillSystem
	agents *AgentSystem
	mcp    *MCPSystem
	hooks  *HookSystem

	// Prompt cache manager for LLM prompt caching
	cacheManager *prompt.PromptCacheManager

	// Session context (new instance per sub-agent)
	session *SessionContext
}

// WithCommands sets the command provider for internal commands.
// This allows the application layer to inject a command.Command implementation.
func WithCommands(cmdProvider commands.Command) RuntimeOption {
	return func(r *AgentRuntime) {
		r.commandProvider = cmdProvider
	}
}

// WithIntentService sets the intent service for natural language command recognition.
func WithIntentService(intentSvc *intent.Service) RuntimeOption {
	return func(r *AgentRuntime) {
		r.intentService = intentSvc
	}
}

// WithLogger sets the logger for the runtime.
func WithLogger(log *logger.Logger) RuntimeOption {
	return func(r *AgentRuntime) {
		r.logger = log
	}
}

// WithEventHandler sets the event handler callback.
func WithEventHandler(handler EventHandler) RuntimeOption {
	return func(r *AgentRuntime) {
		r.onEvent = handler
	}
}

// WithConfirmationHandler sets the confirmation handler callback.
func WithConfirmationHandler(handler EventConfirmationHandler) RuntimeOption {
	return func(r *AgentRuntime) {
		r.onConfirm = handler
	}
}

// WithSessionStore sets the session store for persistence.
func WithSessionStore(store *jsonl.MessageStore) RuntimeOption {
	return func(r *AgentRuntime) {
		r.sessionStore = store
	}
}

// WithDataDir sets the session data directory for task persistence.
func WithDataDir(dataDir string) RuntimeOption {
	return func(r *AgentRuntime) {
		r.dataDir = dataDir
	}
}

// WithSessionID sets the session ID.
func WithSessionID(id string) RuntimeOption {
	return func(r *AgentRuntime) {
		r.sessionID = id
	}
}

// WithUserID sets the user ID.
func WithUserID(userID string) RuntimeOption {
	return func(r *AgentRuntime) {
		r.userID = userID
	}
}

// WithMode sets the runtime mode.
func WithMode(mode RuntimeMode) RuntimeOption {
	return func(r *AgentRuntime) {
		r.mode = mode
	}
}

// WithMCPManager sets the MCP manager for dynamic tool loading.
func WithMCPManager(mgr *mcpmanager.Manager) RuntimeOption {
	return func(r *AgentRuntime) {
		r.mcpManager = mgr
	}
}

// WithAutoApprove enables auto-approve mode for all tool executions.
// When enabled, all permissions default to "allow" - no confirmation required.
// Useful for SDK usage without interactive environment.
func WithAutoApprove() RuntimeOption {
	return func(r *AgentRuntime) {
		r.config.AutoApprove = true
	}
}

// New creates a new agent runtime.
func New(cfg *RuntimeConfig, opts ...RuntimeOption) (*AgentRuntime, error) {
	r := &AgentRuntime{
		config:    cfg,
		sessionID: cfg.SessionID,
		mode:      RuntimeModeCLI, // Default to CLI mode
	}

	for _, opt := range opts {
		opt(r)
	}

	// Use fallback logger if not injected (TUI mode should inject via WithLogger)
	if r.logger == nil {
		r.logger = logger.NewNamed(logger.Config{Level: "info", Format: "text", Output: "stdout", Module: "runtime"})
	}

	return r, nil
}

// NewSubAgentRuntime creates a lightweight sub-agent runtime that shares
// expensive resources with the parent runtime via component pointers.
//
// It shares: SharedResources, SkillSystem, AgentSystem, MCPSystem, HookSystem.
// It creates: new SessionContext, new memory, new Engine instance.
//
// The subCfg should be pre-built via buildSubAgentConfig() with all overrides applied.
// If the config contains an LLM model override (different from parent), the caller
// should use the full delegation path instead.
//
// If delegationLogger is provided, the sub-agent will log to its independent file.
// Otherwise, it shares the parent's logger.
func NewSubAgentRuntime(parent *AgentRuntime, subCfg *RuntimeConfig, disabledTools []string, delegationLogger *logger.DelegationFileLogger) (*AgentRuntime, error) {
	// Validate parent runtime is properly initialized
	if parent.shared == nil {
		return nil, fmt.Errorf("parent runtime not properly initialized: shared resources nil")
	}

	subLogger := parent.logger
	if delegationLogger != nil {
		subLogger = delegationLogger.Logger().WithModule("sub-agent")
	}

	// Inherit tool allowlist from parent (if parent engine exists)
	var parentAllowlist []string
	if parent.agent != nil {
		parentAllowlist = parent.agent.GetToolAllowlist()
	}

	// Handle permission inheritance
	var subPermMgr *permissions.Manager
	var subShared *SharedResources
	permMode := permissions.ParsePermissionInheritanceStrategy(subCfg.PermissionMode)
	switch permMode {
	case permissions.PermissionInheritDowngrade:
		// Create downgraded permission manager
		permLevel := permissions.ParsePermissionControlLevel(subCfg.PermissionLevel)
		if permLevel == permissions.ControlDeny {
			// Deny is most restrictive - useful for read-only sub-agents
			subPermMgr = parent.shared.ClonePermMgrWithDowngrade(permLevel)
		} else if permLevel == permissions.ControlAsk {
			// Ask requires confirmation for all operations
			subPermMgr = parent.shared.ClonePermMgrWithDowngrade(permLevel)
		} else {
			// Allow is same as inherit, use parent's permMgr
			subPermMgr = parent.shared.GetPermMgr()
		}
		// Create new SharedResources with cloned permMgr
		subShared = &SharedResources{
			llmClient:       parent.shared.llmClient,
			httpClient:      parent.shared.httpClient,
			webHttpClient:   parent.shared.webHttpClient,
			permMgr:         subPermMgr,
			commandProvider: parent.shared.commandProvider,
		}
	case permissions.PermissionIndependent:
		// Create independent permission manager with default config
		// Note: This creates a new manager with default settings, not inheriting from parent
		permLevel := permissions.ParsePermissionControlLevel(subCfg.PermissionLevel)
		if permLevel != "" {
			// Create from parent with the specified level
			subPermMgr = parent.shared.ClonePermMgrWithDowngrade(permLevel)
		} else {
			// Use default config
			subPermMgr, _ = permissions.NewManager(permissions.DefaultPermissionConfig())
		}
		subShared = &SharedResources{
			llmClient:       parent.shared.llmClient,
			httpClient:      parent.shared.httpClient,
			webHttpClient:   parent.shared.webHttpClient,
			permMgr:         subPermMgr,
			commandProvider: parent.shared.commandProvider,
		}
	case permissions.PermissionReadonly:
		// Auto readonly mode: inherit_downgrade with ControlDeny
		// Only read-only tools are allowed, all write operations are blocked
		subPermMgr = parent.shared.ClonePermMgrWithDowngrade(permissions.ControlDeny)
		subShared = &SharedResources{
			llmClient:       parent.shared.llmClient,
			httpClient:      parent.shared.httpClient,
			webHttpClient:   parent.shared.webHttpClient,
			permMgr:         subPermMgr,
			commandProvider: parent.shared.commandProvider,
		}
	default: // PermissionInherit
		// Share parent's permission manager (default behavior)
		subPermMgr = parent.shared.GetPermMgr()
		subShared = parent.shared
	}

	subAgent := &AgentRuntime{
		config:              subCfg,
		sessionID:           "",
		mode:                parent.mode,      // Inherit parent runtime mode (TUI/CLI/API)
		onConfirm:           parent.onConfirm, // Share confirmation handler
		logger:              subLogger,
		skipInitialize:      true,
		parentToolAllowlist: parentAllowlist, // Inherit tool allowlist from parent

		// Share or create components based on permission mode
		shared: subShared,
		skills: parent.skills,
		agents: parent.agents,
		mcp:    parent.mcp, // Note: started=false prevents sub-agent from stopping MCP
		hooks:  parent.hooks,

		// Keep original fields for backward compatibility during migration
		httpClient:      parent.httpClient,
		webHttpClient:   parent.webHttpClient,
		llmClient:       parent.llmClient,
		permMgr:         subPermMgr,
		mcpManager:      parent.mcpManager,
		skillLoader:     parent.skillLoader,
		skillInjector:   parent.skillInjector,
		agentLoader:     parent.agentLoader,
		commandProvider: parent.commandProvider,
		hookEngine:      parent.hookEngine,
	}

	// Filter and share tools from parent
	var filteredTools []tools.Tool
	if parent.agent != nil {
		parentTools := parent.agent.GetTools()
		filteredTools = filterToolsByNames(parentTools, disabledTools)
	}
	subAgent.preBuiltTools = filteredTools

	return subAgent, nil
}

// Shutdown shuts down the runtime.
// Note: This does NOT clear the persistent storage. Session data is preserved
// so it can be loaded again when the server restarts.
// Sub-agent runtimes (skipInitialize=true) do NOT stop MCP servers as they
// are owned by the parent runtime.
func (r *AgentRuntime) Shutdown() {
	// Wait for all pending memory persistence operations to complete
	if r.memory != nil {
		r.memory.Shutdown()
	}

	// Fire SessionEnd hook via component (non-blocking)
	sessionID := r.sessionID
	if r.session != nil {
		sessionID = r.session.GetSessionID()
	}
	r.hooks.Fire(context.Background(), hooks.EventSessionEnd, map[string]any{
		"session_id": sessionID,
	})

	// Only stop MCP servers if this is the parent runtime (owns them)
	// MCPSystem.StopAll checks started flag internally
	r.mcp.StopAll(context.Background())

	// Shut down hooks engine (closes file watchers if any)
	// Only parent should shutdown hooks (sub-agent shares pointer but shouldn't close)
	if !r.skipInitialize {
		r.hooks.Shutdown()
	}

	// Mark as not initialized
	r.initialized = false
	if r.session != nil {
		r.session.SetInitialized(false)
	}

	// Stop cleanup goroutine
	r.stopCleanupGoroutine()
}

// startCleanupGoroutine starts a background goroutine that periodically cleans up
// stale memory based on RetentionConfig. Only runs for parent runtime (not sub-agents).
// Sub-agents inherit parent's memory lifecycle; running independent cleanup could race
// with parent's cleanup goroutine.
func (r *AgentRuntime) startCleanupGoroutine() {
	if r.skipInitialize {
		// Sub-agents don't run cleanup - they share parent's memory lifecycle
		return
	}

	retention := r.config.Memory.Retention
	if retention.CleanupInterval <= 0 {
		r.logger.Debug().Msg("Memory cleanup disabled (cleanup_interval not set)")
		return
	}

	r.cleanupMu.Lock()
	r.cleanupCtx, r.cleanupCancel = context.WithCancel(context.Background())
	r.cleanupDone = make(chan struct{})
	r.cleanupMu.Unlock()

	go func() {
		defer func() {
			r.cleanupMu.Lock()
			if r.cleanupDone != nil {
				close(r.cleanupDone)
			}
			r.cleanupMu.Unlock()
		}()
		ticker := time.NewTicker(retention.CleanupInterval)
		defer ticker.Stop()

		r.logger.Info().Dur("interval", retention.CleanupInterval).Msg("Memory cleanup goroutine started")

		for {
			select {
			case <-r.cleanupCtx.Done():
				r.logger.Debug().Msg("Memory cleanup goroutine stopped")
				return
			case <-ticker.C:
				r.cleanupStaleMemory(retention.MaxInactiveAge)
			}
		}
	}()
}

// stopCleanupGoroutine stops the background cleanup goroutine safely.
// Uses mutex to prevent race conditions when Shutdown is called concurrently.
func (r *AgentRuntime) stopCleanupGoroutine() {
	r.cleanupMu.Lock()
	if r.cleanupCancel == nil {
		r.cleanupMu.Unlock()
		return
	}
	r.cleanupCancel()
	cancelDone := r.cleanupDone
	r.cleanupCancel = nil
	r.cleanupDone = nil
	r.cleanupMu.Unlock()

	// Wait for goroutine to finish outside lock
	if cancelDone != nil {
		<-cancelDone
	}
}

// cleanupStaleMemory removes stale messages from memory based on threshold.
// Preserves summary if memory supports SummarizingMemory interface.
func (r *AgentRuntime) cleanupStaleMemory(threshold time.Duration) {
	if r.memory == nil || threshold <= 0 {
		return
	}

	if r.memory.IsStale(threshold) {
		r.logger.Info().Dur("threshold", threshold).Msg("Memory is stale, clearing messages (preserving summary if available)")
		// SessionMemory embeds BaseMemory which has ClearPreserveSummary
		r.memory.ClearPreserveSummary()
	}
}

// GetToolNames returns the list of registered tool names.
func (r *AgentRuntime) GetToolNames() []string {
	r.toolNamesMu.RLock()
	defer r.toolNamesMu.RUnlock()
	result := make([]string, len(r.toolNames))
	copy(result, r.toolNames)
	return result
}

// GetAgent returns the underlying agent instance.
func (r *AgentRuntime) GetAgent() *enginepkg.Engine {
	return r.agent
}

// GetMemory returns the session memory.
func (r *AgentRuntime) GetMemory() *memory.SessionMemory {
	return r.memory
}

// GetSummarizer returns the summarizer from session memory.
func (r *AgentRuntime) GetSummarizer() *memory.Summarizer {
	if r.memory != nil {
		return r.memory.GetSummarizer()
	}
	return nil
}

// AddTool adds a tool to the agent after initialization.
// This is useful for adding UI-specific tools like internal commands.
func (r *AgentRuntime) AddTool(tool tools.Tool) error {
	if r.agent == nil {
		return fmt.Errorf("agent not initialized")
	}
	r.toolNamesMu.Lock()
	r.agent.AddTool(tool)
	r.toolNames = append(r.toolNames, tool.Name())
	r.toolNamesMu.Unlock()
	return nil
}

// SetEventHandler sets the event handler callback.
// This allows dynamic configuration of event handling, useful for adapters.
func (r *AgentRuntime) SetEventHandler(handler EventHandler) {
	r.handlerMu.Lock()
	r.onEvent = handler
	r.handlerMu.Unlock()
}

// SetConfirmationHandler sets the confirmation handler callback.
// This allows dynamic configuration of confirmation handling.
func (r *AgentRuntime) SetConfirmationHandler(handler EventConfirmationHandler) {
	r.handlerMu.Lock()
	r.onConfirm = handler
	r.handlerMu.Unlock()
}

// PlanReviewHandler returns a handler that triggers plan review confirmation
// via the onConfirm callback and blocks waiting for user approval.
// If AutoApprove is enabled, returns true immediately without triggering confirmation.
func (r *AgentRuntime) PlanReviewHandler() func(ctx context.Context, goal string, steps []string) (bool, error) {
	return func(ctx context.Context, goal string, steps []string) (bool, error) {
		// AutoApprove: skip confirmation and return true immediately
		if r.config.AutoApprove {
			r.logger.Debug().Str("module", "runtime").Msg("PlanReviewHandler: auto-approving plan (AutoApprove enabled)")
			return true, nil
		}

		respCh := make(chan bool, 1)
		r.handlerMu.RLock()
		handler := r.onConfirm
		if handler == nil {
			r.handlerMu.RUnlock()
			return false, fmt.Errorf("plan review: no confirmation handler configured")
		}
		r.handlerMu.RUnlock()
		handler(ConfirmationRequest{
			Type:       "plan_review",
			PlanGoal:   goal,
			PlanSteps:  steps,
			ResponseCh: respCh,
		})
		confirmCtx, confirmCancel := context.WithTimeout(ctx, 120*time.Second)
		defer confirmCancel()
		select {
		case approved := <-respCh:
			return approved, nil
		case <-confirmCtx.Done():
			return false, confirmCtx.Err()
		}
	}
}

// SetAgentDelegateFn sets the agent delegation function callback.
// This allows the command provider to trigger agent delegation.
func (r *AgentRuntime) SetAgentDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error)) {
	r.agentDelegateFn = fn
	// Also set in AgentSystem component
	if r.agents != nil {
		r.agents.SetDelegateFn(fn)
	}
}

// GetAgentDelegateFn returns the agent delegation function.
func (r *AgentRuntime) GetAgentDelegateFn() func(ctx context.Context, agentName string, task string) (string, error) {
	// Prefer component if available
	if r.agents != nil {
		return r.agents.GetDelegateFn()
	}
	return r.agentDelegateFn
}

// IsSkillsEnabled returns whether the skill system is enabled in the runtime config.
func (r *AgentRuntime) IsSkillsEnabled() bool {
	// Use component if available
	if r.skills != nil {
		return r.skills.IsEnabled()
	}
	return r.config.Skills.Enabled && len(r.config.Skills.Directories) > 0
}

// GetSkillDirectories returns the configured skill directories from the runtime config.
func (r *AgentRuntime) GetSkillDirectories() []string {
	// Use component if available
	if r.skills != nil {
		return r.skills.GetDirectories()
	}
	return r.config.Skills.Directories
}

// GetUserID returns the bound user ID for multi-user isolation.
func (r *AgentRuntime) GetUserID() string {
	// Use component if available
	if r.session != nil {
		return r.session.GetUserID()
	}
	return r.userID
}

// ClearTasks clears all tasks from memory and disk via the Engine.
func (r *AgentRuntime) ClearTasks() {
	if r.agent != nil {
		r.agent.ClearTasks()
	}
}

// Clear satisfies cmds.TaskProvider interface (Clear() alias for ClearTasks).
func (r *AgentRuntime) Clear() {
	r.ClearTasks()
}

// GetSystemPrompt returns the current system prompt for debugging/display purposes.
// Returns a formatted string showing all prompt layers and their cache status.
func (r *AgentRuntime) GetSystemPrompt() string {
	var result string

	// Show base system prompt
	result += "=== System Prompt ===\n\n"
	if r.config.SystemPrompt != "" {
		result += r.config.SystemPrompt
	} else if r.config.Role != "" {
		result += r.promptBuilder.BuildWithRole(r.config.Role)
	} else {
		result += r.promptBuilder.BuildWithConfig(r.config.Config)
	}

	// Show Skills block
	if r.skillLoader != nil && len(r.skillLoader.GetSkills()) > 0 {
		result += "\n\n=== Skills Block ===\n\n"
		for _, skill := range r.skillLoader.GetSkills() {
			result += fmt.Sprintf("- %s: %s\n", skill.Name, skill.Description)
		}
	}

	// Show Agents block
	if r.agentLoader != nil && len(r.agentLoader.GetAgents()) > 0 && r.config.EnableSubAgent {
		result += "\n\n=== Agents Block ===\n\n"
		for _, agent := range r.agentLoader.GetAgents() {
			result += fmt.Sprintf("- %s: %s\n", agent.Name, agent.Description)
		}
	}

	// Show AURA.md block
	auraMd := r.loadProjectAuraMd()
	if auraMd != "" {
		result += "\n\n=== AURA.md Block ===\n\n"
		result += auraMd
	}

	// Show cache status
	if r.cacheManager != nil && r.cacheManager.Enabled() {
		result += "\n\n=== Cache Status ===\n\n"
		result += "Prompt caching: ENABLED\n"
		result += fmt.Sprintf("Layers: StaticSystem(0), Tools(1), Skills(2), Agents(3), ProjectAura(4)\n")
	} else {
		result += "\n\n=== Cache Status ===\n\n"
		result += "Prompt caching: DISABLED\n"
	}

	return result
}
