// Package retrieval provides RAG (Retrieval Augmented Generation) functionality.
package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// RAG implements Retrieval Augmented Generation.
type RAG struct {
	collection storage.Collection
	topK       int
}

// Option configures a RAG instance.
type Option func(*RAG)

// WithTopK sets the number of documents to retrieve.
func WithTopK(k int) Option {
	return func(r *RAG) { r.topK = k }
}

// New creates a new RAG instance.
func New(collection storage.Collection, opts ...Option) *RAG {
	r := &RAG{collection: collection, topK: 5}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Query retrieves relevant documents and returns formatted context + raw results.
func (r *RAG) Query(ctx context.Context, query string) (string, []storage.Result, error) {
	results, err := r.collection.Query(ctx, query, r.topK)
	if err != nil {
		return "", nil, fmt.Errorf("query collection: %w", err)
	}
	if len(results) == 0 {
		return "", nil, nil
	}

	var sb strings.Builder
	sb.WriteString("Relevant knowledge:\n\n")
	for i, res := range results {
		source, _ := res.Document.Metadata["source"].(string)
		if source == "" {
			source = res.Document.ID
		}
		sb.WriteString(fmt.Sprintf("[%d] Source: %s (similarity: %.2f)\n%s\n\n",
			i+1, source, res.Score, res.Document.Content))
	}
	return sb.String(), results, nil
}

// AugmentPrompt prepends retrieved knowledge to a system prompt.
func (r *RAG) AugmentPrompt(ctx context.Context, query, systemPrompt string) (string, error) {
	knowledge, _, err := r.Query(ctx, query)
	if err != nil || knowledge == "" {
		return systemPrompt, nil
	}
	return systemPrompt + "\n\n" + knowledge, nil
}
