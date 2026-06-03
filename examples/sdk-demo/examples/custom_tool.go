package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
	demotools "github.com/oneliang/aura-sdk-demo/tools"
)

// CustomTool demonstrates registering and using custom tools with the new event stream pattern.
func CustomTool() error {
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

	// Register custom tool after initialization
	weatherTool := demotools.NewWeatherTool()
	if err := runtime.AddTool(weatherTool); err != nil {
		return fmt.Errorf("add tool: %w", err)
	}

	fmt.Println("Registered custom tool: weather")

	// Start event stream
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	defer runtime.Stop(ctx)

	// Get output event stream
	events := runtime.Events()

	// Generate request ID
	requestID := uuid.New().String()

	// Send user input
	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "What's the weather in Tokyo?", requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
	}

	var response strings.Builder
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeToolStart:
			extra := ev.Extra()
			if toolName, ok := extra["tool"].(string); ok {
				fmt.Printf("\n[Tool executing: %s]\n", toolName)
			}
		case sdk.EventTypeToolEnd:
			fmt.Printf("[Tool result: %s]\n", ev.Content())
		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeDone:
			break
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}