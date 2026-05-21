package lsp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tools "github.com/oneliang/aura/tools/pkg"
)

// TestCodeToolName tests the code tool name.
func TestCodeToolName(t *testing.T) {
	tool := NewCodeTool(".")

	if tool.Name() != "code_navigate" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "code_navigate")
	}
}

// TestCodeToolDescription tests the code tool description.
func TestCodeToolDescription(t *testing.T) {
	tool := NewCodeTool(".")

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	// Check for supported operations
	requiredOps := []string{"definition", "references", "symbols", "format", "diagnostics", "rename"}
	for _, op := range requiredOps {
		if !containsString(desc, op) {
			t.Errorf("Description() should mention operation: %s", op)
		}
	}
}

// TestCodeToolExecuteValidation tests parameter validation.
func TestCodeToolExecuteValidation(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing operation",
			params:  map[string]any{},
			wantErr: true,
			errMsg:  "operation is required",
		},
		{
			name: "missing file",
			params: map[string]any{
				"operation": "definition",
			},
			wantErr: true,
			errMsg:  "file path is required",
		},
		{
			name: "unknown operation",
			params: map[string]any{
				"operation": "unknown",
				"file":      "test.go",
			},
			wantErr: true,
			errMsg:  "unknown operation",
		},
		{
			name: "definition missing line",
			params: map[string]any{
				"operation": "definition",
				"file":      "test.go",
				"column":    1,
			},
			wantErr: true,
			errMsg:  "line and column are required",
		},
		{
			name: "definition missing column",
			params: map[string]any{
				"operation": "definition",
				"file":      "test.go",
				"line":      1.0,
			},
			wantErr: true,
			errMsg:  "line and column are required",
		},
		{
			name: "references missing line",
			params: map[string]any{
				"operation": "references",
				"file":      "test.go",
				"column":    1,
			},
			wantErr: true,
			errMsg:  "line and column are required",
		},
		{
			name: "rename missing newName",
			params: map[string]any{
				"operation": "rename",
				"file":      "test.go",
				"line":      1.0,
				"column":    1,
			},
			wantErr: true,
			errMsg:  "newName are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)

			if tt.wantErr {
				hasError := err != nil || (result != nil && result.Status == tools.ToolStatusError)
				if !hasError {
					t.Error("Execute() expected error, got nil")
				}
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
			}

			if tt.wantErr && tt.errMsg != "" {
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else if result != nil {
					errMsg = result.Error
				}
				if errMsg != "" && !containsString(errMsg, tt.errMsg) {
					t.Errorf("Execute() error = %q, want to contain %q", errMsg, tt.errMsg)
				}
			}
		})
	}
}

// TestCodeToolExecuteOperations tests operation execution (error paths since gopls may not be installed).
func TestCodeToolExecuteOperations(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	// Create a temporary Go file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		operation string
		params    map[string]any
	}{
		{
			name:      "definition",
			operation: "definition",
			params: map[string]any{
				"operation": "definition",
				"file":      tmpFile,
				"line":      3.0,
				"column":    6.0,
			},
		},
		{
			name:      "references",
			operation: "references",
			params: map[string]any{
				"operation": "references",
				"file":      tmpFile,
				"line":      3.0,
				"column":    6.0,
			},
		},
		{
			name:      "symbols",
			operation: "symbols",
			params: map[string]any{
				"operation": "symbols",
				"file":      tmpFile,
			},
		},
		{
			name:      "format",
			operation: "format",
			params: map[string]any{
				"operation": "format",
				"file":      tmpFile,
			},
		},
		{
			name:      "diagnostics",
			operation: "diagnostics",
			params: map[string]any{
				"operation": "diagnostics",
				"file":      tmpFile,
			},
		},
		{
			name:      "rename",
			operation: "rename",
			params: map[string]any{
				"operation": "rename",
				"file":      tmpFile,
				"line":      3.0,
				"column":    6.0,
				"newName":   "main2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)

			// If gopls is not installed, expect error about command not found
			if err != nil {
				// Check if error is about gopls not found (acceptable)
				if containsString(err.Error(), "executable file not found") ||
					containsString(err.Error(), "gopls") {
					// This is expected when gopls is not installed
					return
				}
				t.Errorf("Execute() unexpected error = %v", err)
			}

			// If successful, result should not be empty
			if result.Content == "" {
				t.Error("Execute() result should not be empty")
			}
		})
	}
}

// TestCodeToolWorkspaceRoot tests workspace root detection.
func TestCodeToolWorkspaceRoot(t *testing.T) {
	// Test with a path that should find a workspace root
	tool := NewCodeTool(".")

	mgr := tool.Manager()
	if mgr == nil {
		t.Error("Manager should not be nil")
	}
}

// TestCodeToolGoplsPath tests gopls path detection.
func TestCodeToolGoplsPath(t *testing.T) {
	tool := NewCodeTool(".")

	// Check that Go client is available via manager
	mgr := tool.Manager()
	goClient, err := mgr.GetClient("go")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}

	// Client should be registered (availability depends on installation)
	if goClient.Language() != "go" {
		t.Errorf("Language() = %q, want %q", goClient.Language(), "go")
	}
}

// TestCodeToolMutex tests concurrent execution safety.
func TestCodeToolMutex(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	// Run multiple operations concurrently to test mutex
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, _ = tool.Execute(ctx, map[string]any{
				"operation": "diagnostics",
				"file":      "nonexistent.go",
			})
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestCodeToolSymbolsOperation tests symbols operation specifically.
func TestCodeToolSymbolsOperation(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	// symbols operation with empty file should return error
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "symbols",
	})

	// Should return error for missing file
	if err == nil && result.Status != tools.ToolStatusError {
		t.Error("Expected error for missing file")
	}
}

// TestCodeToolWithNonExistentFile tests behavior with non-existent files.
func TestCodeToolWithNonExistentFile(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"operation": "definition",
		"file":      "/nonexistent/file.go",
		"line":      1.0,
		"column":    1.0,
	})

	// Should fail (either gopls error via ToolResult or Go error)
	hasError := err != nil || (result != nil && result.Status == tools.ToolStatusError)
	if !hasError {
		t.Error("Execute() expected error for non-existent file")
	}
}

// TestCodeToolMultiLanguage tests multi-language support.
func TestCodeToolMultiLanguage(t *testing.T) {
	tool := NewCodeTool(".")
	ctx := context.Background()

	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{"go file", "main.go", false},
		{"rust file", "main.rs", false},  // placeholder client, returns not available error
		{"typescript file", "index.ts", false}, // placeholder client
		{"python file", "main.py", false}, // placeholder client
		{"unknown file", "file.xyz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"operation": "symbols",
				"file":      tt.file,
			})

			if tt.wantErr {
				if err == nil && result.Status != tools.ToolStatusError {
					t.Error("Expected error for unknown language")
				}
				return
			}

			// For known languages, either success or "not available" error is acceptable
			if err != nil {
				// Check error is about LSP server not available
				if result == nil || result.Status != tools.ToolStatusError {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
