package tui

import "testing"

func TestFormatToolParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "task create with content",
			input:    `{"action":"create","content":"当前目录项目调研"}`,
			expected: "create: 当前目录项目调研",
		},
		{
			name:     "task update with content",
			input:    `{"action":"update","task_id":1,"status":"in_progress","notes":"working on it"}`,
			expected: "update: working on it",
		},
		{
			name:     "file read with path",
			input:    `{"path":"/tmp/test.go"}`,
			expected: "/tmp/test.go",
		},
		{
			name:     "calculator with expression",
			input:    `{"expression":"2+2"}`,
			expected: "2+2",
		},
		{
			name:     "action only no content",
			input:    `{"action":"list"}`,
			expected: "list",
		},
		{
			name:     "invalid JSON falls back to raw",
			input:    `{invalid}`,
			expected: "{invalid}",
		},
		{
			name:     "empty JSON falls back to raw",
			input:    `{}`,
			expected: "{}",
		},
		{
			name:     "query field",
			input:    `{"query":"what is go"}`,
			expected: "what is go",
		},
		{
			name:     "command with content",
			input:    `{"command":"search","query":"hello world"}`,
			expected: "search: hello world",
		},
		{
			name:     "non-string values ignored",
			input:    `{"action":"create","count":5,"flag":true}`,
			expected: "create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolParams(tt.input)
			if result != tt.expected {
				t.Errorf("formatToolParams(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
