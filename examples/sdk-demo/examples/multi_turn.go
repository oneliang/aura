package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/core/pkg/sdk"
)

// MultiTurn demonstrates multi-turn conversation with session persistence.
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

	// Conversation loop
	inputs := []string{
		"My name is Alice",
		"What's my name?",
		"Tell me what we discussed earlier",
	}

	for i, input := range inputs {
		fmt.Printf("\n=== Turn %d ===\n", i+1)
		fmt.Printf("User: %s\n", input)

		events, err := runtime.Process(ctx, input)
		if err != nil {
			return fmt.Errorf("process turn %d: %w", i+1, err)
		}

		var response strings.Builder
		for ev := range events {
			if ev.Type() == sdk.EventTypeResponse || ev.Type() == sdk.EventTypeResponseChunk {
				response.WriteString(ev.Content())
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