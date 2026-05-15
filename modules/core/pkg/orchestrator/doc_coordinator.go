package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CoordinationEvent represents an event emitted by the coordinator.
type CoordinationEvent struct {
	Type      string    `yaml:"type"`
	DocID     string    `yaml:"doc_id"`
	AgentID   string    `yaml:"agent_id,omitempty"`
	Status    DocStatus `yaml:"status,omitempty"`
	Message   string    `yaml:"message"`
	Timestamp time.Time `yaml:"timestamp"`
}

// CoordinationEventType defines possible coordinator events.
type CoordinationEventType string

const (
	EventDocCreated        CoordinationEventType = "doc_created"
	EventDocAssigned       CoordinationEventType = "doc_assigned"
	EventDocStarted        CoordinationEventType = "doc_started"
	EventDocCompleted      CoordinationEventType = "doc_completed"
	EventDocRejected       CoordinationEventType = "doc_rejected"
	EventDocBlocked        CoordinationEventType = "doc_blocked"
	EventDocUnblocked      CoordinationEventType = "doc_unblocked"
	EventAgentRegistered   CoordinationEventType = "agent_registered"
	EventAgentUnregistered CoordinationEventType = "agent_unregistered"
)

// CoordinationEventHandler is the callback type for coordinator events.
type CoordinationEventHandler func(event CoordinationEvent)

// DocCoordinator manages the lifecycle of collaboration documents:
// polls DocStore, checks dependencies, routes to per-agent WorkQueues.
type DocCoordinator struct {
	store        *DocStore
	registry     *TaskRegistry
	agentQueues  map[string]*WorkQueue // keyed by agent ID
	scanInterval time.Duration
	onEvent      CoordinationEventHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
}

// NewDocCoordinator creates a new document coordinator.
func NewDocCoordinator(store *DocStore, registry *TaskRegistry, interval time.Duration, onEvent CoordinationEventHandler) *DocCoordinator {
	return &DocCoordinator{
		store:        store,
		registry:     registry,
		agentQueues:  make(map[string]*WorkQueue),
		scanInterval: interval,
		onEvent:      onEvent,
	}
}

// RegisterAgent creates a work queue for the given agent ID.
func (c *DocCoordinator) RegisterAgent(agentID string) *WorkQueue {
	c.mu.Lock()
	defer c.mu.Unlock()

	if queue, exists := c.agentQueues[agentID]; exists {
		return queue
	}

	queue := NewWorkQueue(agentID)
	c.agentQueues[agentID] = queue

	c.emitEvent(CoordinationEvent{
		Type:      string(EventAgentRegistered),
		AgentID:   agentID,
		Message:   fmt.Sprintf("Agent %s registered", agentID),
		Timestamp: time.Now(),
	})

	return queue
}

// UnregisterAgent removes the work queue for the agent.
func (c *DocCoordinator) UnregisterAgent(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.agentQueues, agentID)

	c.emitEvent(CoordinationEvent{
		Type:      string(EventAgentUnregistered),
		AgentID:   agentID,
		Message:   fmt.Sprintf("Agent %s unregistered", agentID),
		Timestamp: time.Now(),
	})
}

// QueueFor returns the WorkQueue for a given agent (nil if not found).
func (c *DocCoordinator) QueueFor(agentID string) *WorkQueue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agentQueues[agentID]
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (c *DocCoordinator) Run(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()

	ticker := time.NewTicker(c.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.scanAndRoute()
		}
	}
}

// scanAndRoute polls the store and routes new documents to agent queues.
func (c *DocCoordinator) scanAndRoute() {
	// Sync registry from store
	if err := c.registry.SyncFromStore(c.store); err != nil {
		c.emitEvent(CoordinationEvent{
			Type:      "error",
			Message:   fmt.Sprintf("Failed to sync registry: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// Get completed IDs for dependency checking
	completedIDs := c.registry.GetCompletedIDs()

	// Get all pending documents
	pendingDocs := c.registry.ListByStatus(DocStatusPending)

	for _, doc := range pendingDocs {
		// Check if already enqueued (skip if doc is in progress)
		if doc.Status == DocStatusInProgress {
			continue
		}

		// Check dependencies
		if doc.IsBlocked(completedIDs) {
			// Mark as blocked if not already
			if doc.Status != DocStatusBlocked {
				doc.Status = DocStatusBlocked
				c.store.UpdateStatus(doc.ID, DocStatusBlocked, "Blocked waiting for dependencies")
			}
			continue
		}

		// Dependencies met - route to appropriate queue
		c.routeToQueue(doc)
	}

	// Check blocked documents that may now be unblocked
	blockedDocs := c.registry.ListByStatus(DocStatusBlocked)
	for _, doc := range blockedDocs {
		if !doc.IsBlocked(completedIDs) {
			// Update status back to pending
			doc.Status = DocStatusPending
			c.store.UpdateStatus(doc.ID, DocStatusPending, "Dependencies satisfied")
			c.emitEvent(CoordinationEvent{
				Type:      string(EventDocUnblocked),
				DocID:     doc.ID,
				Message:   fmt.Sprintf("Document %s unblocked", doc.ID),
				Timestamp: time.Now(),
			})
		}
	}
}

// routeToQueue routes a document to the appropriate agent queue.
func (c *DocCoordinator) routeToQueue(doc *CollaboDoc) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	targetAgent := doc.To
	if targetAgent == "" || targetAgent == "any" {
		// Route to any available agent (simple round-robin or first available)
		targetAgent = c.findAvailableAgent()
	}

	if targetAgent == "" {
		// No agents available
		return
	}

	queue, exists := c.agentQueues[targetAgent]
	if !exists {
		// Agent not registered, skip
		return
	}

	// Update document status and enqueue
	doc.Status = DocStatusPending
	queue.Enqueue(doc)

	c.emitEvent(CoordinationEvent{
		Type:      string(EventDocAssigned),
		DocID:     doc.ID,
		AgentID:   targetAgent,
		Status:    doc.Status,
		Message:   fmt.Sprintf("Document %s assigned to agent %s", doc.ID, targetAgent),
		Timestamp: time.Now(),
	})
}

// findAvailableAgent returns the first agent with the shortest queue.
func (c *DocCoordinator) findAvailableAgent() string {
	if len(c.agentQueues) == 0 {
		return ""
	}

	var bestAgent string
	minLen := -1

	for agentID, queue := range c.agentQueues {
		len := queue.Len()
		if minLen == -1 || len < minLen {
			minLen = len
			bestAgent = agentID
		}
	}

	return bestAgent
}

// Submit writes a new doc to the store and registers it.
func (c *DocCoordinator) Submit(doc *CollaboDoc) error {
	if err := c.store.Save(doc); err != nil {
		return err
	}

	c.registry.Register(doc)

	c.emitEvent(CoordinationEvent{
		Type:      string(EventDocCreated),
		DocID:     doc.ID,
		AgentID:   doc.From,
		Status:    doc.Status,
		Message:   fmt.Sprintf("Document %s created: %s", doc.ID, doc.Title),
		Timestamp: time.Now(),
	})

	return nil
}

// MarkInProgress atomically updates doc status in store and registry.
func (c *DocCoordinator) MarkInProgress(id, agentID string) error {
	doc, exists := c.registry.Get(id)
	if !exists {
		return fmt.Errorf("document not found: %s", id)
	}

	doc.Status = DocStatusInProgress
	doc.AddHistory(agentID, "started", "")

	if err := c.store.Save(doc); err != nil {
		return err
	}

	// Remove from queue (will be re-added if rejected)
	if queue := c.QueueFor(agentID); queue != nil {
		queue.Remove(id)
	}

	c.emitEvent(CoordinationEvent{
		Type:      string(EventDocStarted),
		DocID:     id,
		AgentID:   agentID,
		Status:    DocStatusInProgress,
		Message:   fmt.Sprintf("Document %s started by agent %s", id, agentID),
		Timestamp: time.Now(),
	})

	return nil
}

// MarkCompleted atomically updates status and triggers dependent doc checks.
func (c *DocCoordinator) MarkCompleted(id, agentID, result string) error {
	doc, exists := c.registry.Get(id)
	if !exists {
		return fmt.Errorf("document not found: %s", id)
	}

	doc.Status = DocStatusCompleted
	doc.AddHistory(agentID, "completed", result)

	if err := c.store.Save(doc); err != nil {
		return err
	}

	c.emitEvent(CoordinationEvent{
		Type:      string(EventDocCompleted),
		DocID:     id,
		AgentID:   agentID,
		Status:    DocStatusCompleted,
		Message:   fmt.Sprintf("Document %s completed by agent %s", id, agentID),
		Timestamp: time.Now(),
	})

	return nil
}

// MarkRejected marks a doc rejected with reason.
func (c *DocCoordinator) MarkRejected(id, agentID, reason string) error {
	doc, exists := c.registry.Get(id)
	if !exists {
		return fmt.Errorf("document not found: %s", id)
	}

	doc.Status = DocStatusRejected
	doc.AddHistory(agentID, "rejected", reason)

	if err := c.store.Save(doc); err != nil {
		return err
	}

	c.emitEvent(CoordinationEvent{
		Type:      string(EventDocRejected),
		DocID:     id,
		AgentID:   agentID,
		Status:    DocStatusRejected,
		Message:   fmt.Sprintf("Document %s rejected by agent %s: %s", id, agentID, reason),
		Timestamp: time.Now(),
	})

	return nil
}

// Stop stops the coordinator polling loop.
func (c *DocCoordinator) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
}

// IsRunning returns true if the coordinator is running.
func (c *DocCoordinator) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// emitEvent emits an event if a handler is registered.
func (c *DocCoordinator) emitEvent(event CoordinationEvent) {
	if c.onEvent != nil {
		c.onEvent(event)
	}
}

// GetQueueStats returns statistics for all agent queues.
func (c *DocCoordinator) GetQueueStats() map[string]struct {
	Urgent int
	Normal int
	Low    int
} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]struct {
		Urgent int
		Normal int
		Low    int
	})

	for agentID, queue := range c.agentQueues {
		urgent, normal, low := queue.Stats()
		stats[agentID] = struct {
			Urgent int
			Normal int
			Low    int
		}{urgent, normal, low}
	}

	return stats
}
