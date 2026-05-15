// Package tasks provides in-memory task tracking for multi-step operations.
package tasks

import (
	"fmt"
	"strings"
	"time"
)

// TaskStatus represents the status of a tracked task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
)

// Task represents a single tracked task item.
type Task struct {
	ID         int        `json:"id"`
	Content    string     `json:"content"`      // Imperative form (shown when pending/completed)
	ActiveForm string     `json:"active_form"`  // Progressive form (shown when in_progress)
	Status     TaskStatus `json:"status"`
	Notes      string     `json:"notes,omitempty"`
	PlanStepID string     `json:"plan_step_id,omitempty"` // Link to plan step ID
	PlanGoal   string     `json:"plan_goal,omitempty"`    // Parent plan goal
	CreatedAt  time.Time  `json:"created_at,omitempty"`
}

// TaskList manages a session-scoped list of tasks.
type TaskList struct {
	tasks  []Task
	nextID int
}

// NewTaskList creates an empty task list.
func NewTaskList() *TaskList {
	return &TaskList{nextID: 1}
}

// Create adds a new task and returns it.
func (tl *TaskList) Create(content string) Task {
	t := Task{
		ID:         tl.nextID,
		Content:    content,
		ActiveForm: deriveActiveForm(content),
		Status:     TaskStatusPending,
	}
	tl.nextID++
	tl.tasks = append(tl.tasks, t)
	return t
}

// Update updates an existing task by ID.
func (tl *TaskList) Update(id int, status TaskStatus, notes string) (Task, error) {
	for i, t := range tl.tasks {
		if t.ID == id {
			t.Status = status
			t.Notes = notes
			tl.tasks[i] = t
			return t, nil
		}
	}
	return Task{}, fmt.Errorf("task not found: %d", id)
}

// List returns all tasks.
func (tl *TaskList) List() []Task {
	return tl.tasks
}

// Reset clears all tasks.
func (tl *TaskList) Reset() {
	tl.tasks = make([]Task, 0)
	tl.nextID = 1
}

// Restore injects a pre-existing task list and sets nextID accordingly.
// Used to restore persisted tasks from disk.
func (tl *TaskList) Restore(ts []Task) {
	tl.tasks = make([]Task, len(ts))
	copy(tl.tasks, ts)
	maxID := 0
	for _, t := range ts {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	tl.nextID = maxID + 1
}

// StatusIcon returns the display icon for a task status.
func StatusIcon(status TaskStatus) string {
	switch status {
	case TaskStatusCompleted:
		return "[✓]"
	case TaskStatusInProgress:
		return "[>]"
	default:
		return "[ ]"
	}
}

// CreateFromPlanStep creates a task linked to a plan step.
func (tl *TaskList) CreateFromPlanStep(stepID, goal, content string) Task {
	t := Task{
		ID:         tl.nextID,
		Content:    content,
		Status:     TaskStatusPending,
		PlanStepID: stepID,
		PlanGoal:   goal,
		CreatedAt:  time.Now(),
	}
	tl.nextID++
	tl.tasks = append(tl.tasks, t)
	return t
}

// FindByPlanStepID finds a task by its linked plan step ID.
func (tl *TaskList) FindByPlanStepID(stepID string) *Task {
	for i := range tl.tasks {
		if tl.tasks[i].PlanStepID == stepID {
			return &tl.tasks[i]
		}
	}
	return nil
}

// FindAllByPlanStepID returns all tasks linked to a plan step ID.
func (tl *TaskList) FindAllByPlanStepID(stepID string) []Task {
	var result []Task
	for _, t := range tl.tasks {
		if t.PlanStepID == stepID {
			result = append(result, t)
		}
	}
	return result
}

// CountByStatus returns the count of tasks in a given status.
func (tl *TaskList) CountByStatus(status TaskStatus) int {
	count := 0
	for _, t := range tl.tasks {
		if t.Status == status {
			count++
		}
	}
	return count
}

// InProgressCount returns the count of in_progress tasks.
func (tl *TaskList) InProgressCount() int {
	return tl.CountByStatus(TaskStatusInProgress)
}

// GetInProgressTask returns the first in_progress task, or nil if none.
func (tl *TaskList) GetInProgressTask() *Task {
	for i := range tl.tasks {
		if tl.tasks[i].Status == TaskStatusInProgress {
			return &tl.tasks[i]
		}
	}
	return nil
}

// deriveActiveForm converts imperative form to progressive form.
// "Fix bug" → "Fixing bug", "Run tests" → "Running tests"
func deriveActiveForm(content string) string {
	if content == "" {
		return ""
	}
	verbs := map[string]string{
		"Fix":       "Fixing",
		"Run":       "Running",
		"Build":     "Building",
		"Add":       "Adding",
		"Update":    "Updating",
		"Create":    "Creating",
		"Delete":    "Deleting",
		"Implement": "Implementing",
		"Write":     "Writing",
		"Review":    "Reviewing",
	}
	for imp, prog := range verbs {
		if strings.HasPrefix(content, imp+" ") {
			return prog + content[len(imp):]
		}
	}
	return content // fallback to original
}
