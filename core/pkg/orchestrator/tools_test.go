package orchestrator

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/workspace"
	"github.com/oneliang/aura/shared/pkg/config"
	tools "github.com/oneliang/aura/tools/pkg"
)

// TestSpawnAgentTool tests the SpawnAgentTool.
func TestSpawnAgentTool(t *testing.T) {
	store, tmpDir, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	// Create workspace isolator
	ws, err := workspace.NewIsolator(filepath.Join(tmpDir, "agents"))
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	orch := &Orchestrator{
		config: &config.OrchestratorConfig{
			MaxSubAgents: 10,
		},
		parentCfg:   &config.Config{},
		workspace:   ws,
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewSpawnAgentTool(orch)

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "spawn_agent" {
			t.Errorf("Name() = %q, want 'spawn_agent'", tool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		if desc == "" {
			t.Error("Description() returned empty string")
		}
	})

	t.Run("Execute missing agent_id", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when agent_id is missing")
		}
	})

	t.Run("Execute empty agent_id", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"agent_id": "",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when agent_id is empty")
		}
	})

	t.Run("Execute invalid agent_id type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"agent_id": 123,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when agent_id is not a string")
		}
	})

	t.Run("Execute success", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"agent_id": "test-agent",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})
}

// TestCreateDocTool tests the CreateDocTool.
func TestCreateDocTool(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewCreateDocTool(orch)

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "create_collaboration_doc" {
			t.Errorf("Name() = %q, want 'create_collaboration_doc'", tool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		if desc == "" {
			t.Error("Description() returned empty string")
		}
	})

	t.Run("Execute missing type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when type is missing")
		}
	})

	t.Run("Execute missing title", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"type": "task",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when title is missing")
		}
	})

	t.Run("Execute missing body", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"type":  "task",
			"title": "Test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when body is missing")
		}
	})

	t.Run("Execute invalid type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"type":  "invalid",
			"title": "Test",
			"body":  "Test body",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status for invalid type")
		}
	})

	t.Run("Execute success with all fields", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"type":         "task",
			"to":           "agent-1",
			"priority":     "urgent",
			"title":        "Test Task",
			"body":         "Test body content",
			"dependencies": []any{"doc-1", "doc-2"},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})

	t.Run("Execute with defaults", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"type":  "info",
			"title": "Info Doc",
			"body":  "Info content",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})
}

// TestProcessDocTool tests the ProcessDocTool.
func TestProcessDocTool(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewProcessDocTool(orch, "test-agent")

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "process_collaboration_doc" {
			t.Errorf("Name() = %q, want 'process_collaboration_doc'", tool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		if desc == "" {
			t.Error("Description() returned empty string")
		}
	})

	t.Run("Execute missing doc_id", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when doc_id is missing")
		}
	})

	t.Run("Execute missing action", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status when action is missing")
		}
	})

	t.Run("Execute invalid action", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc",
			"action": "invalid",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status for invalid action")
		}
	})

	t.Run("Execute non-existent doc", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "non-existent",
			"action": "start",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != tools.ToolStatusError {
			t.Error("Execute() should return error status for non-existent doc")
		}
	})

	t.Run("Execute action start", func(t *testing.T) {
		// First create a doc
		doc := &CollaboDoc{
			ID:        "test-doc-start",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "test-agent",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "Test",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc)

		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc-start",
			"action": "start",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})

	t.Run("Execute action complete", func(t *testing.T) {
		// First create a doc
		doc := &CollaboDoc{
			ID:        "test-doc-complete",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "test-agent",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "Test",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc)

		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc-complete",
			"action": "complete",
			"note":   "Task completed successfully",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})

	t.Run("Execute action reject", func(t *testing.T) {
		// First create a doc
		doc := &CollaboDoc{
			ID:        "test-doc-reject",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "test-agent",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "Test",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc)

		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc-reject",
			"action": "reject",
			"note":   "Cannot complete - missing requirements",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})

	t.Run("Execute action reject without note", func(t *testing.T) {
		// First create a doc
		doc := &CollaboDoc{
			ID:        "test-doc-reject-nonote",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "test-agent",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "Test",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc)

		result, err := tool.Execute(context.Background(), map[string]any{
			"doc_id": "test-doc-reject-nonote",
			"action": "reject",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})
}

// TestQueryQueueTool tests the QueryQueueTool.
func TestQueryQueueTool(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewQueryQueueTool(orch)

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "query_work_queue" {
			t.Errorf("Name() = %q, want 'query_work_queue'", tool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		if desc == "" {
			t.Error("Description() returned empty string")
		}
	})

	t.Run("Execute empty queue", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content != "No documents found" {
			t.Errorf("Execute() = %q, want 'No documents found'", result.Content)
		}
	})

	t.Run("Execute with status filter", func(t *testing.T) {
		// First create some docs
		doc1 := &CollaboDoc{
			ID:        "doc-pending",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "agent-1",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "Pending Doc",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		doc2 := &CollaboDoc{
			ID:        "doc-completed",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "agent-1",
			Priority:  PriorityNormal,
			Status:    DocStatusCompleted,
			Title:     "Completed Doc",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc1)
		_, _ = orch.CreateDoc(doc2)

		result, err := tool.Execute(context.Background(), map[string]any{
			"status": "pending",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if result == nil || result.Content == "" {
			t.Error("Execute() returned empty result")
		}
	})

	t.Run("ExecuteJSON empty queue", func(t *testing.T) {
		result, err := tool.ExecuteJSON(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("ExecuteJSON() error = %v", err)
		}
		if result == "" {
			t.Error("ExecuteJSON() returned empty result")
		}
	})

	t.Run("ExecuteJSON with agent_id", func(t *testing.T) {
		// Register agent and add some docs
		orch.coordinator.RegisterAgent("agent-json")

		doc := &CollaboDoc{
			ID:        "doc-json",
			Type:      DocTypeTaskAssign,
			From:      "orchestrator",
			To:        "agent-json",
			Priority:  PriorityNormal,
			Status:    DocStatusPending,
			Title:     "JSON Test",
			Body:      "Test body",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, _ = orch.CreateDoc(doc)

		result, err := tool.ExecuteJSON(context.Background(), map[string]any{
			"agent_id": "agent-json",
		})
		if err != nil {
			t.Fatalf("ExecuteJSON() error = %v", err)
		}
		if result == "" {
			t.Error("ExecuteJSON() returned empty result")
		}
	})

	t.Run("ExecuteJSON with status filter", func(t *testing.T) {
		result, err := tool.ExecuteJSON(context.Background(), map[string]any{
			"status": "completed",
		})
		if err != nil {
			t.Fatalf("ExecuteJSON() error = %v", err)
		}
		if result == "" {
			t.Error("ExecuteJSON() returned empty result")
		}
	})
}

// TestRegisterToolsForAgent tests RegisterToolsForAgent function.
func TestRegisterToolsForAgent(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	// Create a mock agent that implements AddTool
	mockAgent := &mockAgentWithTools{
		toolsAdded: 0,
	}

	RegisterToolsForAgent(mockAgent, orch)

	if mockAgent.toolsAdded != 4 {
		t.Errorf("RegisterToolsForAgent() added %d tools, want 4", mockAgent.toolsAdded)
	}
}

// TestRegisterToolsForAgent_invalidAgent tests RegisterToolsForAgent with invalid agent.
func TestRegisterToolsForAgent_invalidAgent(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
	}

	// Agent without AddTool method - should not panic
	RegisterToolsForAgent("not an agent", orch)
	// Should silently return without error
}

// mockAgentWithTools is a mock agent for testing.
type mockAgentWithTools struct {
	toolsAdded int
}

func (m *mockAgentWithTools) AddTool(tool interface{}) {
	m.toolsAdded++
}

// TestQueryQueueResult_JSON tests QueryQueueResult JSON marshaling.
func TestQueryQueueResult_JSON(t *testing.T) {
	_ = QueryQueueResult{
		Count: 2,
		Docs: []SimpleDocInfo{
			{ID: "doc-1", Title: "Doc 1", Type: "task", Status: "pending", Priority: "normal"},
			{ID: "doc-2", Title: "Doc 2", Type: "request", Status: "completed", Priority: "urgent"},
		},
	}

	// The result should be marshalable to JSON
	// This is tested indirectly through ExecuteJSON
}

// TestQueryQueueTool_Execute_MoreStatusFilters tests QueryQueueTool with more status filters.
func TestQueryQueueTool_Execute_MoreStatusFilters(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewQueryQueueTool(orch)

	// Create docs with different statuses
	docs := []*CollaboDoc{
		{ID: "doc-pending", Type: DocTypeTaskAssign, From: "orchestrator", To: "agent-1", Priority: PriorityNormal, Status: DocStatusPending, Title: "Pending", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "doc-in-progress", Type: DocTypeTaskAssign, From: "orchestrator", To: "agent-1", Priority: PriorityNormal, Status: DocStatusInProgress, Title: "In Progress", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "doc-completed", Type: DocTypeTaskAssign, From: "orchestrator", To: "agent-1", Priority: PriorityNormal, Status: DocStatusCompleted, Title: "Completed", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "doc-rejected", Type: DocTypeTaskAssign, From: "orchestrator", To: "agent-1", Priority: PriorityNormal, Status: DocStatusRejected, Title: "Rejected", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "doc-blocked", Type: DocTypeTaskAssign, From: "orchestrator", To: "agent-1", Priority: PriorityNormal, Status: DocStatusBlocked, Title: "Blocked", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, doc := range docs {
		_, _ = orch.CreateDoc(doc)
	}

	// Test each status filter
	statuses := []string{"pending", "in_progress", "completed", "rejected", "blocked"}
	for _, status := range statuses {
		result, err := tool.Execute(context.Background(), map[string]any{
			"status": status,
		})
		if err != nil {
			t.Errorf("Execute() with status=%s error = %v", status, err)
		}
		if result == nil || result.Content == "" {
			t.Errorf("Execute() with status=%s returned empty result", status)
		}
	}
}

// TestQueryQueueTool_ExecuteJSON_WithInvalidStatus tests ExecuteJSON with invalid status.
func TestQueryQueueTool_ExecuteJSON_WithInvalidStatus(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewQueryQueueTool(orch)

	// Invalid status should return all docs
	result, err := tool.ExecuteJSON(context.Background(), map[string]any{
		"status": "invalid_status",
	})
	if err != nil {
		t.Fatalf("ExecuteJSON() with invalid status error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteJSON() with invalid status returned empty result")
	}
}

// TestQueryQueueTool_Execute_AgentError tests Execute with agent that returns error.
func TestQueryQueueTool_Execute_AgentError(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	registry := NewTaskRegistry()
	coordinator := NewDocCoordinator(store, registry, 100*time.Millisecond, nil)

	orch := &Orchestrator{
		docStore:    store,
		registry:    registry,
		coordinator: coordinator,
		subAgents:   make(map[string]*SubAgent),
	}

	tool := NewQueryQueueTool(orch)

	// Query with non-existent agent should return error
	result, err := tool.Execute(context.Background(), map[string]any{
		"agent_id": "non-existent-agent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Error("Execute() with non-existent agent should return error status")
	}
}
