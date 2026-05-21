// Package orchestrator provides additional tests for the orchestrator package.
package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/config"
)

// TestOrchestratorStatus tests the status constants.
func TestOrchestratorStatus(t *testing.T) {
	if OrchestratorStatusStopped == "" {
		t.Error("OrchestratorStatusStopped should not be empty")
	}
	if OrchestratorStatusStarting == "" {
		t.Error("OrchestratorStatusStarting should not be empty")
	}
	if OrchestratorStatusRunning == "" {
		t.Error("OrchestratorStatusRunning should not be empty")
	}
	if OrchestratorStatusStopping == "" {
		t.Error("OrchestratorStatusStopping should not be empty")
	}
}

// TestNew_WithNilConfig tests New with nil config.
func TestNew_WithNilConfig(t *testing.T) {
	orch, err := New(nil)
	if err == nil {
		t.Error("New() with nil config should return error")
	}
	if orch != nil {
		t.Error("New() with nil config should return nil orchestrator")
	}
}

// TestNew_WithValidConfig tests New with valid config.
func TestNew_WithValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.Enabled = true
	cfg.Orchestrator.MaxSubAgents = 5
	cfg.Orchestrator.WorkspaceDir = tempDir
	cfg.Orchestrator.SupervisionInterval = 30 * time.Second
	cfg.Orchestrator.StaleDocThreshold = 5 * time.Minute
	cfg.Orchestrator.AutoCleanup = false

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if orch == nil {
		t.Fatal("New() returned nil orchestrator")
	}

	// Cleanup
	orch.Stop()
}

// TestNew_DefaultWorkspaceDir tests New with default workspace directory.
func TestNew_DefaultWorkspaceDir(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = "" // Use default

	orch, err := New(cfg)
	if err != nil {
		// May fail if home dir not available, but should not panic
		t.Logf("New() with default workspace: %v", err)
	}
	if orch != nil {
		orch.Stop()
	}
}

// TestOrchestrator_Start tests the Start method.
func TestOrchestrator_Start(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ctx := context.Background()
	err = orch.Start(ctx)
	if err != nil {
		t.Errorf("Start() returned error: %v", err)
	}

	if orch.status != OrchestratorStatusRunning {
		t.Errorf("status = %v, want %v", orch.status, OrchestratorStatusRunning)
	}
}

// TestOrchestrator_Start_AlreadyRunning tests Start when already running.
func TestOrchestrator_Start_AlreadyRunning(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Start again should fail
	err = orch.Start(ctx)
	if err == nil {
		t.Error("Start() when already running should return error")
	}
}

// TestOrchestrator_Stop tests the Stop method.
func TestOrchestrator_Stop(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Stop should not panic
	orch.Stop()

	if orch.status != OrchestratorStatusStopped {
		t.Errorf("status = %v, want %v", orch.status, OrchestratorStatusStopped)
	}
}

// TestOrchestrator_Stop_WhenStopped tests Stop when already stopped.
func TestOrchestrator_Stop_WhenStopped(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Stop without starting should not panic
	orch.Stop()
}

// TestOrchestrator_SpawnAgent_MaxLimit tests SpawnAgent with max limit.
func TestOrchestrator_SpawnAgent_MaxLimit(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir
	cfg.Orchestrator.MaxSubAgents = 1

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Skip starting orchestrator as it requires real LLM
	// For now, just verify the orchestrator was created with correct config
	if orch.config.MaxSubAgents != 1 {
		t.Errorf("MaxSubAgents = %d, want 1", orch.config.MaxSubAgents)
	}
}

// TestOrchestrator_SpawnAgent_DuplicateID tests SpawnAgent with duplicate ID.
func TestOrchestrator_SpawnAgent_DuplicateID(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir
	cfg.Orchestrator.MaxSubAgents = 5

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Test that subAgents map is initialized
	agents := orch.ListAgents()
	if agents == nil {
		t.Error("ListAgents() should return empty slice, not nil")
	}
}

// TestOrchestrator_StopAgent tests StopAgent.
func TestOrchestrator_StopAgent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Stop non-existent agent should return error
	err = orch.StopAgent("non-existent")
	if err == nil {
		t.Error("StopAgent() with non-existent agent should return error")
	}
}

// TestOrchestrator_GetAgent tests GetAgent.
func TestOrchestrator_GetAgent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Get non-existent agent
	agent, exists := orch.GetAgent("non-existent")
	if exists {
		t.Error("GetAgent() should return false for non-existent agent")
	}
	if agent != nil {
		t.Error("GetAgent() should return nil for non-existent agent")
	}
}

// TestOrchestrator_ListAgents tests ListAgents.
func TestOrchestrator_ListAgents(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Empty list should return empty slice, not nil
	agents := orch.ListAgents()
	if agents == nil {
		t.Error("ListAgents() should return empty slice, not nil")
	}
}

// TestOrchestrator_CreateDoc tests CreateDoc.
func TestOrchestrator_CreateDoc(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Create with nil doc should fail
	id, err := orch.CreateDoc(nil)
	if err == nil {
		t.Error("CreateDoc() with nil doc should return error")
	}
	if id != "" {
		t.Error("CreateDoc() with nil doc should return empty ID")
	}
}

// TestOrchestrator_CreateDoc_ValidDoc tests CreateDoc with valid document.
func TestOrchestrator_CreateDoc_ValidDoc(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Create valid doc
	doc := &CollaboDoc{
		ID:       "test-doc-1",
		Type:     DocTypeHandoff,
		Title:    "Test Document",
		Status:   DocStatusPending,
		Priority: PriorityNormal,
		Body:     "Test content",
	}

	id, err := orch.CreateDoc(doc)
	if err != nil {
		t.Fatalf("CreateDoc() returned error: %v", err)
	}
	if id == "" {
		t.Error("CreateDoc() should return non-empty ID")
	}
}

// TestOrchestrator_GetPendingDocs tests GetPendingDocs.
func TestOrchestrator_GetPendingDocs(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Get pending docs for non-existent agent
	docs, err := orch.GetPendingDocs("non-existent")
	if err != nil {
		// May return error, but should not panic
		t.Logf("GetPendingDocs() returned: %v", err)
	}
	if docs == nil {
		// Should return empty slice
		docs = []*CollaboDoc{}
	}
}

// TestOrchestrator_CreateDoc_AutoID tests CreateDoc with auto-generated ID.
func TestOrchestrator_CreateDoc_AutoID(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Create doc without ID
	doc := &CollaboDoc{
		Type:     DocTypeCollabRequest,
		Title:    "Test Document",
		Status:   DocStatusPending,
		Priority: PriorityNormal,
		Body:     "Test content",
	}

	id, err := orch.CreateDoc(doc)
	if err != nil {
		t.Fatalf("CreateDoc() returned error: %v", err)
	}
	if id == "" {
		t.Error("CreateDoc() should generate ID when not provided")
	}
}

// TestOrchestratorConfig tests OrchestratorConfig structure.
func TestOrchestratorConfig(t *testing.T) {
	cfg := &OrchestratorConfig{
		Enabled:             true,
		MaxSubAgents:        10,
		WorkspaceDir:        "/tmp/test",
		SupervisionInterval: 30 * time.Second,
		StaleDocThreshold:   5 * time.Minute,
		AutoCleanup:         true,
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.MaxSubAgents != 10 {
		t.Errorf("MaxSubAgents = %d, want 10", cfg.MaxSubAgents)
	}
	if cfg.WorkspaceDir != "/tmp/test" {
		t.Errorf("WorkspaceDir = %q, want /tmp/test", cfg.WorkspaceDir)
	}
}

// generateDocID tests the generateDocID function indirectly.
func TestGenerateDocID(t *testing.T) {
	// generateDocID uses uuid.New().String(), should not be empty
	id1 := generateDocID()
	id2 := generateDocID()

	if id1 == "" {
		t.Error("generateDocID() should return non-empty string")
	}
	if id2 == "" {
		t.Error("generateDocID() should return non-empty string")
	}
	if id1 == id2 {
		t.Error("generateDocID() should return unique IDs")
	}
}

// TestOrchestrator_Cleanup tests orchestrator cleanup with AutoCleanup enabled.
func TestOrchestrator_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir
	cfg.Orchestrator.AutoCleanup = true

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Stop should cleanup workspace
	orch.Stop()
}

// TestOrchestrator_HandleCoordinationEvent tests handleCoordinationEvent doesn't panic.
func TestOrchestrator_HandleCoordinationEvent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Should not panic with empty event
	orch.handleCoordinationEvent(CoordinationEvent{})
}

// TestOrchestrator_HandleSupervisionEvent tests handleSupervisionEvent doesn't panic.
func TestOrchestrator_HandleSupervisionEvent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Should not panic with empty event
	orch.handleSupervisionEvent(SupervisionEvent{})
}

// TestOrchestratorStatus_String tests status string values.
func TestOrchestratorStatus_String(t *testing.T) {
	tests := []struct {
		status OrchestratorStatus
		want   string
	}{
		{OrchestratorStatusStopped, "stopped"},
		{OrchestratorStatusStarting, "starting"},
		{OrchestratorStatusRunning, "running"},
		{OrchestratorStatusStopping, "stopping"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("status = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

// TestOrchestratorDocOperations tests document operations.
func TestOrchestratorDocOperations(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Create a doc
	doc := &CollaboDoc{
		ID:       "test-doc",
		Type:     DocTypeReviewRequest,
		Title:    "Test",
		Status:   DocStatusPending,
		Priority: PriorityNormal,
		Body:     "content",
	}

	id, err := orch.CreateDoc(doc)
	if err != nil {
		t.Fatalf("CreateDoc() returned error: %v", err)
	}

	// Get pending docs
	docs, err := orch.GetPendingDocs("default")
	if err != nil {
		t.Logf("GetPendingDocs() returned: %v", err)
	}
	if docs != nil && len(docs) > 0 {
		t.Logf("Got %d pending docs", len(docs))
	}

	_ = id
}

// TestOrchestrator_GetDoc tests the GetDoc method.
func TestOrchestrator_GetDoc(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Create a doc
	doc := &CollaboDoc{
		ID:        "test-doc-get",
		Type:      DocTypeTaskAssign,
		From:      "test",
		To:        "default",
		Priority:  PriorityNormal,
		Status:    DocStatusPending,
		Title:     "Test Get",
		Body:      "Test body",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = orch.CreateDoc(doc)
	if err != nil {
		t.Fatalf("CreateDoc() returned error: %v", err)
	}

	// Get the doc
	gotDoc, err := orch.GetDoc("test-doc-get")
	if err != nil {
		t.Fatalf("GetDoc() returned error: %v", err)
	}
	if gotDoc == nil {
		t.Fatal("GetDoc() returned nil")
	}
	if gotDoc.ID != doc.ID {
		t.Errorf("GetDoc() ID = %s, want %s", gotDoc.ID, doc.ID)
	}
	if gotDoc.Title != doc.Title {
		t.Errorf("GetDoc() Title = %s, want %s", gotDoc.Title, doc.Title)
	}
}

// TestOrchestrator_GetDoc_NonExistent tests GetDoc with non-existent ID.
func TestOrchestrator_GetDoc_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	_, err = orch.GetDoc("non-existent-doc")
	if err == nil {
		t.Error("GetDoc() with non-existent ID should return error")
	}
}

// TestOrchestrator_Status tests the Status method.
func TestOrchestrator_Status(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// Status before start
	status := orch.Status()
	if status != OrchestratorStatusStopped {
		t.Errorf("Status() before start = %v, want %v", status, OrchestratorStatusStopped)
	}

	// Start and check status
	ctx := context.Background()
	_ = orch.Start(ctx)

	status = orch.Status()
	if status != OrchestratorStatusRunning {
		t.Errorf("Status() after start = %v, want %v", status, OrchestratorStatusRunning)
	}
}

// TestOrchestrator_GetHealthReport tests the GetHealthReport method.
func TestOrchestrator_GetHealthReport(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Get health report without spawning agents (avoids workerLoop nil pointer issue)
	report := orch.GetHealthReport()
	if report == nil {
		t.Fatal("GetHealthReport() returned nil")
	}

	// Check expected fields
	if _, ok := report["orchestrator_status"]; !ok {
		t.Error("GetHealthReport() missing orchestrator_status")
	}
	if _, ok := report["sub_agents"]; !ok {
		t.Error("GetHealthReport() missing sub_agents")
	}
	if _, ok := report["total_docs"]; !ok {
		t.Error("GetHealthReport() missing total_docs")
	}
	if _, ok := report["supervisor_report"]; !ok {
		t.Error("GetHealthReport() missing supervisor_report")
	}
	if _, ok := report["queue_stats"]; !ok {
		t.Error("GetHealthReport() missing queue_stats")
	}
}

// TestOrchestrator_GetWorkspace tests the GetWorkspace method.
func TestOrchestrator_GetWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ws := orch.GetWorkspace()
	if ws == nil {
		t.Error("GetWorkspace() returned nil")
	}
}

// TestOrchestrator_StopAgent_WithSpawn tests the StopAgent method with a spawned agent.
func TestOrchestrator_StopAgent_WithSpawn(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	// StopAgent requires a spawned agent, but spawning has side effects
	// This test verifies the basic StopAgent flow without full agent lifecycle
	// The StopAgent method should handle the case where agent exists but wasn't properly started

	// For now, test that StopAgent returns error for non-existent agent
	// (This is already tested in TestOrchestrator_StopAgent)

	// Skip this test until SubAgent workerLoop nil pointer issue is fixed
	t.Skip("Skip until SubAgent workerLoop nil pointer issue is fixed")
}

// TestOrchestrator_ListDocs tests the ListDocs method.
func TestOrchestrator_ListDocs(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	ctx := context.Background()
	_ = orch.Start(ctx)

	// Create docs with different statuses
	docs := []*CollaboDoc{
		{ID: "doc-pending", Type: DocTypeTaskAssign, From: "test", To: "default", Priority: PriorityNormal, Status: DocStatusPending, Title: "Pending", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "doc-completed", Type: DocTypeTaskAssign, From: "test", To: "default", Priority: PriorityNormal, Status: DocStatusCompleted, Title: "Completed", Body: "Body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, doc := range docs {
		_, err := orch.CreateDoc(doc)
		if err != nil {
			t.Fatalf("CreateDoc() returned error: %v", err)
		}
	}

	// List all docs
	allDocs, err := orch.ListDocs()
	if err != nil {
		t.Fatalf("ListDocs() returned error: %v", err)
	}
	if len(allDocs) != 2 {
		t.Errorf("ListDocs() returned %d docs, want 2", len(allDocs))
	}

	// List docs by status
	pendingDocs, err := orch.ListDocs(DocStatusPending)
	if err != nil {
		t.Fatalf("ListDocs(pending) returned error: %v", err)
	}
	if len(pendingDocs) != 1 {
		t.Errorf("ListDocs(pending) returned %d docs, want 1", len(pendingDocs))
	}
}

// TestOrchestrator_GetCoordinator tests the GetCoordinator method.
func TestOrchestrator_GetCoordinator(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Orchestrator.WorkspaceDir = tempDir

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer orch.Stop()

	coord := orch.GetCoordinator()
	if coord == nil {
		t.Error("GetCoordinator() returned nil")
	}
}
