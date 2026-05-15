package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
)

// EventStreaming demonstrates real-time event handling.
func EventStreaming() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()

	// Create event handler for real-time updates
	eventHandler := func(ev sdk.Event) {
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
		}
	}

	runtime, err := sdk.NewRuntime(cfg,
		sdk.WithEventHandler(eventHandler),
	)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	if err := runtime.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	defer runtime.Shutdown()

	fmt.Println("Processing with real-time event display...")

	events, err := runtime.Process(ctx, "List files in the current directory and summarize what you find")
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}

	// Wait for completion
	for ev := range events {
		if ev.Type() == sdk.EventTypeDone {
			fmt.Println("\n[Processing complete]")
		}
	}

	return nil
}