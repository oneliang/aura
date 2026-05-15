// Package retrieval provides dynamic RAG (Retrieval Augmented Generation) with token-aware injection.
package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// DynamicRAG extends RAG with dynamic topK adjustment based on token budget.
type DynamicRAG struct {
	collection   storage.Collection
	baseTopK     int // Base number of documents to retrieve
	currentUsage int // Current token usage
	maxBudget    int // Maximum token budget for context
}

// DynamicRAGConfig holds configuration for DynamicRAG.
type DynamicRAGConfig struct {
	BaseTopK  int // Base number of documents (default: 5)
	MaxBudget int // Maximum token budget (0=disabled, dynamic adjustment off)
}

// DefaultDynamicRAGConfig returns the default DynamicRAG configuration.
func DefaultDynamicRAGConfig() DynamicRAGConfig {
	return DynamicRAGConfig{
		BaseTopK:  5,
		MaxBudget: 8000, // Default 8k token budget
	}
}

// NewDynamicRAG creates a new DynamicRAG instance.
func NewDynamicRAG(collection storage.Collection, config DynamicRAGConfig) *DynamicRAG {
	if config.BaseTopK <= 0 {
		config.BaseTopK = DefaultDynamicRAGConfig().BaseTopK
	}
	if config.MaxBudget <= 0 {
		config.MaxBudget = DefaultDynamicRAGConfig().MaxBudget
	}

	return &DynamicRAG{
		collection: collection,
		baseTopK:   config.BaseTopK,
		maxBudget:  config.MaxBudget,
	}
}

// SetTokenBudget updates the current token usage and maximum budget.
// This is used to dynamically adjust the topK for retrieval.
func (r *DynamicRAG) SetTokenBudget(currentUsage, maxBudget int) {
	r.currentUsage = currentUsage
	if maxBudget > 0 {
		r.maxBudget = maxBudget
	}
}

// getDynamicTopK calculates the optimal topK based on current token usage.
// Strategy:
//   - < 50% usage: use baseTopK
//   - 50-70% usage: baseTopK - 1
//   - 70-85% usage: baseTopK - 2
//   - > 85% usage: 1 (minimum to save tokens)
func (r *DynamicRAG) getDynamicTopK() int {
	if r.maxBudget <= 0 {
		return r.baseTopK // Budget disabled, use base
	}

	ratio := float64(r.currentUsage) / float64(r.maxBudget)

	var topK int
	switch {
	case ratio < 0.5:
		topK = r.baseTopK
	case ratio < 0.7:
		topK = max(1, r.baseTopK-1)
	case ratio < 0.85:
		topK = max(1, r.baseTopK-2)
	default:
		topK = 1 // Token budget is tight, minimize retrieval
	}

	return topK
}

// Query retrieves documents with dynamic topK adjustment.
func (r *DynamicRAG) Query(ctx context.Context, query string) (string, []storage.Result, error) {
	topK := r.getDynamicTopK()

	results, err := r.collection.Query(ctx, query, topK)
	if err != nil {
		return "", nil, fmt.Errorf("query collection: %w", err)
	}
	if len(results) == 0 {
		return "", nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Relevant knowledge (retrieved %d documents, token budget: %d/%d):\n\n",
		len(results), r.currentUsage, r.maxBudget))
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

// AugmentQuery prepends retrieved knowledge to a query with token-aware injection.
func (r *DynamicRAG) AugmentQuery(ctx context.Context, query, systemPrompt string) (string, error) {
	knowledge, _, err := r.Query(ctx, query)
	if err != nil || knowledge == "" {
		return query, nil
	}

	// If system prompt is provided, prepend knowledge to it
	if systemPrompt != "" {
		return systemPrompt + "\n\n" + knowledge + "\n\nUser query: " + query, nil
	}

	return knowledge + "\n\n" + query, nil
}

// GetConfig returns the current DynamicRAG configuration.
func (r *DynamicRAG) GetConfig() DynamicRAGConfig {
	return DynamicRAGConfig{
		BaseTopK:  r.baseTopK,
		MaxBudget: r.maxBudget,
	}
}

// GetStats returns current RAG statistics.
func (r *DynamicRAG) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"base_top_k":    r.baseTopK,
		"dynamic_top_k": r.getDynamicTopK(),
		"current_usage": r.currentUsage,
		"max_budget":    r.maxBudget,
		"usage_ratio":   float64(r.currentUsage) / float64(r.maxBudget),
	}
}

// Helper function for max (Go 1.21+ has built-in max)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
