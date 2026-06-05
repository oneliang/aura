// Package orchestrator provides the top-level Orchestrator for multi-agent coordination.
package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/workspace"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/logger"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// OrchestratorStatus represents the overall orchestrator state.
type OrchestratorStatus string

const (
	// OrchestratorStatusStopped orchestrator is not running.
	OrchestratorStatusStopped OrchestratorStatus = "stopped"
	// OrchestratorStatusStarting orchestrator is starting up.
	OrchestratorStatusStarting OrchestratorStatus = "starting"
	// OrchestratorStatusRunning orchestrator is running normally.
	OrchestratorStatusRunning OrchestratorStatus = "running"
	// OrchestratorStatusStopping orchestrator is shutting down.
	OrchestratorStatusStopping OrchestratorStatus = "stopping"
)

// Orchestrator is the top-level multi-agent orchestrator.
// It manages sub-agents, collaboration documents, and supervision.
type Orchestrator struct {
	config      *config.OrchestratorConfig
	parentCfg   *config.Config
	workspace   *workspace.Isolator
	docStore    *DocStore
	registry    *TaskRegistry
	coordinator *DocCoordinator
	supervisor  *Supervisor
	subAgents   map[string]*SubAgent
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	status      OrchestratorStatus
	logger      *logger.Logger
}

// OrchestratorConfig holds the full configuration for the orchestrator.
type OrchestratorConfig struct {
	ParentConfig        *config.Config
	Enabled             bool
	MaxSubAgents        int
	WorkspaceDir        string
	SupervisionInterval time.Duration
	StaleDocThreshold   time.Duration
	AutoCleanup         bool
}

// New creates a new orchestrator.
func New(cfg *config.Config) (*Orchestrator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	orchCfg := cfg.Orchestrator
	if orchCfg.WorkspaceDir == "" {
		orchCfg.WorkspaceDir = ffp.MustAuraHomePath("workspace")
	}

	// Create workspace isolator
	wsIsolator, err := workspace.NewIsolator(orchCfg.WorkspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create document store
	docsDir := wsIsolator.SharedDocsDir()
	docStore, err := NewDocStore(docsDir)
	if err != nil {
		wsIsolator.CleanupAll()
		return nil, fmt.Errorf("failed to create doc store: %w", err)
	}

	// Create task registry
	registry := NewTaskRegistry()

	// Create logger from config
	log := logger.NewNamed(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
		Path:   cfg.Log.Path,
		Module: "orchestrator",
	})

	o := &Orchestrator{
		config:    &orchCfg,
		parentCfg: cfg,
		workspace: wsIsolator,
		docStore:  docStore,
		registry:  registry,
		subAgents: make(map[string]*SubAgent),
		status:    OrchestratorStatusStopped,
		logger:    log,
	}

	// Create coordinator with event handler
	o.coordinator = NewDocCoordinator(docStore, registry, 5*time.Second, o.handleCoordinationEvent)

	// Create supervisor
	supCfg := DefaultSupervisorConfig()
	supCfg.Interval = orchCfg.SupervisionInterval
	supCfg.StaleThreshold = orchCfg.StaleDocThreshold
	o.supervisor = NewSupervisor(supCfg, registry, o.coordinator, o.handleSupervisionEvent)

	return o, nil
}

// Start begins the orchestrator and all its components.
func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status == OrchestratorStatusRunning {
		return fmt.Errorf("orchestrator already running")
	}

	o.ctx, o.cancel = context.WithCancel(ctx)
	o.status = OrchestratorStatusStarting

	// Start coordinator in a goroutine
	go o.coordinator.Run(o.ctx)

	// Start supervisor in a goroutine
	go o.supervisor.Start(o.ctx)

	o.status = OrchestratorStatusRunning
	return nil
}

// Stop terminates the orchestrator and all sub-agents.
func (o *Orchestrator) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status == OrchestratorStatusStopped {
		return
	}

	o.status = OrchestratorStatusStopping

	// Stop all sub-agents
	for _, agent := range o.subAgents {
		agent.Stop()
	}

	// Stop coordinator
	o.coordinator.Stop()

	// Stop supervisor
	o.supervisor.Stop()

	// Cancel context
	if o.cancel != nil {
		o.cancel()
	}

	// Cleanup workspaces if configured
	if o.config.AutoCleanup {
		o.workspace.CleanupAll()
	}

	o.subAgents = make(map[string]*SubAgent)
	o.status = OrchestratorStatusStopped
}

// SpawnAgent creates and starts a new sub-agent.
func (o *Orchestrator) SpawnAgent(ctx context.Context, agentID string, llmOverride *config.LLMConfig) (*SubAgent, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.subAgents) >= o.config.MaxSubAgents {
		return nil, fmt.Errorf("max sub-agents limit reached (%d)", o.config.MaxSubAgents)
	}

	if _, exists := o.subAgents[agentID]; exists {
		return nil, fmt.Errorf("agent %s already exists", agentID)
	}

	// Create workspace for agent
	ws, err := o.workspace.Create(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Register agent with coordinator
	queue := o.coordinator.RegisterAgent(agentID)

	// Create sub-agent
	sa, err := NewSubAgent(&SubAgentConfig{
		ID:           agentID,
		ParentConfig: o.parentCfg,
		LLMOverride:  llmOverride,
		Workspace:    ws,
		WorkQueue:    queue,
		Coordinator:  o.coordinator,
		Logger:       o.logger,
	})
	if err != nil {
		o.coordinator.UnregisterAgent(agentID)
		return nil, fmt.Errorf("failed to create sub-agent: %w", err)
	}

	// Start the agent
	if err := sa.Start(ctx); err != nil {
		o.coordinator.UnregisterAgent(agentID)
		return nil, fmt.Errorf("failed to start sub-agent: %w", err)
	}

	o.subAgents[agentID] = sa

	return sa, nil
}

// StopAgent stops and removes a sub-agent.
func (o *Orchestrator) StopAgent(agentID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	sa, exists := o.subAgents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Stop the agent
	sa.Stop()

	// Unregister from coordinator
	o.coordinator.UnregisterAgent(agentID)

	// Cleanup workspace
	if err := o.workspace.Cleanup(agentID); err != nil {
		// Log but don't fail
	}

	delete(o.subAgents, agentID)
	return nil
}

// GetAgent returns a sub-agent by ID.
func (o *Orchestrator) GetAgent(agentID string) (*SubAgent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	sa, exists := o.subAgents[agentID]
	return sa, exists
}

// ListAgents returns all sub-agents.
func (o *Orchestrator) ListAgents() []*SubAgent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]*SubAgent, 0, len(o.subAgents))
	for _, sa := range o.subAgents {
		agents = append(agents, sa)
	}
	return agents
}

// CreateDoc creates a new collaboration document.
func (o *Orchestrator) CreateDoc(doc *CollaboDoc) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("doc is nil")
	}

	// Generate ID if not provided
	if doc.ID == "" {
		doc.ID = generateDocID()
	}

	// Set defaults
	if doc.Status == "" {
		doc.Status = DocStatusPending
	}
	if doc.Priority == "" {
		doc.Priority = PriorityNormal
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = time.Now()
	}

	// Submit to coordinator
	if err := o.coordinator.Submit(doc); err != nil {
		return "", err
	}

	return doc.ID, nil
}

// GetPendingDocs returns pending documents for an agent.
func (o *Orchestrator) GetPendingDocs(agentID string) ([]*CollaboDoc, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	queue := o.coordinator.QueueFor(agentID)
	if queue == nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return queue.Snapshot(), nil
}

// UpdateDocStatus updates the status of a document.
func (o *Orchestrator) UpdateDocStatus(id, agentID string, status DocStatus, note string) error {
	switch status {
	case DocStatusInProgress:
		return o.coordinator.MarkInProgress(id, agentID)
	case DocStatusCompleted:
		return o.coordinator.MarkCompleted(id, agentID, note)
	case DocStatusRejected:
		return o.coordinator.MarkRejected(id, agentID, note)
	default:
		return o.docStore.UpdateStatus(id, status, note)
	}
}

// GetDoc retrieves a document by ID.
func (o *Orchestrator) GetDoc(id string) (*CollaboDoc, error) {
	return o.docStore.Load(id)
}

// ListDocs lists documents with optional status filter.
func (o *Orchestrator) ListDocs(status ...DocStatus) ([]*CollaboDoc, error) {
	return o.docStore.List(status...)
}

// Status returns the orchestrator status.
func (o *Orchestrator) Status() OrchestratorStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

// GetHealthReport returns a comprehensive health report.
func (o *Orchestrator) GetHealthReport() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agentStatus := make([]map[string]interface{}, 0, len(o.subAgents))
	for _, sa := range o.subAgents {
		agentStatus = append(agentStatus, sa.Stats())
	}

	return map[string]interface{}{
		"orchestrator_status": o.status,
		"sub_agents":          len(o.subAgents),
		"agent_status":        agentStatus,
		"total_docs":          o.registry.Count(),
		"supervisor_report":   o.supervisor.GetHealthReport(),
		"queue_stats":         o.coordinator.GetQueueStats(),
		"workspace_dir":       o.workspace.BaseDir(),
	}
}

// handleCoordinationEvent handles events from the coordinator.
func (o *Orchestrator) handleCoordinationEvent(event CoordinationEvent) {
	o.logger.Debug("orchestrator coordination event", "type", string(event.Type), "message", event.Message)
}

// handleSupervisionEvent handles events from the supervisor.
func (o *Orchestrator) handleSupervisionEvent(event SupervisionEvent) {
	o.logger.Debug("orchestrator supervision event", "type", string(event.Type), "message", event.Message)
}

// generateDocID generates a unique document ID.
func generateDocID() string {
	return "doc-" + uuid.New().String()[:8]
}

// GetWorkspace returns the workspace isolator.
func (o *Orchestrator) GetWorkspace() *workspace.Isolator {
	return o.workspace
}

// GetCoordinator returns the document coordinator.
func (o *Orchestrator) GetCoordinator() *DocCoordinator {
	return o.coordinator
}
