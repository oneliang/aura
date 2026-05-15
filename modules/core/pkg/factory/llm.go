// Package factory provides factories for creating core components.
package factory

import (
	"net/http"
	"sync"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/llm/anthropic"
	"github.com/oneliang/aura/core/pkg/llm/ollama"
	"github.com/oneliang/aura/core/pkg/llm/openai"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/httpclient"
)

// LLMFactory creates LLM clients based on configuration.
type LLMFactory struct {
	config     *config.LLMConfig
	httpClient *http.Client

	// Cache for single instance
	client    llm.Client
	clientErr error
	once      sync.Once
}

// LLMFactoryOption is a configuration option for LLMFactory.
type LLMFactoryOption func(*LLMFactory)

// WithHTTPClient sets the HTTP client for LLM requests.
func WithHTTPClient(client *http.Client) LLMFactoryOption {
	return func(f *LLMFactory) {
		f.httpClient = client
	}
}

// NewLLMFactory creates a new LLM factory.
func NewLLMFactory(cfg *config.LLMConfig, opts ...LLMFactoryOption) *LLMFactory {
	f := &LLMFactory{
		config:     cfg,
		httpClient: httpclient.DefaultLLMClient(),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Create creates an LLM client based on configuration.
// Uses cached client if already created.
// Decorator chain: LoggingClient → RetryClient → (OpenAI|Ollama)Client
func (f *LLMFactory) Create() (llm.Client, error) {
	f.once.Do(func() {
		var bare llm.Client
		switch f.config.Provider {
		case "openai":
			bare = openai.New(
				openai.WithBaseURL(f.config.BaseURL),
				openai.WithModel(f.config.Model),
				openai.WithAPIKey(f.config.APIKey),
				openai.WithHTTPClient(f.httpClient),
			)
		case "anthropic":
			bare = anthropic.New(
				anthropic.WithBaseURL(f.config.BaseURL),
				anthropic.WithModel(f.config.Model),
				anthropic.WithAPIKey(f.config.APIKey),
				anthropic.WithHTTPClient(f.httpClient),
			)
		default: // ollama
			bare = ollama.New(
				ollama.WithBaseURL(f.config.BaseURL),
				ollama.WithModel(f.config.Model),
				ollama.WithHTTPClient(f.httpClient),
			)
		}

		// Apply retry wrapper.
		retryCfg := llm.DefaultRetryConfig()
		if f.config.Retry.MaxRetries > 0 {
			retryCfg.MaxRetries = f.config.Retry.MaxRetries
		}
		if f.config.Retry.InitialDelay > 0 {
			retryCfg.InitialDelay = f.config.Retry.InitialDelay
		}
		if f.config.Retry.MaxDelay > 0 {
			retryCfg.MaxDelay = f.config.Retry.MaxDelay
		}
		wrapped := llm.NewRetryClient(bare, retryCfg)

		// Apply logging wrapper (outermost).
		f.client = llm.NewLoggingClient(wrapped, f.config.Provider, f.config.Model, "")
	})
	return f.client, f.clientErr
}
