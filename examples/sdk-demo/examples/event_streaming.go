package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// EventStreaming demonstrates real-time event handling with the new event stream pattern.
func EventStreaming() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()

	runtime, err := sdk.NewRuntime(cfg)
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

	// Get output event stream
	events := runtime.Events()

	// Generate request ID
	requestID := uuid.New().String()

	fmt.Println("Processing with real-time event display...")

	// Send user input
	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "List files in the current directory and summarize what you find", requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
	}

	// Process events in real-time
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeThinkingStart:
			fmt.Print("\n[Thinking started...]\n")
		case sdk.EventTypeThinkingChunk:
			fmt.Printf("  %s", ev.Content())
		case sdk.EventTypeThinkingEnd:
			fmt.Print("\n[Thinking complete]\n")
		case sdk.EventTypeToolStart:
			extra := ev.Extra()
			if toolName, ok := extra["tool"].(string); ok {
				fmt.Printf("\n[Tool: %s]\n", toolName)
			}
		case sdk.EventTypeToolEnd:
			fmt.Printf("[Tool done]\n")
		case sdk.EventTypeResponseStart:
			fmt.Print("\n[Response: ")
		case sdk.EventTypeResponseChunk:
			fmt.Print(ev.Content())
		case sdk.EventTypeResponseEnd:
			fmt.Print("]\n")
		case sdk.EventTypeStep:
			extra := ev.Extra()
			if step, ok := extra["step"].(int); ok {
				fmt.Printf("\n[Step %d]\n", step)
			}
		case sdk.EventTypeDone:
			fmt.Println("\n[Processing complete]")
			return nil
		}
	}

	return nil
}