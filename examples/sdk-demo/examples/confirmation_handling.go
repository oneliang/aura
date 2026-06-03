package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/events"
)

// ConfirmationHandling demonstrates handling interaction requests via event stream.
func ConfirmationHandling() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()
	// Set permission level to ask for sensitive operations
	cfg.Permissions.DefaultLevel = "ask"

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
	eventStream := runtime.Events()

	// Generate request ID
	requestID := uuid.New().String()

	// Send user input
	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "Create a file named test.txt with content 'Hello World'", requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
	}

	// Process events and handle interaction requests
	var response strings.Builder
	for ev := range eventStream {
		switch ev.Type() {
		case sdk.EventTypeInteractionRequest:
			// Handle interaction request (tool confirmation, plan review, etc.)
			interactionType := ev.InteractionType()
			fmt.Printf("\n=== Interaction Request ===\n")
			fmt.Printf("Type: %s\n", interactionType)
			fmt.Printf("Request ID: %s\n", ev.RequestID())

			if extra := ev.Extra(); extra != nil {
				if toolName, ok := extra["tool_name"].(string); ok {
					fmt.Printf("Tool: %s\n", toolName)
				}
				if toolParams, ok := extra["tool_params"].(map[string]any); ok {
					fmt.Printf("Params: %v\n", toolParams)
				}
				if planGoal, ok := extra["plan_goal"].(string); ok {
					fmt.Printf("Plan Goal: %s\n", planGoal)
				}
				if planSteps, ok := extra["plan_steps"].([]string); ok {
					for i, step := range planSteps {
						fmt.Printf("  Step %d: %s\n", i+1, step)
					}
				}
			}

			fmt.Printf("\nAuto-approving for demo...\n")

			// Send approval response
			respExtra := map[string]any{
				"approved": true,
				"type":     interactionType,
			}
			respEvent := events.NewEventWithExtra(events.EventTypeInteractionResponse, "", respExtra, ev.RequestID())
			if err := runtime.SendEvent(ctx, respEvent); err != nil {
				fmt.Printf("Error sending response: %v\n", err)
			}

		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseChunk:
			response.WriteString(ev.Content())
		case sdk.EventTypeError:
			fmt.Printf("Error: %s\n", ev.Content())
		case sdk.EventTypeDone:
			break
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\nResponse: %s\n", response.String())
	}

	return nil
}