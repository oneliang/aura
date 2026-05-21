package orchestrator

import (
	"context"
	"sync"
	"time"
)

// SupervisionEvent represents an event emitted by the supervisor.
type SupervisionEvent struct {
	Type         string     `yaml:"type"`
	DocIDs       []string   `yaml:"doc_ids,omitempty"`
	AgentIDs     []string   `yaml:"agent_ids,omitempty"`
	CircularDeps [][]string `yaml:"circular_deps,omitempty"`
	Message      string     `yaml:"message"`
	Timestamp    time.Time  `yaml:"timestamp"`
}

// SupervisionEventType defines possible supervisor events.
type SupervisionEventType string

const (
	EventStaleDocsDetected    SupervisionEventType = "stale_docs_detected"
	EventCircularDepsDetected SupervisionEventType = "circular_deps_detected"
	EventOrphanDocsDetected   SupervisionEventType = "orphan_docs_detected"
	EventQueueImbalance       SupervisionEventType = "queue_imbalance"
)

// SupervisionEventHandler is the callback type for supervisor events.
type SupervisionEventHandler func(event SupervisionEvent)

// SupervisorConfig holds configuration for the supervisor.
type SupervisorConfig struct {
	Interval       time.Duration // Supervision check interval
	StaleThreshold time.Duration // Time threshold for stale documents
	EnableChecks   []string      // Which checks to enable
}

// DefaultSupervisorConfig returns default supervisor configuration.
func DefaultSupervisorConfig() *SupervisorConfig {
	return &SupervisorConfig{
		Interval:       30 * time.Second,
		StaleThreshold: 5 * time.Minute,
		EnableChecks:   []string{"stale_docs", "circular_deps", "queue_imbalance"},
	}
}

// Supervisor periodically checks system health and handles edge cases.
type Supervisor struct {
	config      *SupervisorConfig
	registry    *TaskRegistry
	coordinator *DocCoordinator
	onEvent     SupervisionEventHandler
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	running     bool
}

// NewSupervisor creates a new supervisor.
func NewSupervisor(
	cfg *SupervisorConfig,
	registry *TaskRegistry,
	coordinator *DocCoordinator,
	onEvent SupervisionEventHandler,
) *Supervisor {
	return &Supervisor{
		config:      cfg,
		registry:    registry,
		coordinator: coordinator,
		onEvent:     onEvent,
	}
}

// Start begins the supervision loop.
func (s *Supervisor) Start(ctx context.Context) {
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runChecks()
		}
	}
}

// Stop terminates the supervision loop.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}

// IsRunning returns true if the supervisor is running.
func (s *Supervisor) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// runChecks executes all enabled checks.
func (s *Supervisor) runChecks() {
	enabled := make(map[string]bool)
	for _, check := range s.config.EnableChecks {
		enabled[check] = true
	}

	if enabled["stale_docs"] {
		s.checkStaleDocs()
	}

	if enabled["circular_deps"] {
		s.checkCircularDeps()
	}

	if enabled["queue_imbalance"] {
		s.checkQueueImbalance()
	}
}

// checkStaleDocs finds documents that haven't been updated recently.
func (s *Supervisor) checkStaleDocs() {
	staleDocs := s.registry.GetStaleDocs(s.config.StaleThreshold)
	if len(staleDocs) == 0 {
		return
	}

	docIDs := make([]string, len(staleDocs))
	for i, doc := range staleDocs {
		docIDs[i] = doc.ID
	}

	s.emitEvent(SupervisionEvent{
		Type:      string(EventStaleDocsDetected),
		DocIDs:    docIDs,
		Message:   "Detected stale documents that may need attention",
		Timestamp: time.Now(),
	})
}

// checkCircularDeps detects circular dependencies between documents.
func (s *Supervisor) checkCircularDeps() {
	allDocs := s.registry.List()

	// Build dependency graph
	graph := make(map[string][]string)
	for _, doc := range allDocs {
		graph[doc.ID] = doc.Dependencies
	}

	// Find cycles using DFS
	cycles := findCycles(graph)
	if len(cycles) == 0 {
		return
	}

	s.emitEvent(SupervisionEvent{
		Type:         string(EventCircularDepsDetected),
		CircularDeps: cycles,
		Message:      "Detected circular dependencies that will block execution",
		Timestamp:    time.Now(),
	})
}

// checkQueueImbalance detects significant imbalance in agent work queues.
func (s *Supervisor) checkQueueImbalance() {
	stats := s.coordinator.GetQueueStats()
	if len(stats) < 2 {
		return // Need at least 2 agents to have imbalance
	}

	var total, max, min int = 0, -1, -1
	for _, s := range stats {
		total += s.Urgent + s.Normal + s.Low
		if max == -1 || s.Urgent+s.Normal+s.Low > max {
			max = s.Urgent + s.Normal + s.Low
		}
		if min == -1 || s.Urgent+s.Normal+s.Low < min {
			min = s.Urgent + s.Normal + s.Low
		}
	}

	if total == 0 {
		return // No work in system
	}

	// If max queue is 5x larger than min queue, report imbalance
	if min == 0 && max > 5 {
		s.emitEvent(SupervisionEvent{
			Type:      string(EventQueueImbalance),
			Message:   "Work queue imbalance detected: some agents are overloaded while others are idle",
			Timestamp: time.Now(),
		})
	} else if min > 0 && max > min*3 {
		s.emitEvent(SupervisionEvent{
			Type:      string(EventQueueImbalance),
			Message:   "Work queue imbalance detected: max queue is 3x larger than min queue",
			Timestamp: time.Now(),
		})
	}
}

// emitEvent emits an event if a handler is registered.
func (s *Supervisor) emitEvent(event SupervisionEvent) {
	if s.onEvent != nil {
		s.onEvent(event)
	}
}

// findCycles finds all cycles in a directed graph using DFS.
// Returns a list of cycles, where each cycle is a list of node IDs.
func findCycles(graph map[string][]string) [][]string {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycles [][]string

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		if inStack[node] {
			// Found a cycle - extract it
			cycleStart := -1
			for i, p := range path {
				if p == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append([]string{}, path[cycleStart:]...)
				cycles = append(cycles, cycle)
			}
			return
		}

		if visited[node] {
			return
		}

		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			dfs(neighbor, path)
		}

		inStack[node] = false
	}

	for node := range graph {
		if !visited[node] {
			dfs(node, []string{})
		}
	}

	return cycles
}

// GetHealthReport returns a system health report.
func (s *Supervisor) GetHealthReport() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	staleDocs := s.registry.GetStaleDocs(s.config.StaleThreshold)
	allDocs := s.registry.List()

	// Build dependency graph for cycle detection
	graph := make(map[string][]string)
	for _, doc := range allDocs {
		graph[doc.ID] = doc.Dependencies
	}
	cycles := findCycles(graph)

	queueStats := s.coordinator.GetQueueStats()

	return map[string]interface{}{
		"running":           s.running,
		"total_docs":        len(allDocs),
		"stale_docs":        len(staleDocs),
		"circular_deps":     len(cycles),
		"queue_stats":       queueStats,
		"supervisor_config": s.config,
	}
}
