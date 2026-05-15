// Package storage provides vector storage implementations.
package storage

import (
	"testing"
)

// Test Document struct
func TestDocument(t *testing.T) {
	doc := Document{
		ID:      "test-1",
		Content: "Test content",
		Metadata: map[string]any{
			"source": "test.txt",
			"chunk":  0,
		},
	}

	if doc.ID != "test-1" {
		t.Errorf("ID = %v, want 'test-1'", doc.ID)
	}
	if doc.Content != "Test content" {
		t.Errorf("Content = %v, want 'Test content'", doc.Content)
	}
	if doc.Metadata["source"] != "test.txt" {
		t.Errorf("Metadata[source] = %v, want 'test.txt'", doc.Metadata["source"])
	}
}

// Test Result struct
func TestResult(t *testing.T) {
	result := Result{
		Document: Document{
			ID:      "test-1",
			Content: "Test content",
		},
		Score: 0.95,
	}

	if result.Document.ID != "test-1" {
		t.Errorf("Document.ID = %v, want 'test-1'", result.Document.ID)
	}
	if result.Score != 0.95 {
		t.Errorf("Score = %v, want 0.95", result.Score)
	}
}
