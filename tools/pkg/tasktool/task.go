// Package tasktool provides a built-in tool for tracking multi-step task progress.
package tasktool

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/tasks"
	tools "github.com/oneliang/aura/tools/pkg"
)

// TaskTool lets the LLM track its progress on multi-step tasks.
// It does NOT produce external side effects — it only updates in-memory state
// and emits events to the per-request event channel.
type TaskTool struct {
	eventsCh   chan<- events.Event
	requestID  string
	taskList   *tasks.TaskList
	onSave     func() // called after state-changing operations
	hookEngine *hooks.Engine
}

// New creates a TaskTool bound to an event channel, request ID, and task list.
// onSave is called after create/update to persist state (can be nil).
func New(eventsCh chan<- events.Event, requestID string, taskList *tasks.TaskList, onSave func(), hookEngine *hooks.Engine) *TaskTool {
	return &TaskTool{
		eventsCh:   eventsCh,
		requestID:  requestID,
		taskList:   taskList,
		onSave:     onSave,
		hookEngine: hookEngine,
	}
}

func (t *TaskTool) Name() string        { return constants.ToolTask }
func (t *TaskTool) Description() string { return toolDescription }
func (t *TaskTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"create": map[string]any{
				"type":     "object",
				"required": []string{"task_id", "content", "status"},
				"properties": map[string]any{
					"task_id": map[string]any{"type": "number"},
					"content": map[string]any{"type": "string"},
					"status":  map[string]any{"type": "string"},
				},
			},
			"update": map[string]any{
				"type":     "object",
				"required": []string{"task_id", "status"},
				"properties": map[string]any{
					"task_id": map[string]any{"type": "number"},
					"status":  map[string]any{"type": "string"},
					"notes":   map[string]any{"type": "string"},
				},
			},
			"list": map[string]any{
				"type":     "object",
				"required": []string{"tasks"},
				"properties": map[string]any{
					"tasks": map[string]any{"type": "array"},
				},
			},
		},
	}
}

// SetRequest updates the per-request event channel and request ID.
// Called by Engine before each ReAct loop iteration.
// Safe: Engine processes requests sequentially (protected by processingMu),
// so there is no concurrent access to eventsCh/requestID.
func (t *TaskTool) SetRequest(eventsCh chan<- events.Event, requestID string) {
	t.eventsCh = eventsCh
	t.requestID = requestID
}

const toolDescription = `Track progress on multi-step tasks.

When to use: For complex tasks requiring 3+ steps or careful planning.
Usage: call with action parameter set to one of: create, update, list.
Examples:
- Action: {"tool": "task", "parameters": {"action": "create", "content": "Research the codebase"}}
- Action: {"tool": "task", "parameters": {"action": "update", "task_id": 1, "status": "completed", "notes": "Done"}}
- Action: {"tool": "task", "parameters": {"action": "list"}}
Parameters:
- action (string, required): "create" | "update" | "list"
- content (string, required for create): task description
- task_id (number, required for update): the numeric task ID to update
- status (string, required for update): "pending" | "in_progress" | "completed"
- notes (string, optional for update): additional context
Constraint: Only one task can be in_progress at a time. Setting a task to in_progress will pause any other in_progress task.`

func (t *TaskTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	action, _ := params["action"].(string)
	if action == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "action parameter is required (create/update/list)"}, nil
	}

	switch action {
	case "create":
		content, ok := params["content"].(string)
		if !ok || content == "" {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: "content parameter is required for create"}, nil
		}
		task := t.taskList.Create(content)
		t.eventsCh <- events.NewEventWithExtra(
			events.EventTypeTaskCreate,
			content,
			map[string]any{"task_id": task.ID},
			t.requestID,
		)
		t.hookEngine.Fire(ctx, hooks.EventTaskCreated, map[string]any{
			"task_id": task.ID,
			"content": content,
			"status":  string(tasks.TaskStatusPending),
		})
		if t.onSave != nil {
			t.onSave()
		}
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: fmt.Sprintf("Task created: [%d] %s", task.ID, content),
			Data: map[string]any{
				"action":  "create",
				"task_id": task.ID,
				"content": content,
				"status":  string(tasks.TaskStatusPending),
			},
		}, nil

	case "update":
		rawID, ok := params["task_id"].(float64)
		if !ok {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: "task_id parameter is required for update"}, nil
		}
		statusStr, ok := params["status"].(string)
		if !ok {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: "status parameter is required for update"}, nil
		}
		notes, _ := params["notes"].(string)
		taskID := int(rawID)

		// Enforce: max 1 in_progress
		if statusStr == string(tasks.TaskStatusInProgress) {
			for _, existing := range t.taskList.List() {
				if existing.Status == tasks.TaskStatusInProgress && existing.ID != taskID {
					if _, err := t.taskList.Update(existing.ID, tasks.TaskStatusPending, ""); err != nil {
						fmt.Printf("Warning: failed to demote task %d from in_progress: %v\n", existing.ID, err)
					}
				}
			}
		}

		status := tasks.TaskStatus(statusStr)
		updated, err := t.taskList.Update(taskID, status, notes)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
		}
		t.eventsCh <- events.NewEventWithExtra(
			events.EventTypeTaskUpdate,
			updated.Content,
			map[string]any{"task_id": updated.ID, "status": string(updated.Status), "notes": updated.Notes},
			t.requestID,
		)
		if t.onSave != nil {
			t.onSave()
		}
		if status == tasks.TaskStatusCompleted {
			t.hookEngine.Fire(ctx, hooks.EventTaskCompleted, map[string]any{
				"task_id": updated.ID,
				"content": updated.Content,
				"status":  string(updated.Status),
				"notes":   updated.Notes,
			})
		}
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: fmt.Sprintf("Task %d updated to %s", updated.ID, updated.Status),
			Data: map[string]any{
				"action":  "update",
				"task_id": updated.ID,
				"status":  string(updated.Status),
				"notes":   updated.Notes,
			},
		}, nil

	case "list":
		allTasks := t.taskList.List()
		t.eventsCh <- events.NewEventWithExtra(
			events.EventTypeTaskList,
			"",
			map[string]any{"tasks": allTasks},
			t.requestID,
		)
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: formatTaskList(allTasks),
			Data:    map[string]any{"action": "list", "tasks": allTasks},
		}, nil

	default:
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("unknown action: %s (expected create/update/list)", action)}, nil
	}
}

func formatTaskList(ts []tasks.Task) string {
	if len(ts) == 0 {
		return "No tasks tracked."
	}
	var b strings.Builder
	b.WriteString("Current tasks:\n")
	for _, task := range ts {
		b.WriteString(fmt.Sprintf("  %s %d. %s", tasks.StatusIcon(task.Status), task.ID, task.Content))
		if task.Notes != "" {
			b.WriteString(fmt.Sprintf(" (%s)", task.Notes))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
