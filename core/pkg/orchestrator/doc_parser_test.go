package orchestrator

import (
	"testing"
	"time"
)

func TestParseDoc_Valid(t *testing.T) {
	raw := []byte(`---
id: doc-123
type: task_assignment
from: orchestrator
to: agent-1
priority: normal
status: pending
title: Test Task
created_at: 2026-01-01T00:00:00Z
updated_at: 2026-01-01T00:00:00Z
---
This is the task body.
Please complete this task.
`)

	doc, err := ParseDoc(raw)
	if err != nil {
		t.Fatalf("ParseDoc() error = %v", err)
	}

	if doc.ID != "doc-123" {
		t.Errorf("ID = %s, want doc-123", doc.ID)
	}
	if doc.Type != DocTypeTaskAssign {
		t.Errorf("Type = %s, want task_assignment", doc.Type)
	}
	if doc.From != "orchestrator" {
		t.Errorf("From = %s, want orchestrator", doc.From)
	}
	if doc.To != "agent-1" {
		t.Errorf("To = %s, want agent-1", doc.To)
	}
	if doc.Priority != PriorityNormal {
		t.Errorf("Priority = %s, want normal", doc.Priority)
	}
	if doc.Status != DocStatusPending {
		t.Errorf("Status = %s, want pending", doc.Status)
	}
	if doc.Title != "Test Task" {
		t.Errorf("Title = %s, want Test Task", doc.Title)
	}
	if doc.Body != "This is the task body.\nPlease complete this task." {
		t.Errorf("Body = %q, want task body", doc.Body)
	}
}

func TestParseDoc_MissingDelimiter(t *testing.T) {
	raw := []byte(`id: doc-123
title: Test
---
Body here`)

	_, err := ParseDoc(raw)
	if err == nil {
		t.Error("ParseDoc() error = nil, want error for missing delimiter")
	}
}

func TestParseDoc_MissingID(t *testing.T) {
	raw := []byte(`---
type: task_assignment
from: orchestrator
status: pending
title: Test
---
Body`)

	_, err := ParseDoc(raw)
	if err == nil {
		t.Error("ParseDoc() error = nil, want error for missing ID")
	}
}

func TestParseDoc_MissingType(t *testing.T) {
	raw := []byte(`---
id: doc-123
from: orchestrator
status: pending
title: Test
---
Body`)

	_, err := ParseDoc(raw)
	if err == nil {
		t.Error("ParseDoc() error = nil, want error for missing type")
	}
}

func TestParseDoc_DefaultPriority(t *testing.T) {
	raw := []byte(`---
id: doc-123
type: task_assignment
from: orchestrator
status: pending
title: Test
---
Body`)

	doc, err := ParseDoc(raw)
	if err != nil {
		t.Fatalf("ParseDoc() error = %v", err)
	}

	if doc.Priority != PriorityNormal {
		t.Errorf("Priority = %s, want normal (default)", doc.Priority)
	}
}

func TestParseDoc_WithDependencies(t *testing.T) {
	raw := []byte(`---
id: doc-123
type: task_assignment
from: orchestrator
status: pending
title: Test
dependencies:
  - doc-001
  - doc-002
---
Body`)

	doc, err := ParseDoc(raw)
	if err != nil {
		t.Fatalf("ParseDoc() error = %v", err)
	}

	if len(doc.Dependencies) != 2 {
		t.Fatalf("Dependencies len = %d, want 2", len(doc.Dependencies))
	}
	if doc.Dependencies[0] != "doc-001" {
		t.Errorf("Dependencies[0] = %s, want doc-001", doc.Dependencies[0])
	}
}

func TestRenderDoc_Roundtrip(t *testing.T) {
	original := &CollaboDoc{
		ID:           "doc-456",
		Type:         DocTypeCollabRequest,
		From:         "agent-1",
		To:           "agent-2",
		Priority:     PriorityUrgent,
		Status:       DocStatusInProgress,
		Title:        "Collaboration Request",
		Body:         "Please help with this task.",
		Dependencies: []string{"doc-001"},
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		History: []HistoryEntry{
			{AgentID: "agent-1", Action: "created", Timestamp: time.Now(), Note: "Initial creation"},
		},
	}

	// Render to bytes
	data, err := RenderDoc(original)
	if err != nil {
		t.Fatalf("RenderDoc() error = %v", err)
	}

	// Parse back
	parsed, err := ParseDoc(data)
	if err != nil {
		t.Fatalf("ParseDoc() after render error = %v", err)
	}

	// Compare fields
	if parsed.ID != original.ID {
		t.Errorf("ID = %s, want %s", parsed.ID, original.ID)
	}
	if parsed.Type != original.Type {
		t.Errorf("Type = %s, want %s", parsed.Type, original.Type)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("Priority = %s, want %s", parsed.Priority, original.Priority)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status = %s, want %s", parsed.Status, original.Status)
	}
	if parsed.Body != original.Body {
		t.Errorf("Body = %q, want %q", parsed.Body, original.Body)
	}
}

func TestCollaboDoc_IsBlocked(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []string
		completedIDs map[string]bool
		wantBlocked  bool
	}{
		{
			name:         "no dependencies",
			dependencies: nil,
			completedIDs: map[string]bool{},
			wantBlocked:  false,
		},
		{
			name:         "all dependencies satisfied",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{"doc-1": true, "doc-2": true},
			wantBlocked:  false,
		},
		{
			name:         "partial dependencies satisfied",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{"doc-1": true},
			wantBlocked:  true,
		},
		{
			name:         "no dependencies satisfied",
			dependencies: []string{"doc-1", "doc-2"},
			completedIDs: map[string]bool{},
			wantBlocked:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &CollaboDoc{
				ID:           "test-doc",
				Dependencies: tt.dependencies,
			}

			gotBlocked := doc.IsBlocked(tt.completedIDs)
			if gotBlocked != tt.wantBlocked {
				t.Errorf("IsBlocked() = %v, want %v", gotBlocked, tt.wantBlocked)
			}
		})
	}
}

func TestCollaboDoc_AddHistory(t *testing.T) {
	doc := &CollaboDoc{
		ID:    "test-doc",
		Title: "Test",
	}

	before := time.Now()
	doc.AddHistory("agent-1", "started", "Beginning work")
	after := time.Now()

	if len(doc.History) != 1 {
		t.Fatalf("History len = %d, want 1", len(doc.History))
	}

	entry := doc.History[0]
	if entry.AgentID != "agent-1" {
		t.Errorf("AgentID = %s, want agent-1", entry.AgentID)
	}
	if entry.Action != "started" {
		t.Errorf("Action = %s, want started", entry.Action)
	}
	if entry.Note != "Beginning work" {
		t.Errorf("Note = %s, want Beginning work", entry.Note)
	}
	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp out of expected range")
	}
}
