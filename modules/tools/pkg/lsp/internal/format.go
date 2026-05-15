// Package internal provides internal utilities for LSP tools.
package internal

import (
	"context"
	"os/exec"
	"strings"
)

// GoplsResult represents the result of a gopls command.
type GoplsResult struct {
	Output string
	Error  error
}

// FormatOutput formats gopls output with a not-found message.
// Returns the output if not empty, otherwise returns notFoundMsg.
func FormatOutput(output []byte, notFoundMsg string) string {
	result := strings.TrimSpace(string(output))
	if result == "" {
		return notFoundMsg
	}
	return result
}

// RunGopls runs a gopls command and returns the output.
func RunGopls(ctx context.Context, goplsPath, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, goplsPath, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// RunGoplsWithResult runs a gopls command and returns formatted result.
func RunGoplsWithResult(ctx context.Context, goplsPath, dir, notFoundMsg string, args ...string) (string, error) {
	output, err := RunGopls(ctx, goplsPath, dir, args...)
	if err != nil {
		return "", err
	}
	return FormatOutput(output, notFoundMsg), nil
}
