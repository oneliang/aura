package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// MultiTurn demonstrates multi-turn conversation with session persistence using event stream.
func MultiTurn() error {
	ctx := context.Background()

	cfg := sdk.DefaultRuntimeConfig()
	cfg.Memory.MaxContext = 50 // Keep conversation history

	sessionID := fmt.Sprintf("demo-%d", time.Now().Unix())

	runtime, err := sdk.NewRuntime(cfg,
		sdk.WithSessionID(sessionID),
		sdk.WithUserID("demo-user"),
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

	// Get output event stream
	events := runtime.Events()

	// Conversation loop
	inputs := []string{
		"My name is Alice",
		"What's my name?",
		"Tell me what we discussed earlier",
	}

	for i, input := range inputs {
		fmt.Printf("\n=== Turn %d ===\n", i+1)
		fmt.Printf("User: %s\n", input)

		// Generate request ID for this turn
		requestID := uuid.New().String()

		// Send user input
		err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, input, requestID))
		if err != nil {
			return fmt.Errorf("send event turn %d: %w", i+1, err)
		}

		// Process events for this turn (match requestID)
		var response strings.Builder
		for ev := range events {
			// Only process events matching our requestID or global events
			if ev.RequestID() == requestID || ev.RequestID() == "" {
				if ev.Type() == sdk.EventTypeResponse || ev.Type() == sdk.EventTypeResponseChunk {
					response.WriteString(ev.Content())
				}
				if ev.Type() == sdk.EventTypeDone {
					break
				}
			}
		}

		fmt.Printf("Aura: %s\n", response.String())

		// Small pause between turns
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Conversation complete ===")
	fmt.Printf("Session ID: %s\n", sessionID)

	return nil
}