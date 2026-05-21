package orchestrator

import (
	"testing"
	"time"
)

func TestTaskRegistry_RegisterGet(t *testing.T) {
	r := NewTaskRegistry()

	doc := &CollaboDoc{
		ID:     "doc-1",
		Title:  "Test",
		Status: DocStatusPending,
	}

	r.Register(doc)

	got, exists := r.Get("doc-1")
	if !exists {
		t.Fatal("Get() returned exists=false, want true")
	}
	if got.ID != doc.ID {
		t.Errorf("Get().ID = %s, want %s", got.ID, doc.ID)
	}

	// Test non-existent
	_, exists = r.Get("doc-999")
	if exists {
		t.Error("Get(non-existent) returned exists=true, want false")
	}
}

func TestTaskRegistry_Unregister(t *testing.T) {
	r := NewTaskRegistry()

	r.Register(&CollaboDoc{ID: "doc-1"})
	r.Unregister("doc-1")

	_, exists := r.Get("doc-1")
	if exists {
		t.Error("Get() after Unregister returned exists=true, want false")
	}
}

func TestTaskRegistry_ListByStatus(t *testing.T) {
	r := NewTaskRegistry()

	r.Register(&CollaboDoc{ID: "pending-1", Status: DocStatusPending})
	r.Register(&CollaboDoc{ID: "pending-2", Status: DocStatusPending})
	r.Register(&CollaboDoc{ID: "completed-1", Status: DocStatusCompleted})
	r.Register(&CollaboDoc{ID: "in_progress-1", Status: DocStatusInProgress})

	pending := r.ListByStatus(DocStatusPending)
	if len(pending) != 2 {
		t.Errorf("ListByStatus(pending) len = %d, want 2", len(pending))
	}

	completed := r.ListByStatus(DocStatusCompleted)
	if len(completed) != 1 {
		t.Errorf("ListByStatus(completed) len = %d, want 1", len(completed))
	}
}

func TestTaskRegistry_GetCompletedIDs(t *testing.T) {
	r := NewTaskRegistry()

	r.Register(&CollaboDoc{ID: "doc-1", Status: DocStatusCompleted})
	r.Register(&CollaboDoc{ID: "doc-2", Status: DocStatusCompleted})
	r.Register(&CollaboDoc{ID: "doc-3", Status: DocStatusPending})

	completed := r.GetCompletedIDs()

	if !completed["doc-1"] {
		t.Error("doc-1 not in completed map")
	}
	if !completed["doc-2"] {
		t.Error("doc-2 not in completed map")
	}
	if completed["doc-3"] {
		t.Error("doc-3 should not be in completed map")
	}
}

func TestTaskRegistry_DepsReady(t *testing.T) {
	r := NewTaskRegistry()

	docWithDeps := &CollaboDoc{
		ID:           "doc-1",
		Dependencies: []string{"doc-a", "doc-b"},
	}

	completedIDs := map[string]bool{
		"doc-a": true,
		"doc-b": true,
		"doc-c": true,
	}

	if !r.DepsReady(docWithDeps, completedIDs) {
		t.Error("DepsReady() returned false, want true (all deps satisfied)")
	}

	// Test with missing dependency
	incompleteIDs := map[string]bool{"doc-a": true}
	if r.DepsReady(docWithDeps, incompleteIDs) {
		t.Error("DepsReady() returned true, want false (missing deps)")
	}
}

func TestTaskRegistry_Count(t *testing.T) {
	r := NewTaskRegistry()

	if r.Count() != 0 {
		t.Errorf("Count() = %d, want 0", r.Count())
	}

	r.Register(&CollaboDoc{ID: "doc-1"})
	r.Register(&CollaboDoc{ID: "doc-2"})
	r.Register(&CollaboDoc{ID: "doc-3"})

	if r.Count() != 3 {
		t.Errorf("Count() = %d, want 3", r.Count())
	}

	r.Unregister("doc-2")
	if r.Count() != 2 {
		t.Errorf("Count() after unregister = %d, want 2", r.Count())
	}
}

func TestTaskRegistry_Clear(t *testing.T) {
	r := NewTaskRegistry()

	r.Register(&CollaboDoc{ID: "doc-1"})
	r.Register(&CollaboDoc{ID: "doc-2"})

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", r.Count())
	}
}

func TestTaskRegistry_ListByAgent(t *testing.T) {
	r := NewTaskRegistry()

	r.Register(&CollaboDoc{ID: "agent1-doc1", To: "agent-1"})
	r.Register(&CollaboDoc{ID: "agent1-doc2", To: "agent-1"})
	r.Register(&CollaboDoc{ID: "agent2-doc1", To: "agent-2"})
	r.Register(&CollaboDoc{ID: "any-doc", To: "any"})
	r.Register(&CollaboDoc{ID: "empty-to", To: ""})

	docs := r.ListByAgent("agent-1")

	if len(docs) != 4 {
		t.Errorf("ListByAgent(agent-1) len = %d, want 4 (includes any and empty)", len(docs))
	}
}

func TestTaskRegistry_GetStaleDocs(t *testing.T) {
	r := NewTaskRegistry()

	now := time.Now()
	threshold := 5 * time.Minute

	// Recent doc (not stale)
	r.Register(&CollaboDoc{
		ID:        "recent-1",
		Status:    DocStatusPending,
		UpdatedAt: now.Add(-1 * time.Minute),
	})

	// Old pending doc (stale)
	r.Register(&CollaboDoc{
		ID:        "stale-1",
		Status:    DocStatusPending,
		UpdatedAt: now.Add(-10 * time.Minute),
	})

	// Old completed doc (should NOT be returned)
	r.Register(&CollaboDoc{
		ID:        "old-completed",
		Status:    DocStatusCompleted,
		UpdatedAt: now.Add(-10 * time.Minute),
	})

	stale := r.GetStaleDocs(threshold)

	if len(stale) != 1 {
		t.Errorf("GetStaleDocs() len = %d, want 1", len(stale))
	}
	if len(stale) > 0 && stale[0].ID != "stale-1" {
		t.Errorf("Stale doc ID = %s, want stale-1", stale[0].ID)
	}
}

// TestTaskRegistry_GetPendingByAgent tests GetPendingByAgent function.
func TestTaskRegistry_GetPendingByAgent(t *testing.T) {
	r := NewTaskRegistry()

	// Register docs for different agents and statuses
	r.Register(&CollaboDoc{ID: "agent1-pending", To: "agent-1", Status: DocStatusPending})
	r.Register(&CollaboDoc{ID: "agent1-completed", To: "agent-1", Status: DocStatusCompleted})
	r.Register(&CollaboDoc{ID: "agent2-pending", To: "agent-2", Status: DocStatusPending})
	r.Register(&CollaboDoc{ID: "any-pending", To: "", Status: DocStatusPending})

	// Get pending docs for agent-1
	pending := r.GetPendingByAgent("agent-1")
	if len(pending) != 2 {
		t.Errorf("GetPendingByAgent(agent-1) len = %d, want 2 (includes any)", len(pending))
	}

	// Get pending docs for agent-2
	pending2 := r.GetPendingByAgent("agent-2")
	if len(pending2) != 2 {
		t.Errorf("GetPendingByAgent(agent-2) len = %d, want 2 (includes any)", len(pending2))
	}

	// Get pending docs for non-existent agent (should only get 'any' docs)
	pendingNone := r.GetPendingByAgent("non-existent")
	if len(pendingNone) != 1 {
		t.Errorf("GetPendingByAgent(non-existent) len = %d, want 1 (only 'any' docs)", len(pendingNone))
	}
}

// TestTaskRegistry_SyncFromStore tests SyncFromStore function.
func TestTaskRegistry_SyncFromStore(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	r := NewTaskRegistry()

	// Create docs in store
	docs := []*CollaboDoc{
		{ID: "sync-1", Type: DocTypeTaskAssign, From: "test", To: "agent-1", Priority: PriorityNormal, Status: DocStatusPending, Title: "Sync 1", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "sync-2", Type: DocTypeTaskAssign, From: "test", To: "agent-2", Priority: PriorityNormal, Status: DocStatusInProgress, Title: "Sync 2", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "sync-3", Type: DocTypeTaskAssign, From: "test", To: "agent-1", Priority: PriorityNormal, Status: DocStatusCompleted, Title: "Sync 3", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, doc := range docs {
		_ = store.Save(doc)
	}

	// Sync registry from store
	err := r.SyncFromStore(store)
	if err != nil {
		t.Fatalf("SyncFromStore() error = %v", err)
	}

	// Verify docs are synced
	for _, doc := range docs {
		synced, exists := r.Get(doc.ID)
		if !exists {
			t.Errorf("SyncFromStore() missing doc %s", doc.ID)
		}
		if synced.Status != doc.Status {
			t.Errorf("SyncFromStore() doc %s status = %v, want %v", doc.ID, synced.Status, doc.Status)
		}
	}
}

// TestTaskRegistry_SyncFromStore_EmptyStore tests SyncFromStore with empty store.
func TestTaskRegistry_SyncFromStore_EmptyStore(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	r := NewTaskRegistry()

	// Sync from empty store should not error
	err := r.SyncFromStore(store)
	if err != nil {
		t.Fatalf("SyncFromStore() with empty store error = %v", err)
	}

	// Registry should be empty
	if r.Count() != 0 {
		t.Errorf("SyncFromStore() registry count = %d, want 0", r.Count())
	}
}

// TestTaskRegistry_DepsReady_MissingDeps tests DepsReady with missing dependencies.
func TestTaskRegistry_DepsReady_MissingDeps(t *testing.T) {
	r := NewTaskRegistry()

	docWithDeps := &CollaboDoc{
		ID:           "doc-1",
		Dependencies: []string{"doc-a", "doc-b", "doc-c"},
	}

	// Only doc-a is completed
	completedIDs := map[string]bool{
		"doc-a": true,
	}

	if r.DepsReady(docWithDeps, completedIDs) {
		t.Error("DepsReady() returned true, want false (missing doc-b and doc-c)")
	}
}

// TestTaskRegistry_DepsReady_EmptyDeps tests DepsReady with no dependencies.
func TestTaskRegistry_DepsReady_EmptyDeps(t *testing.T) {
	r := NewTaskRegistry()

	docNoDeps := &CollaboDoc{
		ID:           "doc-1",
		Dependencies: nil,
	}

	completedIDs := map[string]bool{}

	if !r.DepsReady(docNoDeps, completedIDs) {
		t.Error("DepsReady() returned false, want true (no dependencies)")
	}
}
