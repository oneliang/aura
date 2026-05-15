// Package factory provides factories for creating core components.
package factory

import (
	"context"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/core/pkg/rollback"
	"github.com/oneliang/aura/core/pkg/skilltool"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/storage/pkg/taskstore"
)

// EngineFactory creates engine instances with all dependencies configured.
type EngineFactory struct {
	llmClient         llm.Client
	config            *config.AgentConfig
	permMgr           *permissions.Manager
	systemPrompt      string
	commands          commands.Command               // Command provider for internal commands
	confirmHandler    engine.ToolConfirmationHandler // Optional confirmation handler
	planReviewFn      engine.PlanReviewHandler       // Optional plan review handler
	logger            *logger.Logger                 // Optional logger for injection
	dataDir           string                         // Session data directory for task persistence
	sessionID         string                         // Session ID for task persistence
	hookEngine        *hooks.Engine                  // Optional hooks engine
	thinkingConfig    *llm.ThinkingConfig            // LLM thinking configuration
	promptCacheConfig *llm.PromptCacheConfig         // Prompt caching configuration
	skillInjector     *skilltool.SkillInjector       // Skill injector for cache-aware skill body retrieval
	toolAllowlist     []string                       // Tool allowlist for phase-based execution
	rollbackMgr       *rollback.Manager              // Rollback manager for plan mode
}

// EngineFactoryOption is a function that configures the EngineFactory.
type EngineFactoryOption func(*EngineFactory)

// WithSystemPrompt sets the system prompt for the engine.
func WithSystemPrompt(prompt string) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.systemPrompt = prompt
	}
}

// WithCommands sets the command provider for the engine.
func WithCommands(cmdProvider commands.Command) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.commands = cmdProvider
	}
}

// WithConfirmationHandler sets the confirmation handler for the engine.
// If provided, this handler will be used instead of the default one created from permission manager.
func WithConfirmationHandler(handler engine.ToolConfirmationHandler) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.confirmHandler = handler
	}
}

// WithPlanReviewHandler sets the plan review handler for the engine.
func WithPlanReviewHandler(handler engine.PlanReviewHandler) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.planReviewFn = handler
	}
}

// WithLogger sets the logger for the engine factory to pass to the engine.
func WithLogger(log *logger.Logger) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.logger = log
	}
}

// WithToolAllowlist sets the tool allowlist for phase-based execution control.
func WithToolAllowlist(names []string) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.toolAllowlist = names
	}
}

// WithRollbackManager sets the rollback manager for plan mode execution.
func WithRollbackManager(mgr *rollback.Manager) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.rollbackMgr = mgr
	}
}

// NewEngineFactory creates a new engine factory.
func NewEngineFactory(
	llmClient llm.Client,
	cfg *config.AgentConfig,
	permMgr *permissions.Manager,
	opts ...EngineFactoryOption,
) *EngineFactory {
	f := &EngineFactory{
		llmClient: llmClient,
		config:    cfg,
		permMgr:   permMgr,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Create creates a new engine with the given memory.
func (f *EngineFactory) Create(mem memory.Memory) (*engine.Engine, error) {
	return f.CreateWithSession(f.sessionID, mem)
}

// CreateWithSession creates a new engine for a specific session.
func (f *EngineFactory) CreateWithSession(sessionID string, mem memory.Memory) (*engine.Engine, error) {
	// Use config values for agent settings, with defaults from constants
	planningMode := engine.ModeImplicit

	if f.config != nil {
		switch f.config.PlanningMode {
		case "explicit":
			planningMode = engine.ModeExplicit
		case "auto":
			planningMode = engine.ModeAuto
		default:
			planningMode = engine.ModeImplicit
		}
	}

	systemPrompt := f.systemPrompt
	if systemPrompt == "" {
		systemPrompt = getDefaultSystemPrompt()
	}

	// Create agent options
	opts := []engine.Option{
		engine.WithClient(f.llmClient),
		engine.WithMemory(mem),
		engine.WithSystemPrompt(systemPrompt),
		engine.WithPlanningMode(planningMode),
		engine.WithPlannerClient(f.llmClient),
	}

	// Create task store if data dir is available
	if f.dataDir != "" && sessionID != "" {
		taskStore := taskstore.New(f.dataDir, sessionID)
		opts = append(opts, engine.WithTaskStore(taskStore))
	}

	// Use external confirmation handler if provided, otherwise create default one
	if f.confirmHandler != nil {
		// Use the externally provided confirmation handler
		opts = append(opts, engine.WithConfirmationHandler(f.confirmHandler))
	} else if f.permMgr != nil {
		// Create default confirmation handler from permission manager
		confirmHandler := f.createConfirmationHandler()
		opts = append(opts, engine.WithConfirmationHandler(confirmHandler))
	}

	// Pass logger to engine if provided
	if f.logger != nil {
		opts = append(opts, engine.WithLogger(f.logger))
	}

	// Pass hooks engine if provided
	if f.hookEngine != nil {
		opts = append(opts, engine.WithHookEngine(f.hookEngine))
	}

	// Pass thinking config if configured
	if f.thinkingConfig != nil {
		opts = append(opts, engine.WithThinking(f.thinkingConfig))
	}

	// Pass prompt cache config if configured
	if f.promptCacheConfig != nil {
		opts = append(opts, engine.WithPromptCacheConfig(f.promptCacheConfig))
	}

	// Pass skill injector if configured (for cache-aware skill body retrieval)
	if f.skillInjector != nil {
		opts = append(opts, engine.WithSkillInjector(f.skillInjector))
	}

	// Pass plan review handler if configured and review is enabled
	if f.planReviewFn != nil && (f.config == nil || f.config.Plan.EnableReview) {
		opts = append(opts, engine.WithPlanReviewHandler(f.planReviewFn))
	}

	// Pass tool allowlist if configured
	if len(f.toolAllowlist) > 0 {
		opts = append(opts, engine.WithToolAllowlist(f.toolAllowlist))
	}

	// Pass plan config from agent config
	if f.config != nil {
		planConfig := engine.PlanConfig{
			VerifyCommands:     f.config.Plan.VerifyCommands,
			UseReviewerAgent:   f.config.Plan.UseReviewerAgent,
			ParallelExplore:    f.config.Plan.ParallelExplore,
			MaxParallelExplore: f.config.Plan.MaxParallelExplore,
		}
		opts = append(opts, engine.WithPlanConfig(planConfig))
	}

	eng, err := engine.New(opts...)
	if err != nil {
		return nil, err
	}

	// Set rollback manager if provided
	if f.rollbackMgr != nil {
		eng.SetRollbackManager(f.rollbackMgr)
	}

	return eng, nil
}

// createConfirmationHandler creates a confirmation handler based on permissions.
func (f *EngineFactory) createConfirmationHandler() engine.ToolConfirmationHandler {
	if f.permMgr == nil {
		return nil
	}
	return func(ctx context.Context, toolName string, params map[string]any) (bool, error) {
		allowed, requiresConfirm, reason := f.permMgr.CheckPermission(ctx, toolName, params)
		if !allowed {
			return false, nil
		}
		if requiresConfirm {
			// Return false to indicate user needs to be asked
			// The actual confirmation will be handled by the caller
			return false, nil
		}
		_ = reason // unused for now
		return true, nil
	}
}

// getDefaultSystemPrompt returns the default system prompt.
func getDefaultSystemPrompt() string {
	// Try to get localized system prompt from i18n
	prompt := i18n.T("engine.system_prompt")
	// If i18n returns the key itself (not initialized or key not found), use fallback
	if prompt == "engine.system_prompt" || prompt == "" {
		return `You are Aura, a personal AI assistant designed to help the user.
You are knowledgeable, helpful, and adapt to the user's communication style.

When responding:
- Be concise but thorough
- Match the user's communication style
- Be helpful and friendly
- Provide accurate and useful information`
	}
	return prompt
}

// WithPermissionManager sets the permission manager for the factory.
func WithPermissionManager(permMgr *permissions.Manager) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.permMgr = permMgr
	}
}

// WithDataDir sets the session data directory for task persistence.
func WithDataDir(dataDir string) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.dataDir = dataDir
	}
}

// WithSessionID sets the session ID for task persistence.
func WithSessionID(sessionID string) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.sessionID = sessionID
	}
}

// WithHookEngine sets the hooks engine for the engine.
func WithHookEngine(hookEngine *hooks.Engine) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.hookEngine = hookEngine
	}
}

// WithThinkingConfig sets the LLM thinking configuration.
func WithThinkingConfig(thinking *llm.ThinkingConfig) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.thinkingConfig = thinking
	}
}

// WithPromptCacheConfig sets the prompt cache configuration for LLM requests.
func WithPromptCacheConfig(cacheConfig *llm.PromptCacheConfig) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.promptCacheConfig = cacheConfig
	}
}

// WithSkillInjector sets the skill injector for cache-aware skill body retrieval.
func WithSkillInjector(injector *skilltool.SkillInjector) EngineFactoryOption {
	return func(f *EngineFactory) {
		f.skillInjector = injector
	}
}
