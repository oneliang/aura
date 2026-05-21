// Package commands provides CLI commands for orchestrator management.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/spf13/cobra"
)

var (
	orchestratorMaxAgents    int
	orchestratorLLMModel     string
	orchestratorWorkspaceDir string
)

// OrchestratorCmd is the root command for orchestrator management.
var OrchestratorCmd = &cobra.Command{
	Use:   "orchestrator",
	Short: "Manage multi-agent orchestrator",
	Long:  `Commands for managing the multi-agent orchestrator system.`,
}

var orchestratorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the orchestrator",
	Run:   runOrchestratorStart,
}

var orchestratorStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the orchestrator",
	Run:   runOrchestratorStop,
}

var orchestratorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show orchestrator status",
	Run:   runOrchestratorStatus,
}

var orchestratorAgentListCmd = &cobra.Command{
	Use:   "agent list",
	Short: "List all sub-agents",
	Run:   runOrchestratorAgentList,
}

var orchestratorAgentStatusCmd = &cobra.Command{
	Use:   "agent status <id>",
	Short: "Show sub-agent status",
	Args:  cobra.ExactArgs(1),
	Run:   runOrchestratorAgentStatus,
}

var orchestratorDocListCmd = &cobra.Command{
	Use:   "doc list",
	Short: "List collaboration documents",
	Run:   runOrchestratorDocList,
}

var orchestratorDocShowCmd = &cobra.Command{
	Use:   "doc show <id>",
	Short: "Show document details",
	Args:  cobra.ExactArgs(1),
	Run:   runOrchestratorDocShow,
}

func init() {
	orchestratorStartCmd.Flags().IntVar(&orchestratorMaxAgents, "max-agents", 5, "Maximum number of sub-agents")
	orchestratorStartCmd.Flags().StringVar(&orchestratorLLMModel, "llm-model", "", "LLM model for sub-agents (optional override)")
	orchestratorStartCmd.Flags().StringVar(&orchestratorWorkspaceDir, "workspace", "", "Custom workspace directory")

	orchestratorDocListCmd.Flags().String("status", "", "Filter by status (pending|in_progress|completed|rejected|blocked)")

	OrchestratorCmd.AddCommand(orchestratorStartCmd)
	OrchestratorCmd.AddCommand(orchestratorStopCmd)
	OrchestratorCmd.AddCommand(orchestratorStatusCmd)
	OrchestratorCmd.AddCommand(orchestratorAgentListCmd)
	OrchestratorCmd.AddCommand(orchestratorAgentStatusCmd)
	OrchestratorCmd.AddCommand(orchestratorDocListCmd)
	OrchestratorCmd.AddCommand(orchestratorDocShowCmd)
}

// loadConfig loads the Aura configuration.
func loadConfig() (*config.Config, error) {
	configPath := ffp.MustAuraHomePath(constants.DefaultConfigFile)
	return config.Load(configPath)
}

func runOrchestratorStart(cmd *cobra.Command, args []string) {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}

	// Apply overrides
	if orchestratorMaxAgents > 0 {
		cfg.Orchestrator.MaxSubAgents = orchestratorMaxAgents
	}
	if orchestratorLLMModel != "" {
		cfg.Orchestrator.SubAgentLLM = &config.LLMConfig{
			Model: orchestratorLLMModel,
		}
	}
	if orchestratorWorkspaceDir != "" {
		cfg.Orchestrator.WorkspaceDir = orchestratorWorkspaceDir
	}

	// Enable orchestrator
	cfg.Orchestrator.Enabled = true

	// Create and start orchestrator
	orch, err := sdk.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating orchestrator: %v\n", err)
		return
	}

	ctx := context.Background()
	if err := orch.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting orchestrator: %v\n", err)
		return
	}

	fmt.Println("Orchestrator started successfully!")
	fmt.Printf("  Workspace: %s\n", orch.GetWorkspace().BaseDir())
	fmt.Printf("  Max agents: %d\n", cfg.Orchestrator.MaxSubAgents)
	fmt.Println("\nPress Ctrl+C to stop")

	// Keep running until interrupted
	select {}
}

func runOrchestratorStop(cmd *cobra.Command, args []string) {
	fmt.Println("Orchestrator stop command - not yet implemented")
	fmt.Println("For now, stop the running orchestrator process (Ctrl+C)")
}

func runOrchestratorStatus(cmd *cobra.Command, args []string) {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}

	if !cfg.Orchestrator.Enabled {
		fmt.Println("Orchestrator is disabled. Enable it in config.yaml or use 'aura orchestrator start'")
		return
	}

	// For now, just show configuration
	// A more advanced implementation would connect to a running orchestrator
	fmt.Println("Orchestrator Configuration:")
	fmt.Printf("  Enabled: %v\n", cfg.Orchestrator.Enabled)
	fmt.Printf("  Max sub-agents: %d\n", cfg.Orchestrator.MaxSubAgents)
	fmt.Printf("  Workspace dir: %s\n", cfg.Orchestrator.WorkspaceDir)
	fmt.Printf("  Supervision interval: %v\n", cfg.Orchestrator.SupervisionInterval)
	fmt.Printf("  Stale doc threshold: %v\n", cfg.Orchestrator.StaleDocThreshold)
	fmt.Printf("  Auto cleanup: %v\n", cfg.Orchestrator.AutoCleanup)

	if cfg.Orchestrator.SubAgentLLM != nil {
		fmt.Printf("  Sub-agent LLM: %s\n", cfg.Orchestrator.SubAgentLLM.Model)
	}
}

func runOrchestratorAgentList(cmd *cobra.Command, args []string) {
	fmt.Println("Agent list command - requires running orchestrator")
	fmt.Println("This command will be implemented when orchestrator runs as a service")
}

func runOrchestratorAgentStatus(cmd *cobra.Command, args []string) {
	fmt.Printf("Agent status for %s - requires running orchestrator\n", args[0])
}

func runOrchestratorDocList(cmd *cobra.Command, args []string) {
	status, _ := cmd.Flags().GetString("status")

	// Create a temporary orchestrator to access docs
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}

	// For read-only operations, we can create a minimal orchestrator
	// Just to access the doc store
	orch, err := sdk.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating orchestrator: %v\n", err)
		return
	}
	defer orch.Stop()

	var docs []*sdk.CollaboDoc
	if status != "" {
		var statusFilter sdk.DocStatus
		switch status {
		case "pending":
			statusFilter = sdk.DocStatusPending
		case "in_progress":
			statusFilter = sdk.DocStatusInProgress
		case "completed":
			statusFilter = sdk.DocStatusCompleted
		case "rejected":
			statusFilter = sdk.DocStatusRejected
		case "blocked":
			statusFilter = sdk.DocStatusBlocked
		default:
			fmt.Fprintf(os.Stderr, "Invalid status: %s\n", status)
			return
		}
		docs, err = orch.ListDocs(statusFilter)
	} else {
		docs, err = orch.ListDocs()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing docs: %v\n", err)
		return
	}

	if len(docs) == 0 {
		fmt.Println("No documents found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tTYPE\tTITLE\tTO")
	fmt.Fprintln(w, "--\t------\t--------\t----\t-----\t--")

	for _, doc := range docs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			doc.ID,
			doc.Status,
			doc.Priority,
			doc.Type,
			utils.Truncate(doc.Title, 30),
			doc.To,
		)
	}
	w.Flush()
}

func runOrchestratorDocShow(cmd *cobra.Command, args []string) {
	docID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}

	orch, err := sdk.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating orchestrator: %v\n", err)
		return
	}
	defer orch.Stop()

	doc, err := orch.GetDoc(docID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading document: %v\n", err)
		return
	}

	// Format as JSON for readability
	data, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Println(string(data))
}

// GetOrchestratorFromContext retrieves orchestrator from command context.
// This is a placeholder for future implementation when orchestrator runs as a service.
func GetOrchestratorFromContext() (*sdk.Orchestrator, error) {
	// For now, create a new orchestrator from config
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return sdk.NewOrchestrator(cfg)
}
