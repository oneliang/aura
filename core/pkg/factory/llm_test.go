package factory

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/config"
)

func TestLLMFactory_Create_Ollama(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "qwen3:8b",
	}

	factory := NewLLMFactory(cfg)
	client, err := factory.Create()

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("Create() returned nil client")
	}

	// Verify it's a LoggingClient wrapping an Ollama client
	_, ok := client.(*llm.LoggingClient)
	if !ok {
		t.Errorf("Create() returned %T, expected *llm.LoggingClient (wrapping *ollama.Client)", client)
	}
}

func TestLLMFactory_Create_OpenAI(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}

	factory := NewLLMFactory(cfg)
	client, err := factory.Create()

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("Create() returned nil client")
	}

	// Verify it's a LoggingClient wrapping an OpenAI client
	_, ok := client.(*llm.LoggingClient)
	if !ok {
		t.Errorf("Create() returned %T, expected *llm.LoggingClient (wrapping *openai.Client)", client)
	}
}

func TestLLMFactory_Create_DefaultProvider(t *testing.T) {
	// Empty provider should default to Ollama
	cfg := &config.LLMConfig{
		Provider: "",
		BaseURL:  "http://localhost:11434",
		Model:    "qwen3:8b",
	}

	factory := NewLLMFactory(cfg)
	client, err := factory.Create()

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("Create() returned nil client")
	}

	// Verify it's a LoggingClient wrapping an Ollama client (default)
	_, ok := client.(*llm.LoggingClient)
	if !ok {
		t.Errorf("Create() returned %T, expected *llm.LoggingClient (default, wrapping *ollama.Client)", client)
	}
}

func TestLLMFactory_Create_WithEmptyConfig(t *testing.T) {
	cfg := &config.LLMConfig{}

	factory := NewLLMFactory(cfg)
	client, err := factory.Create()

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("Create() returned nil client")
	}
}

func TestNewLLMFactory(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "ollama",
	}

	factory := NewLLMFactory(cfg)

	if factory == nil {
		t.Fatal("NewLLMFactory() returned nil")
	}

	if factory.config != cfg {
		t.Error("NewLLMFactory() did not store config correctly")
	}
}
