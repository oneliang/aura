// Package commands provides CLI commands for aura.
package commands

import (
	"context"
	"fmt"
	"os"

	initpkg "github.com/oneliang/aura/commands/pkg/init"
	"github.com/spf13/cobra"
)

// InitCmd is the command for initializing AURA.md.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize AURA.md with codebase documentation",
	Long: `Scan project structure and generate AURA.md file.

The generated file includes:
- Build & Development Commands (from Makefile/package.json)
- Architecture Overview (module structure)
- Key Interfaces (with file paths)
- Configuration (config file locations)
- Development Patterns (project-specific)

This command helps Aura quickly understand your project structure
and coding conventions, reducing repetitive context setup.`,
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

	// Execute init handler directly
	handler := &initpkg.Handler{}

	params := map[string]any{
		"cwd":   cwd,
		"force": initForce,
	}

	result, err := handler.Execute(ctx, "command_init", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}