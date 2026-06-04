// Package runtime provides the unified runtime for the agent system.
package runtime

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"

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

	// ===== 新架构：统一事件流 =====

	// OUT: 发送事件通道
	eventOutCh chan Event

	// IN: 交互请求-响应匹配
	interactionMu     sync.RWMutex
	interactionPending map[string]chan events.InteractionResponse  // RequestID → ResponseCh

	// 运行状态
	running bool
	runMu   sync.Mutex

	// 输入队列（顺序处理，避免嵌套事件循环）
	inputQueue  chan inputRequest
	processWg   sync.WaitGroup // 等待处理完成
	processCancel context.CancelFunc // Cancel function for processInputQueue context

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

// inputRequest 封装用户输入请求
type inputRequest struct {
	Input     string
	RequestID string
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
	// Stop Engine's processInputQueue goroutine first
	if r.agent != nil {
		r.agent.Shutdown()
	}

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

// PlanReviewHandler returns a handler that triggers plan review confirmation
// via the event stream and blocks waiting for user approval.
// If AutoApprove is enabled, returns true immediately without triggering confirmation.
func (r *AgentRuntime) PlanReviewHandler() func(ctx context.Context, goal string, steps []string) (bool, error) {
	return func(ctx context.Context, goal string, steps []string) (bool, error) {
		// AutoApprove: skip confirmation and return true immediately
		if r.config.AutoApprove {
			r.logger.Debug().Str("module", "runtime").Msg("PlanReviewHandler: auto-approving plan (AutoApprove enabled)")
			return true, nil
		}

		// Use event stream for plan review confirmation
		req := events.InteractionRequest{
			Type:      events.InteractionTypePlanReview,
			PlanGoal:  goal,
			PlanSteps: steps,
			Timeout:   120 * time.Second,
		}

		resp := r.requestInteraction(ctx, req)
		if resp.Error != nil {
			return false, resp.Error
		}
		return resp.Approved, nil
	}
}

// RollbackConfirmHandler returns a rollback confirm handler that uses the event stream.
func (r *AgentRuntime) RollbackConfirmHandler() func(ctx context.Context, snapshotID string, files []string, reason string) (bool, error) {
	return func(ctx context.Context, snapshotID string, files []string, reason string) (bool, error) {
		req := events.InteractionRequest{
			Type:           events.InteractionTypeRollbackConfirm,
			RollbackReason: reason,
			Timeout:        60 * time.Second,
		}
		resp := r.requestInteraction(ctx, req)
		if resp.Error != nil {
			return false, resp.Error
		}
		return resp.Approved, nil
	}
}

// AskUserQuestionHandler returns an ask user question handler that uses the event stream.
func (r *AgentRuntime) AskUserQuestionHandler() func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error) {
	return func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error) {
		req := events.InteractionRequest{
			Type:         events.InteractionTypeAskUserQuestion,
			Question:     question,
			QuestionType: questionType,
			Options:      options,
			Timeout:      120 * time.Second,
		}
		resp := r.requestInteraction(ctx, req)
		if resp.Cancelled {
			return nil, fmt.Errorf("user cancelled the question")
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return &events.QuestionResponse{
			Answer:    resp.AnswerText,
			Answers:   resp.Selections,
			Cancelled: resp.Cancelled,
		}, nil
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

// ===== 新架构：统一事件流接口 =====

// Start 启动Agent（初始化事件通道）
func (r *AgentRuntime) Start(ctx context.Context) error {
	r.runMu.Lock()
	if r.running {
		r.runMu.Unlock()
		return fmt.Errorf("agent already running")
	}
	r.running = true
	r.eventOutCh = make(chan Event, 100)
	r.interactionPending = make(map[string]chan events.InteractionResponse)
	r.inputQueue = make(chan inputRequest, 10)

	// Create cancellable context for processInputQueue
	processCtx, processCancel := context.WithCancel(ctx)
	r.processCancel = processCancel

	r.runMu.Unlock()

	// 启动输入处理goroutine（顺序处理，避免嵌套）
	r.processWg.Add(1) // Add before go to avoid race
	go r.processInputQueue(processCtx)

	// 发送启动事件
	r.sendEvent(events.NewEvent(events.EventTypeAgentStart, "", ""))

	return nil
}

// Stop 停止Agent
func (r *AgentRuntime) Stop(ctx context.Context) error {
	r.runMu.Lock()
	if !r.running {
		r.runMu.Unlock()
		return nil
	}
	eventOutCh := r.eventOutCh
	r.eventOutCh = nil
	inputQueue := r.inputQueue
	r.inputQueue = nil
	processCancel := r.processCancel
	r.processCancel = nil
	r.running = false
	r.runMu.Unlock()

	// Cancel processInputQueue context first (stops goroutine waiting in select)
	if processCancel != nil {
		processCancel()
	}

	// 关闭输入队列，等待处理goroutine退出
	if inputQueue != nil {
		close(inputQueue)
	}
	r.processWg.Wait()

	// 发送停止事件（在unlock后，使用缓存的channel）
	if eventOutCh != nil {
		select {
		case eventOutCh <- events.NewEvent(events.EventTypeAgentStop, "", ""):
		default:
			// 通道满或已关闭，忽略
		}
		close(eventOutCh)
	}

	return nil
}

// SendEvent 统一入口：接收事件（IN）
// 所有输入都通过这个方法：用户文本、交互响应、系统命令等
func (r *AgentRuntime) SendEvent(ctx context.Context, event Event) error {
	switch event.Type() {
	case events.EventTypeUserInput:
		// 用户文本输入
		return r.handleUserInput(ctx, event.Content())

	case events.EventTypeUserMessage:
		// 用户消息（带metadata）
		return r.handleUserMessage(ctx, event)

	case events.EventTypeInteractionResponse:
		// 交互响应
		return r.handleInteractionResponse(event)

	case events.EventTypeSystemCommand:
		// 系统命令
		return r.handleSystemCommand(ctx, event)

	default:
		// 未知事件类型
		return fmt.Errorf("unknown event type: %s", event.Type())
	}
}

// Events 获取输出事件流（OUT）
func (r *AgentRuntime) Events() <-chan Event {
	return r.eventOutCh
}

// handleUserInput 处理用户文本输入
func (r *AgentRuntime) handleUserInput(ctx context.Context, input string) error {
	if !r.initialized {
		return fmt.Errorf("runtime not initialized")
	}

	// 提交到输入队列，由processInputQueue顺序处理
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	req := inputRequest{Input: input, RequestID: requestID}

	select {
	case r.inputQueue <- req:
		return nil
	default:
		return fmt.Errorf("input queue full")
	}
}

// handleUserMessage 处理用户消息（带metadata）
func (r *AgentRuntime) handleUserMessage(ctx context.Context, event Event) error {
	if !r.initialized {
		return fmt.Errorf("runtime not initialized")
	}

	// 从Extra中提取输入内容
	input := event.Content()
	if input == "" {
		if content, ok := event.Extra()["content"].(string); ok {
			input = content
		}
	}

	// 提交到输入队列，使用事件的RequestID
	requestID := event.RequestID()
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	req := inputRequest{Input: input, RequestID: requestID}

	select {
	case r.inputQueue <- req:
		return nil
	default:
		return fmt.Errorf("input queue full")
	}
}

// handleInteractionResponse 处理交互响应
func (r *AgentRuntime) handleInteractionResponse(event Event) error {
	// 解析响应
	resp := events.InteractionResponse{
		RequestID:  event.RequestID(),
	 Approved:   false,
	}

	// 从Extra中提取响应数据
	if extra := event.Extra(); extra != nil {
		if approved, ok := extra["approved"].(bool); ok {
			resp.Approved = approved
		}
		if cancelled, ok := extra["cancelled"].(bool); ok {
			resp.Cancelled = cancelled
		}
		if answer, ok := extra["answer"].(string); ok {
			resp.AnswerText = answer
		}
		if selection, ok := extra["selection"].(string); ok {
			resp.Selection = selection
		}
		if selections, ok := extra["selections"].([]string); ok {
			resp.Selections = selections
		}
		if err, ok := extra["error"].(error); ok {
			resp.Error = err
		}
		if typ, ok := extra["type"].(events.InteractionType); ok {
			resp.Type = typ
		} else if typStr, ok := extra["type"].(string); ok {
			// Fallback: handle string type (from JSON deserialization or cross-boundary)
			resp.Type = events.InteractionType(typStr)
		}
	}

	// 找到匹配的响应channel并发送响应（在锁内发送避免竞态）
	r.interactionMu.Lock()
	respCh, exists := r.interactionPending[resp.RequestID]
	if exists {
		select {
		case respCh <- resp:
			// 发送成功，从pending中移除
			delete(r.interactionPending, resp.RequestID)
		default:
			// channel已满或已关闭，清理
			delete(r.interactionPending, resp.RequestID)
		}
	}
	r.interactionMu.Unlock()

	return nil
}

// handleSystemCommand 处理系统命令
func (r *AgentRuntime) handleSystemCommand(ctx context.Context, event Event) error {
	// 系统命令处理（可扩展）
	return nil
}

// sendEvent 发送事件到OUT通道
func (r *AgentRuntime) sendEvent(event Event) {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	if !r.running || r.eventOutCh == nil {
		return
	}

	select {
	case r.eventOutCh <- event:
	default:
		// 通道满，丢弃事件（或记录警告）
		r.logger.Warn().Str("type", string(event.Type())).Msg("Event channel full, dropping event")
	}
}

// processInputQueue 顺序处理输入队列
// 这避免了嵌套事件循环问题，确保一个请求完成后再处理下一个
func (r *AgentRuntime) processInputQueue(ctx context.Context) {
	defer r.processWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-r.inputQueue:
			if !ok {
				return
			}
			// 处理单个请求
			r.sendEvent(events.NewEvent(events.EventTypeThinkingStart, "Processing user input...", req.RequestID))

			// Add user message to memory before processing
			// This is essential for buildReActMessages to include user input
			r.memory.AddWithType(sharedmemory.RoleUser, req.Input, sharedmemory.MessageTypeUser)

			// 直接调用Engine.Run而非Process包装器
			eventsCh, err := r.agent.Run(ctx, req.Input)
			if err != nil {
				r.sendEvent(events.NewEvent(events.EventTypeError, err.Error(), req.RequestID))
				r.sendEvent(events.NewEvent(events.EventTypeDone, "", req.RequestID))
				continue
			}

			// 转发事件到OUT通道，同时响应context取消
			eventLoop:
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-eventsCh:
					if !ok {
						break eventLoop
					}
					// 设置RequestID以确保事件流匹配
					if ev.RequestID() == "" {
						ev = events.NewEvent(ev.Type(), ev.Content(), req.RequestID)
						if extra := ev.Extra(); extra != nil {
							ev = events.NewEventWithExtra(ev.Type(), ev.Content(), extra, req.RequestID)
						}
					}
					r.sendEvent(ev)
				}
			}

			// 确保发送Done事件
			r.sendEvent(events.NewEvent(events.EventTypeDone, "", req.RequestID))
		}
	}
}

// requestInteraction 发送交互请求（Agent内部调用）
// 用于工具确认、Plan审核、回滚确认等场景
func (r *AgentRuntime) requestInteraction(ctx context.Context, req events.InteractionRequest) events.InteractionResponse {
	// 生成请求ID
	if req.ID == "" {
		req.ID = fmt.Sprintf("interaction_%d", time.Now().UnixNano())
	}

	// 设置默认超时
	if req.Timeout == 0 {
		req.Timeout = 60 * time.Second
	}

	// 创建响应channel
	respCh := make(chan events.InteractionResponse, 1)
	r.interactionMu.Lock()
	r.interactionPending[req.ID] = respCh
	r.interactionMu.Unlock()

	// 发送交互请求事件到OUT
	extra := map[string]any{
		"tool_name":   req.ToolName,
		"tool_params": req.ToolParams,
		"plan_goal":   req.PlanGoal,
		"plan_steps":  req.PlanSteps,
		"plan_files":  req.PlanFiles,
		"question":    req.Question,
		"question_type": req.QuestionType,
		"options":     req.Options,
		"default_answer": req.DefaultAnswer,
		"rollback_reason": req.RollbackReason,
		"rollback_target": req.RollbackTarget,
	}

	event := events.NewInteractionEvent(
		events.EventTypeInteractionRequest,
		req.Type,
		req.ID,
		extra,
	)

	r.sendEvent(event)

	// 等待响应（带超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	select {
	case resp := <-respCh:
		r.interactionMu.Lock()
		delete(r.interactionPending, req.ID)
		r.interactionMu.Unlock()
		return resp
	case <-timeoutCtx.Done():
		r.interactionMu.Lock()
		delete(r.interactionPending, req.ID)
		r.interactionMu.Unlock()
		// 超时 → 中断（而非自动批准）
		// 用户后续可继续对话，LLM会重触发未完成内容
		r.logger.Warn().Str("request_id", req.ID).Msg("Interaction request timeout, interrupting")
		return events.InteractionResponse{
			RequestID: req.ID,
			Type:      req.Type,
			Approved:  false, // 不批准
			Cancelled: true,  // 标记为中断
			Error:     timeoutCtx.Err(),
		}
	}
}
