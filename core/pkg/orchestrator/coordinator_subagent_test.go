package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/workspace"
	"github.com/oneliang/aura/shared/pkg/config"
)

func TestDocCoordinator_UnregisterAgent(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Register an agent first
	queue := coordinator.RegisterAgent("agent-1")
	if queue == nil {
		t.Fatal("RegisterAgent() returned nil")
	}

	// Verify agent is registered
	if coordinator.QueueFor("agent-1") == nil {
		t.Error("Agent queue should exist after registration")
	}

	// Unregister the agent
	coordinator.UnregisterAgent("agent-1")

	// Verify agent is unregistered
	if coordinator.QueueFor("agent-1") != nil {
		t.Error("Agent queue should be nil after unregister")
	}
}

func TestDocCoordinator_QueueFor(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Non-existent agent
	queue := coordinator.QueueFor("non-existent")
	if queue != nil {
		t.Error("QueueFor() should return nil for non-existent agent")
	}

	// Existing agent
	coordinator.RegisterAgent("agent-1")
	queue = coordinator.QueueFor("agent-1")
	if queue == nil {
		t.Error("QueueFor() should return queue for registered agent")
	}
}

func TestDocCoordinator_Run(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 50*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Run in goroutine
	done := make(chan bool)
	go func() {
		coordinator.Run(ctx)
		done <- true
	}()

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Run to exit
	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		// If it doesn't exit naturally, call Stop
		coordinator.Stop()
		// Give it time to stop
		time.Sleep(100 * time.Millisecond)
	}

	if coordinator.IsRunning() {
		t.Error("Coordinator should stop after context cancellation or Stop()")
	}
}

func TestDocCoordinator_Submit(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Register an agent
	coordinator.RegisterAgent("agent-1")

	// Create a doc with all required fields
	doc := &CollaboDoc{
		ID:        "test-doc",
		Type:      DocTypeTaskAssign,
		From:      "orchestrator",
		To:        "agent-1",
		Priority:  PriorityNormal,
		Status:    DocStatusPending,
		Title:     "Test",
		Body:      "Test body",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Submit should save doc to store and register in registry
	err := coordinator.Submit(doc)
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	// Verify doc was saved
	loadedDoc, err := store.Load(doc.ID)
	if err != nil {
		t.Fatalf("Store.Load() error = %v", err)
	}
	if loadedDoc.ID != doc.ID {
		t.Errorf("Loaded doc ID = %q, want %q", loadedDoc.ID, doc.ID)
	}

	// Verify doc was registered
	registeredDoc, ok := registry.Get(doc.ID)
	if !ok {
		t.Error("Doc should be registered in registry")
	}
	if registeredDoc.ID != doc.ID {
		t.Errorf("Registered doc ID = %q, want %q", registeredDoc.ID, doc.ID)
	}
}

func TestDocCoordinator_Submit_NilDoc(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	err := coordinator.Submit(nil)
	if err == nil {
		t.Error("Submit(nil) should return error")
	}
}

func TestDocCoordinator_MarkInProgress(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Create and save a doc
	doc := &CollaboDoc{
		ID:       "test-doc",
		Type:     DocTypeTaskAssign,
		From:     "orchestrator",
		To:       "agent-1",
		Priority: PriorityNormal,
		Status:   DocStatusPending,
		Title:    "Test",
		Body:     "Test body",
	}
	coordinator.Submit(doc)

	// Mark as in progress
	err := coordinator.MarkInProgress(doc.ID, "agent-1")
	if err != nil {
		t.Fatalf("MarkInProgress() error = %v", err)
	}

	// Verify status updated
	loadedDoc, err := store.Load(doc.ID)
	if err != nil {
		t.Fatalf("Store.Load() error = %v", err)
	}
	if loadedDoc.Status != DocStatusInProgress {
		t.Errorf("Status = %q, want %q", loadedDoc.Status, DocStatusInProgress)
	}
}

func TestDocCoordinator_MarkInProgress_NonExistent(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	err := coordinator.MarkInProgress("non-existent-doc", "agent-1")
	if err == nil {
		t.Error("MarkInProgress() for non-existent doc should return error")
	}
}

func TestDocCoordinator_MarkCompleted(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Create and save a doc
	doc := &CollaboDoc{
		ID:       "test-doc",
		Type:     DocTypeTaskAssign,
		From:     "orchestrator",
		To:       "agent-1",
		Priority: PriorityNormal,
		Status:   DocStatusInProgress,
		Title:    "Test",
		Body:     "Test body",
	}
	coordinator.Submit(doc)

	// Mark as completed
	err := coordinator.MarkCompleted(doc.ID, "agent-1", "Result summary")
	if err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}

	// Verify status updated
	loadedDoc, err := store.Load(doc.ID)
	if err != nil {
		t.Fatalf("Store.Load() error = %v", err)
	}
	if loadedDoc.Status != DocStatusCompleted {
		t.Errorf("Status = %q, want %q", loadedDoc.Status, DocStatusCompleted)
	}
}

func TestDocCoordinator_MarkRejected(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Create and save a doc
	doc := &CollaboDoc{
		ID:       "test-doc",
		Type:     DocTypeTaskAssign,
		From:     "orchestrator",
		To:       "agent-1",
		Priority: PriorityNormal,
		Status:   DocStatusInProgress,
		Title:    "Test",
		Body:     "Test body",
	}
	coordinator.Submit(doc)

	// Mark as rejected
	err := coordinator.MarkRejected(doc.ID, "agent-1", "Reason for rejection")
	if err != nil {
		t.Fatalf("MarkRejected() error = %v", err)
	}

	// Verify status updated
	loadedDoc, err := store.Load(doc.ID)
	if err != nil {
		t.Fatalf("Store.Load() error = %v", err)
	}
	if loadedDoc.Status != DocStatusRejected {
		t.Errorf("Status = %q, want %q", loadedDoc.Status, DocStatusRejected)
	}
}

func TestDocCoordinator_IsRunning(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	if coordinator.IsRunning() {
		t.Error("IsRunning() should return false before Run()")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go coordinator.Run(ctx)

	// Let it start
	time.Sleep(20 * time.Millisecond)

	if !coordinator.IsRunning() {
		t.Error("IsRunning() should return true after Run()")
	}

	cancel()

	// Wait for it to stop
	time.Sleep(50 * time.Millisecond)

	if coordinator.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestDocCoordinator_Stop(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	ctx := context.Background()
	go coordinator.Run(ctx)

	// Let it start
	time.Sleep(20 * time.Millisecond)

	// Stop
	coordinator.Stop()

	// Wait for it to stop
	time.Sleep(50 * time.Millisecond)

	if coordinator.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestDocCoordinator_GetQueueStats(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()
	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Empty stats
	stats := coordinator.GetQueueStats()
	if len(stats) != 0 {
		t.Errorf("Expected empty stats, got %d entries", len(stats))
	}

	// Register agents and add work
	coordinator.RegisterAgent("agent-1")
	coordinator.RegisterAgent("agent-2")

	// Add docs to queues
	doc1 := &CollaboDoc{ID: "doc-1", Priority: PriorityNormal}
	doc2 := &CollaboDoc{ID: "doc-2", Priority: PriorityUrgent}
	coordinator.agentQueues["agent-1"].Enqueue(doc1)
	coordinator.agentQueues["agent-2"].Enqueue(doc2)

	stats = coordinator.GetQueueStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 agent stats, got %d", len(stats))
	}
}

func TestSubAgent_Creation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SubAgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SubAgentConfig{
				ID:           "agent-1",
				ParentConfig: &config.Config{},
				Workspace:    &workspace.Workspace{Dir: "/tmp/test"},
				WorkQueue:    NewWorkQueue("agent-1"),
				Coordinator:  NewDocCoordinator(nil, nil, 0, nil),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			config: &SubAgentConfig{
				ParentConfig: &config.Config{},
				Workspace:    &workspace.Workspace{Dir: "/tmp/test"},
			},
			wantErr: true,
		},
		{
			name: "missing parent config",
			config: &SubAgentConfig{
				ID:        "agent-1",
				Workspace: &workspace.Workspace{Dir: "/tmp/test"},
			},
			wantErr: true,
		},
		{
			name: "missing workspace",
			config: &SubAgentConfig{
				ID:           "agent-1",
				ParentConfig: &config.Config{},
			},
			wantErr: true,
		},
		{
			name: "missing work queue",
			config: &SubAgentConfig{
				ID:           "agent-1",
				ParentConfig: &config.Config{},
				Workspace:    &workspace.Workspace{Dir: "/tmp/test"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewSubAgent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSubAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && agent == nil {
				t.Error("NewSubAgent() returned nil agent without error")
			}
		})
	}
}

func TestSubAgent_Accessors(t *testing.T) {
	// This test verifies the accessor methods work correctly
	// Note: Full integration testing would require actual runtime setup

	cfg := &SubAgentConfig{
		ID:           "test-agent",
		ParentConfig: &config.Config{},
		Workspace:    &workspace.Workspace{Dir: "/tmp/test-workspace"},
		WorkQueue:    NewWorkQueue("test-agent"),
		Coordinator:  NewDocCoordinator(nil, nil, 0, nil),
	}

	agent, err := NewSubAgent(cfg)
	if err != nil {
		t.Fatalf("NewSubAgent() error = %v", err)
	}

	// Test ID
	if agent.ID != "test-agent" {
		t.Errorf("ID = %q, want 'test-agent'", agent.ID)
	}

	// Test Workspace
	ws := agent.Workspace()
	if ws == nil {
		t.Error("Workspace() returned nil")
	}

	// Test Status (should be idle initially)
	if agent.Status() != AgentStatusIdle {
		t.Errorf("Status = %q, want %q", agent.Status(), AgentStatusIdle)
	}

	// Test IsAvailable (should be true when idle)
	if !agent.IsAvailable() {
		t.Error("IsAvailable() should return true when idle")
	}

	// Test CurrentDoc (should be nil initially)
	if agent.CurrentDoc() != nil {
		t.Error("CurrentDoc() should return nil initially")
	}

	// Test LastActive (should be set)
	if agent.LastActive().IsZero() {
		t.Error("LastActive() should return non-zero time")
	}

	// Test Stats
	stats := agent.Stats()
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}
	if stats["id"] != "test-agent" {
		t.Errorf("Stats[id] = %v, want 'test-agent'", stats["id"])
	}
	if stats["status"] != AgentStatusIdle {
		t.Errorf("Stats[status] = %v, want %q", stats["status"], AgentStatusIdle)
	}
}

func TestSubAgent_AssignDoc(t *testing.T) {
	cfg := &SubAgentConfig{
		ID:           "test-agent",
		ParentConfig: &config.Config{},
		Workspace:    &workspace.Workspace{Dir: "/tmp/test-workspace"},
		WorkQueue:    NewWorkQueue("test-agent"),
		Coordinator:  NewDocCoordinator(nil, nil, 0, nil),
	}

	agent, err := NewSubAgent(cfg)
	if err != nil {
		t.Fatalf("NewSubAgent() error = %v", err)
	}

	// Create a doc
	doc := &CollaboDoc{
		ID:       "test-doc",
		Type:     DocTypeTaskAssign,
		Priority: PriorityNormal,
		Title:    "Test",
	}

	// Assign doc - should enqueue it
	agent.AssignDoc(doc)

	// Verify doc is in queue
	queuedDoc, ok := agent.workQueue.Dequeue()
	if !ok {
		t.Error("WorkQueue should have doc")
	}
	if queuedDoc.ID != doc.ID {
		t.Errorf("Dequeued doc ID = %q, want %q", queuedDoc.ID, doc.ID)
	}
}

func TestSubAgent_Stop(t *testing.T) {
	cfg := &SubAgentConfig{
		ID:           "test-agent",
		ParentConfig: &config.Config{},
		Workspace:    &workspace.Workspace{Dir: "/tmp/test-workspace"},
		WorkQueue:    NewWorkQueue("test-agent"),
		Coordinator:  NewDocCoordinator(nil, nil, 0, nil),
	}

	agent, err := NewSubAgent(cfg)
	if err != nil {
		t.Fatalf("NewSubAgent() error = %v", err)
	}

	// Stop should not panic
	agent.Stop()

	// Status should be Done
	if agent.Status() != AgentStatusDone {
		t.Errorf("Status after Stop = %q, want %q", agent.Status(), AgentStatusDone)
	}
}

func TestCollaboDoc_IsBlocked_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []string
		completedIDs map[string]bool
		wantBlocked  bool
	}{
		{
			name:         "empty dependencies",
			dependencies: nil,
			completedIDs: map[string]bool{},
			wantBlocked:  false,
		},
		{
			name:         "all deps completed",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{"doc-1": true, "doc-2": true},
			wantBlocked:  false,
		},
		{
			name:         "no deps completed",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{},
			wantBlocked:  true,
		},
		{
			name:         "partial deps completed",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{"doc-1": true},
			wantBlocked:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &CollaboDoc{
				ID:           "test-doc",
				Dependencies: tt.dependencies,
			}
			got := doc.IsBlocked(tt.completedIDs)
			if got != tt.wantBlocked {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.wantBlocked)
			}
		})
	}
}
