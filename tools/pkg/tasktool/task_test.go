package tasktool

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/tasks"
	tools "github.com/oneliang/aura/tools/pkg"
)

func TestTaskTool_Name(t *testing.T) {
	tl := tasks.NewTaskList()
	tool := New(make(chan<- events.Event, 10), "req-1", tl, nil, nil)
	if tool.Name() != "task" {
		t.Fatalf("expected name 'task', got %q", tool.Name())
	}
}

func TestTaskTool_Description(t *testing.T) {
	tl := tasks.NewTaskList()
	tool := New(make(chan<- events.Event, 10), "req-1", tl, nil, nil)
	desc := tool.Description()
	if !strings.Contains(desc, "create") || !strings.Contains(desc, "update") || !strings.Contains(desc, "list") {
		t.Fatalf("description should mention create/update/list actions, got %q", desc)
	}
}

func TestTaskTool_Execute_Create(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "create",
		"content": "Write tests",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "Task created") || !strings.Contains(result.Content, "Write tests") {
		t.Fatalf("unexpected result: %q", result)
	}

	// Verify event emitted
	ev := <-eventCh
	if ev.Type() != events.EventTypeTaskCreate {
		t.Fatalf("expected task_create event, got %q", ev.Type())
	}
	if ev.Content() != "Write tests" {
		t.Fatalf("expected event content 'Write tests', got %q", ev.Content())
	}
	taskID, _ := ev.Extra()["task_id"].(int)
	if taskID != 1 {
		t.Fatalf("expected task_id 1, got %v", taskID)
	}
}

func TestTaskTool_Execute_Update(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	// Create first
	tl.Create("Write tests")

	// Then update
	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "update",
		"task_id": float64(1),
		"status":  "in_progress",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "updated to in_progress") {
		t.Fatalf("unexpected result: %q", result)
	}

	// Verify event
	ev := <-eventCh
	if ev.Type() != events.EventTypeTaskUpdate {
		t.Fatalf("expected task_update event, got %q", ev.Type())
	}
	status, _ := ev.Extra()["status"].(string)
	if status != "in_progress" {
		t.Fatalf("expected status in_progress, got %q", status)
	}
}

func TestTaskTool_Execute_List(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	tl.Create("Write tests")
	tl.Create("Review code")

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "Write tests") || !strings.Contains(result.Content, "Review code") {
		t.Fatalf("unexpected result: %q", result)
	}

	// Verify event
	ev := <-eventCh
	if ev.Type() != events.EventTypeTaskList {
		t.Fatalf("expected task_list event, got %q", ev.Type())
	}
}

func TestTaskTool_Execute_NoAction(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	result, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Fatal("expected error for missing action")
	}
	if result != nil && !strings.Contains(result.Error, "action parameter is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskTool_Execute_UnknownAction(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "unknown",
	})
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Fatal("expected error for unknown action")
	}
	if result != nil && !strings.Contains(result.Error, "unknown action") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskTool_Execute_UpdateNotFound(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "update",
		"task_id": float64(999),
		"status":  "completed",
	})
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Fatal("expected error for non-existent task")
	}
}

func TestFormatTaskList(t *testing.T) {
	tl := tasks.NewTaskList()
	tl.Create("Task 1")
	tl.Create("Task 2")
	tl.Update(1, tasks.TaskStatusInProgress, "")
	tl.Update(2, tasks.TaskStatusCompleted, "Done")

	result := formatTaskList(tl.List())
	if !strings.Contains(result, "[>] 1. Task 1") {
		t.Fatalf("expected in_progress icon for task 1, got %q", result)
	}
	if !strings.Contains(result, "[✓] 2. Task 2") {
		t.Fatalf("expected completed icon for task 2, got %q", result)
	}
	if !strings.Contains(result, "Done") {
		t.Fatalf("expected notes for task 2, got %q", result)
	}
}

func TestTaskTool_Execute_InProgressConstraint(t *testing.T) {
	eventCh := make(chan events.Event, 10)
	tl := tasks.NewTaskList()
	tool := New(eventCh, "req-1", tl, nil, nil)

	// Create two tasks
	tl.Create("Research")
	tl.Create("Write code")

	// Set first to in_progress
	_, _ = tool.Execute(context.Background(), map[string]any{
		"action":  "update",
		"task_id": float64(1),
		"status":  "in_progress",
	})
	// Drain event
	<-eventCh

	// Set second to in_progress — should auto-pause first
	_, _ = tool.Execute(context.Background(), map[string]any{
		"action":  "update",
		"task_id": float64(2),
		"status":  "in_progress",
	})
	<-eventCh

	list := tl.List()
	if list[0].Status != tasks.TaskStatusPending {
		t.Fatalf("expected task 1 to be paused to pending, got %q", list[0].Status)
	}
	if list[1].Status != tasks.TaskStatusInProgress {
		t.Fatalf("expected task 2 to be in_progress, got %q", list[1].Status)
	}
}
