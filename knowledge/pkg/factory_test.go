package knowledge

import (
	"context"
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

// TestNewDefaultCollectionFactory tests NewDefaultCollectionFactory function.
func TestNewDefaultCollectionFactory(t *testing.T) {
	factory := NewDefaultCollectionFactory(nil)
	if factory == nil {
		t.Fatal("NewDefaultCollectionFactory() returned nil")
	}
}

// TestNewDefaultCollectionFactory_WithConfig tests with config.
func TestNewDefaultCollectionFactory_WithConfig(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			EmbeddingModel: "test-model",
			BaseURL:        "http://localhost:11434",
		},
	}

	factory := NewDefaultCollectionFactory(cfg)
	if factory == nil {
		t.Fatal("NewDefaultCollectionFactory() returned nil")
	}
	if factory.config != cfg {
		t.Error("Config not set correctly")
	}
}

// TestDefaultCollectionFactory_NewCollection tests NewCollection method.
// Note: This test requires Ollama to be running, so we test the interface compliance.
func TestDefaultCollectionFactory_NewCollection(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			EmbeddingModel: "nomic-embed-text",
			BaseURL:        "http://localhost:11434",
		},
	}

	factory := NewDefaultCollectionFactory(cfg)
	ctx := context.Background()

	// This will likely fail if Ollama is not running, but we test the interface
	collection, err := factory.NewCollection(ctx, "")
	// Just verify the method can be called without panic
	_ = collection
	_ = err
}

// TestCollectionFactory_Interface tests that DefaultCollectionFactory implements CollectionFactory.
func TestCollectionFactory_Interface(t *testing.T) {
	var _ CollectionFactory = (*DefaultCollectionFactory)(nil)
}
