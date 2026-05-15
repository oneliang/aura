package tasks

import (
	"testing"
)

func TestTaskList_Create(t *testing.T) {
	tl := NewTaskList()
	task := tl.Create("Write documentation")

	if task.ID != 1 {
		t.Fatalf("expected ID 1, got %d", task.ID)
	}
	if task.Content != "Write documentation" {
		t.Fatalf("expected content 'Write documentation', got %q", task.Content)
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("expected status pending, got %q", task.Status)
	}
}

func TestTaskList_Update(t *testing.T) {
	tl := NewTaskList()
	tl.Create("Write documentation")

	updated, err := tl.Update(1, TaskStatusInProgress, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != TaskStatusInProgress {
		t.Fatalf("expected in_progress, got %q", updated.Status)
	}
}

func TestTaskList_UpdateNotFound(t *testing.T) {
	tl := NewTaskList()
	_, err := tl.Update(999, TaskStatusCompleted, "")
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestTaskList_List(t *testing.T) {
	tl := NewTaskList()
	tl.Create("Task 1")
	tl.Create("Task 2")

	list := tl.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}
}

func TestTaskList_Reset(t *testing.T) {
	tl := NewTaskList()
	tl.Create("Task 1")
	tl.Reset()

	if len(tl.List()) != 0 {
		t.Fatal("expected empty list after reset")
	}

	task := tl.Create("New task after reset")
	if task.ID != 1 {
		t.Fatalf("expected ID to reset to 1, got %d", task.ID)
	}
}

func TestTaskList_InProgressConstraint(t *testing.T) {
	// The constraint is enforced by TaskTool.Execute(), not TaskList.Update() directly.
	// This test verifies the underlying Update works correctly when called with in_progress.
	tl := NewTaskList()
	t1 := tl.Create("Research")
	t2 := tl.Create("Write code")

	// Directly set both to in_progress (tool-level logic prevents this, but Update() is generic)
	tl.Update(t1.ID, TaskStatusInProgress, "")
	tl.Update(t2.ID, TaskStatusInProgress, "")

	// Both can be in_progress at the TaskList level — the tool enforces the constraint
	allTasks := tl.List()
	if len(allTasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(allTasks))
	}
}

func TestTaskList_Complete(t *testing.T) {
	tl := NewTaskList()
	task := tl.Create("Write code")
	tl.Update(task.ID, TaskStatusCompleted, "All tests pass")

	list := tl.List()
	if list[0].Status != TaskStatusCompleted {
		t.Fatalf("expected completed, got %q", list[0].Status)
	}
	if list[0].Notes != "All tests pass" {
		t.Fatalf("expected notes 'All tests pass', got %q", list[0].Notes)
	}
}

func TestTaskList_Restore(t *testing.T) {
	tl := NewTaskList()
	tl.Restore([]Task{
		{ID: 1, Content: "A", Status: TaskStatusCompleted, Notes: "done"},
		{ID: 3, Content: "B", Status: TaskStatusPending},
	})

	list := tl.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}
	if list[0].ID != 1 || list[0].Content != "A" || list[0].Status != TaskStatusCompleted {
		t.Fatalf("task 0 mismatch: %+v", list[0])
	}
	if list[0].Notes != "done" {
		t.Fatalf("task 0 notes mismatch: %q", list[0].Notes)
	}
	if list[1].ID != 3 || list[1].Content != "B" {
		t.Fatalf("task 1 mismatch: %+v", list[1])
	}

	// nextID should be maxID + 1 = 4
	newTask := tl.Create("C")
	if newTask.ID != 4 {
		t.Fatalf("expected next ID 4, got %d", newTask.ID)
	}
}

func TestTaskList_CreateFromPlanStep(t *testing.T) {
	tl := NewTaskList()
	task := tl.CreateFromPlanStep("step-1", "Build feature", "Implement API endpoint")

	if task.ID != 1 {
		t.Fatalf("expected ID 1, got %d", task.ID)
	}
	if task.PlanStepID != "step-1" {
		t.Fatalf("expected plan_step_id 'step-1', got %q", task.PlanStepID)
	}
	if task.PlanGoal != "Build feature" {
		t.Fatalf("expected plan_goal 'Build feature', got %q", task.PlanGoal)
	}
	if task.Content != "Implement API endpoint" {
		t.Fatalf("expected content 'Implement API endpoint', got %q", task.Content)
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("expected status pending, got %q", task.Status)
	}
	if task.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
}

func TestTaskList_FindByPlanStepID(t *testing.T) {
	tl := NewTaskList()
	tl.CreateFromPlanStep("step-1", "Goal A", "Task A")
	tl.CreateFromPlanStep("step-2", "Goal A", "Task B")
	tl.Create("Orphan task")

	// Find existing
	task := tl.FindByPlanStepID("step-2")
	if task == nil {
		t.Fatal("expected to find task for step-2")
	}
	if task.Content != "Task B" {
		t.Fatalf("expected content 'Task B', got %q", task.Content)
	}

	// Find non-existent
	task = tl.FindByPlanStepID("nonexistent")
	if task != nil {
		t.Fatal("expected nil for non-existent step")
	}
}

func TestTaskList_FindAllByPlanStepID(t *testing.T) {
	tl := NewTaskList()
	tl.CreateFromPlanStep("step-1", "Goal A", "Task A")
	tl.CreateFromPlanStep("step-2", "Goal A", "Task B")
	tl.Create("Orphan task")

	// Find all for step-1
	results := tl.FindAllByPlanStepID("step-1")
	if len(results) != 1 {
		t.Fatalf("expected 1 task, got %d", len(results))
	}
	if results[0].Content != "Task A" {
		t.Fatalf("expected content 'Task A', got %q", results[0].Content)
	}

	// Find all for non-existent
	results = tl.FindAllByPlanStepID("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(results))
	}
}

func TestTaskList_CountByStatus(t *testing.T) {
	tl := NewTaskList()
	t1 := tl.Create("Task 1")
	t2 := tl.Create("Task 2")
	tl.Create("Task 3") // remains pending

	tl.Update(t1.ID, TaskStatusCompleted, "")
	tl.Update(t2.ID, TaskStatusInProgress, "")
	// t3 remains pending

	if tl.CountByStatus(TaskStatusCompleted) != 1 {
		t.Fatal("expected 1 completed")
	}
	if tl.CountByStatus(TaskStatusInProgress) != 1 {
		t.Fatal("expected 1 in_progress")
	}
	if tl.CountByStatus(TaskStatusPending) != 1 {
		t.Fatal("expected 1 pending")
	}
	if tl.CountByStatus("unknown") != 0 {
		t.Fatal("expected 0 for unknown status")
	}

	// Empty list
	emptyTl := NewTaskList()
	if emptyTl.CountByStatus(TaskStatusPending) != 0 {
		t.Fatal("expected 0 for empty list")
	}
}

func TestTask_ActiveForm_DerivedOnCreate(t *testing.T) {
	tl := NewTaskList()
	task := tl.Create("Fix authentication bug")

	if task.Content != "Fix authentication bug" {
		t.Fatalf("expected content 'Fix authentication bug', got %q", task.Content)
	}
	// ActiveForm should be derived: "Fixing authentication bug"
	if task.ActiveForm != "Fixing authentication bug" {
		t.Fatalf("expected active_form 'Fixing authentication bug', got %q", task.ActiveForm)
	}
}

func TestTask_ActiveForm_CommonVerbs(t *testing.T) {
	tests := []struct {
		content    string
		activeForm string
	}{
		{"Run tests", "Running tests"},
		{"Build the project", "Building the project"},
		{"Add user authentication", "Adding user authentication"},
		{"Update config file", "Updating config file"},
		{"Create new module", "Creating new module"},
		{"Delete old files", "Deleting old files"},
		{"Implement feature X", "Implementing feature X"},
		{"Write documentation", "Writing documentation"},
		{"Review the code", "Reviewing the code"},
	}

	for _, tc := range tests {
		tl := NewTaskList()
		task := tl.Create(tc.content)
		if task.ActiveForm != tc.activeForm {
			t.Errorf("content %q: expected active_form %q, got %q", tc.content, tc.activeForm, task.ActiveForm)
		}
	}
}

func TestTask_ActiveForm_Fallback(t *testing.T) {
	tl := NewTaskList()
	// Unknown verb — should fallback to original content
	task := tl.Create("Analyze performance metrics")

	if task.ActiveForm != "Analyze performance metrics" {
		t.Fatalf("expected active_form to fallback to content, got %q", task.ActiveForm)
	}
}

func TestDeriveActiveForm(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"Fix bug", "Fixing bug"},
		{"Run tests", "Running tests"},
		{"Build app", "Building app"},
		{"Add feature", "Adding feature"},
		{"Update code", "Updating code"},
		{"Create file", "Creating file"},
		{"Delete item", "Deleting item"},
		{"Implement logic", "Implementing logic"},
		{"Write code", "Writing code"},
		{"Review PR", "Reviewing PR"},
		{"Unknown action", "Unknown action"}, // fallback
		{"", ""},                             // empty
	}

	for _, tc := range tests {
		result := deriveActiveForm(tc.input)
		if result != tc.output {
			t.Errorf("deriveActiveForm(%q) = %q, want %q", tc.input, result, tc.output)
		}
	}
}

func TestTaskList_InProgressCount(t *testing.T) {
	tl := NewTaskList()
	t1 := tl.Create("Task 1")
	t2 := tl.Create("Task 2")
	tl.Create("Task 3")

	// Initially 0 in_progress
	if tl.InProgressCount() != 0 {
		t.Fatal("expected 0 in_progress initially")
	}

	tl.Update(t1.ID, TaskStatusInProgress, "")
	if tl.InProgressCount() != 1 {
		t.Fatal("expected 1 in_progress after updating t1")
	}

	tl.Update(t2.ID, TaskStatusInProgress, "")
	if tl.InProgressCount() != 2 {
		t.Fatal("expected 2 in_progress (TaskList allows multiple, tool enforces constraint)")
	}

	tl.Update(t1.ID, TaskStatusCompleted, "")
	if tl.InProgressCount() != 1 {
		t.Fatal("expected 1 in_progress after completing t1")
	}
}

func TestTaskList_GetInProgressTask(t *testing.T) {
	tl := NewTaskList()
	t1 := tl.Create("Task 1")
	t2 := tl.Create("Task 2")

	// Initially no in_progress
	task := tl.GetInProgressTask()
	if task != nil {
		t.Fatal("expected nil when no in_progress tasks")
	}

	tl.Update(t1.ID, TaskStatusInProgress, "")
	task = tl.GetInProgressTask()
	if task == nil {
		t.Fatal("expected to find in_progress task")
	}
	if task.ID != t1.ID {
		t.Fatalf("expected task ID %d, got %d", t1.ID, task.ID)
	}

	// Multiple in_progress (TaskList allows, tool enforces) — returns first found
	tl.Update(t2.ID, TaskStatusInProgress, "")
	task = tl.GetInProgressTask()
	if task == nil {
		t.Fatal("expected to find at least one in_progress task")
	}
}
