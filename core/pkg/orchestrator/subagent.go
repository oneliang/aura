package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oneliang/aura/core/pkg/runtime"
	"github.com/oneliang/aura/core/pkg/workspace"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// AgentStatus represents the lifecycle state of a sub-agent.
type AgentStatus string

const (
	// AgentStatusIdle agent is idle, waiting for work.
	AgentStatusIdle AgentStatus = "idle"
	// AgentStatusBusy agent is processing a document.
	AgentStatusBusy AgentStatus = "busy"
	// AgentStatusDone agent has finished its task.
	AgentStatusDone AgentStatus = "done"
	// AgentStatusError agent encountered an error.
	AgentStatusError AgentStatus = "error"
)

// SubAgent represents a sub-agent with isolated workspace and memory.
type SubAgent struct {
	ID            string
	cfg           *config.Config
	workspace     *workspace.Workspace
	memory        memory.Memory
	status        AgentStatus
	workQueue     *WorkQueue
	runtime       *runtime.AgentRuntime
	coordinator   *DocCoordinator
	currentDoc    *CollaboDoc
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	lastActive    time.Time
	logger        *logger.Logger
	disableTools  bool // Whether tools are disabled for this agent
}

// SubAgentConfig holds configuration for creating a sub-agent.
type SubAgentConfig struct {
	ID           string
	ParentConfig *config.Config
	LLMOverride  *config.LLMConfig // Optional LLM config override
	Workspace    *workspace.Workspace
	WorkQueue    *WorkQueue
	Coordinator  *DocCoordinator
	SystemPrompt string // Optional custom system prompt
	DisableTools bool   // If true, disable tools for this agent
	Logger       *logger.Logger
}

// NewSubAgent creates a new sub-agent.
func NewSubAgent(cfg *SubAgentConfig) (*SubAgent, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	if cfg.ParentConfig == nil {
		return nil, fmt.Errorf("parent config is required")
	}
	if cfg.Workspace == nil {
		return nil, fmt.Errorf("workspace is required")
	}
	if cfg.WorkQueue == nil {
		return nil, fmt.Errorf("work queue is required")
	}

	// Clone parent config and apply LLM override if provided — merge non-empty fields only.
	agentCfg := *cfg.ParentConfig
	if cfg.LLMOverride != nil {
		if cfg.LLMOverride.Model != "" {
			agentCfg.LLM.Model = cfg.LLMOverride.Model
		}
		if cfg.LLMOverride.Provider != "" {
			agentCfg.LLM.Provider = cfg.LLMOverride.Provider
		}
		if cfg.LLMOverride.BaseURL != "" {
			agentCfg.LLM.BaseURL = cfg.LLMOverride.BaseURL
		}
		if cfg.LLMOverride.APIKey != "" {
			agentCfg.LLM.APIKey = cfg.LLMOverride.APIKey
		}
		if cfg.LLMOverride.EmbeddingModel != "" {
			agentCfg.LLM.EmbeddingModel = cfg.LLMOverride.EmbeddingModel
		}
	}

	// Apply custom system prompt if provided
	if cfg.SystemPrompt != "" {
		// Note: This would require adding a field to Config or a different approach
		// For now, we'll handle this in the runtime options
	}

	sa := &SubAgent{
		ID:           cfg.ID,
		cfg:          &agentCfg,
		workspace:    cfg.Workspace,
		workQueue:    cfg.WorkQueue,
		coordinator:  cfg.Coordinator,
		status:       AgentStatusIdle,
		lastActive:   time.Now(),
		logger:       cfg.Logger,
		disableTools: cfg.DisableTools,
	}

	return sa, nil
}

// Start initializes the agent and starts processing work from the queue.
func (sa *SubAgent) Start(ctx context.Context) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.ctx != nil {
		return fmt.Errorf("agent already started")
	}

	sa.ctx, sa.cancel = context.WithCancel(ctx)

	// Create runtime
	rtCfg := &runtime.RuntimeConfig{
		Config:       sa.cfg,
		SessionID:    "", // Use default session
		Role:         "sub-agent",
		DisableTools: sa.disableTools,
	}

	// Set custom system prompt if needed
	if sa.workQueue != nil {
		// Agent-specific prompt can be injected here
	}

	var err error
	sa.runtime, err = runtime.New(rtCfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	if err := sa.runtime.Initialize(sa.ctx); err != nil {
		return fmt.Errorf("failed to initialize runtime: %w", err)
	}

	// Use the runtime's memory directly (it implements agent.Memory interface)
	sa.memory = sa.runtime.GetMemory()

	// Start worker goroutine
	go sa.workerLoop()

	return nil
}

// Stop terminates the agent and releases resources.
func (sa *SubAgent) Stop() {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.cancel != nil {
		sa.cancel()
	}

	if sa.runtime != nil {
		sa.runtime.Shutdown()
	}

	sa.status = AgentStatusDone
	sa.ctx = nil
	sa.cancel = nil
}

// Status returns the current agent status.
func (sa *SubAgent) Status() AgentStatus {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.status
}

// Workspace returns the agent's workspace.
func (sa *SubAgent) Workspace() *workspace.Workspace {
	return sa.workspace
}

// Memory returns the agent's memory.
func (sa *SubAgent) Memory() memory.Memory {
	return sa.memory
}

// CurrentDoc returns the document currently being processed.
func (sa *SubAgent) CurrentDoc() *CollaboDoc {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.currentDoc
}

// LastActive returns the last activity timestamp.
func (sa *SubAgent) LastActive() time.Time {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.lastActive
}

// workerLoop continuously processes documents from the work queue.
func (sa *SubAgent) workerLoop() {
	for {
		select {
		case <-sa.ctx.Done():
			return
		default:
		}

		// Get next document from queue
		doc, ok := sa.workQueue.Dequeue()
		if !ok {
			// No work available, wait a bit
			select {
			case <-sa.ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		// Process the document
		sa.processDocument(doc)
	}
}

// processDocument processes a single collaboration document.
func (sa *SubAgent) processDocument(doc *CollaboDoc) {
	sa.mu.Lock()
	sa.status = AgentStatusBusy
	sa.currentDoc = doc
	sa.lastActive = time.Now()
	sa.mu.Unlock()

	// Mark document as in progress
	if err := sa.coordinator.MarkInProgress(doc.ID, sa.ID); err != nil {
		sa.mu.Lock()
		sa.status = AgentStatusError
		sa.mu.Unlock()
		return
	}

	// Build the prompt from the document
	prompt := sa.buildPromptFromDoc(doc)

	// Process through the agent runtime
	events, err := sa.runtime.Process(sa.ctx, prompt)
	if err != nil {
		sa.handleProcessingError(doc, err)
		return
	}

	// Collect results from event stream
	var result string
	for ev := range events {
		switch ev.Type() {
		case runtime.EventTypeResponse:
			result = ev.Content()
		case runtime.EventTypeError:
			sa.handleProcessingError(doc, fmt.Errorf("%s", ev.Content()))
			return
		}
	}

	// Mark document as completed
	if err := sa.coordinator.MarkCompleted(doc.ID, sa.ID, result); err != nil {
		sa.mu.Lock()
		sa.status = AgentStatusError
		sa.mu.Unlock()
		return
	}

	sa.mu.Lock()
	sa.status = AgentStatusIdle
	sa.currentDoc = nil
	sa.mu.Unlock()
}

// buildPromptFromDoc builds a prompt from the collaboration document.
func (sa *SubAgent) buildPromptFromDoc(doc *CollaboDoc) string {
	prompt := fmt.Sprintf(`## Task Assignment

**Document ID:** %s
**Title:** %s
**From:** %s

## Instructions

%s

## Response Format

Please complete this task and provide a clear summary of your work.
`, doc.ID, doc.Title, doc.From, doc.Body)

	return prompt
}

// handleProcessingError handles errors during document processing.
func (sa *SubAgent) handleProcessingError(doc *CollaboDoc, err error) {
	sa.mu.Lock()
	sa.status = AgentStatusError
	sa.mu.Unlock()

	// Mark document as rejected with error reason
	if markErr := sa.coordinator.MarkRejected(doc.ID, sa.ID, err.Error()); markErr != nil {
		if sa.logger != nil {
			sa.logger.Error().Err(markErr).
				Str("doc_id", doc.ID).
				Str("agent_id", sa.ID).
				Msg("Failed to mark document as rejected")
		}
	}
}

// AssignDoc assigns a specific document to this agent.
func (sa *SubAgent) AssignDoc(doc *CollaboDoc) {
	sa.workQueue.Enqueue(doc)
}

// IsAvailable returns true if the agent is available for work.
func (sa *SubAgent) IsAvailable() bool {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.status == AgentStatusIdle
}

// Stats returns agent statistics.
func (sa *SubAgent) Stats() map[string]interface{} {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	return map[string]interface{}{
		"id":          sa.ID,
		"status":      sa.status,
		"workspace":   sa.workspace.Dir,
		"last_active": sa.lastActive,
		"current_doc": func() string {
			if sa.currentDoc != nil {
				return sa.currentDoc.ID
			}
			return ""
		}(),
	}
}
