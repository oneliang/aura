// Package knowledgetool provides knowledge base tools for the agent.
package knowledgetool

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/knowledge/pkg/importer"
	"github.com/oneliang/aura/knowledge/pkg/retrieval"
	"github.com/oneliang/aura/knowledge/pkg/storage"
	tools "github.com/oneliang/aura/tools/pkg"
)

// SearchTool searches the knowledge base.
type SearchTool struct {
	rag *retrieval.RAG
}

// NewSearchTool creates a new knowledge search tool.
func NewSearchTool(collection storage.Collection, opts ...retrieval.Option) *SearchTool {
	return &SearchTool{rag: retrieval.New(collection, opts...)}
}

func (t *SearchTool) Name() string { return "knowledge_search" }
func (t *SearchTool) Description() string {
	return "Search the personal knowledge base. Parameters: query (string, required)"
}

// Execute searches the knowledge base and returns matching content.
func (t *SearchTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "query parameter is required"}, nil
	}
	context, results, err := t.rag.Query(ctx, query)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}
	if len(results) == 0 {
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "No relevant knowledge found."}, nil
	}
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: context}, nil
}

// ImportTool imports documents into the knowledge base.
type ImportTool struct {
	imp *importer.Importer
}

// NewImportTool creates a new knowledge import tool.
func NewImportTool(collection storage.Collection, opts ...importer.Option) *ImportTool {
	return &ImportTool{imp: importer.New(collection, opts...)}
}

func (t *ImportTool) Name() string { return "knowledge_import" }
func (t *ImportTool) Description() string {
	return "Import a file or directory into the knowledge base. Parameters: path (string, required)"
}

// Execute imports a path (file or directory) into the knowledge base.
func (t *ImportTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "path parameter is required"}, nil
	}

	// Detect file vs directory
	var n int
	var err error
	n, err = t.imp.ImportDir(ctx, path)
	if err != nil {
		// Try as single file
		n, err = t.imp.ImportFile(ctx, path)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("import failed: %v", err)}, nil
		}
	}
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Successfully imported %d chunks from %s", n, path)}, nil
}
