package commands

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/llm/ollama"
	"github.com/oneliang/aura/core/pkg/llm/openai"
	"github.com/oneliang/aura/shared/pkg/config"
)

func TestCreateLLMClient(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "qwen3:8b",
		},
	}

	client := createTestLLMClient(cfg)
	if client == nil {
		t.Error("createTestLLMClient() returned nil client")
	}
}

func TestCreateLLMClient_OpenAI(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
			APIKey:   "test-key",
		},
	}

	client := createTestLLMClient(cfg)
	if client == nil {
		t.Error("createTestLLMClient() returned nil client for OpenAI")
	}
}

func TestCreateLLMClient_DefaultProvider(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "unknown",
			Model:    "test",
		},
	}

	// Should default to Ollama
	client := createTestLLMClient(cfg)
	if client == nil {
		t.Error("createTestLLMClient() returned nil client for unknown provider")
	}
}

// createTestLLMClient creates an LLM client for testing purposes.
func createTestLLMClient(cfg *config.Config) llm.Client {
	switch cfg.LLM.Provider {
	case "openai":
		return openai.New(
			openai.WithBaseURL(cfg.LLM.BaseURL),
			openai.WithModel(cfg.LLM.Model),
			openai.WithAPIKey(cfg.LLM.APIKey),
		)
	default: // ollama
		return ollama.New(
			ollama.WithBaseURL(cfg.LLM.BaseURL),
			ollama.WithModel(cfg.LLM.Model),
		)
	}
}
