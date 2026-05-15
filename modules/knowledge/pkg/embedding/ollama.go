// Package embedding provides embedding function adapters for chromem-go.
package embedding

import (
	"context"
	"fmt"
	"strings"

	chromem "github.com/philippgille/chromem-go"
)

// OllamaEmbeddingFunc returns a chromem-compatible embedding function
// that calls an Ollama /api/embeddings endpoint.
// chromem-go appends "/embeddings" to the baseURL, so we ensure the
// URL ends with "/api" (e.g. http://host:11434/api).
func OllamaEmbeddingFunc(baseURL, model string) chromem.EmbeddingFunc {
	// chromem.NewEmbeddingFuncOllama appends "/embeddings" to baseURL,
	// so we need baseURL to be "http://host:port/api".
	apiURL := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(apiURL, "/api") {
		apiURL += "/api"
	}
	return chromem.NewEmbeddingFuncOllama(model, apiURL)
}

// ValidateEmbedding checks that the embedding function works.
func ValidateEmbedding(ctx context.Context, fn chromem.EmbeddingFunc) error {
	vec, err := fn(ctx, "test")
	if err != nil {
		return fmt.Errorf("embedding function failed: %w", err)
	}
	if len(vec) == 0 {
		return fmt.Errorf("embedding returned empty vector")
	}
	return nil
}
