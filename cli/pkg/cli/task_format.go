package cli

import (
	"fmt"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/tasks"
)

// formatTaskEvent formats a task event for CLI console output.
// Returns the formatted string with ANSI color codes.
func formatTaskEvent(ev sdk.Event) string {
	switch ev.Type() {
	case sdk.EventTypeTaskCreate:
		taskID, _ := ev.Extra()["task_id"].(int)
		return fmt.Sprintf("\033[36m[+] Task #%d: %s\033[0m\n", taskID, ev.Content())

	case sdk.EventTypeTaskUpdate:
		taskID, _ := ev.Extra()["task_id"].(int)
		status, _ := ev.Extra()["status"].(string)
		return fmt.Sprintf("\033[36m[~] Task #%d -> %s\033[0m\n", taskID, status)

	case sdk.EventTypeTaskList:
		// Direct path: engine creates []tasks.Task, runtime passes Extra() through
		if rawTasks, ok := ev.Extra()["tasks"].([]tasks.Task); ok {
			s := fmt.Sprintf("\033[36mTasks (%d):\033[0m\n", len(rawTasks))
			for _, t := range rawTasks {
				s += fmt.Sprintf("  %-3d %-15s %v", t.ID, t.Status, t.Content)
				if t.Notes != "" {
					s += fmt.Sprintf(" (%s)", t.Notes)
				}
				s += "\n"
			}
			return s
		}
		// SSE path: JSON serialization converts slices to []interface{}
		if rawTasks, ok := ev.Extra()["tasks"].([]interface{}); ok {
			s := fmt.Sprintf("\033[36mTasks (%d):\033[0m\n", len(rawTasks))
			for _, t := range rawTasks {
				if tm, ok := t.(map[string]interface{}); ok {
					id, _ := tm["ID"].(float64)
					s += fmt.Sprintf("  %-3d %-15s %v\n", int(id), tm["Status"], tm["Content"])
				}
			}
			return s
		}
	}
	return ""
}
