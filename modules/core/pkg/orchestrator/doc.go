// Package orchestrator provides multi-agent coordination capabilities.
package orchestrator

import "time"

// DocStatus represents the lifecycle state of a collaboration document.
type DocStatus string

const (
	// DocStatusPending document is waiting to be picked up.
	DocStatusPending DocStatus = "pending"
	// DocStatusInProgress document is being processed by an agent.
	DocStatusInProgress DocStatus = "in_progress"
	// DocStatusCompleted document processing is complete.
	DocStatusCompleted DocStatus = "completed"
	// DocStatusRejected document was rejected (with reason).
	DocStatusRejected DocStatus = "rejected"
	// DocStatusBlocked document is blocked waiting for dependencies.
	DocStatusBlocked DocStatus = "blocked"
)

// DocType distinguishes different types of collaboration documents.
type DocType string

const (
	// DocTypeCollabRequest inter-agent collaboration request.
	DocTypeCollabRequest DocType = "collaboration_request"
	// DocTypeTaskAssign task assignment from orchestrator to agent.
	DocTypeTaskAssign DocType = "task_assignment"
	// DocTypeHandoff handoff from one agent to another.
	DocTypeHandoff DocType = "handoff"
	// DocTypeReviewRequest request for review/feedback.
	DocTypeReviewRequest DocType = "review_request"
	// DocTypeInfoShare information sharing (FYI, no action needed).
	DocTypeInfoShare DocType = "info_share"
)

// Priority defines work queue ordering.
type Priority string

const (
	// PriorityUrgent high priority, processed first.
	PriorityUrgent Priority = "urgent"
	// PriorityNormal standard priority.
	PriorityNormal Priority = "normal"
	// PriorityLow low priority, processed when queue is empty.
	PriorityLow Priority = "low"
)

// HistoryEntry represents a single action in document history.
type HistoryEntry struct {
	AgentID   string    `yaml:"agent_id"`
	Action    string    `yaml:"action"` // e.g., "created", "started", "completed", "rejected"
	Timestamp time.Time `yaml:"timestamp"`
	Note      string    `yaml:"note,omitempty"`
}

// CollaboDoc represents a collaboration document with YAML front-matter.
type CollaboDoc struct {
	ID           string         `yaml:"id"`
	Type         DocType        `yaml:"type"`
	From         string         `yaml:"from"` // agent ID or "orchestrator"
	To           string         `yaml:"to,omitempty"`
	Priority     Priority       `yaml:"priority"`
	Status       DocStatus      `yaml:"status"`
	Title        string         `yaml:"title"`
	Dependencies []string       `yaml:"dependencies,omitempty"`
	CreatedAt    time.Time      `yaml:"created_at"`
	UpdatedAt    time.Time      `yaml:"updated_at"`
	Body         string         `yaml:"-"` // Markdown body after front-matter
	History      []HistoryEntry `yaml:"history,omitempty"`
}

// IsBlocked returns true if the document has unmet dependencies.
func (d *CollaboDoc) IsBlocked(completedIDs map[string]bool) bool {
	if len(d.Dependencies) == 0 {
		return false
	}
	for _, depID := range d.Dependencies {
		if !completedIDs[depID] {
			return true
		}
	}
	return false
}

// AddHistory adds a history entry to the document.
func (d *CollaboDoc) AddHistory(agentID, action, note string) {
	d.History = append(d.History, HistoryEntry{
		AgentID:   agentID,
		Action:    action,
		Timestamp: time.Now(),
		Note:      note,
	})
	d.UpdatedAt = time.Now()
}
