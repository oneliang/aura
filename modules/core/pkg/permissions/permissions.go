// Package permissions provides multi-level permission control for tools.
package permissions

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandRestrictions holds command whitelist/blacklist for execute-level tools.
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

// PermissionConfig represents the permission configuration.
type PermissionConfig struct {
	// DefaultLevel is the default permission control level for unknown tools.
	// Values: "allow", "ask", "deny"
	DefaultLevel PermissionControlLevel `mapstructure:"default_level"`

	// Tools maps tool names to their permission control level.
	// Values: "allow", "ask", "deny"
	Tools map[string]PermissionControlLevel `mapstructure:"tools"`

	// ShellRestrictions holds command restrictions for bash.
	ShellRestrictions CommandRestrictions `mapstructure:"shell_restrictions"`

	// SSHRestrictions holds SSH-specific restrictions.
	SSHRestrictions SSHRestrictions `mapstructure:"ssh"`

	// TrustedDirs holds list of trusted directory paths.
	TrustedDirs []string `mapstructure:"trusted_dirs"`

	// AutoAskTrust indicates whether to auto-ask for trust in CLI/TUI modes.
	AutoAskTrust bool `mapstructure:"auto_ask_trust"`
}

// DefaultPermissionConfig returns the default permission configuration.
func DefaultPermissionConfig() *PermissionConfig {
	return &PermissionConfig{
		DefaultLevel: ControlAsk,
		Tools: map[string]PermissionControlLevel{
			"file_read":   ControlAllow,
			"file_list":   ControlAllow,
			"file_search": ControlAllow,
			// Search tools (Claude Code style, read-only)
			"glob":             ControlAllow,
			"grep":             ControlAllow,
			"file_write":       ControlAsk,
			"bash":             ControlAsk,
			"ssh_exec":         ControlAsk,
			"datetime":         ControlAllow,
			"calculator":       ControlAllow,
			"text":             ControlAllow,
			"web_fetch":        ControlAllow,
			"web_search":       ControlAllow,
			"knowledge_import": ControlAsk,
			"knowledge_search": ControlAllow,
		},
		ShellRestrictions: CommandRestrictions{
			// Safe default allowed commands for common development tasks
			AllowedCommands: []string{
				// File operations
				"ls", "dir", "find", "du", "df",
				"cat", "head", "tail", "less", "more",
				"cp", "mv", "rename",
				"mkdir", "rmdir", "touch",
				"chmod", "chown",
				// Text processing
				"grep", "sed", "awk", "cut", "sort", "uniq", "wc",
				"jq",            // JSON processing
				"diff", "patch", // Code operations
				"git",                 // Version control
				"go",                  // Go development
				"npm", "yarn", "pnpm", // Node.js development
				"make", "cmake", "ninja", // Build tools
				// System info
				"pwd", "whoami", "id", "uname", "hostname",
				"ps", "top", "htop", "free", "uptime",
				"echo", "printf",
				// Network (read-only)
				"curl", "wget", "ping", "netstat", "ss",
				// Development tools
				"docker", "kubectl", "helm",
			},
			DeniedCommands: []string{
				// Destructive operations
				"rm -rf /*",
				"rm -rf /",
				"rm -rf ~",
				"rm -rf $HOME",
				"mkfs *",
				"dd if=*",
				"dd of=/dev/*",
				// Remote code execution
				"curl * | sh",
				"curl * | bash",
				"wget * | sh",
				"wget * | bash",
				// Privilege escalation
				"sudo *",
				"su -",
				"su root",
				// System modification
				"chmod -R 777 *",
				"chown -R *:* /",
				"mount * /",
				"umount -l *",
				// Dangerous writes
				"echo * > /etc/*",
				"echo * > /proc/*",
				"echo * > /sys/*",
				// Data exfiltration
				"cat /etc/shadow",
				"cat /etc/passwd | *",
				// Fork bomb
				":(){ :|:& };:",
				// System shutdown
				"shutdown -h now",
				"reboot",
				"poweroff",
				"halt",
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
	}
}

// CommandChecker checks if a command is allowed based on restrictions.
type CommandChecker struct {
	allowedPatterns []*regexp.Regexp
	deniedPatterns  []*regexp.Regexp
}

// NewCommandChecker creates a new command checker from restrictions.
func NewCommandChecker(r CommandRestrictions) (*CommandChecker, error) {
	checker := &CommandChecker{
		allowedPatterns: make([]*regexp.Regexp, 0),
		deniedPatterns:  make([]*regexp.Regexp, 0),
	}

	// Compile allowed patterns
	for _, pattern := range r.AllowedCommands {
		re, err := patternToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed pattern %q: %w", pattern, err)
		}
		checker.allowedPatterns = append(checker.allowedPatterns, re)
	}

	// Compile denied patterns
	for _, pattern := range r.DeniedCommands {
		re, err := patternToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid denied pattern %q: %w", pattern, err)
		}
		checker.deniedPatterns = append(checker.deniedPatterns, re)
	}

	return checker, nil
}

// IsAllowed checks if a command is allowed.
// Uses deny-by-default security model:
// - If denied patterns match, reject
// - If allowed patterns are configured, only allow matching commands
// - If no allowed patterns configured, deny by default (security first)
func (c *CommandChecker) IsAllowed(command string) (bool, string) {
	// Check denied patterns first (deny takes precedence)
	for _, re := range c.deniedPatterns {
		if re.MatchString(command) {
			return false, fmt.Sprintf("command denied by pattern: %s", re.String())
		}
	}

	// If no allowed patterns configured, deny by default (deny-by-default security model)
	if len(c.allowedPatterns) == 0 {
		return false, fmt.Sprintf("command not in allowed list (deny-by-default): %s", command)
	}

	// Check if command matches any allowed pattern
	for _, re := range c.allowedPatterns {
		if re.MatchString(command) {
			return true, ""
		}
	}

	return false, fmt.Sprintf("command not in allowed list: %s", command)
}

// NewPermissionConfig creates a new PermissionConfig from raw config values.
// This handles the conversion of tool permission levels from strings to PermissionControlLevel.
func NewPermissionConfig(
	defaultLevel string,
	tools map[string]string,
	shellRestrictions CommandRestrictions,
	sshRestrictions SSHRestrictions,
	trustedDirs []string,
	autoAskTrust bool,
) *PermissionConfig {
	// Convert tools map from string to PermissionControlLevel
	convertedTools := make(map[string]PermissionControlLevel, len(tools))
	for toolName, level := range tools {
		convertedTools[toolName] = ParsePermissionControlLevel(level)
	}

	return &PermissionConfig{
		DefaultLevel:      ParsePermissionControlLevel(defaultLevel),
		Tools:             convertedTools,
		ShellRestrictions: shellRestrictions,
		SSHRestrictions:   sshRestrictions,
		TrustedDirs:       trustedDirs,
		AutoAskTrust:      autoAskTrust,
	}
}

// patternToRegex converts a shell-like pattern to a regex.
// Supports:
// - * matches any sequence of characters (except spaces by default)
// - Patterns are anchored at start and end
// - Shell operators (|, ;, &, >, <, etc.) are treated as literals for matching
func patternToRegex(pattern string) (*regexp.Regexp, error) {
	// Escape all special regex characters first
	escaped := regexp.QuoteMeta(pattern)

	// Convert escaped \* back to .* for wildcard matching
	// This allows patterns like "rm *" to match "rm -rf /tmp"
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")

	// Handle common shell pattern variations
	// Convert multiple spaces to single space matcher
	escaped = strings.ReplaceAll(escaped, "  ", " +")

	// Anchor the pattern to match full command
	regexStr := "^" + escaped + "$"

	// Compile with case-insensitive flag for more flexible matching
	return regexp.Compile(regexStr)
}

// patternToHostRegex converts a hostname pattern to a regex.
// Hostname patterns are simpler - just support * wildcard.
func patternToHostRegex(pattern string) (*regexp.Regexp, error) {
	// Escape all special regex characters first
	escaped := regexp.QuoteMeta(pattern)

	// Convert escaped \* back to .* for wildcard matching
	// This allows patterns like "*.example.com" to match "server.example.com"
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")

	// Anchor the pattern to match full hostname
	regexStr := "^" + escaped + "$"

	return regexp.Compile(regexStr)
}

// SSHChecker checks SSH-specific restrictions.
type SSHChecker struct {
	allowedHostPatterns    []*regexp.Regexp
	deniedHostPatterns     []*regexp.Regexp
	allowedCommandPatterns []*regexp.Regexp
	deniedCommandPatterns  []*regexp.Regexp
}

// NewSSHChecker creates a new SSH checker from restrictions.
func NewSSHChecker(r SSHRestrictions) (*SSHChecker, error) {
	checker := &SSHChecker{
		allowedHostPatterns:    make([]*regexp.Regexp, 0),
		deniedHostPatterns:     make([]*regexp.Regexp, 0),
		allowedCommandPatterns: make([]*regexp.Regexp, 0),
		deniedCommandPatterns:  make([]*regexp.Regexp, 0),
	}

	// Compile host patterns using host-specific regex converter
	for _, pattern := range r.AllowedHosts {
		re, err := patternToHostRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed host pattern %q: %w", pattern, err)
		}
		checker.allowedHostPatterns = append(checker.allowedHostPatterns, re)
	}

	for _, pattern := range r.DeniedHosts {
		re, err := patternToHostRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid denied host pattern %q: %w", pattern, err)
		}
		checker.deniedHostPatterns = append(checker.deniedHostPatterns, re)
	}

	// Compile command patterns using shell pattern regex converter
	for _, pattern := range r.AllowedCommands {
		re, err := patternToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed command pattern %q: %w", pattern, err)
		}
		checker.allowedCommandPatterns = append(checker.allowedCommandPatterns, re)
	}

	for _, pattern := range r.DeniedCommands {
		re, err := patternToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid denied command pattern %q: %w", pattern, err)
		}
		checker.deniedCommandPatterns = append(checker.deniedCommandPatterns, re)
	}

	return checker, nil
}

// IsHostAllowed checks if a host is allowed.
func (c *SSHChecker) IsHostAllowed(host string) (bool, string) {
	// Check denied hosts first
	for _, re := range c.deniedHostPatterns {
		if re.MatchString(host) {
			return false, fmt.Sprintf("SSH host denied: %s", host)
		}
	}

	// If no allowed hosts configured, allow by default
	if len(c.allowedHostPatterns) == 0 {
		return true, ""
	}

	// Check if host matches any allowed pattern
	for _, re := range c.allowedHostPatterns {
		if re.MatchString(host) {
			return true, ""
		}
	}

	return false, fmt.Sprintf("SSH host not in allowed list: %s", host)
}

// IsCommandAllowed checks if an SSH command is allowed.
func (c *SSHChecker) IsCommandAllowed(command string) (bool, string) {
	// Check denied commands first
	for _, re := range c.deniedCommandPatterns {
		if re.MatchString(command) {
			return false, fmt.Sprintf("SSH command denied: %s", command)
		}
	}

	// If no allowed commands configured, allow by default
	if len(c.allowedCommandPatterns) == 0 {
		return true, ""
	}

	// Check if command matches any allowed pattern
	for _, re := range c.allowedCommandPatterns {
		if re.MatchString(command) {
			return true, ""
		}
	}

	return false, fmt.Sprintf("SSH command not in allowed list: %s", command)
}
