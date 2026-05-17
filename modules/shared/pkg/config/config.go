// Package config provides configuration management for the agent system.
package config

import (
	"errors"
	"fmt"
	"os"
	stdpath "path/filepath"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/spf13/viper"
)

// Default value constants for configuration.
const (
	defaultLLMProvider = "ollama"
	// defaultLLMBaseURL, defaultLLMModel are defined in constants package
	defaultMaxTokens       = 8000
	defaultMaxContext      = 50
	defaultTokenEstimator  = "simple"
	defaultLogLevel        = "info"
	defaultLogFormat       = "text"
	defaultLogOutput       = "stdout"
	defaultPermissionLevel = PermLevelAsk
	defaultPlanningMode    = "implicit"
	defaultWorkspaceDir    = "workspace"
)

// Permission control level constants for configuration.
// These are used in DefaultConfig() for setting default permission values.
const (
	PermLevelAllow = "allow"
	PermLevelAsk   = "ask"
	PermLevelDeny  = "deny"
)

// Environment variable names for configuration.
const (
	EnvLLMProvider = "AURA_LLM_PROVIDER"
	EnvLLMBaseURL  = "AURA_LLM_BASE_URL"
	EnvLLMModel    = "AURA_LLM_MODEL"
	EnvLLMAPIKey   = "AURA_LLM_API_KEY"
)

// Config represents the main configuration structure.
// Each field corresponds to a subsystem config fragment.
// New subsystems should add their own nested struct here,
// NOT top-level fields, to keep the config tree balanced.
type Config struct {
	Version      string             `mapstructure:"version"` // Config schema version, default "1"
	LLM          LLMConfig          `mapstructure:"llm"`
	Memory       MemoryConfig       `mapstructure:"memory"`
	Tools        ToolsConfig        `mapstructure:"tools"`
	Log          LogConfig          `mapstructure:"log"`
	SSH          SSHConfig          `mapstructure:"ssh"`
	Permissions  PermissionsConfig  `mapstructure:"permissions"`
	Skills       SkillsConfig       `mapstructure:"skills"`
	Agent        AgentConfig        `mapstructure:"agent"`
	API          APIConfig          `mapstructure:"api"`
	Debug        DebugConfig        `mapstructure:"debug"`
	TUI          TUIConfig          `mapstructure:"tui"`
	Adapters     AdaptersConfig     `mapstructure:"adapters"`
	Knowledge    KnowledgeConfig    `mapstructure:"knowledge"`
	I18n         I18nConfig         `mapstructure:"i18n"`
	Orchestrator OrchestratorConfig `mapstructure:"orchestrator"`
	Agents       AgentsConfig       `mapstructure:"agents"`
	Intent       IntentConfig       `mapstructure:"intent"`
	Users        UsersConfig        `mapstructure:"users"`
	Habit        HabitConfig        `mapstructure:"habit"`
	Location     LocationConfig     `mapstructure:"location"`
	LSP          LSPConfig          `mapstructure:"lsp"`
}

// APIConfig represents API server configuration.
type APIConfig struct {
	Port string `mapstructure:"port"` // Port to listen on (default: 8080)
}

// LLMConfig represents LLM provider configuration.
type LLMConfig struct {
	Provider         string         `mapstructure:"provider"` // ollama, openai, anthropic
	BaseURL          string         `mapstructure:"base_url"`
	Model            string         `mapstructure:"model"`
	APIKey           string         `mapstructure:"api_key"`         // optional for local providers
	EmbeddingModel   string         `mapstructure:"embedding_model"` // model used for embeddings
	Retry            RetryConfig    `mapstructure:"retry"`           // retry configuration for LLM requests
	Thinking         ThinkingConfig `mapstructure:"thinking"`        // thinking/reasoning configuration
	EnablePromptCache bool          `mapstructure:"enable_prompt_cache"` // enable prompt caching
}

// RetryConfig represents retry configuration for LLM requests.
type RetryConfig struct {
	MaxRetries   int           `mapstructure:"max_retries"`   // maximum number of retries (0 = disabled)
	InitialDelay time.Duration `mapstructure:"initial_delay"` // initial backoff delay
	MaxDelay     time.Duration `mapstructure:"max_delay"`     // maximum backoff delay
}

// ThinkingConfig represents thinking/reasoning configuration for LLM requests.
type ThinkingConfig struct {
	Enabled         bool   `mapstructure:"enabled"`          // enable native thinking mode
	ReasoningEffort string `mapstructure:"reasoning_effort"` // low/medium/high (OpenAI)
	BudgetTokens    int    `mapstructure:"budget_tokens"`    // max thinking tokens (Anthropic)
}

// MemoryConfig represents memory system configuration.
type MemoryConfig struct {
	Type             string        `mapstructure:"type"`              // sqlite, memory
	StorageDir       string        `mapstructure:"storage_dir"`       // data/memory
	MaxContext       int           `mapstructure:"max_context"`       // max context messages (legacy, use max_tokens)
	MaxTokens        int           `mapstructure:"max_tokens"`        // max tokens (0=use max_context fallback)
	TokenEstimator   string        `mapstructure:"token_estimator"`   // simple, tiktoken
	SummaryThreshold float64       `mapstructure:"summary_threshold"` // token ratio to trigger summarization (0.0-1.0)
	ContextThreshold float64       `mapstructure:"context_threshold"` // token ratio for context window (0.0-1.0)
	Retention        RetentionConfig `mapstructure:"retention"`         // retention policy for staleness
}

// RetentionConfig represents memory retention policy configuration.
type RetentionConfig struct {
	MaxAge          time.Duration `mapstructure:"max_age"`           // Maximum message age before cleanup (0 = no limit)
	MaxInactiveAge  time.Duration `mapstructure:"max_inactive_age"`  // Maximum inactive time before stale (0 = no staleness check)
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`  // Background cleanup interval (0 = no cleanup)
}

// DebugConfig represents debug configuration.
type DebugConfig struct {
	ShowTokens         bool `mapstructure:"show_tokens"`          // Show token usage in TUI
	LogTokens          bool `mapstructure:"log_tokens"`           // Log token changes
	LogLLMInteractions bool `mapstructure:"log_llm_interactions"` // Log LLM requests/responses to file
}

// ToolsConfig represents tools configuration.
type ToolsConfig struct {
	Enabled        []string      `mapstructure:"enabled"`
	DefaultTimeout time.Duration `mapstructure:"default_timeout"` // default tool execution timeout
	ShellTimeout   time.Duration `mapstructure:"shell_timeout"`   // shell command timeout
	SSHTimeout     time.Duration `mapstructure:"ssh_timeout"`     // SSH command timeout
	WebTimeout     time.Duration `mapstructure:"web_timeout"`     // web fetch/search timeout
}

// LogConfig represents logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, text
	Output string `mapstructure:"output"` // stdout, file
	Path   string `mapstructure:"path"`   // log file path
}

// SSHConfig represents SSH configuration.
type SSHConfig struct {
	Servers []SSHServerConfig `mapstructure:"servers"`
}

// SSHServerConfig represents a single SSH server configuration.
type SSHServerConfig struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	KeyPath  string `mapstructure:"key_path"`
	Password string `mapstructure:"password"` // Consider using environment variable instead
}

// PermissionsConfig represents permissions configuration.
type PermissionsConfig struct {
	DefaultLevel      string              `mapstructure:"default_level"`
	Tools             map[string]string   `mapstructure:"tools"`
	ShellRestrictions CommandRestrictions `mapstructure:"shell_restrictions"`
	SSHRestrictions   SSHRestrictions     `mapstructure:"ssh"`
	TrustedDirs       []string            `mapstructure:"trusted_dirs"`   // List of trusted directory paths
	AutoAskTrust      bool                `mapstructure:"auto_ask_trust"` // Auto-ask trust in CLI/TUI modes
}

// CommandRestrictions holds command whitelist/blacklist.
type CommandRestrictions struct {
	AllowedCommands []string `mapstructure:"allowed_commands"`
	DeniedCommands  []string `mapstructure:"denied_commands"`
}

// SSHRestrictions holds SSH-specific restrictions.
type SSHRestrictions struct {
	AllowedHosts    []string `mapstructure:"allowed_hosts"`
	DeniedHosts     []string `mapstructure:"denied_hosts"`
	AllowedCommands []string `mapstructure:"allowed_commands"`
	DeniedCommands  []string `mapstructure:"denied_commands"`
}

// SkillsConfig represents skills configuration.
type SkillsConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Directories []string `mapstructure:"directories"`
}

// AgentConfig represents agent configuration.
type AgentConfig struct {
	PlanningMode   string     `mapstructure:"planning_mode" yaml:"planning_mode"` // planning mode: implicit, explicit, or auto
	Temperature    float64    `mapstructure:"temperature" yaml:"temperature"`     // LLM temperature for generation (0.0-1.0)
	SummaryTemp    float64    `mapstructure:"summary_temp" yaml:"summary_temp"`   // LLM temperature for summarization
	EnableSubAgent bool       `mapstructure:"enable_sub_agent" yaml:"enable_sub_agent"` // enable sub-agent delegation (default: true)
	Plan           PlanConfig `mapstructure:"plan"`                               // plan-specific settings
}

// PlanConfig represents plan system configuration.
type PlanConfig struct {
	EnableReview      bool     `mapstructure:"enable_review"`      // enable plan review before execution
	VerifyCommands    []string `mapstructure:"verify_commands"`    // commands to run in verify phase (e.g., "make test")
	UseReviewerAgent  bool     `mapstructure:"use_reviewer_agent"` // delegate to code-reviewer agent in verify phase
	ParallelExplore   bool     `mapstructure:"parallel_explore"`   // enable parallel exploration with multiple agents
	MaxParallelExplore int     `mapstructure:"max_parallel_explore"` // max concurrent exploration agents (default: 3)
}

// ValidateAgentConfig validates agent configuration parameters.
// This function is reusable for both main AgentConfig and SubAgent meta.
func ValidateAgentConfig(planningMode string, temperature, summaryTemp float64) error {
	// Validate temperature (0.0-1.0)
	if temperature < 0 || temperature > 1 {
		return fmt.Errorf(i18n.T("error.config.temperature_range"), temperature)
	}

	// Validate summary_temp (0.0-1.0)
	if summaryTemp < 0 || summaryTemp > 1 {
		return fmt.Errorf(i18n.T("error.config.summary_temp_range"), summaryTemp)
	}

	// Validate planning_mode
	validModes := []string{"implicit", "explicit", "auto", ""}
	valid := false
	for _, mode := range validModes {
		if planningMode == mode {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf(i18n.T("error.config.invalid_planning_mode"), planningMode)
	}

	return nil
}

// Validate validates the AgentConfig.
func (c *AgentConfig) Validate() error {
	return ValidateAgentConfig(c.PlanningMode, c.Temperature, c.SummaryTemp)
}

// TUIConfig represents terminal UI configuration.
type TUIConfig struct {
	DebugMode bool `mapstructure:"debug_mode"` // Show step counters and detailed logs
}

// AdaptersConfig represents external platform adapters configuration.
type AdaptersConfig struct {
	Enabled         bool                `mapstructure:"enabled"`          // Master switch for all adapters
	EnabledAdapters []string            `mapstructure:"enabled_adapters"` // List of enabled adapter names
	DataDir         string              `mapstructure:"data_dir"`         // Data directory for adapter storage
	Feishu          FeishuAdapterConfig `mapstructure:"feishu"`           // Feishu adapter configuration
}

// FeishuAdapterConfig represents Feishu adapter configuration.
type FeishuAdapterConfig struct {
	Enabled                 bool   `mapstructure:"enabled"`                   // Enable/disable Feishu adapter
	AppID                   string `mapstructure:"app_id"`                    // Feishu App ID
	AppSecret               string `mapstructure:"app_secret"`                // Feishu App Secret
	EncryptKey              string `mapstructure:"encrypt_key"`               // Message encryption key (optional)
	VerificationToken       string `mapstructure:"verification_token"`        // Webhook verification token (optional)
	WebhookPath             string `mapstructure:"webhook_path"`              // Webhook endpoint path
	Port                    string `mapstructure:"port"`                      // HTTP server port
	AsyncProcessing         bool   `mapstructure:"async_processing"`          // Process messages asynchronously
	AutoReply               bool   `mapstructure:"auto_reply"`                // Automatically reply to messages
	ShowProcessingIndicator bool   `mapstructure:"show_processing_indicator"` // Show "processing" indicator message
}

// KnowledgeConfig represents knowledge base configuration.
type KnowledgeConfig struct {
	TopK          int     `mapstructure:"top_k"`           // number of results to retrieve from knowledge base
	RAGTokenRatio float64 `mapstructure:"rag_token_ratio"` // max ratio of context tokens for RAG results
}

// I18nConfig represents internationalization configuration.
type I18nConfig struct {
	Locale     string `mapstructure:"locale"`      // current locale (e.g., en, zh-CN)
	LocalesDir string `mapstructure:"locales_dir"` // custom locales directory (optional)
	Fallback   string `mapstructure:"fallback"`    // fallback locale (default: en)
}

// OrchestratorConfig represents multi-agent orchestrator configuration.
type OrchestratorConfig struct {
	Enabled             bool          `mapstructure:"enabled"`              // Enable orchestrator mode
	MaxSubAgents        int           `mapstructure:"max_sub_agents"`       // Maximum number of sub-agents
	WorkspaceDir        string        `mapstructure:"workspace_dir"`        // Base directory for agent workspaces
	SupervisionInterval time.Duration `mapstructure:"supervision_interval"` // Interval for supervision checks
	StaleDocThreshold   time.Duration `mapstructure:"stale_doc_threshold"`  // Time threshold for stale document detection
	AutoCleanup         bool          `mapstructure:"auto_cleanup"`         // Auto-cleanup finished agents
	SubAgentLLM         *LLMConfig    `mapstructure:"sub_agent_llm"`        // LLM config for sub-agents (nil = inherit)
}

// AgentsConfig represents LLM-triggered SubAgent configuration.
type AgentsConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Directories []string `mapstructure:"directories"`
}

// IntentConfig represents intent recognition configuration.
type IntentConfig struct {
	Enabled             bool    `mapstructure:"enabled"`              // Enable intent recognition
	ConfidenceThreshold float64 `mapstructure:"confidence_threshold"` // Minimum confidence threshold (0.0-1.0)
}

// HabitConfig represents habit tracking configuration.
type HabitConfig struct {
	Enabled          bool    `mapstructure:"enabled"`             // Enable habit tracking
	MinOccurrences   int     `mapstructure:"min_occurrences"`     // Minimum pattern appearances to form a habit
	ConfThreshold    float64 `mapstructure:"conf_threshold"`      // Minimum confidence for valid habit
	MaxActionAgeDays int     `mapstructure:"max_action_age_days"` // Max action age in days
	AnalysisLimit    int     `mapstructure:"analysis_limit"`      // Max actions to analyze
}

// LocationConfig represents location detection configuration.
type LocationConfig struct {
	FixedCity    string `mapstructure:"fixed_city"`    // Fixed city name (overrides auto-detection)
	FixedCountry string `mapstructure:"fixed_country"` // Fixed country name
	AutoDetect   bool   `mapstructure:"auto_detect"`   // Enable IP-based auto-detection
}

// LSPConfig represents LSP server configuration.
type LSPConfig struct {
	Enabled         bool                      `mapstructure:"enabled"`          // Enable LSP support
	Servers         map[string]LSPServerEntry `mapstructure:"servers"`          // language → config
	DefaultTimeout  time.Duration             `mapstructure:"default_timeout"`  // default timeout for LSP operations
}

// LSPServerEntry represents a single LSP server configuration.
type LSPServerEntry struct {
	Command    string            `mapstructure:"command"`           // LSP server command (e.g., "gopls")
	Args       []string          `mapstructure:"args"`              // Additional arguments
	Env        map[string]string `mapstructure:"env,omitempty"`     // Environment variables
	Extensions []string          `mapstructure:"extensions"`        // File extensions (e.g., [".go"])
	Disabled   bool              `mapstructure:"disabled,omitempty"` // Disable this server
}

// UsersConfig represents multi-user configuration.
type UsersConfig struct {
	Default     string       `mapstructure:"default"`     // Default user ID (for CLI mode)
	Definitions []UserConfig `mapstructure:"definitions"` // User definitions
}

// UserConfig represents a single user configuration.
type UserConfig struct {
	ID              string   `mapstructure:"id" yaml:"id"`                                  // User unique identifier
	Name            string   `mapstructure:"name" yaml:"name"`                              // Display name
	APIToken        string   `mapstructure:"api_token" yaml:"api_token"`                    // API authentication token
	ProfilePath     string   `mapstructure:"profile_path" yaml:"profile_path"`              // Path to user's profile.yaml
	KnowledgeDirs   []string `mapstructure:"knowledge_dirs" yaml:"knowledge_dirs"`          // User's knowledge directories (private + shared)
	AllowedSharedKB []string `mapstructure:"allowed_shared_kb" yaml:"allowed_shared_kb"`    // Other users' shared KB this user can access
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	homeDir := filepath.GetHomeDir()

	return &Config{
		Version: "1",
		LLM: LLMConfig{
			Provider:       defaultLLMProvider,
			BaseURL:        constants.DefaultLLMBaseURL,
			Model:          constants.DefaultLLMModel,
			EmbeddingModel: "",
			Retry: RetryConfig{
				MaxRetries:   3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     30 * time.Second,
			},
			Thinking: ThinkingConfig{
				Enabled:         constants.DefaultThinkingEnabled,
				ReasoningEffort: constants.DefaultThinkingEffort,
				BudgetTokens:    constants.DefaultThinkingBudget,
			},
		},
		Memory: MemoryConfig{
			Type:             "sqlite",
			StorageDir:       stdpath.Join(homeDir, constants.DirMemory),
			MaxContext:       defaultMaxContext,
			MaxTokens:        defaultMaxTokens,
			TokenEstimator:   defaultTokenEstimator,
			SummaryThreshold: 0.7,
			ContextThreshold: 0.85,
			Retention: RetentionConfig{
				MaxInactiveAge:  24 * time.Hour, // Default: 24 hours before staleness
				CleanupInterval: 0,             // Default: disabled (explicit config required)
			},
		},
		Tools: ToolsConfig{
			Enabled:        []string{"file_read", "file_write", "file_search", "file_list", "bash", "datetime", "calculator", "web_fetch", "knowledge_search", "knowledge_import", "code_navigate", "text", "glob", "grep", "location"},
			DefaultTimeout: constants.DefaultToolTimeout,
			ShellTimeout:   constants.DefaultShellTimeout,
			SSHTimeout:     constants.DefaultSSHTimeout,
			WebTimeout:     constants.DefaultWebTimeout,
		},
		Log: LogConfig{
			Level:  defaultLogLevel,
			Format: defaultLogFormat,
			Output: defaultLogOutput,
		},
		Permissions: PermissionsConfig{
			DefaultLevel: defaultPermissionLevel,
			Tools: map[string]string{
				"file_read":        PermLevelAllow,
				"file_list":        PermLevelAllow,
				"file_search":      PermLevelAllow,
				"file_write":       PermLevelAsk,
				"bash":             PermLevelAsk,
				"ssh_exec":         PermLevelAsk,
				"datetime":         PermLevelAllow,
				"calculator":       PermLevelAllow,
				"text":             PermLevelAllow,
				"web_fetch":        PermLevelAllow,
				"web_search":       PermLevelAllow,
				"knowledge_search": PermLevelAllow,
				"knowledge_import": PermLevelAsk,
				"code_navigate":    PermLevelAllow,
				"glob":             PermLevelAllow,
				"grep":             PermLevelAllow,
				"location":         PermLevelAllow,
			},
			ShellRestrictions: CommandRestrictions{
				AllowedCommands: []string{},
				DeniedCommands: []string{
					"rm -rf /",
					"rm -rf /*",
					"mkfs *",
					"dd if=*",
					"curl * | sh",
					"curl * | bash",
					"wget * | sh",
					"wget * | bash",
				},
			},
			SSHRestrictions: SSHRestrictions{
				AllowedHosts:    []string{},
				DeniedHosts:     []string{},
				AllowedCommands: []string{},
				DeniedCommands:  []string{},
			},
			TrustedDirs:  []string{}, // Empty by default
			AutoAskTrust: true,       // Auto-ask trust in CLI/TUI modes
		},
		Skills: SkillsConfig{
			Enabled: true,
			Directories: []string{
				stdpath.Join(homeDir, constants.DefaultHomeDir, constants.DirSkills),
			},
		},
		Agent: AgentConfig{
			PlanningMode:   defaultPlanningMode,
			Temperature:    0.7,
			SummaryTemp:    0.3,
			EnableSubAgent: true,
			Plan: PlanConfig{
				EnableReview:       true,
				VerifyCommands:     []string{"make test"},
				UseReviewerAgent:   false,
				ParallelExplore:    false,
				MaxParallelExplore: 3,
			},
		},
		API: APIConfig{
			Port: fmt.Sprintf("%d", constants.DefaultAPIPort),
		},
		Debug: DebugConfig{
			ShowTokens:         true,
			LogTokens:          true,
			LogLLMInteractions: true,
		},
		TUI: TUIConfig{
			DebugMode: false,
		},
		Adapters: AdaptersConfig{
			Enabled:         false,
			DataDir:         stdpath.Join(homeDir, constants.DirData),
			EnabledAdapters: []string{},
			Feishu: FeishuAdapterConfig{
				Enabled:         false,
				WebhookPath:     "/webhook/feishu",
				Port:            "8080",
				AsyncProcessing: true,
				AutoReply:       true,
			},
		},
		Knowledge: KnowledgeConfig{
			TopK:          5,
			RAGTokenRatio: 0.5,
		},
		I18n: I18nConfig{
			Locale:   "en",
			Fallback: "en",
		},
		Orchestrator: OrchestratorConfig{
			Enabled:             false,
			MaxSubAgents:        5,
			WorkspaceDir:        stdpath.Join(homeDir, constants.DefaultHomeDir, defaultWorkspaceDir),
			SupervisionInterval: 30 * time.Second,
			StaleDocThreshold:   5 * time.Minute,
			AutoCleanup:         true,
		},
		Agents: AgentsConfig{
			Enabled: true,
			Directories: []string{
				stdpath.Join(homeDir, constants.DefaultHomeDir, constants.DirAgents),
			},
		},
		Intent: IntentConfig{
			Enabled:             true,
			ConfidenceThreshold: 0.7,
		},
		Users: UsersConfig{
			Default:     "default",
			Definitions: []UserConfig{},
		},
		Habit: HabitConfig{
			Enabled:          true,
			MinOccurrences:   3,
			ConfThreshold:    0.3,
			MaxActionAgeDays: 30,
			AnalysisLimit:    500,
		},
		Location: LocationConfig{
			AutoDetect: true,
		},
		LSP: LSPConfig{
			Enabled:        true,
			DefaultTimeout: 30 * time.Second,
			Servers:        map[string]LSPServerEntry{}, // Use built-in defaults
		},
	}
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	if provider := os.Getenv(EnvLLMProvider); provider != "" {
		cfg.LLM.Provider = provider
	}
	if baseURL := os.Getenv(EnvLLMBaseURL); baseURL != "" {
		cfg.LLM.BaseURL = baseURL
	}
	if model := os.Getenv(EnvLLMModel); model != "" {
		cfg.LLM.Model = model
	}
	if apiKey := os.Getenv(EnvLLMAPIKey); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
}

// Load loads configuration from file.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		// Try default locations
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		configPath = stdpath.Join(homeDir, constants.DefaultHomeDir, constants.DefaultConfigFile)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Apply environment overrides even if config file doesn't exist
		applyEnvOverrides(cfg)
		return cfg, nil // Return default config if file doesn't exist
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("error.config.read_file"), err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("error.config.unmarshal"), err)
	}

	// Fix nested permissions fields that viper may not parse correctly
	// Viper sometimes flattens snake_case to camelCase (e.g., trusted_dirs -> trusteddirs)
	// This fallback ensures fields are loaded even when nested structure parsing fails
	if len(cfg.Permissions.TrustedDirs) == 0 {
		cfg.Permissions.TrustedDirs = v.GetStringSlice("permissions.trusteddirs")
	}
	if !cfg.Permissions.AutoAskTrust {
		cfg.Permissions.AutoAskTrust = v.GetBool("permissions.autoasktrust")
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Initialize i18n system
	localesDir := cfg.I18n.LocalesDir
	if localesDir == "" {
		// Use embedded locales directory
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" {
			homeDir = "."
		}
		localesDir = stdpath.Join(homeDir, constants.DefaultHomeDir, "locales")
	}
	// Initialize i18n (non-fatal: missing locale files fall back to default language)
	if err := i18n.Init(localesDir, cfg.I18n.Locale); err != nil {
		// Non-fatal: continue with default locale
		_ = err
	}

	return cfg, nil
}

// Save saves configuration to file.
func (c *Config) Save(configPath string) error {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("error.config.home_dir"), err)
		}
		configPath = stdpath.Join(homeDir, constants.DefaultHomeDir, constants.DefaultConfigFile)
	}

	// Ensure directory exists
	dir := stdpath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("error.config.create_dir"), err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	v.Set("llm", c.LLM)
	v.Set("memory", c.Memory)
	v.Set("tools", c.Tools)
	v.Set("log", c.Log)
	v.Set("ssh", c.SSH)
	v.Set("permissions", c.Permissions)
	v.Set("skills", c.Skills)
	v.Set("agent", c.Agent)
	v.Set("api", c.API)
	v.Set("tui", c.TUI)
	v.Set("adapters", c.Adapters)
	v.Set("knowledge", c.Knowledge)
	v.Set("i18n", c.I18n)
	v.Set("orchestrator", c.Orchestrator)
	v.Set("intent", c.Intent)
	v.Set("users", c.Users)
	v.Set("habit", c.Habit)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("error.config.write_file"), err)
	}

	return nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Backward compat: default version to "1" if empty
	if c.Version == "" {
		c.Version = "1"
	}
	if c.Version != "1" {
		return fmt.Errorf("%s (got: %s, expected: %s)", i18n.T("error.config.unsupported_version"), c.Version, "1")
	}

	if c.LLM.Provider == "" {
		return errors.New(i18n.T("error.config.llm_provider_required"))
	}

	if c.LLM.BaseURL == "" {
		return errors.New(i18n.T("error.config.llm_baseurl_required"))
	}

	if c.LLM.Model == "" {
		return errors.New(i18n.T("error.config.llm_model_required"))
	}

	// Validate provider
	switch c.LLM.Provider {
	case "ollama", "openai", "anthropic":
		// Valid providers
	default:
		return fmt.Errorf("%s (got: %s)", i18n.T("error.config.invalid_llm_provider"), c.LLM.Provider)
	}

	// Validate log level
	switch c.Log.Level {
	case "debug", "info", "warn", "error", "":
		// Valid levels
	default:
		return fmt.Errorf("%s (got: %s)", i18n.T("error.config.invalid_log_level"), c.Log.Level)
	}

	// Validate Feishu adapter configuration if enabled
	if c.Adapters.Enabled && c.Adapters.Feishu.Enabled {
		if c.Adapters.Feishu.AppID == "" {
			return errors.New(i18n.T("error.config.feishu_appid_empty"))
		}
		if c.Adapters.Feishu.AppSecret == "" {
			return errors.New(i18n.T("error.config.feishu_appsecret_empty"))
		}
	}

	return nil
}
