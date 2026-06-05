package factory

import (
	"net/http"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
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
		BaseURL:  constants.DefaultOpenAIBaseURL,
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

func TestLLMFactory_WithTimeout(t *testing.T) {
	// Test with custom timeout
	cfg := &config.LLMConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "qwen3:8b",
		Timeout:  300 * time.Second, // 5 minutes
	}

	factory := NewLLMFactory(cfg)
	client, err := factory.Create()

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if client == nil {
		t.Fatal("Create() returned nil client")
	}

	// Verify HTTP client has configured timeout
	if factory.httpClient == nil {
		t.Fatal("Factory HTTP client is nil")
	}

	if factory.httpClient.Timeout != 300*time.Second {
		t.Errorf("HTTP client timeout = %v, expected 300s", factory.httpClient.Timeout)
	}
}

func TestLLMFactory_WithZeroTimeout(t *testing.T) {
	// Test with zero timeout (should use default)
	cfg := &config.LLMConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "qwen3:8b",
		Timeout:  0, // Zero timeout
	}

	factory := NewLLMFactory(cfg)

	// Verify HTTP client uses default timeout
	if factory.httpClient == nil {
		t.Fatal("Factory HTTP client is nil")
	}

	if factory.httpClient.Timeout != constants.DefaultLLMTimeout {
		t.Errorf("HTTP client timeout = %v, expected default %v", factory.httpClient.Timeout, constants.DefaultLLMTimeout)
	}
}

func TestLLMFactory_WithCustomHTTPClient(t *testing.T) {
	// Test WithHTTPClient option overrides timeout config
	cfg := &config.LLMConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "qwen3:8b",
		Timeout:  300 * time.Second, // Config timeout
	}

	customClient := &http.Client{
		Timeout: 600 * time.Second, // Custom timeout (different from config)
	}

	factory := NewLLMFactory(cfg, WithHTTPClient(customClient))

	// Verify HTTP client is the custom one (not from config)
	if factory.httpClient != customClient {
		t.Error("Factory HTTP client should be the custom client")
	}

	if factory.httpClient.Timeout != 600*time.Second {
		t.Errorf("HTTP client timeout = %v, expected 600s (custom)", factory.httpClient.Timeout)
	}
}
