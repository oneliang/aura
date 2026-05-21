// Package internal provides tests for the internal package.
package internal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestFormatOutput tests FormatOutput function.
func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      []byte
		notFoundMsg string
		want        string
	}{
		{
			name:        "with output",
			output:      []byte("some output"),
			notFoundMsg: "not found",
			want:        "some output",
		},
		{
			name:        "with output and newlines",
			output:      []byte("line1\nline2\n"),
			notFoundMsg: "not found",
			want:        "line1\nline2",
		},
		{
			name:        "empty output returns not found message",
			output:      []byte{},
			notFoundMsg: "not found",
			want:        "not found",
		},
		{
			name:        "whitespace only returns not found message",
			output:      []byte("   \n  \n  "),
			notFoundMsg: "not found",
			want:        "not found",
		},
		{
			name:        "empty not found message",
			output:      []byte{},
			notFoundMsg: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatOutput(tt.output, tt.notFoundMsg)
			if got != tt.want {
				t.Errorf("FormatOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunGopls tests RunGopls function.
func TestRunGopls(t *testing.T) {
	ctx := context.Background()

	// Find gopls or skip test
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("gopls not found, skipping test")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Test with version command
	output, err := RunGopls(ctx, goplsPath, tmpDir, "version")
	if err != nil {
		t.Fatalf("RunGopls() error = %v", err)
	}
	if len(output) == 0 {
		t.Error("RunGopls() should return output for version command")
	}
}

// TestRunGopls_ContextCancellation tests RunGopls with context cancellation.
func TestRunGopls_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Use a command that would take time
	output, err := RunGopls(ctx, "sleep", ".", "1")
	if err == nil {
		t.Error("RunGopls() should return error when context is cancelled")
	}
	if len(output) > 0 {
		t.Log("RunGopls() may have partial output:", string(output))
	}
}

// TestRunGopls_InvalidCommand tests RunGopls with invalid command.
func TestRunGopls_InvalidCommand(t *testing.T) {
	ctx := context.Background()

	_, err := RunGopls(ctx, "nonexistent_command_12345", ".", "arg1")
	if err == nil {
		t.Error("RunGopls() should return error for non-existent command")
	}
}

// TestRunGoplsWithResult tests RunGoplsWithResult function.
func TestRunGoplsWithResult(t *testing.T) {
	ctx := context.Background()

	// Find gopls or skip test
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("gopls not found, skipping test")
	}

	tmpDir := t.TempDir()

	// Test successful command
	result, err := RunGoplsWithResult(ctx, goplsPath, tmpDir, "not found", "version")
	if err != nil {
		t.Fatalf("RunGoplsWithResult() error = %v", err)
	}
	if result == "not found" {
		t.Error("RunGoplsWithResult() should return actual output, not not-found message")
	}
	if result == "" {
		t.Error("RunGoplsWithResult() should return non-empty output")
	}
}

// TestRunGoplsWithResult_Error tests RunGoplsWithResult with error.
func TestRunGoplsWithResult_Error(t *testing.T) {
	ctx := context.Background()

	_, err := RunGoplsWithResult(ctx, "nonexistent_command", ".", "not found", "arg")
	if err == nil {
		t.Error("RunGoplsWithResult() should return error for non-existent command")
	}
}

// TestRunGoplsWithResult_EmptyOutput tests RunGoplsWithResult with empty output.
func TestRunGoplsWithResult_EmptyOutput(t *testing.T) {
	ctx := context.Background()

	// Create a script that outputs nothing
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "empty.sh")
	script := "@echo off\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	// On Unix, use a shell command that produces no output
	result, err := RunGoplsWithResult(ctx, "true", tmpDir, "no output")
	if err != nil {
		t.Fatalf("RunGoplsWithResult() error = %v", err)
	}
	if result != "no output" {
		t.Errorf("RunGoplsWithResult() = %q, want no output", result)
	}
}

// TestRunGopls_WithTimeout tests RunGopls with timeout context.
func TestRunGopls_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use sleep which will exceed the timeout
	_, err := RunGopls(ctx, "sleep", ".", "1")
	if err == nil {
		t.Error("RunGopls() should return error when timeout exceeded")
	}
}

// TestRunGoplsWithResult_WithTimeout tests RunGoplsWithResult with timeout context.
func TestRunGoplsWithResult_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := RunGoplsWithResult(ctx, "sleep", ".", "timeout msg", "1")
	if err == nil {
		t.Error("RunGoplsWithResult() should return error when timeout exceeded")
	}
}
