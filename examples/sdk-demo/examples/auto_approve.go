package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
)

// AutoApprove demonstrates SDK usage without interactive confirmation.
// Useful for background automation, CI/CD pipelines, or non-interactive environments.
//
// Key points:
// - All tool executions are automatically approved
// - Shell restrictions (denied commands) are still enforced for safety
// - No confirmation handler needed
func AutoApprove() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()

	// Create runtime with auto-approve mode
	// This bypasses all tool confirmation prompts
	// BUT safety restrictions (shell_restrictions.denied_commands) are still enforced
	runtime, err := sdk.NewRuntime(cfg,
		sdk.WithAutoApprove(),
	)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	if err := runtime.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	defer runtime.Shutdown()

	fmt.Println("Auto-approve mode enabled - no confirmation prompts")
	fmt.Println("Safety restrictions still active (dangerous commands blocked)")

	// Process request that would normally require confirmation
	// (e.g., file_write, bash commands)
	events, err := runtime.Process(ctx, "Create a file named demo.txt with content 'Auto-approve test'")
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}

	var response strings.Builder
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeToolStart:
			extra := ev.Extra()
			if toolName, ok := extra["tool"].(string); ok {
				fmt.Printf("[Tool auto-approved: %s]\n", toolName)
			}
		case sdk.EventTypeToolEnd:
			fmt.Printf("[Tool completed]\n")
		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseChunk:
			response.WriteString(ev.Content())
		case sdk.EventTypeDone:
			fmt.Println("\nProcessing complete")
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}