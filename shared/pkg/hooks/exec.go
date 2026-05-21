package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// maxOutputBytes limits hook stdout/stderr capture to 1MB.
	maxOutputBytes = 1 * 1024 * 1024
	// defaultHookTimeout is the default timeout for hook execution.
	defaultHookTimeout = 30 * time.Second
)

// executeCommand forks a subprocess to run the hook command.
// It passes the input as JSON on stdin, captures stdout/stderr, and
// attempts to parse stdout as a HookOutput JSON object.
func executeCommand(ctx context.Context, cmd string, input any, timeoutSec int) (*HookResult, error) {
	timeout := defaultHookTimeout
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdinData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook input: %w", err)
	}

	sh := exec.CommandContext(cmdCtx, "sh", "-c", cmd)
	sh.Stdin = bytes.NewReader(stdinData)

	var stdoutBuf, stderrBuf bytes.Buffer
	sh.Stdout = &stdoutBuf
	sh.Stderr = &stderrBuf

	if err := sh.Run(); err != nil {
		// Context timeout
		if cmdCtx.Err() == context.DeadlineExceeded {
			return &HookResult{
				ExitCode: -1,
				Stderr:   "hook timed out",
			}, nil
		}
		// Get exit code if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			result := &HookResult{
				ExitCode: exitErr.ExitCode(),
				Stdout:   truncateString(stdoutBuf.String(), maxOutputBytes),
				Stderr:   truncateString(stderrBuf.String(), maxOutputBytes),
			}
			result.Parsed = parseHookOutput(result.Stdout)
			return result, nil
		}
		return nil, fmt.Errorf("hook command failed: %w", err)
	}

	result := &HookResult{
		ExitCode: 0,
		Stdout:   truncateString(stdoutBuf.String(), maxOutputBytes),
		Stderr:   truncateString(stderrBuf.String(), maxOutputBytes),
	}
	result.Parsed = parseHookOutput(result.Stdout)
	return result, nil
}

// parseHookOutput attempts to parse stdout as a HookOutput JSON object.
// If parsing fails, returns nil — the raw stdout is still available in HookResult.Stdout.
func parseHookOutput(stdout string) *HookOutput {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil
	}
	var output HookOutput
	if err := json.Unmarshal([]byte(trimmed), &output); err != nil {
		return nil
	}
	return &output
}

// truncateString truncates s to at most maxBytes.
func truncateString(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "... (truncated)"
}
