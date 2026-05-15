package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
)

// ConfirmationHandling demonstrates handling sensitive operation confirmations.
func ConfirmationHandling() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg := sdk.DefaultRuntimeConfig()
	// Set permission level to ask for sensitive operations
	cfg.Permissions.DefaultLevel = "ask"

	// Create confirmation handler
	confirmHandler := func(req sdk.ConfirmationRequest) {
		fmt.Printf("\n=== Confirmation Request ===\n")
		fmt.Printf("Type: %s\n", req.Type)
		if req.ToolName != "" {
			fmt.Printf("Tool: %s\n", req.ToolName)
			fmt.Printf("Params: %v\n", req.Params)
		}
		if len(req.PlanSteps) > 0 {
			fmt.Printf("Plan Goal: %s\n", req.PlanGoal)
			for i, step := range req.PlanSteps {
				fmt.Printf("  Step %d: %s\n", i+1, step)
			}
		}
		fmt.Printf("\nAuto-approving for demo...\n")

		// Send approval response
		req.ResponseCh <- true
	}

	// Create runtime with confirmation handler
	runtime, err := sdk.NewRuntime(cfg,
		sdk.WithConfirmationHandler(confirmHandler),
	)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	if err := runtime.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	defer runtime.Shutdown()

	// Process potentially sensitive operation
	events, err := runtime.Process(ctx, "Create a file named test.txt with content 'Hello World'")
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}

	var response strings.Builder
	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeConfirmationRequest:
			fmt.Println("Confirmation event received")
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