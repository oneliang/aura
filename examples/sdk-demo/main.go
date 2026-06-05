package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oneliang/aura-sdk-demo/examples"
)

func main() {
	example := flag.String("example", "basic", "Example to run: basic, tool, confirm, stream, conversation, timeout, auto")
	flag.Parse()

	fmt.Printf("=== Aura SDK Demo: %s ===\n\n", *example)

	var err error
	switch *example {
	case "basic":
		err = examples.BasicUsage()
	case "tool":
		err = examples.CustomTool()
	case "confirm":
		err = examples.ConfirmationHandling()
	case "stream":
		err = examples.EventStreaming()
	case "conversation":
		err = examples.MultiTurn()
	case "timeout":
		err = examples.TimeoutConfig()
	case "auto":
		err = examples.AutoApprove()
	default:
		fmt.Printf("Unknown example: %s\n\nAvailable examples:\n", *example)
		fmt.Println("  basic        - Minimal SDK integration")
		fmt.Println("  tool         - Custom tool registration")
		fmt.Println("  confirm      - Confirmation handling")
		fmt.Println("  stream       - Real-time event display")
		fmt.Println("  conversation - Multi-turn chat")
		fmt.Println("  timeout      - LLM timeout configuration (for long-running tasks)")
		fmt.Println("  auto         - Auto-approve mode (non-interactive)")
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Demo complete ===")
}