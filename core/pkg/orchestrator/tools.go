package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tools "github.com/oneliang/aura/tools/pkg"
)

// SpawnAgentTool is a tool for spawning sub-agents.
type SpawnAgentTool struct {
	orchestrator *Orchestrator
}

// NewSpawnAgentTool creates a new spawn agent tool.
func NewSpawnAgentTool(orch *Orchestrator) *SpawnAgentTool {
	return &SpawnAgentTool{orchestrator: orch}
}

// Name returns the tool name.
func (t *SpawnAgentTool) Name() string {
	return "spawn_agent"
}

// Description returns the tool description.
func (t *SpawnAgentTool) Description() string {
	return "Spawn a new sub-agent with optional LLM configuration. Parameters: agent_id (string, required)"
}

// Execute spawns a new sub-agent.
func (t *SpawnAgentTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	agentID, ok := params["agent_id"].(string)
	if !ok || agentID == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "agent_id is required and must be a string"}, nil
	}

	agent, err := t.orchestrator.SpawnAgent(ctx, agentID, nil)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to spawn agent: %v", err)}, nil
	}

	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Agent %s spawned successfully (workspace: %s)", agent.ID, agent.Workspace().Dir)}, nil
}

// CreateDocTool is a tool for creating collaboration documents.
type CreateDocTool struct {
	orchestrator *Orchestrator
}

// NewCreateDocTool creates a new create collaboration document tool.
func NewCreateDocTool(orch *Orchestrator) *CreateDocTool {
	return &CreateDocTool{orchestrator: orch}
}

// Name returns the tool name.
func (t *CreateDocTool) Name() string {
	return "create_collaboration_doc"
}

// Description returns the tool description.
func (t *CreateDocTool) Description() string {
	return "Create a collaboration document for inter-agent communication. Parameters: type (task|request|handoff|review|info), to (agent_id or 'any'), priority (urgent|normal|low), title (string), body (markdown), dependencies (optional string array of doc IDs)"
}

// Execute creates a new collaboration document.
func (t *CreateDocTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Extract and validate parameters
	docType, ok := params["type"].(string)
	if !ok || docType == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "type is required (task|request|handoff|review|info)"}, nil
	}

	to, _ := params["to"].(string) // Optional, defaults to "any"
	if to == "" {
		to = "any"
	}

	priority, ok := params["priority"].(string)
	if !ok || priority == "" {
		priority = "normal"
	}

	title, ok := params["title"].(string)
	if !ok || title == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "title is required"}, nil
	}

	body, ok := params["body"].(string)
	if !ok || body == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "body is required"}, nil
	}

	deps, _ := params["dependencies"].([]any) // Optional
	var dependencies []string
	for _, d := range deps {
		if s, ok := d.(string); ok {
			dependencies = append(dependencies, s)
		}
	}

	// Map string type to DocType
	var docDocType DocType
	switch docType {
	case "task":
		docDocType = DocTypeTaskAssign
	case "request":
		docDocType = DocTypeCollabRequest
	case "handoff":
		docDocType = DocTypeHandoff
	case "review":
		docDocType = DocTypeReviewRequest
	case "info":
		docDocType = DocTypeInfoShare
	default:
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid type: %s (must be task|request|handoff|review|info)", docType)}, nil
	}

	// Map string priority to Priority
	var docPriority Priority
	switch priority {
	case "urgent":
		docPriority = PriorityUrgent
	case "low":
		docPriority = PriorityLow
	default:
		docPriority = PriorityNormal
	}

	// Create document
	doc := &CollaboDoc{
		Type:         docDocType,
		From:         "orchestrator",
		To:           to,
		Priority:     docPriority,
		Status:       DocStatusPending,
		Title:        title,
		Dependencies: dependencies,
		Body:         body,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	docID, err := t.orchestrator.CreateDoc(doc)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to create document: %v", err)}, nil
	}

	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Document created: %s (%s)", docID, title)}, nil
}

// ProcessDocTool is a tool for processing collaboration documents.
type ProcessDocTool struct {
	orchestrator *Orchestrator
	agentID      string // The agent using this tool
}

// NewProcessDocTool creates a new process collaboration document tool.
func NewProcessDocTool(orch *Orchestrator, agentID string) *ProcessDocTool {
	return &ProcessDocTool{
		orchestrator: orch,
		agentID:      agentID,
	}
}

// Name returns the tool name.
func (t *ProcessDocTool) Name() string {
	return "process_collaboration_doc"
}

// Description returns the tool description.
func (t *ProcessDocTool) Description() string {
	return "Update the status of a collaboration document. Parameters: doc_id (string, required), action (start|complete|reject, required), note (string, optional - result or rejection reason)"
}

// Execute processes a collaboration document.
func (t *ProcessDocTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	docID, ok := params["doc_id"].(string)
	if !ok || docID == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "doc_id is required"}, nil
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "action is required (start|complete|reject)"}, nil
	}

	note, _ := params["note"].(string) // Optional

	switch action {
	case "start":
		err := t.orchestrator.UpdateDocStatus(docID, t.agentID, DocStatusInProgress, "")
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to start document: %v", err)}, nil
		}
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Document %s marked as in_progress", docID)}, nil

	case "complete":
		err := t.orchestrator.UpdateDocStatus(docID, t.agentID, DocStatusCompleted, note)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to complete document: %v", err)}, nil
		}
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Document %s completed: %s", docID, note)}, nil

	case "reject":
		if note == "" {
			note = "No reason provided"
		}
		err := t.orchestrator.UpdateDocStatus(docID, t.agentID, DocStatusRejected, note)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to reject document: %v", err)}, nil
		}
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Document %s rejected: %s", docID, note)}, nil

	default:
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid action: %s (must be start|complete|reject)", action)}, nil
	}
}

// QueryQueueTool is a tool for querying work queues.
type QueryQueueTool struct {
	orchestrator *Orchestrator
}

// NewQueryQueueTool creates a new query work queue tool.
func NewQueryQueueTool(orch *Orchestrator) *QueryQueueTool {
	return &QueryQueueTool{orchestrator: orch}
}

// Name returns the tool name.
func (t *QueryQueueTool) Name() string {
	return "query_work_queue"
}

// Description returns the tool description.
func (t *QueryQueueTool) Description() string {
	return "Query the work queue for pending documents. Parameters: agent_id (string, optional - defaults to caller's queue), status (string, optional - pending|in_progress|completed|rejected)"
}

// Execute queries the work queue.
func (t *QueryQueueTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	agentID, _ := params["agent_id"].(string) // Optional
	status, _ := params["status"].(string)    // Optional

	var docs []*CollaboDoc
	var err error

	if agentID != "" {
		// Get pending docs for specific agent
		docs, err = t.orchestrator.GetPendingDocs(agentID)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to get pending docs: %v", err)}, nil
		}
	} else {
		// List all docs with optional status filter
		var statusFilter []DocStatus
		if status != "" {
			switch status {
			case "pending":
				statusFilter = []DocStatus{DocStatusPending}
			case "in_progress":
				statusFilter = []DocStatus{DocStatusInProgress}
			case "completed":
				statusFilter = []DocStatus{DocStatusCompleted}
			case "rejected":
				statusFilter = []DocStatus{DocStatusRejected}
			case "blocked":
				statusFilter = []DocStatus{DocStatusBlocked}
			}
		}

		if len(statusFilter) > 0 {
			docs, err = t.orchestrator.ListDocs(statusFilter...)
		} else {
			docs, err = t.orchestrator.ListDocs()
		}
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to list docs: %v", err)}, nil
		}
	}

	// Format output
	if len(docs) == 0 {
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "No documents found"}, nil
	}

	// Build formatted output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d document(s):\n\n", len(docs)))

	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, doc.Priority, doc.Title))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", doc.ID))
		sb.WriteString(fmt.Sprintf("   Type: %s\n", doc.Type))
		sb.WriteString(fmt.Sprintf("   Status: %s\n", doc.Status))
		sb.WriteString(fmt.Sprintf("   From: %s -> To: %s\n", doc.From, doc.To))
		if len(doc.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("   Dependencies: %v\n", doc.Dependencies))
		}
		sb.WriteString("\n")
	}

	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: sb.String()}, nil
}

// QueryQueueResult represents a structured query result for JSON output.
type QueryQueueResult struct {
	Count      int             `json:"count"`
	Docs       []SimpleDocInfo `json:"docs"`
	QueueStats map[string]struct {
		Urgent int `json:"urgent"`
		Normal int `json:"normal"`
		Low    int `json:"low"`
	} `json:"queue_stats,omitempty"`
}

// SimpleDocInfo is a simplified document info for JSON output.
type SimpleDocInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	From     string `json:"from"`
	To       string `json:"to,omitempty"`
}

// ExecuteJSON executes the query and returns JSON output.
func (t *QueryQueueTool) ExecuteJSON(ctx context.Context, params map[string]any) (string, error) {
	agentID, _ := params["agent_id"].(string)
	status, _ := params["status"].(string)

	var docs []*CollaboDoc
	var err error

	if agentID != "" {
		docs, err = t.orchestrator.GetPendingDocs(agentID)
	} else if status != "" {
		var statusFilter []DocStatus
		switch status {
		case "pending":
			statusFilter = []DocStatus{DocStatusPending}
		case "in_progress":
			statusFilter = []DocStatus{DocStatusInProgress}
		case "completed":
			statusFilter = []DocStatus{DocStatusCompleted}
		case "rejected":
			statusFilter = []DocStatus{DocStatusRejected}
		}
		docs, err = t.orchestrator.ListDocs(statusFilter...)
	} else {
		docs, err = t.orchestrator.ListDocs()
	}

	if err != nil {
		return "", fmt.Errorf("failed to query docs: %w", err)
	}

	// Build structured result
	simpleDocs := make([]SimpleDocInfo, len(docs))
	for i, doc := range docs {
		simpleDocs[i] = SimpleDocInfo{
			ID:       doc.ID,
			Title:    doc.Title,
			Type:     string(doc.Type),
			Status:   string(doc.Status),
			Priority: string(doc.Priority),
			From:     doc.From,
			To:       doc.To,
		}
	}

	// Convert queue stats to type with JSON tags
	queueStats := t.orchestrator.GetCoordinator().GetQueueStats()
	statsWithTags := make(map[string]struct {
		Urgent int `json:"urgent"`
		Normal int `json:"normal"`
		Low    int `json:"low"`
	}, len(queueStats))
	for k, v := range queueStats {
		statsWithTags[k] = struct {
			Urgent int `json:"urgent"`
			Normal int `json:"normal"`
			Low    int `json:"low"`
		}{Urgent: v.Urgent, Normal: v.Normal, Low: v.Low}
	}

	result := QueryQueueResult{
		Count:      len(docs),
		Docs:       simpleDocs,
		QueueStats: statsWithTags,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// RegisterToolsForAgent registers all orchestrator tools to an agent.
// This is a helper function to avoid circular dependency with factory package.
func RegisterToolsForAgent(ag interface{}, orch *Orchestrator) {
	// Use reflection-free approach with type assertion
	type AgentAddTool interface {
		AddTool(tool interface{})
	}

	a, ok := ag.(AgentAddTool)
	if !ok {
		return
	}

	a.AddTool(NewSpawnAgentTool(orch))
	a.AddTool(NewCreateDocTool(orch))
	a.AddTool(NewProcessDocTool(orch, "orchestrator"))
	a.AddTool(NewQueryQueueTool(orch))
}
