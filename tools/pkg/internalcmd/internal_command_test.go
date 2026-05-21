// Package internalcmd provides tests for the internalcmd package.
package internalcmd

import (
	"context"
	"testing"

	tools "github.com/oneliang/aura/tools/pkg"
)

// TestInternalCommandTool_Name tests Name method.
func TestInternalCommandTool_Name(t *testing.T) {
	tool := New(nil)
	if tool.Name() != "internal_command" {
		t.Errorf("Name() = %q, want internal_command", tool.Name())
	}
}

// TestInternalCommandTool_Description tests Description method.
func TestInternalCommandTool_Description(t *testing.T) {
	tool := New(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

// TestInternalCommandTool_Execute tests Execute method.
func TestInternalCommandTool_Execute(t *testing.T) {
	executed := false
	mockExecutor := func(ctx context.Context, cmdName string, params map[string]any) (string, error) {
		executed = true
		if cmdName != "test_cmd" {
			t.Errorf("executor got cmdName %q, want test_cmd", cmdName)
		}
		if len(params) != 1 {
			t.Errorf("executor got %d params, want 1", len(params))
		}
		return "result", nil
	}

	tool := New(mockExecutor)
	ctx := context.Background()
	params := map[string]any{
		"command": "test_cmd",
		"params":  map[string]any{"key": "value"},
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result == nil || result.Content != "result" {
		t.Errorf("Execute() result = %q, want result", result)
	}
	if !executed {
		t.Error("Executor was not called")
	}
}

// TestInternalCommandTool_Execute_NoExecutor tests Execute without executor.
func TestInternalCommandTool_Execute_NoExecutor(t *testing.T) {
	tool := New(nil)
	ctx := context.Background()
	params := map[string]any{
		"command": "test_cmd",
	}

	result, err := tool.Execute(ctx, params)
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Error("Execute() should return error when executor is nil")
	}
}

// TestInternalCommandTool_Execute_MissingCommand tests Execute with missing command.
func TestInternalCommandTool_Execute_MissingCommand(t *testing.T) {
	executed := false
	mockExecutor := func(ctx context.Context, cmdName string, params map[string]any) (string, error) {
		executed = true
		return "", nil
	}

	tool := New(mockExecutor)
	ctx := context.Background()
	params := map[string]any{}

	result, err := tool.Execute(ctx, params)
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Error("Execute() should return error when command is missing")
	}
	if executed {
		t.Error("Executor should not be called when command is missing")
	}
}

// TestInternalCommandTool_Execute_InvalidCommandType tests Execute with invalid command type.
func TestInternalCommandTool_Execute_InvalidCommandType(t *testing.T) {
	executed := false
	mockExecutor := func(ctx context.Context, cmdName string, params map[string]any) (string, error) {
		executed = true
		return "", nil
	}

	tool := New(mockExecutor)
	ctx := context.Background()
	params := map[string]any{
		"command": 123, // Invalid type
	}

	result, err := tool.Execute(ctx, params)
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Error("Execute() should return error when command is not a string")
	}
	if executed {
		t.Error("Executor should not be called when command type is invalid")
	}
}

// TestInternalCommandTool_Execute_NoParams tests Execute without params.
func TestInternalCommandTool_Execute_NoParams(t *testing.T) {
	executorCalled := false
	mockExecutor := func(ctx context.Context, cmdName string, params map[string]any) (string, error) {
		executorCalled = true
		if params == nil {
			t.Error("params should not be nil")
		}
		return "result", nil
	}

	tool := New(mockExecutor)
	ctx := context.Background()
	params := map[string]any{
		"command": "test_cmd",
		// No params key
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result == nil || result.Content != "result" {
		t.Errorf("Execute() result = %q, want result", result)
	}
	if !executorCalled {
		t.Error("Executor should have been called")
	}
}

// TestInternalCommandTool_Execute_InvalidParamsType tests Execute with invalid params type.
func TestInternalCommandTool_Execute_InvalidParamsType(t *testing.T) {
	executorCalled := false
	mockExecutor := func(ctx context.Context, cmdName string, params map[string]any) (string, error) {
		executorCalled = true
		return "result", nil
	}

	tool := New(mockExecutor)
	ctx := context.Background()
	params := map[string]any{
		"command": "test_cmd",
		"params":  "invalid", // Should be map[string]any
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result == nil || result.Content != "result" {
		t.Errorf("Execute() result = %q, want result", result)
	}
	if !executorCalled {
		t.Error("Executor should have been called with empty params")
	}
}
