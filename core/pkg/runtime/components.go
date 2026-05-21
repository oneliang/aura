// Package runtime provides component definitions for AgentRuntime.
package runtime

import (
	"context"
	"net/http"
	"sync"

	agentpkg "github.com/oneliang/aura/agent/pkg/agent"
	agentloader "github.com/oneliang/aura/agent/pkg/loader"
	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/intent"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/core/pkg/prompt"
	"github.com/oneliang/aura/core/pkg/skilltool"
	"github.com/oneliang/aura/habit/pkg/manager"
	mcpmanager "github.com/oneliang/aura/mcp/pkg/manager"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/tasks"
	"github.com/oneliang/aura/skill/pkg/loader"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	tools "github.com/oneliang/aura/tools/pkg"
)

// SharedResources holds read-only resources shared with sub-agents.
// Thread-safe after initialization; immutable during runtime operation.
type SharedResources struct {
	llmClient       llm.Client
	httpClient      *http.Client
	webHttpClient   *http.Client
	permMgr         *permissions.Manager
	commandProvider commands.Command
}

// GetPermMgr returns the permission manager.
func (s *SharedResources) GetPermMgr() *permissions.Manager {
	if s == nil {
		return nil
	}
	return s.permMgr
}

// ClonePermMgrWithDowngrade creates a downgraded permission manager for sub-agent.
// Returns nil if SharedResources is nil, permMgr is nil, or clone fails.
// On clone failure, returns nil instead of parent to prevent security bypass.
func (s *SharedResources) ClonePermMgrWithDowngrade(level permissions.PermissionControlLevel) *permissions.Manager {
	if s == nil || s.permMgr == nil {
		return nil
	}
	cloned, err := s.permMgr.CloneWithDowngrade(level)
	if err != nil {
		// Return nil on error to prevent sub-agent from inheriting parent's full permissions
		// Caller should handle nil case appropriately (e.g., use restrictive default)
		return nil
	}
	return cloned
}

// SkillSystem manages skill loading and progressive disclosure.
// Nil when Skills.Enabled=false in config.
type SkillSystem struct {
	loader      *loader.Loader
	injector    *skilltool.SkillInjector
	intentSvc   *intent.Service
	directories []string
}

// IsEnabled returns whether the skill system is active.
func (s *SkillSystem) IsEnabled() bool {
	return s != nil && s.loader != nil
}

// GetDirectories returns configured skill directories.
func (s *SkillSystem) GetDirectories() []string {
	if s == nil {
		return nil
	}
	return s.directories
}

// GetLoader returns the skill loader for system prompt building.
func (s *SkillSystem) GetLoader() *loader.Loader {
	if s == nil {
		return nil
	}
	return s.loader
}

// GetInjector returns the skill injector for skill_activate tool.
func (s *SkillSystem) GetInjector() *skilltool.SkillInjector {
	if s == nil {
		return nil
	}
	return s.injector
}

// AgentSystem manages agent loading and delegation.
// Nil when Agents.Enabled=false in config.
type AgentSystem struct {
	loader     *agentloader.Loader
	delegateFn func(ctx context.Context, agentName string, task string) (string, error)
	mu         sync.RWMutex // protects delegateFn
}

// FindAgent locates an agent by name from the loader.
func (a *AgentSystem) FindAgent(agentName string) (*agentpkg.Agent, error) {
	if a == nil || a.loader == nil {
		return nil, nil
	}
	for i := range a.loader.GetAgents() {
		if a.loader.GetAgents()[i].Name == agentName {
			return &a.loader.GetAgents()[i], nil
		}
	}
	return nil, nil
}

// GetLoader returns the agent loader for system prompt building.
func (a *AgentSystem) GetLoader() *agentloader.Loader {
	if a == nil {
		return nil
	}
	return a.loader
}

// SetDelegateFn sets the agent delegation function.
func (a *AgentSystem) SetDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error)) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.delegateFn = fn
	a.mu.Unlock()
}

// GetDelegateFn returns the agent delegation function.
func (a *AgentSystem) GetDelegateFn() func(ctx context.Context, agentName string, task string) (string, error) {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	fn := a.delegateFn
	a.mu.RUnlock()
	return fn
}

// MCPSystem manages MCP server lifecycle.
// Nil if not injected via WithMCPManager option.
type MCPSystem struct {
	manager *mcpmanager.Manager
	started bool // Prevents sub-agent from stopping parent's MCP servers
}

// StartAll starts all MCP servers and returns discovered tools.
// Sets started=true after successful start.
func (m *MCPSystem) StartAll(ctx context.Context) ([]tools.Tool, error) {
	if m == nil || m.manager == nil {
		return nil, nil
	}
	tools, err := m.manager.StartAll(ctx)
	if err == nil && len(tools) > 0 {
		m.started = true
	}
	return tools, err
}

// StopAll stops all MCP servers (only if this instance started them).
// Sub-agents (started=false) cannot stop parent's servers.
func (m *MCPSystem) StopAll(ctx context.Context) error {
	if m == nil || m.manager == nil || !m.started {
		return nil
	}
	return m.manager.StopAll(ctx)
}

// GetManager returns the MCP manager for tool registration.
func (m *MCPSystem) GetManager() *mcpmanager.Manager {
	if m == nil {
		return nil
	}
	return m.manager
}

// HookSystem manages event-driven subprocess integration.
// Nil if hooks.yaml does not exist.
type HookSystem struct {
	engine *hooks.Engine
}

// Fire fires a hook event (non-blocking).
func (h *HookSystem) Fire(ctx context.Context, event hooks.HookEventType, data map[string]any) {
	if h == nil || h.engine == nil {
		return
	}
	h.engine.Fire(ctx, event, data)
}

// FireWithToolName fires a hook event with tool name context.
func (h *HookSystem) FireWithToolName(ctx context.Context, event hooks.HookEventType, toolName string, data map[string]any) {
	if h == nil || h.engine == nil {
		return
	}
	h.engine.FireWithToolName(ctx, event, toolName, data)
}

// FireBlocking fires a hook and waits for result.
func (h *HookSystem) FireBlocking(ctx context.Context, event hooks.HookEventType, data map[string]any) (*hooks.HookResult, error) {
	if h == nil || h.engine == nil {
		return nil, nil
	}
	return h.engine.FireBlocking(ctx, event, data)
}

// FireBlockingWithToolName fires a hook with tool name and waits for result.
func (h *HookSystem) FireBlockingWithToolName(ctx context.Context, event hooks.HookEventType, toolName string, data map[string]any) (*hooks.HookResult, error) {
	if h == nil || h.engine == nil {
		return nil, nil
	}
	return h.engine.FireBlockingWithToolName(ctx, event, toolName, data)
}

// Shutdown closes the hook engine.
func (h *HookSystem) Shutdown() {
	if h != nil && h.engine != nil {
		h.engine.Shutdown()
	}
}

// GetEngine returns the hook engine for memory system.
func (h *HookSystem) GetEngine() *hooks.Engine {
	if h == nil {
		return nil
	}
	return h.engine
}

// SessionContext holds session-specific state (isolated per sub-agent).
type SessionContext struct {
	sessionID     string
	userID        string
	mode          RuntimeMode
	sessionStore  *jsonl.MessageStore
	dataDir       string
	initialized   bool
	logger        *logger.Logger
	promptBuilder *prompt.PromptBuilder
	habitManager  *manager.Manager
	taskList      *tasks.TaskList
}

// GetUserID returns the bound user ID for multi-user isolation.
func (s *SessionContext) GetUserID() string {
	if s == nil {
		return ""
	}
	return s.userID
}

// GetSessionID returns the session identifier.
func (s *SessionContext) GetSessionID() string {
	if s == nil {
		return ""
	}
	return s.sessionID
}

// GetMode returns the runtime mode.
func (s *SessionContext) GetMode() RuntimeMode {
	if s == nil {
		return RuntimeModeCLI
	}
	return s.mode
}

// GetDataDir returns the session data directory.
func (s *SessionContext) GetDataDir() string {
	if s == nil {
		return ""
	}
	return s.dataDir
}

// GetLogger returns the session logger.
func (s *SessionContext) GetLogger() *logger.Logger {
	if s == nil {
		return nil
	}
	return s.logger
}

// IsInitialized returns whether the session is initialized.
func (s *SessionContext) IsInitialized() bool {
	if s == nil {
		return false
	}
	return s.initialized
}

// SetInitialized marks the session as initialized.
func (s *SessionContext) SetInitialized(val bool) {
	if s != nil {
		s.initialized = val
	}
}

// GetPromptBuilder returns the prompt builder.
func (s *SessionContext) GetPromptBuilder() *prompt.PromptBuilder {
	if s == nil {
		return nil
	}
	return s.promptBuilder
}

// GetHabitManager returns the habit manager.
func (s *SessionContext) GetHabitManager() *manager.Manager {
	if s == nil {
		return nil
	}
	return s.habitManager
}

// GetTaskList returns the task list.
func (s *SessionContext) GetTaskList() *tasks.TaskList {
	if s == nil {
		return nil
	}
	return s.taskList
}

// GetSessionStore returns the session store for persistence.
func (s *SessionContext) GetSessionStore() *jsonl.MessageStore {
	if s == nil {
		return nil
	}
	return s.sessionStore
}
