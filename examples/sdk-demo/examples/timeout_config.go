package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// TimeoutConfig demonstrates LLM timeout configuration for long-running tasks.
// Shows how to configure HTTP client timeout separately from context timeout.
func TimeoutConfig() error {
	// Context timeout for the entire operation (15 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// 1. Create runtime configuration
	cfg := sdk.DefaultRuntimeConfig()

	// 2. Configure LLM timeout for long-running reasoning tasks
	// This is the HTTP client timeout for LLM API calls
	cfg.LLM.Timeout = 600 * time.Second // 10 minutes

	fmt.Println("=== LLM Timeout Configuration Demo ===")
	fmt.Printf("LLM HTTP timeout: %v (for API calls)\n", cfg.LLM.Timeout)
	fmt.Printf("Context timeout: %v (for entire operation)\n", 15*time.Minute)

	// 3. Create runtime
	runtime, err := sdk.NewRuntime(cfg)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	// 4. Initialize runtime
	if err := runtime.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	defer runtime.Shutdown()

	// 5. Start event stream
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	defer runtime.Stop(ctx)

	// 6. Get output event stream
	events := runtime.Events()

	// 7. Generate request ID
	requestID := uuid.New().String()

	// 8. Send a complex reasoning task that may take longer
	task := "Analyze the following problem step by step and provide a detailed solution: " +
		"Design a microservices architecture for an e-commerce platform with high scalability requirements. " +
		"Consider: service boundaries, data consistency, fault tolerance, and performance optimization."

	fmt.Printf("\nTask: %s\n\n", task)

	err = runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, task, requestID))
	if err != nil {
		return fmt.Errorf("send event: %w", err)
	}

	// 9. Consume event stream
	var response strings.Builder
	startTime := time.Now()

	for ev := range events {
		switch ev.Type() {
		case sdk.EventTypeThinkingStart:
			fmt.Println("[Thinking...]")
		case sdk.EventTypeThinkingEnd:
			fmt.Println("[Thinking complete]")
		case sdk.EventTypeResponse:
			response.WriteString(ev.Content())
		case sdk.EventTypeResponseChunk:
			chunk := ev.Content()
			response.WriteString(chunk)
			fmt.Print(chunk) // Stream output in real-time
		case sdk.EventTypeError:
			fmt.Printf("\nError: %s\n", ev.Content())
			return fmt.Errorf("runtime error: %s", ev.Content())
		case sdk.EventTypeDone:
			duration := time.Since(startTime)
			fmt.Printf("\n\nProcessing complete in %v\n", duration)
			return nil
		}
	}

	if response.Len() > 0 {
		fmt.Printf("\n\nFull response length: %d characters\n", response.Len())
		fmt.Printf("Processing time: %v\n", time.Since(startTime))
	}

	return nil
}