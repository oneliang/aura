package constants

import "time"

// Version information
const (
	Version     = "0.1.0"
	BuildCommit = "dev"
)

// Default values - can be overridden by configuration
var (
	// LLM related
	DefaultLLMBaseURL  = "http://localhost:11434"
	DefaultLLMModel    = "qwen3:8b"
	DefaultOllamaModel = "llama3.2"

	// OpenAI
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "gpt-4o-mini"

	// Anthropic
	DefaultAnthropicBaseURL    = "https://api.anthropic.com/v1"
	DefaultAnthropicModel      = "claude-sonnet-4-20250514"
	DefaultAnthropicAPIVersion = "2024-10-22"

	DefaultLLMTimeout = 120 * time.Second

	// Tools related
	DefaultToolTimeout  = 30 * time.Second
	DefaultShellTimeout = 30 * time.Second
	DefaultSSHTimeout   = 30 * time.Second
	DefaultWebTimeout   = 30 * time.Second

	// Thinking related
	DefaultThinkingEnabled = true
	DefaultThinkingEffort  = "medium"
	DefaultThinkingBudget  = 2048

	// API related
	DefaultAPIPort = 8080
)
