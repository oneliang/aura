// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/knowledge/pkg"
	"github.com/oneliang/aura/knowledge/pkg/importer"
	"github.com/oneliang/aura/knowledge/pkg/retrieval"
)

// KnowledgeExecutor handles knowledge base commands.
type KnowledgeExecutor struct {
	factory knowledge.CollectionFactory
	userID  string
}

// NewKnowledgeExecutor creates a new knowledge executor.
func NewKnowledgeExecutor(factory knowledge.CollectionFactory, userID string) *KnowledgeExecutor {
	return &KnowledgeExecutor{
		factory: factory,
		userID:  userID,
	}
}

// ExecuteCommand executes a knowledge command.
// Commands: search, import, stats
func (e *KnowledgeExecutor) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	switch cmd {
	case "search":
		query, _ := params["query"].(string)
		return e.search(ctx, query)
	case "import":
		path, _ := params["path"].(string)
		return e.importKnowledge(ctx, path)
	case "stats":
		return e.stats(ctx)
	default:
		return "", fmt.Errorf("unknown knowledge command: %s", cmd)
	}
}

// search searches the knowledge base.
func (e *KnowledgeExecutor) search(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Open knowledge collection
	col, err := e.factory.NewCollection(ctx, e.userID)
	if err != nil {
		return "", fmt.Errorf("failed to open knowledge base: %w", err)
	}

	// Use RAG for retrieval
	rag := retrieval.New(col, retrieval.WithTopK(5))
	contextText, results, err := rag.Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return "No results found.", nil
	}

	return contextText, nil
}

// importKnowledge imports a file into the knowledge base.
func (e *KnowledgeExecutor) importKnowledge(ctx context.Context, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Open knowledge collection
	col, err := e.factory.NewCollection(ctx, e.userID)
	if err != nil {
		return "", fmt.Errorf("failed to open knowledge base: %w", err)
	}

	// Create importer with collection
	imp := importer.New(col)
	count, err := imp.ImportFile(ctx, path)
	if err != nil {
		return "", fmt.Errorf("import failed: %w", err)
	}

	return fmt.Sprintf("Imported %d documents from %s", count, path), nil
}

// stats returns knowledge base statistics.
func (e *KnowledgeExecutor) stats(ctx context.Context) (string, error) {
	col, err := e.factory.NewCollection(ctx, e.userID)
	if err != nil {
		return "", fmt.Errorf("failed to open knowledge base: %w", err)
	}

	// Get document count
	count, err := col.Count(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get count: %w", err)
	}

	return fmt.Sprintf("Knowledge base statistics:\n  Documents: %d", count), nil
}
