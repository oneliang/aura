// Package commands provides CLI commands for aura.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	initpkg "github.com/oneliang/aura/commands/pkg/init"
	"github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/spf13/cobra"
)

// InitCmd is the command for initializing AURA.md.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize AURA.md with workspace documentation",
	Long: `Explore the current workspace and generate AURA.md file.

The generated file adapts to the workspace type:
- Code projects: Build commands, architecture, entry points
- Document libraries: Document organization, key files, purpose
- Mixed/Other: Appropriate structure based on content

This command helps Aura quickly understand your workspace
and reduces repetitive context setup.`,
	Run: runInit,
}

var initForce bool

func init() {
	InitCmd.Flags().BoolVar(&initForce, "force", false, "Force regeneration even if AURA.md exists")
}

func runInit(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	auraMdPath := filepath.Join(cwd, "AURA.md")

	// Check if AURA.md already exists
	if !initForce {
		if _, err := os.Stat(auraMdPath); err == nil {
			fmt.Printf("AURA.md already exists at: %s\nUse --force to regenerate.\n", auraMdPath)
			return
		}
	}

	// Load config (empty string uses default path ~/.aura/config.yaml)
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Build init-specific runtime config
	initCfg := sdk.FromConfig(cfg)
	initCfg.SystemPrompt = initpkg.InitSystemPrompt
	initCfg.EnableSubAgent = false
	initCfg.SessionID = "" // No persistence
	initCfg.Agent.PlanningMode = string(engine.ModeImplicit)

	// Create logger
	log := logger.NewNamed(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
		Module: "init",
	})

	// Create runtime
	rt, err := sdk.NewRuntime(initCfg,
		sdk.WithAutoApprove(),
		sdk.WithLogger(log),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating runtime: %v\n", err)
		os.Exit(1)
	}

	// Initialize runtime
	if err := rt.Initialize(ctx); err != nil {
		rt.Shutdown()
		fmt.Fprintf(os.Stderr, "Error initializing runtime: %v\n", err)
		os.Exit(1)
	}

	// Build prompt
	prompt := initpkg.BuildInitPrompt(cwd)

	fmt.Println("Exploring workspace...")

	// Process and collect events
	// Start runtime event stream
	if err := rt.Start(ctx); err != nil {
		rt.Shutdown()
		fmt.Fprintf(os.Stderr, "Error starting runtime: %v\n", err)
		os.Exit(1)
	}

	// Send user input event
	requestID := fmt.Sprintf("init_%d", time.Now().UnixNano())
	userEvent := events.NewEvent(events.EventTypeUserInput, prompt, requestID)
	if err := rt.SendEvent(ctx, userEvent); err != nil {
		rt.Stop(ctx)
		rt.Shutdown()
		fmt.Fprintf(os.Stderr, "Error sending event: %v\n", err)
		os.Exit(1)
	}

	// Collect response content
	var contentBuilder strings.Builder
eventLoop:
	for event := range rt.Events() {
		switch event.Type() {
		case events.EventTypeResponse, events.EventTypeResponseChunk:
			if event.Content() != "" {
				contentBuilder.WriteString(event.Content())
			}
		case events.EventTypeError:
			fmt.Fprintf(os.Stderr, "Error: %s\n", event.Content())
		case events.EventTypeDone:
			break eventLoop
		}
	}
	rt.Stop(ctx)

	// Shutdown runtime
	rt.Shutdown()

	content := contentBuilder.String()
	if content == "" {
		fmt.Fprintf(os.Stderr, "Error: No content generated\n")
		os.Exit(1)
	}

	// Write AURA.md
	if err := os.WriteFile(auraMdPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing AURA.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated AURA.md at: %s\n\nPreview:\n%s\n", auraMdPath, truncatePreview(content, 500))
}

func truncatePreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}