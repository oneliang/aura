// Package embedding provides embedding function adapters for chromem-go.
package embedding

import (
	"context"
	"strings"
	"testing"
)

func TestOllamaEmbeddingFunc(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantURL string
		model   string
	}{
		{"with /api suffix", "http://localhost:11434/api", "http://localhost:11434/api", "nomic-embed-text"},
		{"without /api suffix", "http://localhost:11434", "http://localhost:11434/api", "nomic-embed-text"},
		{"with trailing slash", "http://localhost:11434/", "http://localhost:11434/api", "nomic-embed-text"},
		{"with multiple trailing slashes", "http://localhost:11434///", "http://localhost:11434/api", "nomic-embed-text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := OllamaEmbeddingFunc(tt.baseURL, tt.model)
			if fn == nil {
				t.Error("OllamaEmbeddingFunc() returned nil")
			}
		})
	}
}

func TestValidateEmbedding(t *testing.T) {
	ctx := context.Background()

	// Test with noop embedding function (returns fixed vector)
	noopFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	err := ValidateEmbedding(ctx, noopFunc)
	if err != nil {
		t.Errorf("ValidateEmbedding() with valid function error = %v", err)
	}
}

func TestValidateEmbeddingEmptyVector(t *testing.T) {
	ctx := context.Background()

	// Test with embedding function that returns empty vector
	emptyFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{}, nil
	}

	err := ValidateEmbedding(ctx, emptyFunc)
	if err == nil {
		t.Error("ValidateEmbedding() should error on empty vector")
	}
	if !strings.Contains(err.Error(), "empty vector") {
		t.Errorf("Error should mention 'empty vector', got: %v", err)
	}
}

func TestValidateEmbeddingError(t *testing.T) {
	ctx := context.Background()

	// Test with embedding function that returns error
	errorFunc := func(ctx context.Context, text string) ([]float32, error) {
		return nil, context.Canceled
	}

	err := ValidateEmbedding(ctx, errorFunc)
	if err == nil {
		t.Error("ValidateEmbedding() should error when function returns error")
	}
	if !strings.Contains(err.Error(), "embedding function failed") {
		t.Errorf("Error should mention 'embedding function failed', got: %v", err)
	}
}

// Test with a local mock embedding function (to avoid network calls)
func TestValidateEmbeddingWithMock(t *testing.T) {
	ctx := context.Background()

	// Use a mock function that returns a fixed vector
	mockFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	err := ValidateEmbedding(ctx, mockFunc)
	if err != nil {
		t.Errorf("ValidateEmbedding() with mock function error = %v", err)
	}
}
