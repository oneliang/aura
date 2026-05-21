// Package storage provides chromem-go backed vector storage.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	chromem "github.com/philippgille/chromem-go"
)

// ChromemCollection implements Collection using chromem-go.
type ChromemCollection struct {
	db         *chromem.DB
	collection *chromem.Collection
	name       string
}

// ChromemOptions configures a ChromemCollection.
type ChromemOptions struct {
	// DataDir is the directory to persist data. Empty means in-memory only.
	DataDir string
	// Name is the collection name.
	Name string
	// EmbeddingFunc is the function used to embed text. If nil, uses Ollama.
	EmbeddingFunc chromem.EmbeddingFunc
}

// NewChromemCollection creates or opens a chromem-go backed collection.
func NewChromemCollection(ctx context.Context, opts ChromemOptions) (*ChromemCollection, error) {
	if opts.Name == "" {
		opts.Name = "default"
	}

	var db *chromem.DB
	var err error

	if opts.DataDir != "" {
		// Persist to disk
		if err = os.MkdirAll(opts.DataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data dir: %w", err)
		}
		dbPath := filepath.Join(opts.DataDir, "chromem.db")
		db, err = chromem.NewPersistentDB(dbPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to open persistent DB: %w", err)
		}
	} else {
		db = chromem.NewDB()
	}

	// Use provided embedding function or noop (will require embeddings pre-computed)
	embeddingFunc := opts.EmbeddingFunc
	if embeddingFunc == nil {
		embeddingFunc = chromem.NewEmbeddingFuncDefault()
	}

	col, err := db.GetOrCreateCollection(opts.Name, nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	return &ChromemCollection{
		db:         db,
		collection: col,
		name:       opts.Name,
	}, nil
}

// Add adds documents to the collection.
func (c *ChromemCollection) Add(ctx context.Context, documents []Document) error {
	chromemDocs := make([]chromem.Document, 0, len(documents))
	for _, doc := range documents {
		// Build metadata as map[string]string (chromem requirement)
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
		chromemDocs = append(chromemDocs, chromem.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: metadata,
		})
	}
	return c.collection.AddDocuments(ctx, chromemDocs, 1)
}

// Query searches for similar documents.
func (c *ChromemCollection) Query(ctx context.Context, query string, nResults int) ([]Result, error) {
	if c.collection.Count() == 0 {
		return nil, nil
	}

	if nResults > c.collection.Count() {
		nResults = c.collection.Count()
	}

	res, err := c.collection.Query(ctx, query, nResults, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	results := make([]Result, 0, len(res))
	for _, r := range res {
		metadata := make(map[string]any)
		for k, v := range r.Metadata {
			metadata[k] = v
		}
		results = append(results, Result{
			Document: Document{
				ID:       r.ID,
				Content:  r.Content,
				Metadata: metadata,
			},
			Score: r.Similarity,
		})
	}
	return results, nil
}

// Delete removes documents by ID.
func (c *ChromemCollection) Delete(ctx context.Context, ids []string) error {
	return c.collection.Delete(ctx, nil, nil, ids...)
}

// Count returns the number of documents.
func (c *ChromemCollection) Count(_ context.Context) (int, error) {
	return c.collection.Count(), nil
}
