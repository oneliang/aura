// Package storage provides vector storage implementations.
package storage

import (
	"context"
)

// Document represents a document in the knowledge base.
type Document struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Result represents a search result.
type Result struct {
	Document Document `json:"document"`
	Score    float32  `json:"score"`
}

// Collection defines the interface for a document collection.
type Collection interface {
	// Add adds documents to the collection.
	Add(ctx context.Context, documents []Document) error

	// Query searches for similar documents.
	Query(ctx context.Context, query string, nResults int) ([]Result, error)

	// Delete removes documents by ID.
	Delete(ctx context.Context, ids []string) error

	// Count returns the number of documents.
	Count(ctx context.Context) (int, error)
}
