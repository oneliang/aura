package tools

import (
	"context"
	"testing"
)

// TestBaseTool tests the BaseTool implementation.
func TestBaseTool(t *testing.T) {
	executorCalled := false
	testExecutor := func(ctx context.Context, params map[string]any) (*ToolResult, error) {
		executorCalled = true
		return &ToolResult{Status: ToolStatusSuccess, Content: "test result"}, nil
	}

	tool := NewBaseTool("test_tool", "A test tool", testExecutor)

	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "test_tool")
	}

	if tool.Description() != "A test tool" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "A test tool")
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{"param": "value"})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result == nil || result.Content != "test result" {
		t.Errorf("Execute() result = %q, want %q", result, "test result")
	}

	if !executorCalled {
		t.Error("executor was not called")
	}
}

// TestBaseToolError tests error handling in BaseTool.
func TestBaseToolError(t *testing.T) {
	testExecutor := func(ctx context.Context, params map[string]any) (*ToolResult, error) {
		return nil, context.Canceled
	}

	tool := NewBaseTool("test_tool", "A test tool", testExecutor)

	ctx := context.Background()
	_, err := tool.Execute(ctx, nil)

	if err != context.Canceled {
		t.Errorf("Execute() error = %v, want %v", err, context.Canceled)
	}
}

// TestToolResult tests the ToolResult type.
func TestToolResult(t *testing.T) {
	tests := []struct {
		name    string
		result  *ToolResult
		wantStr string
	}{
		{
			name: "success",
			result: &ToolResult{
				Status:  ToolStatusSuccess,
				Content: "command output",
			},
			wantStr: "command output",
		},
		{
			name: "error",
			result: &ToolResult{
				Status: ToolStatusError,
				Error:  "something went wrong",
			},
			wantStr: "Error: something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.result.String()
			if str != tt.wantStr {
				t.Errorf("String() = %q, want %q", str, tt.wantStr)
			}

			jsonStr := tt.result.JSON()
			if jsonStr == "" {
				t.Error("JSON() returned empty string")
			}
		})
	}
}
