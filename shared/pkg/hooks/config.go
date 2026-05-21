package hooks

// HooksConfig is the top-level hooks configuration loaded from config.yaml.
// This is a server-wide configuration — both CLI and Server modes load from
// the same config file.
type HooksConfig struct {
	Enabled          bool        `yaml:"enabled" mapstructure:"enabled"`
	SessionStart     []HookEvent `yaml:"SessionStart" mapstructure:"SessionStart"`
	UserPromptSubmit []HookEvent `yaml:"UserPromptSubmit" mapstructure:"UserPromptSubmit"`
	PreToolUse       []HookEvent `yaml:"PreToolUse" mapstructure:"PreToolUse"`
	PostToolUse      []HookEvent `yaml:"PostToolUse" mapstructure:"PostToolUse"`
	PostToolUseFail  []HookEvent `yaml:"PostToolUseFailure" mapstructure:"PostToolUseFailure"`
	Stop             []HookEvent `yaml:"Stop" mapstructure:"Stop"`
	StopFailure      []HookEvent `yaml:"StopFailure" mapstructure:"StopFailure"`
	SubagentStop     []HookEvent `yaml:"SubagentStop" mapstructure:"SubagentStop"`
	PreCompact       []HookEvent `yaml:"PreCompact" mapstructure:"PreCompact"`
	PostCompact      []HookEvent `yaml:"PostCompact" mapstructure:"PostCompact"`
	TaskCreated      []HookEvent `yaml:"TaskCreated" mapstructure:"TaskCreated"`
	TaskCompleted    []HookEvent `yaml:"TaskCompleted" mapstructure:"TaskCompleted"`
	FileChanged      []HookEvent `yaml:"FileChanged" mapstructure:"FileChanged"`
	SessionEnd       []HookEvent `yaml:"SessionEnd" mapstructure:"SessionEnd"`
	PreResponse      []HookEvent `yaml:"PreResponse" mapstructure:"PreResponse"` // Blocking hook before final response
	WatchPaths       []string    `yaml:"watch_paths" mapstructure:"watch_paths"`
}

// HookEvent represents a single hook event configuration with optional tool name
// matching (for tool-related events like PreToolUse, PostToolUse).
type HookEvent struct {
	Matcher string       `yaml:"matcher" mapstructure:"matcher"`
	Hooks   []HookConfig `yaml:"hooks" mapstructure:"hooks"`
}

// HookConfig represents a single command hook.
type HookConfig struct {
	Type    string `yaml:"type" mapstructure:"type"`
	Command string `yaml:"command" mapstructure:"command"`
	Timeout int    `yaml:"timeout" mapstructure:"timeout"`
}
