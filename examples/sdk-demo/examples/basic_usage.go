package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// BasicUsage demonstrates minimal SDK integration with the new event stream pattern.
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

	// 4. Start event stream
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	defer runtime.Stop(ctx)

	// 5. Get output event stream
	events := runtime.Events()

	// 6. Generate request ID for this interaction
	requestID := uuid.New().String()

	// 7. Send user input event
	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "Hello, Aura! What can you help me with?", requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
	}

	// 8. Consume event stream
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
			return nil
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}