// Package commands provides CLI commands for knowledge base management.
package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/oneliang/aura/knowledge/pkg/embedding"
	"github.com/oneliang/aura/knowledge/pkg/importer"
	"github.com/oneliang/aura/knowledge/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/user"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/spf13/cobra"
)

// KnowledgeCmd is the root command for knowledge base management.
var KnowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Manage the personal knowledge base",
}

var importCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import a file or directory into the knowledge base",
	Args:  cobra.ExactArgs(1),
	Run:   runImport,
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the personal knowledge base",
	Args:  cobra.ExactArgs(1),
	Run:   runSearch,
}

var kbStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show knowledge base statistics",
	Run:   runKbStats,
}

func init() {
	KnowledgeCmd.AddCommand(importCmd)
	KnowledgeCmd.AddCommand(searchCmd)
	KnowledgeCmd.AddCommand(kbStatsCmd)
}

// openCollection opens (or creates) the default knowledge collection for the current user.
func openCollection(ctx context.Context) (*storage.ChromemCollection, error) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Initialize user context if not already set
	InitUserContext(cmdCtx)

	userID := cmdCtx.UserID

	// Determine collection name using helper
	collectionName := user.GetUserCollectionName(userID)

	// Determine data directory
	dataDir := ffp.MustAuraHomePath(constants.DirKnowledge)
	if userID != "" {
		dataDir = ffp.MustAuraHomePath(constants.DirUsers, userID, "knowledge", user.KnowledgePrivate)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create knowledge directory: %w", err)
	}

	embFn := embedding.OllamaEmbeddingFunc(cmdCtx.Config.LLM.BaseURL, "nomic-embed-text")

	return cmdCtx.KnowledgeStoreFactory.NewCollection(ctx, storage.ChromemOptions{
		DataDir:       dataDir,
		Name:          collectionName,
		EmbeddingFunc: embFn,
	})
}

func runImport(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Initialize user context
	InitUserContext(cmdCtx)

	path := args[0]

	// CLI-specific: Check if path is a file or directory and handle accordingly
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// For directory imports, we need to handle it at CLI level since
	// the CommandProvider only handles single file imports
	ctx := context.Background()
	if info.IsDir() {
		// Directory import - handle directly in CLI layer
		col, err := openCollection(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Opening knowledge base...\n")
		fmt.Printf("Importing directory %s...\n", path)

		imp := importer.New(col)
		n, err := imp.ImportDir(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Import error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Done. Imported %d chunks from directory.\n", n)
		return
	}

	// File import - use CommandProvider
	params := map[string]any{"path": path}
	result, err := cmdCtx.CommandProvider.Execute(ctx, "command_knowledge_import", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runSearch(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Initialize user context
	InitUserContext(cmdCtx)

	query := args[0]
	params := map[string]any{"query": query}

	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_knowledge_search", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runKbStats(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Initialize user context
	InitUserContext(cmdCtx)

	// Determine storage path (user-aware)
	userID := cmdCtx.UserID
	var dataDir string
	if userID != "" {
		dataDir = ffp.MustAuraHomePath(constants.DirUsers, userID, "knowledge", user.KnowledgePrivate)
	} else {
		dataDir = ffp.MustAuraHomePath(constants.DirKnowledge)
	}

	// Try to use CommandProvider for stats, fallback to direct display
	if cmdCtx.CommandProvider != nil {
		result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_knowledge_stats", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(result)
	} else {
		// Fallback: just show storage info
		fmt.Println("Knowledge Base Statistics")
		fmt.Println("=========================")
		fmt.Println("(Stats not available in this mode)")
	}
	fmt.Printf("  Storage: %s\n", dataDir)
}
