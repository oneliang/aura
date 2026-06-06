// Package engine provides the core engine implementation for AI agent execution.
package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/tasks"
	"github.com/oneliang/aura/storage/pkg/taskstore"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/llm"
	corememory "github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/core/pkg/planner"
	"github.com/oneliang/aura/core/pkg/rollback"
	"github.com/oneliang/aura/core/pkg/skilltool"
	"github.com/oneliang/aura/knowledge/pkg/retrieval"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/tasktool"
)

// Precompiled regex for removing final answer prefix.
var finalAnswerPattern = regexp.MustCompile(`(?i)^(Final Answer:|Final Response:)\s*`)

// Precompiled regex for stripping code fence tags (e.g., ```tool_call, ```).
// Matches lines that are only a code fence delimiter, optionally with language identifier.
var codeFencePattern = regexp.MustCompile(`(?m)^\s*\x60{3}\w*\s*$`)

// thinkingFilter strips LLM thinking content from stream chunks.
// Handles tags split across chunks for the <think>...</think> pattern.
type thinkingFilter struct {
	inThinking bool
	pending    strings.Builder // partial content when tags are split
}

// stripThinking removes thinking tag content from a chunk.
// Returns (cleanedContent, capturedThinking) — captured thinking is returned
// immediately when a closing tag is found.
func (f *thinkingFilter) stripThinking(chunk string) (content, thinking string) {
	if chunk == "" {
		return "", ""
	}

	const (
		openTag1  = "<think>"
		closeTag1 = "</think>"
	)

	f.pending.WriteString(chunk)
	data := f.pending.String()
	f.pending.Reset()

	var out strings.Builder

	for len(data) > 0 {
		if f.inThinking {
			// Look for closing tag
			idx := strings.Index(data, closeTag1)
			if idx >= 0 {
				// Captured the thinking content — return it immediately
				thinking = data[:idx]
				f.inThinking = false
				data = data[idx+len(closeTag1):]
			} else {
				// Closing tag not yet possible — all data is thinking content
				f.pending.WriteString(data)
				return "", ""
			}
		} else {
			// Look for opening tag
			idx := strings.Index(data, openTag1)
			if idx >= 0 {
				// Found opening tag — output content before it
				out.WriteString(data[:idx])
				f.inThinking = true
				data = data[idx+len(openTag1):]
			} else {
				// No opening tag found — all content is regular text
				out.WriteString(data)
				break
			}
		}
	}

	return out.String(), thinking
}

// extractThinkingContent returns any remaining buffered thinking content.
func (f *thinkingFilter) extractThinkingContent() string {
	s := f.pending.String()
	f.pending.Reset()
	return s
}

// IsInThinking returns true if currently inside a thinking tag.
func (f *thinkingFilter) IsInThinking() bool {
	return f.inThinking
}

// SensitiveTool marks tools that require user confirmation before execution.
type SensitiveTool interface {
	tools.Tool
	RequiresConfirmation() bool
}

// PermissionTool marks tools that declare a permission level.
type PermissionTool interface {
	tools.Tool
	PermissionLevel() string
}

// CommandRestrictionTool marks tools that have command restrictions.
type CommandRestrictionTool interface {
	tools.Tool
	CheckCommand(command string) (bool, string)
}

// InputSchemaProvider is an optional interface tools can implement to declare
// their JSON input schema, used for parameter validation and LLM tool description.
type InputSchemaProvider interface {
	InputSchema() map[string]any
}

// ToolConfirmationHandler is called to get user permission for sensitive tool operations.
type ToolConfirmationHandler func(ctx context.Context, toolName string, params map[string]any) (bool, error)

// PlanReviewHandler is called to get user approval for a generated plan before execution.
type PlanReviewHandler func(ctx context.Context, goal string, steps []string) (bool, error)

// RollbackConfirmHandler is called to get user confirmation for rollback after execution/verification failure.
type RollbackConfirmHandler func(ctx context.Context, snapshotID string, files []string, reason string) (bool, error)

// AskUserQuestionHandler is called to prompt the user with a question during plan review.
type AskUserQuestionHandler func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error)

// PlanningMode represents the planning strategy mode.
type PlanningMode string

const (
	// ModeImplicit - LLM implicitly plans during ReAct loop (default, fast)
	ModeImplicit PlanningMode = "implicit"
	// ModeExplicit - Create explicit plan first, then execute step by step
	ModeExplicit PlanningMode = "explicit"
	// ModeAuto - Auto-detect based on task complexity
	ModeAuto PlanningMode = "auto"
)

// PlanModeState represents the current phase within plan mode.
type PlanModeState string

const (
	// PlanModeStateNone - not in plan mode
	PlanModeStateNone PlanModeState = "none"
	// PlanModeStateExplore - in exploration phase (read-only)
	PlanModeStateExplore PlanModeState = "explore"
	// PlanModeStateDesign - in design phase (creating plan)
	PlanModeStateDesign PlanModeState = "design"
	// PlanModeStateReview - in review phase (waiting for user approval)
	PlanModeStateReview PlanModeState = "review"
	// PlanModeStateExecute - in execution phase (executing plan steps)
	PlanModeStateExecute PlanModeState = "execute"
	// PlanModeStateVerify - in verification phase (running verify commands)
	PlanModeStateVerify PlanModeState = "verify"
)

// Config represents agent configuration.
type EngineConfig struct {
	SystemPrompt        string
	Tools               []tools.Tool
	Commands            commands.Command // Command provider for internal commands
	ConfirmationHandler ToolConfirmationHandler
	PlanningMode        PlanningMode // Planning mode: implicit, explicit, or auto
	PlannerClient       llm.Client   // LLM client for planner (can be same as main client)
	PlanConfig          PlanConfig   // Plan-specific settings (verify commands, etc.)

	// Context optimization
	EnableSummarization bool                   // Enable automatic conversation summarization
	Summarizer          *corememory.Summarizer // Summarizer for context compression
	EnableDynamicRAG    bool                   // Enable dynamic RAG with token-aware injection
	DynamicRAG          *retrieval.DynamicRAG

	// Loop control
	MaxSteps int // Max ReAct loop iterations (0 = unlimited)

	// MaxParallelTools controls how many tools can execute concurrently in a single ReAct step.
	// 0 means default (5), 1 means serial execution.
	MaxParallelTools int

	// Thinking controls native LLM thinking/reasoning mode
	Thinking *llm.ThinkingConfig

	// Self-verification
	EnableReflection bool    // Enable reflection step before final response
	ReflectionConfig ReflectionConfig // Reflection settings

	// Prompt caching
	EnablePromptCache bool                // Enable prompt caching for LLM requests
	PromptCacheConfig *llm.PromptCacheConfig // Pre-built cache configuration (from PromptCacheManager)

	// SkillInjector for retrieving activated skill bodies (cache-aware)
	SkillInjector *skilltool.SkillInjector
}

// ReflectionConfig represents reflection/self-verification settings.
type ReflectionConfig struct {
	PromptTemplate string // Custom prompt template for reflection (supports %s for response)
}

// PlanConfig represents plan-specific settings for the Engine.
type PlanConfig struct {
	VerifyCommands     []string // Commands to run in verify phase
	UseReviewerAgent   bool     // Delegate to code-reviewer agent in verify phase
	ParallelExplore    bool     // Enable parallel exploration
	MaxParallelExplore int      // Max concurrent exploration agents
}

// Agent represents an AI agent.
type Engine struct {
	client     llm.Client
	memory     memory.Memory
	regTools   map[string]tools.Tool // renamed from 'tools' to avoid conflict with tools package
	config     EngineConfig
	planner    *planner.Planner
	logger     *logger.Logger
	hookEngine *hooks.Engine

	// Current plan (for explicit planning mode)
	currentPlan *planner.Plan

	// Current step index being executed in the planning loop
	currentStepIndex int

	// Current request ID being processed
	currentRequestID string

	// Input queue management - sequential processing for single session
	inputMu      sync.Mutex
	inputQueue   chan InputRequest
	ctx          context.Context
	cancel       context.CancelFunc
	processingMu sync.Mutex // Mutex for processing state

	// Session-scoped task list and persistence
	taskList  *tasks.TaskList
	taskStore *taskstore.TaskStore

	// Tool allowlist for phase-based execution control (nil = all tools allowed)
	toolAllowlist []string

	// Current execution phase
	currentPhase Phase

	// Saved tool allowlist for plan mode restore
	savedToolAllowlist []string

	// Plan mode state tracking
	planModeState PlanModeState

	// Plan file path for write guard
	planFilePath string

	// Plan review handler (for explicit planning mode)
	planReviewFn PlanReviewHandler

	// Agent delegation function (for verify phase and parallel exploration)
	agentDelegateFn func(ctx context.Context, agentName string, task string) (string, error)

	// Rollback manager for plan mode execution
	rollbackMgr *rollback.Manager

	// Rollback snapshot ID for current plan execution
	rollbackSnapshotID string

	// Rollback confirm handler (for execution/verification failure recovery)
	rollbackConfirmFn RollbackConfirmHandler

	// Ask user question handler (for plan review clarifying questions)
	askUserQuestionFn AskUserQuestionHandler

	// Shutdown guard
	shutdownOnce sync.Once

	// Goroutine lifecycle management
	processWg sync.WaitGroup
}

// Option is an agent configuration option.
type Option func(*Engine)

// WithClient sets the LLM client.
func WithClient(client llm.Client) Option {
	return func(e *Engine) {
		e.client = client
	}
}

// WithMemory sets the memory.
func WithMemory(mem memory.Memory) Option {
	return func(e *Engine) {
		e.memory = mem
	}
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(e *Engine) {
		e.config.SystemPrompt = prompt
	}
}

// WithConfirmationHandler sets the confirmation handler for sensitive tools.
func WithConfirmationHandler(handler ToolConfirmationHandler) Option {
	return func(e *Engine) {
		e.config.ConfirmationHandler = handler
	}
}

// WithPlanningMode sets the planning mode.
func WithPlanningMode(mode PlanningMode) Option {
	return func(e *Engine) {
		e.config.PlanningMode = mode
	}
}

// WithPlannerClient sets the LLM client for the planner.
func WithPlannerClient(client llm.Client) Option {
	return func(e *Engine) {
		e.config.PlannerClient = client
	}
}

// WithToolAllowlist sets the allowed tool names for phase-based execution.
// Nil or empty list means all registered tools are available.
func WithToolAllowlist(names []string) Option {
	return func(e *Engine) {
		e.toolAllowlist = names
	}
}

// WithPlanConfig sets the plan-specific configuration.
func WithPlanConfig(planConfig PlanConfig) Option {
	return func(e *Engine) {
		e.config.PlanConfig = planConfig
	}
}

// WithAgentDelegateFn sets the agent delegation function for sub-agent calls.
func WithAgentDelegateFn(delegateFn func(ctx context.Context, agentName string, task string) (string, error)) Option {
	return func(e *Engine) {
		e.agentDelegateFn = delegateFn
	}
}

// SetAgentDelegateFn sets the agent delegation function after engine creation.
func (e *Engine) SetAgentDelegateFn(delegateFn func(ctx context.Context, agentName string, task string) (string, error)) {
	e.agentDelegateFn = delegateFn
}

// WithLogger sets the logger.
func WithLogger(log *logger.Logger) Option {
	return func(e *Engine) {
		e.logger = log
	}
}

// WithMaxSteps sets the max ReAct loop iterations.
func WithMaxSteps(maxSteps int) Option {
	return func(e *Engine) {
		e.config.MaxSteps = maxSteps
	}
}

// WithMaxParallelTools sets the max concurrent tool executions.
func WithMaxParallelTools(n int) Option {
	return func(e *Engine) {
		e.config.MaxParallelTools = n
	}
}

// WithThinking sets the thinking configuration for LLM requests.
func WithThinking(thinking *llm.ThinkingConfig) Option {
	return func(e *Engine) {
		e.config.Thinking = thinking
	}
}

// WithPromptCacheConfig sets the prompt cache configuration for LLM requests.
func WithPromptCacheConfig(cacheConfig *llm.PromptCacheConfig) Option {
	return func(e *Engine) {
		e.config.EnablePromptCache = cacheConfig != nil && cacheConfig.Enabled
		e.config.PromptCacheConfig = cacheConfig
	}
}

// WithSkillInjector sets the skill injector for cache-aware skill body retrieval.
func WithSkillInjector(injector *skilltool.SkillInjector) Option {
	return func(e *Engine) {
		e.config.SkillInjector = injector
	}
}

// WithRollbackManager sets the rollback manager for plan mode execution.
func WithRollbackManager(mgr *rollback.Manager) Option {
	return func(e *Engine) {
		e.rollbackMgr = mgr
	}
}

// SetRollbackManager sets the rollback manager after engine creation.
func (e *Engine) SetRollbackManager(mgr *rollback.Manager) {
	e.rollbackMgr = mgr
}

// WithTaskStore sets the task persistence store for the session.
func WithTaskStore(store *taskstore.TaskStore) Option {
	return func(e *Engine) {
		e.taskStore = store
	}
}

// WithHookEngine sets the hooks engine for the engine.
func WithHookEngine(hookEngine *hooks.Engine) Option {
	return func(e *Engine) {
		e.hookEngine = hookEngine
	}
}

// WithPlanReviewHandler sets the plan review handler for explicit planning mode.
func WithPlanReviewHandler(handler PlanReviewHandler) Option {
	return func(e *Engine) {
		e.planReviewFn = handler
	}
}

// WithRollbackConfirmHandler sets the rollback confirm handler for plan mode rollback recovery.
func WithRollbackConfirmHandler(handler RollbackConfirmHandler) Option {
	return func(e *Engine) {
		e.rollbackConfirmFn = handler
	}
}

// WithAskUserQuestionHandler sets the ask user question handler for plan review clarifying questions.
func WithAskUserQuestionHandler(handler AskUserQuestionHandler) Option {
	return func(e *Engine) {
		e.askUserQuestionFn = handler
	}
}

// New creates a new engine.
func New(opts ...Option) (*Engine, error) {
	e := &Engine{
		regTools:        make(map[string]tools.Tool),
		planModeState:   PlanModeStateNone, // Initialize to "none" to avoid blocking tools when not in plan mode
		config: EngineConfig{
			PlanningMode: ModeImplicit, // Default to implicit planning
		},
		inputQueue: make(chan InputRequest, 10), // Buffer for input queue
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.client == nil {
		return nil, fmt.Errorf("llm client is required")
	}
	if e.memory == nil {
		return nil, fmt.Errorf("memory is required")
	}
	// Initialize default logger if not set
	// CLI mode: fallback logger outputs to file to avoid cluttering user interface
	if e.logger == nil {
		e.logger = logger.NewNamed(logger.Config{Level: "info", Format: "text", Output: "file", Module: "engine"})
	}

	// Initialize planner if planner client is provided
	if e.config.PlannerClient != nil {
		e.planner = planner.New(e.config.PlannerClient)
	}

	// Initialize empty task list (tasks are NOT auto-loaded from previous sessions).
	// Users can load incomplete tasks on demand via LoadTasks() when needed.
	e.taskList = tasks.NewTaskList()

	// Start input processing loop
	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.processWg.Add(1)
	go e.processInputQueue()

	return e, nil
}

// AddTool adds a tool to the agent.
func (e *Engine) AddTool(tool tools.Tool) {
	e.regTools[tool.Name()] = tool
}

// saveTasks persists the current task list to disk.
func (e *Engine) saveTasks() {
	if e.taskStore != nil {
		if err := e.taskStore.Save(e.taskList.List()); err != nil {
			e.logger.Warn("saveTasks: failed to persist tasks", "error", err.Error())
		}
	}
}

// ClearTasks clears all tasks from memory and disk.
func (e *Engine) ClearTasks() {
	e.taskList.Reset()
	e.saveTasks()
}

// LoadTasks loads persisted tasks from disk into memory.
// Used for on-demand task restoration (not auto-loaded at startup).
func (e *Engine) LoadTasks() error {
	if e.taskStore == nil {
		return nil
	}
	saved, err := e.taskStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load persisted tasks: %w", err)
	}
	if len(saved) > 0 {
		e.taskList.Restore(saved)
	}
	return nil
}

// GetTaskList returns the shared task list for tool registration.
func (e *Engine) GetTaskList() *tasks.TaskList {
	return e.taskList
}

// GetSaveTasksFunc returns the callback for persisting tasks.
// Used by task tool to save after changes.
func (e *Engine) GetSaveTasksFunc() func() {
	return e.saveTasks
}

// Run submits an input to the processing queue.
// Returns a per-request event channel for receiving events.
// Inputs are processed sequentially in the order they are received.
// Each input is assigned a unique request ID for event grouping and tracing.
// The returned channel is closed when the request is complete (including done event).
func (e *Engine) Run(ctx context.Context, input string) (<-chan events.Event, error) {
	requestID := generateRequestID()

	// Create per-request event channel
	perRequestEventCh := make(chan events.Event, constants.AgentEventBufferSize)

	request := InputRequest{
		Input:     input,
		RequestID: requestID,
		EventChan: perRequestEventCh,
	}

	e.logger.Info("Run: submitting request", "requestID", requestID, "input", input)

	select {
	case e.inputQueue <- request:
		return perRequestEventCh, nil
	default:
		close(perRequestEventCh)
		return nil, fmt.Errorf("input queue full")
	}
}

// processInputQueue processes inputs sequentially.
// This ensures only one ReAct loop runs at a time per Engine instance.
// Events are sent directly to the per-request event channel.
func (e *Engine) processInputQueue() {
	defer e.processWg.Done()
	for request := range e.inputQueue {
		e.processingMu.Lock()

		// Store current request ID for tracking
		e.currentRequestID = request.RequestID

		// Create a context for this processing round
		procCtx, procCancel := context.WithCancel(e.ctx)

		// Use per-request event channel
		eventCh := request.EventChan

		// Note: User message is added to memory by event stream handler (handleUserInput)
		// when EventTypeUserInput is received via SendEvent. We don't add it again here
		// to avoid duplication in session storage.

		e.logger.Info("processInputQueue: starting ReAct loop", "requestID", request.RequestID)

		// Start ReAct loop in background
		go func() {
			defer func() {
				if r := recover(); r != nil {
					e.logger.Error(fmt.Sprintf("ReAct loop goroutine panicked: %v", r), "requestID", request.RequestID)
					select {
					case eventCh <- events.NewEvent(events.EventTypeError, fmt.Sprintf("Internal error: %v", r), request.RequestID):
					default:
					}
				}
			}()
			defer procCancel()

			// Update task tool with per-request event channel (if registered)
			if taskTool, ok := e.regTools["task"].(*tasktool.TaskTool); ok {
				taskTool.SetRequest(eventCh, request.RequestID)
			}

			// Send current task list to TUI at start of each request.
			// Only send incomplete tasks — completed tasks are hidden to avoid clutter.
			incompleteTasks := make([]tasks.Task, 0)
			for _, t := range e.taskList.List() {
				if t.Status != tasks.TaskStatusCompleted {
					incompleteTasks = append(incompleteTasks, t)
				}
			}
			if len(incompleteTasks) > 0 {
				eventCh <- events.NewEventWithExtra(
					events.EventTypeTaskList,
					"",
					map[string]any{"tasks": incompleteTasks},
					request.RequestID,
				)
			}

			// If no tools registered, use simple streaming
			if len(e.regTools) == 0 {
				e.runSimple(procCtx, eventCh, request.RequestID)
				return
			}

			// Check planning mode and route accordingly
			switch e.config.PlanningMode {
			case ModeExplicit:
				e.runExplicitPlanningLoop(procCtx, eventCh, request.Input, request.RequestID)
			case ModeAuto:
				if e.isComplexTask(request.Input) {
					e.runExplicitPlanningLoop(procCtx, eventCh, request.Input, request.RequestID)
				} else {
					e.runReActLoop(procCtx, eventCh, request.RequestID)
				}
			default:
				e.runReActLoop(procCtx, eventCh, request.RequestID)
			}
		}()

		// Wait for ReAct loop to complete
		<-procCtx.Done()

		// Auto-complete any remaining tasks before finishing the request
		for _, task := range e.taskList.List() {
			if task.Status != tasks.TaskStatusCompleted {
				e.taskList.Update(task.ID, tasks.TaskStatusCompleted, "auto-completed on request end")
			}
		}
		e.saveTasks()

		// Fire Stop hook (non-blocking) — request completed normally
		e.hookEngine.Fire(e.ctx, hooks.EventStop, map[string]any{
			"request_id": request.RequestID,
		})

		// Send done event to signal request completion
		doneEvent := events.NewEvent(events.EventTypeDone, "", request.RequestID)
		e.logger.Info("processInputQueue: sending done event", "requestID", request.RequestID)
		select {
		case eventCh <- doneEvent:
			e.logger.Debug("processInputQueue: done event sent", "requestID", request.RequestID)
		case <-time.After(5 * time.Second):
			e.logger.Warn("processInputQueue: done event timed out", "requestID", request.RequestID)
		}

		// Close event channel after done event
		close(eventCh)

		e.logger.Info("processInputQueue: request completed", "requestID", request.RequestID)
		e.processingMu.Unlock()
	}
}

// Shutdown shuts down the engine and stops processing.
// Safe to call multiple times — subsequent calls are no-ops.
func (e *Engine) Shutdown() {
	e.shutdownOnce.Do(func() {
		e.cancel()
		close(e.inputQueue)
		e.processWg.Wait() // Wait for processInputQueue goroutine to complete
	})
}

// SetSystemPrompt sets the system prompt.
func (e *Engine) SetSystemPrompt(prompt string) {
	e.config.SystemPrompt = prompt
}

// buildToolSchemas converts registered tools to LLM tool schemas.
// Respects toolAllowlist — only allowed tools are exposed to the LLM.
func (e *Engine) buildToolSchemas() []llm.ToolSchema {
	allowed := e.isToolAllowed
	schemas := make([]llm.ToolSchema, 0, len(e.regTools))
	for name, tool := range e.regTools {
		if !allowed(name) {
			continue
		}
		schemas = append(schemas, llm.ToolSchema{
			Name:        name,
			Description: tool.Description(),
			Parameters:  e.buildToolInputSchema(tool),
		})
	}
	return schemas
}

// buildToolInputSchema extracts JSON input schema from a tool's description.
// If the tool implements InputSchema, use it directly.
// Otherwise, derive a minimal schema from the description.
func (e *Engine) buildToolInputSchema(tool tools.Tool) map[string]any {
	if isp, ok := tool.(InputSchemaProvider); ok {
		if schema := isp.InputSchema(); schema != nil {
			return schema
		}
	}
	// Fallback: empty schema (LLM will infer from description)
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// GetTools returns the list of available tools.
func (e *Engine) GetTools() []tools.Tool {
	toolList := make([]tools.Tool, 0, len(e.regTools))
	for _, tool := range e.regTools {
		toolList = append(toolList, tool)
	}
	return toolList
}

// SetToolAllowlist sets the allowed tool names for the current execution phase.
// Nil or empty list means all registered tools are available.
func (e *Engine) SetToolAllowlist(names []string) {
	e.toolAllowlist = names
}

// GetToolAllowlist returns the current tool allowlist.
// Returns nil if all tools are allowed.
func (e *Engine) GetToolAllowlist() []string {
	return e.toolAllowlist
}

// GetPhase returns the current execution phase.
func (e *Engine) GetPhase() Phase {
	if e.currentPhase == 0 {
		return PhaseNormal // Default
	}
	return e.currentPhase
}

// setPhase sets the current execution phase (internal method).
func (e *Engine) setPhase(phase Phase) {
	e.currentPhase = phase
}

// GetPlanModeState returns the current plan mode state.
func (e *Engine) GetPlanModeState() PlanModeState {
	return e.planModeState
}

// GetPlanFilePath returns the current plan file path.
func (e *Engine) GetPlanFilePath() string {
	return e.planFilePath
}

// SetPlanFilePath sets the plan file path for write guard.
func (e *Engine) SetPlanFilePath(path string) {
	e.planFilePath = path
}

// enterPlanMode transitions into plan mode with permission downgrade.
// Saves current tool allowlist and restricts to read-only tools.
func (e *Engine) enterPlanMode() {
	// Set phase to exploration
	e.setPhase(PhaseExploration)

	// Save current allowlist for restore
	e.savedToolAllowlist = e.toolAllowlist

	// Restrict to read-only exploration tools
	e.SetToolAllowlist(explorationTools)

	// Update state
	e.planModeState = PlanModeStateExplore
}

// exitPlanMode transitions out of plan mode with permission restore.
// Restores saved tool allowlist and sets state to execute.
func (e *Engine) exitPlanMode() {
	// Set phase back to normal
	e.setPhase(PhaseNormal)

	// Restore saved allowlist
	if e.savedToolAllowlist != nil {
		e.SetToolAllowlist(e.savedToolAllowlist)
	} else {
		e.SetToolAllowlist(nil)
	}

	// Clear saved state
	e.savedToolAllowlist = nil

	// Update state to execute
	e.planModeState = PlanModeStateExecute
}

// transitionPlanModeState updates the plan mode state to a new phase.
func (e *Engine) transitionPlanModeState(newState PlanModeState) {
	e.planModeState = newState
}

// isToolAllowed checks if a tool is in the current allowlist.
// Nil or empty allowlist means all tools are allowed.
func (e *Engine) isToolAllowed(name string) bool {
	if len(e.toolAllowlist) == 0 {
		return true
	}
	for _, allowed := range e.toolAllowlist {
		if allowed == name {
			return true
		}
	}
	return false
}

// recordObservation records a structured tool observation in memory.
// Single-tool: wraps in "[Observation: {tool}]" format.
// If the result contains image DataURI, uses multi-part message.
func (e *Engine) recordObservation(result, toolName string) {
	// Check if result contains image DataURI
	if strings.Contains(result, constants.ObservationDataImageHint) {
		lines := strings.Split(result, "\n")
		var dataURI string
		for _, line := range lines {
			if strings.HasPrefix(line, "DataURI: ") {
				dataURI = strings.TrimPrefix(line, "DataURI: ")
				break
			}
		}
		if dataURI != "" {
			e.memory.AddWithParts(memory.RoleUser, []memory.MessagePart{
				{Type: "text", Text: fmt.Sprintf("[Observation: %s]: Image loaded successfully. Analyze this image and provide a description.", toolName)},
				{Type: "image_url", ImageURL: &memory.ImageURL{URL: dataURI}},
			}, memory.MessageTypeObservation)
			return
		}
	}
	e.memory.AddWithType(memory.RoleUser, fmt.Sprintf(constants.ObservationFormat, toolName, result), memory.MessageTypeObservation)
}

// recordObservationError records a structured tool error in memory.
func (e *Engine) recordObservationError(err error, toolName string) {
	e.memory.AddWithType(memory.RoleUser, fmt.Sprintf(constants.ObservationErrorFormat, toolName, err), memory.MessageTypeObservation)
}

// recordStructuredObservation records a tool observation, including structured
// Data if available. When Data is present, it is serialized as JSON and appended
// to the observation so the LLM can reason over structured results.
func (e *Engine) recordStructuredObservation(result *tools.ToolResult, toolName string) {
	if result == nil {
		return
	}

	// Special handling for skill_activate: acknowledge only, body retrieved via SkillInjector
	// This ensures skill bodies don't pollute memory history, preserving cache prefix matching
	if toolName == "skill_activate" && result.Data != nil {
		if injected, ok := result.Data["injected"].(bool); ok && injected {
			// DO NOT inject skill body to memory - it's retrieved in buildReActMessages
			// via SkillInjector.GetInjectedBodies() for cache-aware message building
			skillName, _ := result.Data["skill_name"].(string)
			e.memory.AddWithType(memory.RoleUser,
				fmt.Sprintf("[Skill activated: %s]", skillName),
				memory.MessageTypeObservation)
			return
		}
	}

	// Image DataURI results use the existing special path
	if strings.Contains(result.Content, constants.ObservationDataImageHint) {
		e.recordObservation(result.Content, toolName)
		return
	}
	if len(result.Data) > 0 {
		dataJSON, err := json.Marshal(result.Data)
		if err == nil {
			e.memory.AddWithType(memory.RoleUser,
				fmt.Sprintf(constants.ObservationWithDataFmt, toolName, result.Content, string(dataJSON)),
				memory.MessageTypeObservation)
			return
		}
	}
	e.memory.AddWithType(memory.RoleUser, fmt.Sprintf(constants.ObservationFormat, toolName, result.Content), memory.MessageTypeObservation)
}

// formatObservationWithStructuredData formats a single tool result for multi-tool
// aggregation. Like recordStructuredObservation but returns a string.
func (e *Engine) formatObservationWithStructuredData(result *tools.ToolResult, toolName string) string {
	if result == nil {
		return fmt.Sprintf(constants.ObservationFormat, toolName, "")
	}
	if strings.Contains(result.Content, constants.ObservationDataImageHint) {
		return fmt.Sprintf(constants.ObservationFormat, toolName, result.Content)
	}
	if len(result.Data) > 0 {
		dataJSON, err := json.Marshal(result.Data)
		if err == nil {
			return fmt.Sprintf(constants.ObservationWithDataFmt, toolName, result.Content, string(dataJSON))
		}
	}
	return fmt.Sprintf(constants.ObservationFormat, toolName, result.Content)
}

// ToolAction represents a parsed tool action.
type ToolAction struct {
	Tool       string         `json:"tool"`
	Parameters map[string]any `json:"parameters"`
}

// InputRequest represents a request to process user input.
type InputRequest struct {
	Input     string
	RequestID string
	EventChan chan<- events.Event // Per-request event channel
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
