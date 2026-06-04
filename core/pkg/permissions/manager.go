// Package permissions provides multi-level permission control for tools.
package permissions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Manager manages permission checking for tools.
type Manager struct {
	mu           sync.RWMutex
	config       *PermissionConfig
	commandCheck *CommandChecker
	sshCheck     *SSHChecker
	sessions     map[string]*SessionPermissions
	trustedDirs  []string // List of trusted directory absolute paths
}

// SessionPermissions holds per-session permission grants.
type SessionPermissions struct {
	SessionID       string
	GrantedTools    map[string]bool
	GrantedCommands map[string]bool // For one-time command grants
}

// NewManager creates a new permission manager.
func NewManager(config *PermissionConfig) (*Manager, error) {
	if config == nil {
		config = DefaultPermissionConfig()
	}

	commandCheck, err := NewCommandChecker(config.ShellRestrictions)
	if err != nil {
		return nil, fmt.Errorf("failed to create command checker: %w", err)
	}

	sshCheck, err := NewSSHChecker(config.SSHRestrictions)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH checker: %w", err)
	}

	// Populate trustedDirs from config
	trustedDirs, err := resolveTrustedDirs(config.TrustedDirs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve trusted dirs: %w", err)
	}

	return &Manager{
		config:       config,
		commandCheck: commandCheck,
		sshCheck:     sshCheck,
		sessions:     make(map[string]*SessionPermissions),
		trustedDirs:  trustedDirs,
	}, nil
}

// CheckPermission checks if a tool can be executed.
// Returns (allowed, requiresConfirmation, reason).
// Per-tool settings take precedence over default level.
func (m *Manager) CheckPermission(ctx context.Context, toolName string, params map[string]any) (bool, bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check tool-specific permission setting FIRST (takes precedence)
	controlLevel, ok := m.config.Tools[toolName]
	if ok {
		switch controlLevel {
		case ControlAllow:
			return true, false, ""
		case ControlDeny:
			return false, false, fmt.Sprintf("tool %q is denied by configuration", toolName)
		case ControlAsk:
			// Check session grants before requiring confirmation
			for _, session := range m.sessions {
				if session.GrantedTools[toolName] {
					return true, false, "" // Granted in session
				}
			}
			return true, true, "" // Requires confirmation
		}
	}

	// Fall back to default level for unknown tools
	if m.config.DefaultLevel == ControlAllow {
		return true, false, ""
	}

	// Check session-based grants for unknown tools
	for _, session := range m.sessions {
		if session.GrantedTools[toolName] {
			return true, false, ""
		}
	}

	// Handle default level for unknown tools
	switch m.config.DefaultLevel {
	case ControlDeny:
		return false, false, fmt.Sprintf("tool %q is denied by default level", toolName)
	case ControlAsk:
		return true, true, ""
	default:
		// Unknown default level - require confirmation
		return true, true, ""
	}
}

// CheckCommand checks if a shell command is allowed.
func (m *Manager) CheckCommand(command string) (bool, string) {
	return m.commandCheck.IsAllowed(command)
}

// CheckSSHHost checks if an SSH host is allowed.
func (m *Manager) CheckSSHHost(host string) (bool, string) {
	return m.sshCheck.IsHostAllowed(host)
}

// CheckSSHCommand checks if an SSH command is allowed.
func (m *Manager) CheckSSHCommand(command string) (bool, string) {
	return m.sshCheck.IsCommandAllowed(command)
}

// GrantSessionPermission grants a tool permission for a session.
func (m *Manager) GrantSessionPermission(sessionID, toolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grantSession(sessionID)
	m.sessions[sessionID].GrantedTools[toolName] = true
}

// GrantSessionCommand grants a one-time command permission for a session.
func (m *Manager) GrantSessionCommand(sessionID, command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grantSession(sessionID)
	m.sessions[sessionID].GrantedCommands[command] = true
}

// grantSession creates a session if it doesn't exist (must be called with lock held).
func (m *Manager) grantSession(sessionID string) {
	if _, ok := m.sessions[sessionID]; !ok {
		m.sessions[sessionID] = &SessionPermissions{
			SessionID:       sessionID,
			GrantedTools:    make(map[string]bool),
			GrantedCommands: make(map[string]bool),
		}
	}
}

// RevokeSessionPermission revokes a tool permission for a session.
func (m *Manager) RevokeSessionPermission(sessionID, toolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		delete(session.GrantedTools, toolName)
	}
}

// ClearSession clears all permissions for a session.
func (m *Manager) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
}

// UpdateConfig updates the permission configuration.
func (m *Manager) UpdateConfig(config *PermissionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	commandCheck, err := NewCommandChecker(config.ShellRestrictions)
	if err != nil {
		return fmt.Errorf("failed to create command checker: %w", err)
	}

	sshCheck, err := NewSSHChecker(config.SSHRestrictions)
	if err != nil {
		return fmt.Errorf("failed to create SSH checker: %w", err)
	}

	m.config = config
	m.commandCheck = commandCheck
	m.sshCheck = sshCheck

	// Update trusted dirs
	m.trustedDirs, err = resolveTrustedDirs(config.TrustedDirs)
	if err != nil {
		return fmt.Errorf("failed to resolve trusted dirs: %w", err)
	}

	return nil
}

// GetConfig returns the current permission configuration.
func (m *Manager) GetConfig() *PermissionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetTrustedDirs sets the trusted directories from config.
func (m *Manager) SetTrustedDirs(dirs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	m.trustedDirs, err = resolveTrustedDirs(dirs)
	if err != nil {
		return fmt.Errorf("failed to resolve trusted dirs: %w", err)
	}
	return nil
}

// IsTrustedPath checks if a path is within any trusted directory (including subdirectories).
// It resolves symlinks to prevent symlink-based bypass attacks.
func (m *Manager) IsTrustedPath(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no trusted dirs configured, allow all (backward compatibility)
	if len(m.trustedDirs) == 0 {
		return true
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Resolve symlinks to get the real path
	// This prevents attacks where a trusted path contains a symlink to a sensitive location
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the path doesn't exist yet (e.g., writing a new file),
		// check the parent directory for symlinks
		parentDir := filepath.Dir(absPath)
		if parentEval, err := filepath.EvalSymlinks(parentDir); err == nil {
			realPath = filepath.Join(parentEval, filepath.Base(absPath))
		} else {
			// Can't resolve, use original path (conservative approach)
			realPath = absPath
		}
	}

	// Use shared utility function for consistent subpath checking
	for _, trustedDir := range m.trustedDirs {
		// Also resolve symlinks in trusted directory
		realTrustedDir, err := filepath.EvalSymlinks(trustedDir)
		if err != nil {
			// If trusted dir can't be resolved, use original
			realTrustedDir = trustedDir
		}

		// Use filepath.Rel for accurate subpath checking
		rel, err := filepath.Rel(realTrustedDir, realPath)
		if err != nil {
			continue
		}
		// If rel doesn't start with "..", the path is within the trusted dir
		if !strings.HasPrefix(rel, "..") {
			return true
		}
	}

	return false
}

// resolveTrustedDirs resolves a list of directory paths to absolute paths.
func resolveTrustedDirs(dirs []string) ([]string, error) {
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dir %q: %w", dir, err)
		}
		result = append(result, absDir)
	}
	return result, nil
}

// AddTrustedDir adds a directory to the trusted list.
func (m *Manager) AddTrustedDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory %q: %w", dir, err)
	}

	// Check if already trusted
	for _, trusted := range m.trustedDirs {
		if trusted == absDir {
			return nil // Already trusted, no-op
		}
	}

	m.trustedDirs = append(m.trustedDirs, absDir)
	return nil
}

// GetTrustedDirs returns a copy of the trusted directories.
func (m *Manager) GetTrustedDirs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.trustedDirs))
	copy(result, m.trustedDirs)
	return result
}

// CheckAndAskTrust checks if the current working directory is trusted.
// Returns (needAsk, currentDir, err).
func (m *Manager) CheckAndAskTrust() (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return false, "", fmt.Errorf("failed to get current directory: %w", err)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return false, "", fmt.Errorf("failed to resolve current directory: %w", err)
	}

	// If no trusted dirs configured, need to ask
	if len(m.trustedDirs) == 0 {
		return true, absCwd, nil
	}

	// Check if current directory is within any trusted directory
	for _, trustedDir := range m.trustedDirs {
		rel, err := filepath.Rel(trustedDir, absCwd)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(rel, "..") {
			// Already trusted
			return false, absCwd, nil
		}
	}

	// Not trusted, need to ask
	return true, absCwd, nil
}

// CloneWithDowngrade creates a new PermissionManager with downgraded control level.
// The cloned manager inherits the parent's configuration but applies a more restrictive
// default control level. This is useful for sub-agents that need limited permissions.
func (m *Manager) CloneWithDowngrade(downgradeLevel PermissionControlLevel) (*Manager, error) {
	// Phase 1: Copy config under lock (fast)
	cfg, trustedDirs := m.cloneConfigValues(downgradeLevel)

	// Phase 2: Create checkers outside lock (may involve I/O)
	commandCheck, err := NewCommandChecker(cfg.ShellRestrictions)
	if err != nil {
		return nil, fmt.Errorf("failed to create command checker: %w", err)
	}

	sshCheck, err := NewSSHChecker(cfg.SSHRestrictions)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH checker: %w", err)
	}

	return &Manager{
		config:       cfg,
		commandCheck: commandCheck,
		sshCheck:     sshCheck,
		sessions:     make(map[string]*SessionPermissions),
		trustedDirs:  trustedDirs,
	}, nil
}

// cloneConfigValues copies config values under read lock.
func (m *Manager) cloneConfigValues(downgradeLevel PermissionControlLevel) (*PermissionConfig, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clonedTools := make(map[string]PermissionControlLevel)
	for tool, level := range m.config.Tools {
		if level == ControlAllow && isReadOnlyTool(tool) {
			clonedTools[tool] = ControlAllow
		} else {
			clonedTools[tool] = downgradeLevel
		}
	}

	clonedCfg := &PermissionConfig{
		DefaultLevel:      downgradeLevel,
		Tools:             clonedTools,
		ShellRestrictions: m.config.ShellRestrictions,
		SSHRestrictions:   m.config.SSHRestrictions,
		TrustedDirs:       m.config.TrustedDirs,
	}

	clonedTrustedDirs := make([]string, len(m.trustedDirs))
	copy(clonedTrustedDirs, m.trustedDirs)

	return clonedCfg, clonedTrustedDirs
}

// isReadOnlyTool checks if a tool name indicates a read-only operation.
func isReadOnlyTool(toolName string) bool {
	readOnlyTools := []string{
		"file_read", "file_list", "file_search",
		"glob", "grep",
		"datetime", "calculator",
		"knowledge_search", "code_navigate",
		"web_fetch", "web_search",
		"location",
	}
	for _, ro := range readOnlyTools {
		if toolName == ro {
			return true
		}
	}
	return false
}
