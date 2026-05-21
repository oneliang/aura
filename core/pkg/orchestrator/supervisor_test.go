package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDefaultSupervisorConfig(t *testing.T) {
	cfg := DefaultSupervisorConfig()

	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}
	if cfg.StaleThreshold != 5*time.Minute {
		t.Errorf("StaleThreshold = %v, want 5m", cfg.StaleThreshold)
	}
	if len(cfg.EnableChecks) != 3 {
		t.Errorf("EnableChecks length = %d, want 3", len(cfg.EnableChecks))
	}
}

func TestNewSupervisor(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	handler := func(event SupervisionEvent) {}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	if s == nil {
		t.Fatal("NewSupervisor() returned nil")
	}
	if s.config != cfg {
		t.Error("config not set correctly")
	}
	if s.registry != registry {
		t.Error("registry not set correctly")
	}
	if s.coordinator != coordinator {
		t.Error("coordinator not set correctly")
	}
}

func TestSupervisor_IsRunning(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	cfg.Interval = 50 * time.Millisecond // Short interval for testing
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	handler := func(event SupervisionEvent) {}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	if s.IsRunning() {
		t.Error("IsRunning() should return false before Start()")
	}

	ctx := context.Background()
	go s.Start(ctx)

	// Give it time to start
	time.Sleep(20 * time.Millisecond)

	if !s.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	s.Stop()

	// Give it time to stop
	time.Sleep(20 * time.Millisecond)

	if s.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestSupervisor_StartStop(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	cfg.Interval = 50 * time.Millisecond
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	handler := func(event SupervisionEvent) {}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)

	// Let it run for a bit
	time.Sleep(150 * time.Millisecond)

	// Stop
	s.Stop()
	cancel()

	// Give it time to stop
	time.Sleep(50 * time.Millisecond)

	if s.IsRunning() {
		t.Error("Supervisor should be stopped")
	}
}

func TestSupervisor_runChecks(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	cfg.Interval = 100 * time.Millisecond
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	var mu sync.Mutex
	events := []SupervisionEvent{}
	handler := func(event SupervisionEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx)

	// Let checks run
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have emitted some events (even if empty results)
	// The checks should run without errors
}

func TestSupervisor_checkStaleDocs(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	cfg.Interval = 100 * time.Hour // Disable periodic checks
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	var mu sync.Mutex
	events := []SupervisionEvent{}
	handler := func(event SupervisionEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	// Register a stale doc
	doc := &CollaboDoc{
		ID:        "test-doc-1",
		Type:      DocTypeTaskAssign,
		From:      "orchestrator",
		To:        "agent-1",
		Priority:  PriorityNormal,
		Status:    DocStatusPending,
		Title:     "Test Doc",
		CreatedAt: time.Now().Add(-10 * time.Minute), // Old document
		UpdatedAt: time.Now().Add(-10 * time.Minute), // Stale
	}
	registry.Register(doc)

	// Run check
	s.checkStaleDocs()

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	} else if events[0].Type != string(EventStaleDocsDetected) {
		t.Errorf("Event type = %q, want %q", events[0].Type, EventStaleDocsDetected)
	}
}

func TestSupervisor_checkCircularDeps(t *testing.T) {
	tests := []struct {
		name      string
		docs      []*CollaboDoc
		wantEvent bool
		eventType SupervisionEventType
	}{
		{
			name:      "no circular deps",
			docs:      []*CollaboDoc{{ID: "doc-1", Dependencies: []string{"doc-2"}}, {ID: "doc-2", Dependencies: nil}},
			wantEvent: false,
		},
		{
			name:      "circular deps A->B->A",
			docs:      []*CollaboDoc{{ID: "doc-A", Dependencies: []string{"doc-B"}}, {ID: "doc-B", Dependencies: []string{"doc-A"}}},
			wantEvent: true,
			eventType: EventCircularDepsDetected,
		},
		{
			name:      "self-referencing doc",
			docs:      []*CollaboDoc{{ID: "doc-self", Dependencies: []string{"doc-self"}}},
			wantEvent: true,
			eventType: EventCircularDepsDetected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultSupervisorConfig()
			cfg.Interval = 100 * time.Hour
			registry := NewTaskRegistry()
			store, _, cleanup := setupTestDocStore(t)
			defer cleanup()
			coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

			var mu sync.Mutex
			events := []SupervisionEvent{}
			handler := func(event SupervisionEvent) {
				mu.Lock()
				defer mu.Unlock()
				events = append(events, event)
			}

			s := NewSupervisor(cfg, registry, coordinator, handler)

			// Register docs
			for _, doc := range tt.docs {
				registry.Register(doc)
			}

			// Run check
			s.checkCircularDeps()

			mu.Lock()
			defer mu.Unlock()

			if tt.wantEvent && len(events) == 0 {
				t.Error("Expected event, got none")
			}
			if !tt.wantEvent && len(events) > 0 {
				t.Errorf("Expected no event, got %d", len(events))
			}
			if tt.wantEvent && len(events) > 0 && events[0].Type != string(tt.eventType) {
				t.Errorf("Event type = %q, want %q", events[0].Type, tt.eventType)
			}
		})
	}
}

func TestSupervisor_checkQueueImbalance(t *testing.T) {
	tests := []struct {
		name        string
		setupQueues func(*DocCoordinator)
		wantEvent   bool
	}{
		{
			name: "balanced queues",
			setupQueues: func(c *DocCoordinator) {
				c.RegisterAgent("agent-1")
				c.RegisterAgent("agent-2")
				// Add docs to both queues
				doc1 := &CollaboDoc{ID: "doc-1", Priority: PriorityNormal}
				doc2 := &CollaboDoc{ID: "doc-2", Priority: PriorityNormal}
				c.agentQueues["agent-1"].Enqueue(doc1)
				c.agentQueues["agent-2"].Enqueue(doc2)
			},
			wantEvent: false,
		},
		{
			name: "one agent idle",
			setupQueues: func(c *DocCoordinator) {
				c.RegisterAgent("agent-1")
				c.RegisterAgent("agent-2")
				// Add 10 docs to agent-1, none to agent-2
				for i := 0; i < 10; i++ {
					c.agentQueues["agent-1"].Enqueue(&CollaboDoc{ID: "doc-" + string(rune(i+'0')), Priority: PriorityNormal})
				}
			},
			wantEvent: true,
		},
		{
			name: "single agent",
			setupQueues: func(c *DocCoordinator) {
				c.RegisterAgent("agent-1")
				for i := 0; i < 10; i++ {
					c.agentQueues["agent-1"].Enqueue(&CollaboDoc{ID: "doc-" + string(rune(i+'0')), Priority: PriorityNormal})
				}
			},
			wantEvent: false,
		},
		{
			name: "no work",
			setupQueues: func(c *DocCoordinator) {
				c.RegisterAgent("agent-1")
				c.RegisterAgent("agent-2")
			},
			wantEvent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultSupervisorConfig()
			cfg.Interval = 100 * time.Hour
			registry := NewTaskRegistry()
			store, _, cleanup := setupTestDocStore(t)
			defer cleanup()
			coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

			// Setup queues
			tt.setupQueues(coordinator)

			var mu sync.Mutex
			events := []SupervisionEvent{}
			handler := func(event SupervisionEvent) {
				mu.Lock()
				defer mu.Unlock()
				events = append(events, event)
			}

			s := NewSupervisor(cfg, registry, coordinator, handler)

			// Run check
			s.checkQueueImbalance()

			mu.Lock()
			defer mu.Unlock()

			if tt.wantEvent && len(events) == 0 {
				t.Error("Expected event, got none")
			}
			if !tt.wantEvent && len(events) > 0 {
				t.Errorf("Expected no event, got %d", len(events))
			}
			if tt.wantEvent && len(events) > 0 && events[0].Type != string(EventQueueImbalance) {
				t.Errorf("Event type = %q, want %q", events[0].Type, EventQueueImbalance)
			}
		})
	}
}

func TestSupervisor_emitEvent(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	var receivedEvent SupervisionEvent
	var eventReceived bool
	handler := func(event SupervisionEvent) {
		receivedEvent = event
		eventReceived = true
	}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	testEvent := SupervisionEvent{
		Type:      "test_event",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	s.emitEvent(testEvent)

	if !eventReceived {
		t.Error("Event handler was not called")
	}
	if receivedEvent.Type != "test_event" {
		t.Errorf("Event type = %q, want 'test_event'", receivedEvent.Type)
	}
	if receivedEvent.Message != "Test message" {
		t.Errorf("Event message = %q, want 'Test message'", receivedEvent.Message)
	}
}

func TestSupervisor_emitEvent_NilHandler(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Create supervisor without event handler
	s := NewSupervisor(cfg, registry, coordinator, nil)

	testEvent := SupervisionEvent{
		Type:      "test_event",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Should not panic
	s.emitEvent(testEvent)
}

func TestSupervisor_GetHealthReport(t *testing.T) {
	cfg := DefaultSupervisorConfig()
	cfg.Interval = 100 * time.Hour
	registry := NewTaskRegistry()
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	handler := func(event SupervisionEvent) {}

	s := NewSupervisor(cfg, registry, coordinator, handler)

	// Register some docs
	registry.Register(&CollaboDoc{ID: "doc-1", Status: DocStatusPending})
	registry.Register(&CollaboDoc{ID: "doc-2", Status: DocStatusPending, Dependencies: []string{"doc-1"}})

	report := s.GetHealthReport()

	if report["running"] != false {
		t.Error("running should be false before Start()")
	}
	if report["total_docs"].(int) != 2 {
		t.Errorf("total_docs = %d, want 2", report["total_docs"])
	}
	if report["stale_docs"] == nil {
		t.Error("stale_docs should be present")
	}
	if report["circular_deps"] == nil {
		t.Error("circular_deps should be present")
	}
	if report["queue_stats"] == nil {
		t.Error("queue_stats should be present")
	}
}

func TestFindCycles(t *testing.T) {
	tests := []struct {
		name    string
		graph   map[string][]string
		wantLen int
	}{
		{
			name:    "no cycles",
			graph:   map[string][]string{"A": {"B"}, "B": {"C"}, "C": {}},
			wantLen: 0,
		},
		{
			name:    "simple cycle A->B->A",
			graph:   map[string][]string{"A": {"B"}, "B": {"A"}},
			wantLen: 1,
		},
		{
			name:    "self-loop",
			graph:   map[string][]string{"A": {"A"}},
			wantLen: 1,
		},
		{
			name:    "three-node cycle",
			graph:   map[string][]string{"A": {"B"}, "B": {"C"}, "C": {"A"}},
			wantLen: 1,
		},
		{
			name:    "empty graph",
			graph:   map[string][]string{},
			wantLen: 0,
		},
		{
			name:    "multiple cycles",
			graph:   map[string][]string{"A": {"B"}, "B": {"A"}, "C": {"D"}, "D": {"C"}},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycles := findCycles(tt.graph)
			if len(cycles) != tt.wantLen {
				t.Errorf("findCycles() returned %d cycles, want %d", len(cycles), tt.wantLen)
			}
		})
	}
}
