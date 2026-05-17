package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/httpclient"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/user"

	agentbuilder "github.com/oneliang/aura/agent/pkg/builder"
	agentloader "github.com/oneliang/aura/agent/pkg/loader"
	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/factory"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/core/pkg/prompt"
	"github.com/oneliang/aura/core/pkg/rollback"
	"github.com/oneliang/aura/core/pkg/skilltool"
	skillbuilder "github.com/oneliang/aura/skill/pkg/builder"
	"github.com/oneliang/aura/skill/pkg/loader"
	tools "github.com/oneliang/aura/tools/pkg"

	"github.com/oneliang/aura/habit/pkg/manager"

	"github.com/oneliang/aura/shared/pkg/hooks"
)

// ReAct format instructions for tool usage (cached as part of tools block).
const reactInstructions = `To use a tool, use the following format:

Action: {"tool": "tool_name", "parameters": {"param1": "value1"}}
Observation: [result of the tool]
... (this Thought/Action/Observation can repeat N times)

When you have a final answer, respond directly without using Action format.

Important rules:
1. Only use tools when necessary
2. Actions MUST use the exact JSON format shown above: Action: {valid JSON}
3. NEVER use XML-style tags, function wrappers, or any other format
4. After receiving an observation, continue thinking or provide final answer
5. Be concise in your responses
6. When multiple independent tools can be called at once, output multiple Action: lines consecutively
7. Tool observations may contain "Structured Data" sections with JSON — use this structured data for next steps`

// Initialize initializes the runtime (LLM, Agent, Tools, etc.).
func (r *AgentRuntime) Initialize(ctx context.Context) error {
	if r.initialized {
		return nil
	}

	// Fast path for sub-agents: skip LLM creation, skill loading, agent loading,
	// tool registration, and MCP startup.
	if r.skipInitialize {
		return r.initializeSubAgent(ctx)
	}

	// Phase 1: Create clients (HTTP, LLM, Permission)
	if err := r.initClients(ctx); err != nil {
		return err
	}

	// Phase 2: Load skills (Loader, Injector for skill_activate tool)
	r.initSkillSystem(ctx)

	// Phase 3: Load agents
	r.initAgentSystem(ctx)

	// Phase 4: Create memory and prompt
	if err := r.initMemoryAndPrompt(ctx); err != nil {
		return err
	}

	// Phase 5: Initialize prompt cache manager (partial - before Engine)
	// Cache manager must be created before initEngine() so config can be passed.
	r.initPromptCachePreEngine(ctx)

	// Phase 6: Create Engine
	if err := r.initEngine(ctx); err != nil {
		return err
	}

	// Phase 7: Register tools
	r.initTools(ctx)

	// Phase 8: Initialize prompt cache manager (complete - set tools block)
	// Tools block requires r.agent.GetTools() which is populated after initTools().
	r.initPromptCachePostTools(ctx)

	// Phase 9: Post-initialization (hooks, delegation, habit)
	if err := r.initPostSetup(ctx); err != nil {
		return err
	}

	r.initialized = true
	if r.session != nil {
		r.session.SetInitialized(true)
	}
	return nil
}

// initClients creates HTTP clients, LLM client, and permission manager.
func (r *AgentRuntime) initClients(ctx context.Context) error {
	// Create shared HTTP clients
	r.httpClient = httpclient.DefaultLLMClient()
	r.webHttpClient = httpclient.DefaultWebClient()

	// Create LLM factory with HTTP client injection
	llmFactory := factory.NewLLMFactory(
		&r.config.LLM,
		factory.WithHTTPClient(r.httpClient),
	)
	var err error
	r.llmClient, err = llmFactory.Create()
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Wrap with logging client if enabled
	if r.config.Debug.LogLLMInteractions {
		r.llmClient = llm.NewLoggingClient(r.llmClient, r.config.LLM.Provider, r.config.LLM.Model, r.sessionID)
	}

	// Create permission manager
	permFactory := factory.NewPermissionManagerFactory()
	r.permMgr, err = permFactory.Create(&r.config.Permissions)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to create permission manager")
	}

	// Create SharedResources component
	r.shared = &SharedResources{
		llmClient:       r.llmClient,
		httpClient:      r.httpClient,
		webHttpClient:   r.webHttpClient,
		permMgr:         r.permMgr,
		commandProvider: r.commandProvider,
	}

	return nil
}

// initSkillSystem loads skills and initializes matcher/injector for Progressive Disclosure.
func (r *AgentRuntime) initSkillSystem(ctx context.Context) {
	if !r.config.Skills.Enabled {
		return
	}

	r.skillLoader = loader.NewLoader(r.config.Skills.Directories)
	if _, err := r.skillLoader.Load(); err != nil {
		r.logger.Warn().Err(err).Msg("Failed to load skills")
	}

	// Initialize skill injector for skill_activate tool deduplication
	if len(r.skillLoader.GetSkills()) > 0 {
		r.skillInjector = skilltool.NewSkillInjector()
		r.logger.Debug().Str("module", "runtime").Int("count", len(r.skillLoader.GetSkills())).Msg("Skills loaded and injector initialized")
		for _, sk := range r.skillLoader.GetSkills() {
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Str("description", sk.Description).Msg("Loaded skill")
		}
	} else {
		r.logger.Debug().Str("module", "runtime").Msg("No skills loaded")
	}

	// Create SkillSystem component
	r.skills = &SkillSystem{
		loader:      r.skillLoader,
		injector:    r.skillInjector,
		intentSvc:   r.intentService,
		directories: r.config.Skills.Directories,
	}
}

// initAgentSystem loads agents from configured directories.
func (r *AgentRuntime) initAgentSystem(ctx context.Context) {
	if !r.config.Agents.Enabled {
		return
	}

	r.agentLoader = agentloader.NewLoader(r.config.Agents.Directories)
	if _, err := r.agentLoader.Load(); err != nil {
		r.logger.Warn().Err(err).Msg("Failed to load agents")
	}

	// Create AgentSystem component
	r.agents = &AgentSystem{
		loader: r.agentLoader,
	}
}

// initMemoryAndPrompt creates prompt builder, memory, hooks engine, and system prompt.
func (r *AgentRuntime) initMemoryAndPrompt(ctx context.Context) error {
	// Create prompt builder
	roleLoader := prompt.NewRoleLoader("")
	r.promptBuilder = prompt.NewPromptBuilder(roleLoader)

	// Create session memory
	var err error
	r.memory, err = r.createMemory()
	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}

	// Create hooks engine from independent config file
	hooksCfg, err := hooks.LoadHooksConfig("")
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to load hooks config, hooks disabled")
	}
	r.hookEngine = hooks.NewEngine(hooksCfg, r.logger)

	// Pass hook engine to session memory (for PreCompact/PostCompact hooks)
	if r.hookEngine != nil && r.memory != nil {
		r.memory.SetHookEngine(r.hookEngine)
	}

	// Create HookSystem component
	r.hooks = &HookSystem{
		engine: r.hookEngine,
	}

	// Create SessionContext component
	r.session = &SessionContext{
		sessionID:     r.sessionID,
		userID:        r.userID,
		mode:          r.mode,
		sessionStore:  r.sessionStore,
		dataDir:       r.dataDir,
		logger:        r.logger,
		promptBuilder: r.promptBuilder,
	}

	return nil
}

// initEngine creates the main Engine instance.
func (r *AgentRuntime) initEngine(ctx context.Context) error {
	// Build system prompt
	systemPrompt := r.buildSystemPrompt()

	// Create confirmation handler that checks for TUI channel at execution time
	confirmHandler := r.createConfirmationHandler()

	// Create agent factory options
	agentFactoryOpts := []factory.EngineFactoryOption{
		factory.WithSystemPrompt(systemPrompt),
		factory.WithConfirmationHandler(confirmHandler),
		factory.WithPlanReviewHandler(r.PlanReviewHandler()),
		factory.WithDataDir(r.dataDir),
		factory.WithSessionID(r.sessionID),
	}
	// Add command provider if configured
	if r.commandProvider != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithCommands(r.commandProvider))
	}
	// Pass logger to EngineFactory
	if r.logger != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithLogger(r.logger))
	}
	// Pass hooks engine to EngineFactory
	if r.hookEngine != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithHookEngine(r.hookEngine))
	}
	// Pass thinking config from LLM config
	agentFactoryOpts = append(agentFactoryOpts, factory.WithThinkingConfig(&llm.ThinkingConfig{
		Enabled:         r.config.LLM.Thinking.Enabled,
		ReasoningEffort: r.config.LLM.Thinking.ReasoningEffort,
		BudgetTokens:    r.config.LLM.Thinking.BudgetTokens,
	}))

	// Pass prompt cache config if caching is enabled
	if r.cacheManager != nil && r.cacheManager.Enabled() {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithPromptCacheConfig(
			&llm.PromptCacheConfig{
				Enabled:      true,
				SystemBlocks: r.cacheManager.BuildSystemBlocks(),
				CacheType:    r.cacheManager.BuildOpenAICacheType(),
			},
		))
	}

	// Pass skill injector if skills are enabled (for cache-aware skill body retrieval)
	if r.skillInjector != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithSkillInjector(r.skillInjector))
	}

	// Initialize rollback manager if plan mode is enabled
	if r.config.Agent.PlanningMode == "explicit" || r.config.Agent.PlanningMode == "auto" {
		workDir, err := os.Getwd()
		if err == nil {
			rollbackMgr := rollback.NewManager(workDir, r.logger)
			agentFactoryOpts = append(agentFactoryOpts, factory.WithRollbackManager(rollbackMgr))
			r.logger.Debug().Str("module", "runtime").Str("workDir", workDir).Msg("rollback manager initialized")
		}
	}

	agentFactory := factory.NewEngineFactory(
		r.llmClient,
		&r.config.Agent,
		r.permMgr,
		agentFactoryOpts...,
	)

	// Create agent
	var err error
	r.agent, err = agentFactory.Create(r.memory)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// initTools registers all tools including MCP tools.
func (r *AgentRuntime) initTools(ctx context.Context) {
	if r.config.DisableTools {
		return
	}

	toolReg := factory.NewToolRegistry(
		&r.config.Tools,
		r.permMgr,
		factory.WithWebHTTPClient(r.webHttpClient),
		factory.WithToolRegistryLogger(r.logger),
	)
	toolReg.RegisterAll(ctx, r.agent, r.config.Config)

	// Register command provider as a tool if available
	if r.commandProvider != nil {
		r.agent.AddTool(factory.NewCommandTool(r.commandProvider))
	}

	// Register skill_activate tool if skills are loaded
	if r.skillLoader != nil && len(r.skillLoader.GetSkills()) > 0 && r.skillInjector != nil {
		skillActivateTool := skilltool.NewSkillActivateTool(r.skillLoader, r.skillInjector)
		r.agent.AddTool(skillActivateTool)
		r.toolNamesMu.Lock()
		r.toolNames = append(r.toolNames, skillActivateTool.Name())
		r.toolNamesMu.Unlock()
		r.logger.Debug().Str("module", "runtime").Msg("skill_activate tool registered")
	}

	r.toolNames = factory.GetToolNames(r.agent)

	// Create MCPSystem component
	r.mcp = &MCPSystem{
		manager: r.mcpManager,
	}

	// Register MCP tools synchronously with timeout
	if r.mcpManager != nil {
		mcpCtx, mcpCancel := context.WithTimeout(ctx, 10*time.Second)
		mcpTools, err := r.mcp.StartAll(mcpCtx)
		mcpCancel()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to start MCP servers")
		} else {
			for _, t := range mcpTools {
				r.agent.AddTool(t)
				r.toolNamesMu.Lock()
				r.toolNames = append(r.toolNames, t.Name())
				r.toolNamesMu.Unlock()
			}
			if len(mcpTools) > 0 {
				r.logger.Info().Int("count", len(mcpTools)).Msg("MCP tools registered")
			}
		}
	}
}

// initPostSetup performs post-initialization: delegation, hooks, habit manager.
func (r *AgentRuntime) initPostSetup(ctx context.Context) error {
	// Initialize delegation audit logger (ensures log file exists before first delegation)
	if dl := logger.GetDelegationAuditLogger(); dl != nil {
		r.logger.Info().Msg("Delegation audit logger initialized")
	}

	// Set up agent delegation function if command provider supports it
	// Skip if sub-agent is disabled (single-agent mode)
	if r.commandProvider != nil && r.config.EnableSubAgent {
		r.agentDelegateFn = r.createAgentDelegateFn(ctx)
		if cp, ok := r.commandProvider.(*commands.CommandProvider); ok {
			cp.SetAgentDelegateFn(r.agentDelegateFn)
		} else {
			r.logger.Warn().Msg("Agent delegation: commandProvider type assertion failed, delegation will not work")
		}
		// Set delegateFn in AgentSystem component
		if r.agents != nil {
			r.agents.SetDelegateFn(r.agentDelegateFn)
		}
		// Set delegateFn in agent (engine) for parallel exploration
		if r.agent != nil {
			r.agent.SetAgentDelegateFn(r.agentDelegateFn)
		}
	}

	// Fire SessionStart hook via component (non-blocking)
	r.hooks.Fire(ctx, hooks.EventSessionStart, map[string]any{
		"session_id": r.sessionID,
		"user_id":    r.userID,
		"mode":       r.mode,
	})

	// Initialize habit manager (after initialized flag so Process() can access it)
	if !user.IsLegacyMode(r.userID) && r.config.Habit.Enabled {
		habitCfg := manager.Config{
			MinOccurrences: r.config.Habit.MinOccurrences,
			ConfThreshold:  r.config.Habit.ConfThreshold,
			MaxActionAge:   time.Duration(r.config.Habit.MaxActionAgeDays) * 24 * time.Hour,
			AnalysisLimit:  r.config.Habit.AnalysisLimit,
		}
		var err error
		r.habitManager, err = manager.New(habitCfg)
		if err != nil {
			r.logger.Warn().Err(err).Str("module", "runtime").Msg("Failed to create habit manager, continuing without habit tracking")
		}
		// Set habitManager in SessionContext component
		if r.session != nil {
			r.session.habitManager = r.habitManager
		}
	}

	// Start background cleanup goroutine for stale memory
	r.startCleanupGoroutine()

	return nil
}

// initializeSubAgent performs lightweight initialization for sub-agent runtimes.
// It creates only what the sub-agent needs: memory and Engine.
func (r *AgentRuntime) initializeSubAgent(ctx context.Context) error {
	// Create session memory (new, empty -- sub-agent has isolated context)
	var err error
	r.memory, err = r.createMemory()
	if err != nil {
		return fmt.Errorf("failed to create sub-agent memory: %w", err)
	}

	// Create SessionContext component for sub-agent
	r.session = &SessionContext{
		sessionID:    r.sessionID,
		userID:       r.userID,
		mode:         r.mode,
		sessionStore: r.sessionStore,
		dataDir:      r.dataDir,
		logger:       r.logger,
	}

	// Build system prompt (already set in config, but inject skills if available)
	systemPrompt := r.config.SystemPrompt
	if r.skillLoader != nil && len(r.skillLoader.GetSkills()) > 0 {
		systemPrompt += skillbuilder.BuildSystemPromptSection(r.skillLoader.GetSkills())
	}
	if r.agentLoader != nil && len(r.agentLoader.GetAgents()) > 0 {
		systemPrompt += agentbuilder.BuildSystemPromptSection(r.agentLoader.GetAgents())
	}

	// Create confirmation handler
	confirmHandler := r.createConfirmationHandler()

	// Create Engine using shared LLM client and permission manager
	r.logger.Info().Str("llmClient_type", fmt.Sprintf("%T", r.llmClient)).Msg("initializeSubAgent: creating engine with shared LLM client")
	agentFactoryOpts := []factory.EngineFactoryOption{
		factory.WithSystemPrompt(systemPrompt),
		factory.WithConfirmationHandler(confirmHandler),
		factory.WithDataDir(r.dataDir),
		factory.WithSessionID(r.sessionID),
	}
	if r.logger != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithLogger(r.logger))
	}
	// Pass command provider if parent has one (for nested delegation)
	if r.commandProvider != nil {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithCommands(r.commandProvider))
	}

	// Pass thinking config from LLM config (sub-agents inherit parent's thinking)
	agentFactoryOpts = append(agentFactoryOpts, factory.WithThinkingConfig(&llm.ThinkingConfig{
		Enabled:         r.config.LLM.Thinking.Enabled,
		ReasoningEffort: r.config.LLM.Thinking.ReasoningEffort,
		BudgetTokens:    r.config.LLM.Thinking.BudgetTokens,
	}))

	// Pass tool allowlist from parent (sub-agents inherit parent's phase constraints)
	if len(r.parentToolAllowlist) > 0 {
		agentFactoryOpts = append(agentFactoryOpts, factory.WithToolAllowlist(r.parentToolAllowlist))
	}

	agentFactory := factory.NewEngineFactory(
		r.llmClient,
		&r.config.Agent,
		r.permMgr,
		agentFactoryOpts...,
	)

	r.agent, err = agentFactory.Create(r.memory)
	if err != nil {
		return fmt.Errorf("failed to create sub-agent engine: %w", err)
	}

	// Register pre-built tools (shared references from parent)
	for _, tool := range r.preBuiltTools {
		r.agent.AddTool(tool)
	}
	r.toolNames = factory.GetToolNames(r.agent)

	// Register command provider as a tool if available (enables nested delegation)
	if r.commandProvider != nil {
		r.agent.AddTool(factory.NewCommandTool(r.commandProvider))
	}

	r.initialized = true
	if r.session != nil {
		r.session.SetInitialized(true)
	}
	return nil
}

// filterToolsByNames filters out disabled tools from the parent's tools.
func filterToolsByNames(parentTools []tools.Tool, disabledTools []string) []tools.Tool {
	if len(disabledTools) == 0 {
		return parentTools
	}
	disabledSet := make(map[string]bool, len(disabledTools))
	for _, name := range disabledTools {
		disabledSet[name] = true
	}
	var filtered []tools.Tool
	for _, tool := range parentTools {
		if !disabledSet[tool.Name()] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// createMemory creates session memory based on configuration.
func (r *AgentRuntime) createMemory() (*memory.SessionMemory, error) {
	if r.sessionStore != nil && r.sessionID != "" {
		return memory.NewSessionMemoryWithConfig(r.sessionID, r.userID, r.sessionStore, memory.SessionMemoryConfig{
			MaxLen: r.config.Memory.MaxContext,
			Source: memory.MessageSource(r.config.MessageSource),
		})
	}
	// Create in-memory session memory for non-persistent mode
	return memory.NewSessionMemoryWithConfig("default", r.userID, nil, memory.SessionMemoryConfig{
		MaxLen: r.config.Memory.MaxContext,
		Source: memory.MessageSource(r.config.MessageSource),
	})
}

// buildSystemPrompt builds the system prompt based on configuration.
func (r *AgentRuntime) buildSystemPrompt() string {
	var prompt string

	if r.config.SystemPrompt != "" {
		prompt = r.config.SystemPrompt
	} else if r.config.Role != "" {
		prompt = r.promptBuilder.BuildWithRole(r.config.Role)
	} else {
		prompt = r.promptBuilder.BuildWithConfig(r.config.Config)
	}

	// Append skills section if skills are loaded
	if r.skillLoader != nil && len(r.skillLoader.GetSkills()) > 0 {
		prompt += skillbuilder.BuildSystemPromptSection(r.skillLoader.GetSkills())
	}

	// Append agents section if agents are loaded and sub-agent is enabled
	if r.agentLoader != nil && len(r.agentLoader.GetAgents()) > 0 && r.config.EnableSubAgent {
		prompt += agentbuilder.BuildSystemPromptSection(r.agentLoader.GetAgents())
	}

	// Append single-agent mode notice if sub-agent is disabled
	if !r.config.EnableSubAgent {
		prompt += "\n\n## Single-Agent Mode\n\nYou are operating in single-agent mode. Do not attempt to delegate tasks to other agents or invoke sub-agents. Complete all tasks yourself using the available tools.\n"
	}

	// Append project-level AURA.md if exists
	prompt += r.loadProjectAuraMd()

	return prompt
}

// loadProjectAuraMd reads project-level AURA.md from current working directory.
func (r *AgentRuntime) loadProjectAuraMd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	auraMdPath := filepath.Join(cwd, "AURA.md")
	content, err := os.ReadFile(auraMdPath)
	if err != nil {
		return ""
	}

	log := logger.RegistryDefault().WithModule("runtime")
	log.Info().Str("path", auraMdPath).Msg("Loaded project-level AURA.md")

	return "\n\n## Project Instructions (AURA.md)\n\n" + string(content)
}

// initPromptCachePreEngine initializes the prompt cache manager before Engine creation.
// Creates the manager and sets static system prompt, skills, and agents blocks.
func (r *AgentRuntime) initPromptCachePreEngine(ctx context.Context) {
	if !r.config.LLM.EnablePromptCache {
		return
	}

	// Create prompt cache manager
	r.cacheManager = prompt.NewPromptCacheManager(true)

	// Build static system prompt
	staticPrompt := r.buildSystemPrompt()
	r.cacheManager.SetStaticSystem(staticPrompt)

	// Build skills metadata block
	if r.skillLoader != nil && len(r.skillLoader.GetSkills()) > 0 {
		skillsBlock := skillbuilder.BuildSystemPromptSection(r.skillLoader.GetSkills())
		r.cacheManager.SetSkillsBlock(skillsBlock)
	}

	// Build agents metadata block
	if r.agentLoader != nil && len(r.agentLoader.GetAgents()) > 0 {
		agentsBlock := agentbuilder.BuildSystemPromptSection(r.agentLoader.GetAgents())
		r.cacheManager.SetAgentsBlock(agentsBlock)
	}
}

// initPromptCachePostTools completes prompt cache initialization after tools are registered.
// Sets the tools block which requires r.agent.GetTools().
func (r *AgentRuntime) initPromptCachePostTools(ctx context.Context) {
	if !r.config.LLM.EnablePromptCache || r.cacheManager == nil {
		return
	}

	// Build tools block after tools are registered
	toolsBlock := r.buildToolsBlock()
	r.cacheManager.SetToolsBlock(toolsBlock)
}

// buildToolsBlock builds the tool descriptions block for caching.
func (r *AgentRuntime) buildToolsBlock() string {
	var sb strings.Builder
	sb.WriteString("You have access to the following tools:\n\n")

	// Get tools and sort by name for consistent cache key
	toolList := r.agent.GetTools()
	names := make([]string, 0, len(toolList))
	toolMap := make(map[string]tools.Tool)
	for _, tool := range toolList {
		name := tool.Name()
		names = append(names, name)
		toolMap[name] = tool
	}
	sort.Strings(names)

	for _, name := range names {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, toolMap[name].Description()))
	}

	// Append ReAct format instructions
	sb.WriteString("\n" + r.getReActInstructions())
	return sb.String()
}

// getReActInstructions returns the ReAct format instructions.
func (r *AgentRuntime) getReActInstructions() string {
	return reactInstructions
}
