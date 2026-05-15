// Package system provides system tools.
package system

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
)

// Test ShellTool
func TestShellToolName(t *testing.T) {
	tool := NewShellTool()
	name := tool.Name()
	if name != constants.ToolShellExec {
		t.Errorf("Name() = %v, want %q", name, constants.ToolShellExec)
	}
}

func TestShellToolDescription(t *testing.T) {
	tool := NewShellTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "command") {
		t.Error("Description() should mention 'command' parameter")
	}
}

func TestShellToolRequiresConfirmation(t *testing.T) {
	tool := NewShellTool()
	if !tool.RequiresConfirmation() {
		t.Error("RequiresConfirmation() should return true for shell execution")
	}
}

func TestShellToolExecute(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	params := map[string]any{"command": "echo Hello"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	// echo adds a newline
	expected := "Hello\n"
	if result == nil || result.Content != expected {
		t.Errorf("Execute() result = %q, want %q", result, expected)
	}
}

func TestShellToolExecuteMissingCommand(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{})
	if err == nil && result.Status != tools.ToolStatusError {
		t.Error("Execute() should error when command is missing")
	}
}

func TestShellToolExecuteInvalidCommandType(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	params := map[string]any{"command": 123}
	result, err := tool.Execute(ctx, params)
	if err == nil && result.Status != tools.ToolStatusError {
		t.Error("Execute() should error when command is not a string")
	}
}

func TestShellToolExecuteWithAllowedCommands(t *testing.T) {
	tool := NewShellTool(WithAllowedCommands([]string{"echo", "ls"}))
	ctx := context.Background()

	// Allowed command
	params := map[string]any{"command": "echo Hello"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !contains(result.Content, "Hello") {
		t.Errorf("Execute() result = %q, should contain 'Hello'", result)
	}
}

func TestShellToolExecuteWithNotAllowedCommand(t *testing.T) {
	tool := NewShellTool(WithAllowedCommands([]string{"echo", "ls"}))
	ctx := context.Background()

	// Not allowed command
	params := map[string]any{"command": "rm -rf /"}
	result, err := tool.Execute(ctx, params)
	if err == nil && result.Status != tools.ToolStatusError {
		t.Error("Execute() should error for not allowed command")
	}
	if result == nil || !contains(result.Error, "not allowed") {
		t.Errorf("Error should mention 'not allowed', got: %v", result)
	}
}

func TestShellToolExecuteWithTimeout(t *testing.T) {
	tool := NewShellTool(WithTimeout(5 * time.Second))
	ctx := context.Background()

	params := map[string]any{"command": "sleep 0.1 && echo done"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !contains(result.Content, "done") {
		t.Errorf("Execute() result = %q, should contain 'done'", result)
	}
}

func TestShellToolExecuteCommandTimeout(t *testing.T) {
	// Use a very short timeout to force timeout
	tool := NewShellTool(WithTimeout(50 * time.Millisecond))
	ctx := context.Background()

	// Command that takes longer than timeout
	params := map[string]any{"command": "sleep 1"}
	result, err := tool.Execute(ctx, params)
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Error("Execute() should error on command timeout")
	}
}

func TestShellToolExecuteCommandWithError(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	// Command that fails
	params := map[string]any{"command": "exit 1"}
	result, err := tool.Execute(ctx, params)
	if err == nil && (result == nil || result.Status != tools.ToolStatusError) {
		t.Error("Execute() should error when command fails")
	}
	if result != nil && !contains(result.Error, "command failed") {
		t.Errorf("Error should mention 'command failed', got: %v", result)
	}
}

func TestShellToolExecuteStderr(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	// Command that outputs to stderr
	params := map[string]any{"command": "echo stderr_message >&2"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !contains(result.Content, "Stderr:") {
		t.Errorf("Result should contain stderr section, got: %q", result)
	}
	if !contains(result.Content, "stderr_message") {
		t.Errorf("Result should contain stderr message, got: %q", result)
	}
}

func TestShellToolExecuteMkdir(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := "/tmp/aura_test_mkdir_" + time.Now().Format("20060102_150405")
	params := map[string]any{"command": "mkdir -p " + tempDir}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Status != tools.ToolStatusSuccess {
		t.Errorf("Execute() status = %v, want success, error: %s", result.Status, result.Error)
	}

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Directory %s was not created", tempDir)
	}

	// Cleanup (ignore error)
	tool.Execute(ctx, map[string]any{"command": "rm -rf " + tempDir})
}

func TestShellToolExecuteMkdirCurrentDir(t *testing.T) {
	tool := NewShellTool()
	ctx := context.Background()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Create directory in current working directory
	testDir := cwd + "/aura_test_mkdir_currdir_" + time.Now().Format("20060102_150405")
	params := map[string]any{"command": "mkdir -p " + testDir}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Status != tools.ToolStatusSuccess {
		t.Errorf("Execute() status = %v, want success, error: %s", result.Status, result.Error)
	}

	// Verify directory was created
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("Directory %s was not created (cwd=%s)", testDir, cwd)
	}

	// Cleanup (ignore error)
	tool.Execute(ctx, map[string]any{"command": "rm -rf " + testDir})
}

func TestShellToolExecuteWithPrefix(t *testing.T) {
	tool := NewShellTool(WithAllowedCommands([]string{"git"}))
	ctx := context.Background()

	// Command with matching first word
	params := map[string]any{"command": "git status ."}
	_, err := tool.Execute(ctx, params)
	// Should be allowed because first word "git" matches allowed "git"
	if err != nil && contains(err.Error(), "not allowed") {
		t.Errorf("Execute() should allow command with matching first word, error = %v", err)
	}

	// Test injection prevention - command with semicolon should still use first word
	tool2 := NewShellTool(WithAllowedCommands([]string{"ls"}))
	params2 := map[string]any{"command": "ls; rm -rf /"}
	_, err2 := tool2.Execute(ctx, params2)
	// Should be allowed because first word is "ls" which matches
	// (The actual command execution may fail, but it should pass the allow check)
	if err2 != nil && contains(err2.Error(), "not allowed") {
		t.Errorf("Execute() should check first word only, error = %v", err2)
	}

	// Test that different first word is blocked
	tool3 := NewShellTool(WithAllowedCommands([]string{"ls"}))
	params3 := map[string]any{"command": "rm -rf /"}
	result3, err3 := tool3.Execute(ctx, params3)
	if err3 == nil && (result3 == nil || result3.Status != tools.ToolStatusError) {
		t.Errorf("Execute() should block command with non-matching first word, error = %v", err3)
	}
	if result3 != nil && !contains(result3.Error, "not allowed") {
		t.Errorf("Error should mention 'not allowed', got: %v", result3)
	}
}

func TestShellToolWithOptions(t *testing.T) {
	tool := NewShellTool(
		WithAllowedCommands([]string{"echo"}),
		WithTimeout(10*time.Second),
	)

	if len(tool.allowedCommands) != 1 {
		t.Errorf("allowedCommands length = %d, want 1", len(tool.allowedCommands))
	}
	if tool.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", tool.timeout)
	}
}

// Test InfoTool
func TestInfoToolName(t *testing.T) {
	tool := &InfoTool{}
	name := tool.Name()
	if name != "system_info" {
		t.Errorf("Name() = %v, want 'system_info'", name)
	}
}

func TestInfoToolDescription(t *testing.T) {
	tool := &InfoTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestInfoToolExecute(t *testing.T) {
	tool := &InfoTool{}
	ctx := context.Background()

	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !contains(result.Content, "System Information") {
		t.Error("Result should contain 'System Information'")
	}
	if !contains(result.Content, "OS:") {
		t.Error("Result should contain OS info")
	}
	if !contains(result.Content, "Architecture:") {
		t.Error("Result should contain Architecture info")
	}
	if !contains(result.Content, "Go Version:") {
		t.Error("Result should contain Go Version info")
	}
}

// Test AllTools
func TestAllTools(t *testing.T) {
	tools := AllTools()

	// Should return 2 tools
	if len(tools) != 2 {
		t.Errorf("AllTools() returned %d tools, want 2", len(tools))
	}

	// Verify tool names
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if !names[constants.ToolShellExec] {
		t.Error("AllTools() missing bash tool")
	}
	if !names["system_info"] {
		t.Error("AllTools() missing system_info tool")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
