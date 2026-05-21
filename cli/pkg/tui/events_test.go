package tui

import "testing"

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		name     string
		extra    map[string]any
		expected int
	}{
		{
			name:     "int task_id",
			extra:    map[string]any{"task_id": 1},
			expected: 1,
		},
		{
			name:     "float64 task_id",
			extra:    map[string]any{"task_id": float64(2)},
			expected: 2,
		},
		{
			name:     "missing task_id",
			extra:    map[string]any{"other": "value"},
			expected: 0,
		},
		{
			name:     "nil extra",
			extra:    nil,
			expected: 0,
		},
		{
			name:     "wrong type task_id",
			extra:    map[string]any{"task_id": "string"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTaskID(tt.extra)
			if result != tt.expected {
				t.Errorf("extractTaskID() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestProcessingWidget_MultiToolEvents simulates the TUI event flow
// for parallel tool execution: multiple ToolStart → multiple ToolEnd.
func TestProcessingWidget_MultiToolEvents(t *testing.T) {
	w := NewProcessingWidget()
	w.Start("file_read")

	// First tool started
	if len(w.activeTools) != 1 {
		t.Errorf("expected 1 active tool after Start, got %d", len(w.activeTools))
	}

	// Second tool starts — AddTool (simulates parallel ToolStart event)
	w.AddTool("bash", `{"command": "ls"}`)
	if len(w.activeTools) != 2 {
		t.Errorf("expected 2 active tools after AddTool, got %d", len(w.activeTools))
	}
	if !w.IsActive() {
		t.Error("ProcessingWidget should be active with 2 tools")
	}

	// First tool ends — RemoveTool
	w.RemoveTool("file_read")
	if len(w.activeTools) != 1 {
		t.Errorf("expected 1 active tool after first RemoveTool, got %d", len(w.activeTools))
	}
	if !w.IsActive() {
		t.Error("ProcessingWidget should be active with 1 tool remaining")
	}

	// Second tool ends — widget should become inactive
	w.RemoveTool("bash")
	if w.IsActive() {
		t.Error("ProcessingWidget should be inactive after all tools complete")
	}
}
