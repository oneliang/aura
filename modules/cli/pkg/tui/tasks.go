// Package tui provides the terminal user interface for Aura.
package tui

import (
	"fmt"
	"strings"

	"github.com/oneliang/aura/shared/pkg/tasks"
)

// TaskWidget manages task list display during agent execution.
type TaskWidget struct {
	tasks   []tasks.Task
	visible bool
	styles  UIStyles
}

// NewTaskWidget creates a new task widget.
func NewTaskWidget(styles UIStyles) *TaskWidget {
	return &TaskWidget{styles: styles}
}

// HandleCreate adds a task from a task_create event.
func (w *TaskWidget) HandleCreate(taskID int, content string, planStepID string) string {
	w.visible = true
	t := tasks.Task{ID: taskID, Content: content, Status: tasks.TaskStatusPending}
	if planStepID != "" {
		t.PlanStepID = planStepID
	}
	w.tasks = append(w.tasks, t)
	return w.Render()
}

// HandleUpdate updates a task from a task_update event.
func (w *TaskWidget) HandleUpdate(taskID int, status, notes string) string {
	for i, t := range w.tasks {
		if t.ID == taskID {
			t.Status = tasks.TaskStatus(status)
			t.Notes = notes
			w.tasks[i] = t
			break
		}
	}
	return w.Render()
}

// HandleList replaces the entire task list from a task_list event.
func (w *TaskWidget) HandleList(newTasks []tasks.Task) string {
	w.tasks = newTasks
	w.visible = len(newTasks) > 0
	return w.Render()
}

// Render returns the formatted task list string.
func (w *TaskWidget) Render() string {
	if !w.visible || len(w.tasks) == 0 {
		return ""
	}

	var b strings.Builder
	for _, t := range w.tasks {
		icon := tasks.StatusIcon(t.Status)
		// Use ActiveForm for in_progress tasks (progressive form)
		displayContent := t.Content
		if t.Status == tasks.TaskStatusInProgress && t.ActiveForm != "" {
			displayContent = t.ActiveForm
		}
		line := fmt.Sprintf("  %s %d. %s", icon, t.ID, displayContent)
		if t.Notes != "" {
			line += fmt.Sprintf(" (%s)", t.Notes)
		}
		// Highlight in_progress task
		if t.Status == tasks.TaskStatusInProgress {
			line = w.styles.TaskInProgress.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// Reset clears the widget for a new interaction.
func (w *TaskWidget) Reset() {
	w.tasks = make([]tasks.Task, 0)
	w.visible = false
}

// RenderStyled returns the task list with visual framing (border + title).
func (w *TaskWidget) RenderStyled() string {
	if !w.visible || len(w.tasks) == 0 {
		return ""
	}

	border := w.styles.TaskWidgetBorder.Render(strings.Repeat("─", TaskWidgetBorderWidth))
	var b strings.Builder
	b.WriteString(border)
	b.WriteByte('\n')
	b.WriteString(w.styles.TaskWidgetTitle.Render("  Tasks"))
	b.WriteByte('\n')
	for _, t := range w.tasks {
		icon := tasks.StatusIcon(t.Status)
		// Use ActiveForm for in_progress tasks (progressive form)
		displayContent := t.Content
		if t.Status == tasks.TaskStatusInProgress && t.ActiveForm != "" {
			displayContent = t.ActiveForm
		}
		line := fmt.Sprintf("  %s %d. %s", icon, t.ID, displayContent)
		if t.Notes != "" {
			line += fmt.Sprintf(" (%s)", t.Notes)
		}
		// Highlight in_progress task
		if t.Status == tasks.TaskStatusInProgress {
			line = w.styles.TaskInProgress.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString(border)
	return b.String()
}
