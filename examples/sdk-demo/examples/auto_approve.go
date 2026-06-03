package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// AutoApprove demonstrates SDK usage without interactive confirmation.
// With the new event stream pattern, interaction requests timeout and auto-approve.
// Useful for background automation, CI/CD pipelines, or non-interactive environments.
func AutoApprove() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()

	// Create runtime with auto-approve mode
	// Interaction requests will timeout and auto-approve
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

	// Start event stream
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	defer runtime.Stop(ctx)

	fmt.Println("Auto-approve mode enabled - interaction requests timeout and auto-approve")
	fmt.Println("Safety restrictions still active (dangerous commands blocked)")

	// Get output event stream
	events := runtime.Events()

	// Generate request ID
	requestID := uuid.New().String()

	// Send user input
	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "Create a file named demo.txt with content 'Auto-approve test'", requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
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
			break
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}