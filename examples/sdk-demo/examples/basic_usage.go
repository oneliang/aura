package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
)

// BasicUsage demonstrates minimal SDK integration.
func BasicUsage() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Create runtime configuration
	cfg := sdk.DefaultRuntimeConfig()

	// 2. Create runtime
	runtime, err := sdk.NewRuntime(cfg)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	// 3. Initialize runtime
	if err := runtime.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	defer runtime.Shutdown()

	// 4. Process input
	events, err := runtime.Process(ctx, "Hello, Aura! What can you help me with?")
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}

	// 5. Consume event stream
	var response strings.Builder
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseChunk:
			response.WriteString(ev.Content())
		case sdk.EventTypeError:
			fmt.Printf("Error: %s\n", ev.Content())
		case sdk.EventTypeDone:
			fmt.Println("\nProcessing complete")
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}