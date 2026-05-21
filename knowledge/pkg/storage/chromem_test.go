// Package storage provides chromem-go backed vector storage.
package storage

import (
	"context"
	"os"
	"testing"
)

func TestNewChromemCollectionInMemory(t *testing.T) {
	ctx := context.Background()

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name: "test-collection",
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}
	if collection == nil {
		t.Fatal("NewChromemCollection() returned nil")
	}
	if collection.name != "test-collection" {
		t.Errorf("collection name = %v, want 'test-collection'", collection.name)
	}
}

func TestNewChromemCollectionDefaultName(t *testing.T) {
	ctx := context.Background()

	collection, err := NewChromemCollection(ctx, ChromemOptions{})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}
	if collection.name != "default" {
		t.Errorf("collection name = %v, want 'default'", collection.name)
	}
}

func TestNewChromemCollectionPersistent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		DataDir: tmpDir,
		Name:    "persistent-test",
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}
	if collection == nil {
		t.Fatal("NewChromemCollection() returned nil")
	}

	// Verify database file was created
	dbPath := tmpDir + "/chromem.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Persistent database file should be created")
	}
}

func TestChromemCollectionAddAndQuery(t *testing.T) {
	ctx := context.Background()

	// Use a mock embedding function to avoid network calls
	mockEmbeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		// Return a fixed vector for testing
		return []float32{0.1, 0.2, 0.3}, nil
	}

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name:          "add-query-test",
		EmbeddingFunc: mockEmbeddingFunc,
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	// Add documents with pre-computed embeddings
	// Since we're using mock embedding, we need to add documents that chromem can handle
	docs := []Document{
		{
			ID:      "doc-1",
			Content: "Go is a programming language developed by Google.",
			Metadata: map[string]any{
				"source": "go.txt",
			},
		},
		{
			ID:      "doc-2",
			Content: "Python is great for data science and machine learning.",
			Metadata: map[string]any{
				"source": "python.txt",
			},
		},
	}

	err = collection.Add(ctx, docs)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}

	// Check count
	count, err := collection.Count(ctx)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestChromemCollectionQueryEmpty(t *testing.T) {
	ctx := context.Background()

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name: "empty-query-test",
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	results, err := collection.Query(ctx, "test query", 5)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	if results != nil {
		t.Error("Query() on empty collection should return nil")
	}
}

func TestChromemCollectionQueryNResults(t *testing.T) {
	ctx := context.Background()

	// Use a mock embedding function to avoid network calls
	mockEmbeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name:          "nresults-test",
		EmbeddingFunc: mockEmbeddingFunc,
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	// Add one document
	docs := []Document{
		{
			ID:       "doc-1",
			Content:  "Test content",
			Metadata: map[string]any{},
		},
	}
	collection.Add(ctx, docs)

	// Query for more results than available
	results, err := collection.Query(ctx, "test", 10)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	// Should return at most the number of documents available
	if len(results) > 1 {
		t.Errorf("Query() returned %d results, should be at most 1", len(results))
	}
}

func TestChromemCollectionDelete(t *testing.T) {
	ctx := context.Background()

	// Use a mock embedding function to avoid network calls
	mockEmbeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name:          "delete-test",
		EmbeddingFunc: mockEmbeddingFunc,
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	// Add documents
	docs := []Document{
		{ID: "doc-1", Content: "Content 1", Metadata: map[string]any{}},
		{ID: "doc-2", Content: "Content 2", Metadata: map[string]any{}},
	}
	collection.Add(ctx, docs)

	// Delete one document
	err = collection.Delete(ctx, []string{"doc-1"})
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Check count
	count, err := collection.Count(ctx)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count() = %d, want 1 after delete", count)
	}
}

func TestChromemCollectionCount(t *testing.T) {
	ctx := context.Background()

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name: "count-test",
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	// Initial count should be 0
	count, err := collection.Count(ctx)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

func TestChromemCollectionMetadataConversion(t *testing.T) {
	ctx := context.Background()

	// Use a mock embedding function to avoid network calls
	mockEmbeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	collection, err := NewChromemCollection(ctx, ChromemOptions{
		Name:          "metadata-test",
		EmbeddingFunc: mockEmbeddingFunc,
	})
	if err != nil {
		t.Fatalf("NewChromemCollection() error = %v", err)
	}

	// Add document with various metadata types
	docs := []Document{
		{
			ID:      "doc-1",
			Content: "Test content",
			Metadata: map[string]any{
				"string":  "value",
				"int":     42,
				"float":   3.14,
				"bool":    true,
				"complex": []string{"a", "b", "c"},
			},
		},
	}

	err = collection.Add(ctx, docs)
	if err != nil {
		t.Errorf("Add() with various metadata types error = %v", err)
	}
}
