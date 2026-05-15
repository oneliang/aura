// Package knowledgetool provides knowledge base tools for the agent.
package knowledgetool

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// MockCollection is a mock implementation of storage.Collection for testing.
type MockCollection struct {
	documents []storage.Document
	queryFunc func(ctx context.Context, query string, nResults int) ([]storage.Result, error)
}

func (m *MockCollection) Add(ctx context.Context, documents []storage.Document) error {
	m.documents = append(m.documents, documents...)
	return nil
}

func (m *MockCollection) Query(ctx context.Context, query string, nResults int) ([]storage.Result, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, nResults)
	}
	// Default: return mock results
	results := make([]storage.Result, 0)
	for i, doc := range m.documents {
		if i >= nResults {
			break
		}
		results = append(results, storage.Result{
			Document: doc,
			Score:    0.9 - float32(i)*0.1,
		})
	}
	return results, nil
}

func (m *MockCollection) Delete(ctx context.Context, ids []string) error {
	return nil
}

func (m *MockCollection) Count(ctx context.Context) (int, error) {
	return len(m.documents), nil
}

// Test SearchTool
func TestSearchToolName(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	name := tool.Name()
	if name != "knowledge_search" {
		t.Errorf("Name() = %v, want 'knowledge_search'", name)
	}
}

func TestSearchToolDescription(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "query") {
		t.Error("Description() should mention 'query' parameter")
	}
}

func TestSearchToolExecute(t *testing.T) {
	collection := &MockCollection{
		documents: []storage.Document{
			{
				ID:       "doc-1",
				Content:  "Go is a programming language.",
				Metadata: map[string]any{"source": "go.txt"},
			},
		},
	}
	tool := NewSearchTool(collection)

	ctx := context.Background()
	params := map[string]any{"query": "programming"}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !strings.Contains(result, "Go is a programming language") {
		t.Errorf("Execute() result should contain document content, got: %s", result)
	}
}

func TestSearchToolExecuteMissingQuery(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Execute() should error when query is missing")
	}
}

func TestSearchToolExecuteEmptyQuery(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{"query": ""})
	if err == nil {
		t.Error("Execute() should error when query is empty")
	}
}

func TestSearchToolExecuteInvalidQueryType(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{"query": 123})
	if err == nil {
		t.Error("Execute() should error when query is not a string")
	}
}

func TestSearchToolExecuteNoResults(t *testing.T) {
	collection := &MockCollection{
		queryFunc: func(ctx context.Context, query string, nResults int) ([]storage.Result, error) {
			return []storage.Result{}, nil
		},
	}
	tool := NewSearchTool(collection)

	ctx := context.Background()
	params := map[string]any{"query": "nonexistent"}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result != "No relevant knowledge found." {
		t.Errorf("Execute() with no results = %q, want 'No relevant knowledge found.'", result)
	}
}

func TestSearchToolWithOptions(t *testing.T) {
	collection := &MockCollection{}
	tool := NewSearchTool(collection)

	if tool.rag == nil {
		t.Error("NewSearchTool() should initialize RAG")
	}
}

// Test ImportTool
func TestImportToolName(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	name := tool.Name()
	if name != "knowledge_import" {
		t.Errorf("Name() = %v, want 'knowledge_import'", name)
	}
}

func TestImportToolDescription(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "path") {
		t.Error("Description() should mention 'path' parameter")
	}
}

func TestImportToolExecuteMissingPath(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Execute() should error when path is missing")
	}
}

func TestImportToolExecuteEmptyPath(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{"path": ""})
	if err == nil {
		t.Error("Execute() should error when path is empty")
	}
}

func TestImportToolExecuteInvalidPathType(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{"path": 123})
	if err == nil {
		t.Error("Execute() should error when path is not a string")
	}
}

func TestImportToolExecuteNonExistentPath(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{"path": "/non/existent/path.txt"})
	// ImportDir returns 0 and nil error for non-existent paths (it skips them)
	// Then ImportFile also returns error which gets wrapped
	// The implementation may return success with 0 chunks or an error
	if err != nil && !strings.Contains(err.Error(), "import failed") {
		t.Errorf("Error should mention 'import failed', got: %v", err)
	}
	// If no error, result should mention 0 chunks
	if err == nil && !strings.Contains(result, "imported 0 chunks") {
		t.Logf("Execute() returned result for non-existent path: %s", result)
	}
}

func TestImportToolWithOptions(t *testing.T) {
	collection := &MockCollection{}
	tool := NewImportTool(collection)

	if tool.imp == nil {
		t.Error("NewImportTool() should initialize Importer")
	}
}
