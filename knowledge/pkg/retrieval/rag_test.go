// Package retrieval provides RAG (Retrieval Augmented Generation) functionality.
package retrieval

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// MockCollection is a mock implementation of storage.Collection for testing.
type MockCollection struct {
	documents []storage.Document
}

func (m *MockCollection) Add(ctx context.Context, documents []storage.Document) error {
	m.documents = append(m.documents, documents...)
	return nil
}

func (m *MockCollection) Query(ctx context.Context, query string, nResults int) ([]storage.Result, error) {
	results := make([]storage.Result, 0)
	for i, doc := range m.documents {
		if i >= nResults {
			break
		}
		// Simple mock: return documents with fixed similarity
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

func TestNewRAG(t *testing.T) {
	collection := &MockCollection{}
	rag := New(collection)

	if rag == nil {
		t.Fatal("New() returned nil")
	}
	if rag.topK != 5 {
		t.Errorf("default topK = %d, want 5", rag.topK)
	}
}

func TestNewRAGWithTopK(t *testing.T) {
	collection := &MockCollection{}
	rag := New(collection, WithTopK(10))

	if rag.topK != 10 {
		t.Errorf("topK = %d, want 10", rag.topK)
	}
}

func TestWithTopK(t *testing.T) {
	rag := &RAG{topK: 5}
	option := WithTopK(3)
	option(rag)

	if rag.topK != 3 {
		t.Errorf("After WithTopK(3), topK = %d, want 3", rag.topK)
	}
}

func TestRAGQuery(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{
		documents: []storage.Document{
			{
				ID:      "doc-1",
				Content: "Go is a programming language.",
				Metadata: map[string]any{
					"source": "go.txt",
				},
			},
			{
				ID:      "doc-2",
				Content: "Python is great for data science.",
				Metadata: map[string]any{
					"source": "python.txt",
				},
			},
		},
	}

	rag := New(collection, WithTopK(2))
	context, results, err := rag.Query(ctx, "programming")
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}

	if results == nil {
		t.Error("Query() should return results")
	}
	if len(results) != 2 {
		t.Errorf("Query() returned %d results, want 2", len(results))
	}

	if !strings.Contains(context, "Relevant knowledge") {
		t.Error("Query() context should contain 'Relevant knowledge'")
	}
	if !strings.Contains(context, "Source: go.txt") {
		t.Error("Query() context should contain source")
	}
	if !strings.Contains(context, "similarity:") {
		t.Error("Query() context should contain similarity score")
	}
}

func TestRAGQueryEmpty(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{}
	rag := New(collection)

	context, results, err := rag.Query(ctx, "test")
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	if context != "" {
		t.Error("Query() on empty collection should return empty context")
	}
	if results != nil {
		t.Error("Query() on empty collection should return nil results")
	}
}

func TestRAGQueryUsesTopK(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{
		documents: []storage.Document{
			{ID: "doc-1", Content: "Content 1", Metadata: map[string]any{}},
			{ID: "doc-2", Content: "Content 2", Metadata: map[string]any{}},
			{ID: "doc-3", Content: "Content 3", Metadata: map[string]any{}},
			{ID: "doc-4", Content: "Content 4", Metadata: map[string]any{}},
			{ID: "doc-5", Content: "Content 5", Metadata: map[string]any{}},
			{ID: "doc-6", Content: "Content 6", Metadata: map[string]any{}},
		},
	}

	rag := New(collection, WithTopK(3))
	_, results, err := rag.Query(ctx, "test")
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Query() with topK=3 returned %d results, want 3", len(results))
	}
}

func TestRAGQueryMetadataWithoutSource(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{
		documents: []storage.Document{
			{
				ID:       "doc-1",
				Content:  "Content without source metadata",
				Metadata: map[string]any{},
			},
		},
	}

	rag := New(collection)
	context, _, err := rag.Query(ctx, "test")
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}

	// Should use document ID as source when metadata source is missing
	if !strings.Contains(context, "Source: doc-1") {
		t.Error("Query() should use document ID as source when metadata source is missing")
	}
}

func TestRAGAugmentPrompt(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{
		documents: []storage.Document{
			{
				ID:       "doc-1",
				Content:  "Relevant knowledge content.",
				Metadata: map[string]any{"source": "test.txt"},
			},
		},
	}

	rag := New(collection)
	systemPrompt := "You are a helpful assistant."

	augmented, err := rag.AugmentPrompt(ctx, "query", systemPrompt)
	if err != nil {
		t.Errorf("AugmentPrompt() error = %v", err)
	}

	if !strings.Contains(augmented, systemPrompt) {
		t.Error("AugmentPrompt() should contain original system prompt")
	}
	if !strings.Contains(augmented, "Relevant knowledge") {
		t.Error("AugmentPrompt() should contain retrieved knowledge")
	}
}

func TestRAGAugmentPromptEmpty(t *testing.T) {
	ctx := context.Background()
	collection := &MockCollection{}
	rag := New(collection)

	systemPrompt := "You are a helpful assistant."
	augmented, err := rag.AugmentPrompt(ctx, "query", systemPrompt)
	if err != nil {
		t.Errorf("AugmentPrompt() error = %v", err)
	}

	// Should return original prompt when no knowledge found
	if augmented != systemPrompt {
		t.Error("AugmentPrompt() should return original prompt when no knowledge found")
	}
}

func TestRAGAugmentPromptError(t *testing.T) {
	ctx := context.Background()

	// Collection that returns error
	errorCollection := &ErrorCollection{}
	rag := New(errorCollection)

	systemPrompt := "You are a helpful assistant."
	augmented, err := rag.AugmentPrompt(ctx, "query", systemPrompt)
	if err != nil {
		t.Errorf("AugmentPrompt() should handle error gracefully, got error = %v", err)
	}
	// Should return original prompt on error
	if augmented != systemPrompt {
		t.Error("AugmentPrompt() should return original prompt on error")
	}
}

// ErrorCollection is a mock that returns errors.
type ErrorCollection struct{}

func (e *ErrorCollection) Add(ctx context.Context, documents []storage.Document) error {
	return nil
}

func (e *ErrorCollection) Query(ctx context.Context, query string, nResults int) ([]storage.Result, error) {
	return nil, context.Canceled
}

func (e *ErrorCollection) Delete(ctx context.Context, ids []string) error {
	return nil
}

func (e *ErrorCollection) Count(ctx context.Context) (int, error) {
	return 0, nil
}
