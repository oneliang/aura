// Package user provides user management for multi-user support.
package user

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"gopkg.in/yaml.v3"
)

// Constants for user management.
const (
	DefaultUserID         = ""         // Empty string means legacy single-user mode
	KnowledgePrivate      = "private"  // Private knowledge directory type
	KnowledgeShared       = "shared"   // Shared knowledge directory type
	DefaultCollectionName = "personal" // Default knowledge collection name
)

// Manager manages user lifecycle and authentication.
type Manager struct {
	mu        sync.RWMutex
	users     map[string]*config.UserConfig // ID -> UserConfig
	tokens    map[string]string             // API token -> User ID
	defaultID string
	baseDir   string
}

// NewManager creates a new user manager.
func NewManager(cfg *config.UsersConfig) (*Manager, error) {
	baseDir := ffp.MustAuraHomePath(constants.DirUsers)

	m := &Manager{
		users:     make(map[string]*config.UserConfig),
		tokens:    make(map[string]string),
		defaultID: cfg.Default,
		baseDir:   baseDir,
	}

	// Index users by ID and token
	for i := range cfg.Definitions {
		user := &cfg.Definitions[i]
		if user.ID == "" {
			return nil, fmt.Errorf("user at index %d has empty ID", i)
		}
		m.users[user.ID] = user
		if user.APIToken != "" {
			m.tokens[user.APIToken] = user.ID
		}
	}

	return m, nil
}

// GetUserByID returns a user by ID.
func (m *Manager) GetUserByID(id string) *config.UserConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[id]
}

// GetUserByToken returns a user by API token.
func (m *Manager) GetUserByToken(token string) *config.UserConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if userID, ok := m.tokens[token]; ok {
		return m.users[userID]
	}
	return nil
}

// GetDefaultUser returns the default user.
func (m *Manager) GetDefaultUser() *config.UserConfig {
	return m.GetUserByID(m.defaultID)
}

// ListUsers returns all users.
func (m *Manager) ListUsers() []config.UserConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]config.UserConfig, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users
}

// CreateUser creates a new user.
func (m *Manager) CreateUser(name string) (*config.UserConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate unique ID
	id := generateUserID(name)

	// Check for duplicates
	if _, exists := m.users[id]; exists {
		// Add random suffix if ID exists
		id = fmt.Sprintf("%s_%s", id, generateRandomString(4))
	}

	// Generate API token
	token := generateAPIToken()

	// Create user directory structure
	userDir := filepath.Join(m.baseDir, id)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user directory: %w", err)
	}

	// Create profile path
	profilePath := filepath.Join(userDir, constants.DefaultProfileFile)

	// Create knowledge directories
	privateKB := filepath.Join(userDir, "knowledge", KnowledgePrivate)
	sharedKB := filepath.Join(userDir, "knowledge", KnowledgeShared)
	if err := os.MkdirAll(privateKB, 0755); err != nil {
		return nil, fmt.Errorf("failed to create private knowledge directory: %w", err)
	}
	if err := os.MkdirAll(sharedKB, 0755); err != nil {
		return nil, fmt.Errorf("failed to create shared knowledge directory: %w", err)
	}

	user := &config.UserConfig{
		ID:            id,
		Name:          name,
		APIToken:      token,
		ProfilePath:   profilePath,
		KnowledgeDirs: []string{privateKB, sharedKB},
	}

	m.users[id] = user
	m.tokens[token] = id

	return user, nil
}

// DeleteUser deletes a user by ID.
func (m *Manager) DeleteUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return fmt.Errorf("user not found: %s", id)
	}

	// Remove token index
	if user.APIToken != "" {
		delete(m.tokens, user.APIToken)
	}

	// Remove user directory
	userDir := filepath.Join(m.baseDir, id)
	if err := os.RemoveAll(userDir); err != nil {
		return fmt.Errorf("failed to remove user directory: %w", err)
	}

	delete(m.users, id)
	return nil
}

// SwitchUser switches the default user.
func (m *Manager) SwitchUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; !exists {
		return fmt.Errorf("user not found: %s", id)
	}

	m.defaultID = id
	return nil
}

// GetDefaultUserDir returns the default user directory.
func (m *Manager) GetDefaultUserDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return filepath.Join(m.baseDir, m.defaultID)
}

// GetUserDir returns a user's directory.
func (m *Manager) GetUserDir(id string) string {
	return filepath.Join(m.baseDir, id)
}

// Save persists the user configuration to disk.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usersCfg := config.UsersConfig{
		Default:     m.defaultID,
		Definitions: make([]config.UserConfig, 0, len(m.users)),
	}

	for _, user := range m.users {
		usersCfg.Definitions = append(usersCfg.Definitions, *user)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(usersCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal users config: %w", err)
	}

	// Write to file
	usersFile := ffp.MustAuraHomePath(constants.DefaultUsersFile)
	dir := filepath.Dir(usersFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create users config directory: %w", err)
	}

	if err := os.WriteFile(usersFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write users config: %w", err)
	}

	return nil
}

// LoadFromConfig loads users from config.
func (m *Manager) LoadFromConfig(cfg *config.UsersConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users = make(map[string]*config.UserConfig)
	m.tokens = make(map[string]string)
	m.defaultID = cfg.Default

	for i := range cfg.Definitions {
		user := &cfg.Definitions[i]
		if user.ID == "" {
			continue
		}
		m.users[user.ID] = user
		if user.APIToken != "" {
			m.tokens[user.APIToken] = user.ID
		}
	}

	return nil
}

// LoadConfig loads users configuration from file.
// Returns nil config when users.yaml doesn't exist (legacy single-user mode).
func LoadConfig() (*config.UsersConfig, error) {
	usersFile := ffp.MustAuraHomePath(constants.DefaultUsersFile)

	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		return &config.UsersConfig{
			Default:     DefaultUserID,
			Definitions: []config.UserConfig{},
		}, nil
	}

	data, err := os.ReadFile(usersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	var usersCfg config.UsersConfig
	if err := yaml.Unmarshal(data, &usersCfg); err != nil {
		return nil, fmt.Errorf("failed to parse users file: %w", err)
	}

	return &usersCfg, nil
}

var (
	defaultUserIDOnce sync.Once
	defaultUserID     = DefaultUserID
)

// GetDefaultUserID returns the default user ID from users.yaml.
// Returns empty string if no users configured (legacy mode).
// Result is cached after first call for performance.
func GetDefaultUserID() string {
	defaultUserIDOnce.Do(func() {
		usersCfg, err := LoadConfig()
		if err == nil && len(usersCfg.Definitions) > 0 {
			defaultUserID = usersCfg.Default
		}
	})
	return defaultUserID
}

// GetDefaultUserName returns the default user's display name from users.yaml.
// Returns "User" if no users configured (legacy mode).
func GetDefaultUserName() string {
	usersCfg, err := LoadConfig()
	if err != nil || len(usersCfg.Definitions) == 0 {
		return "User"
	}
	for _, u := range usersCfg.Definitions {
		if u.ID == usersCfg.Default {
			if u.Name != "" {
				return u.Name
			}
			break
		}
	}
	return "User"
}

// GetUserCollectionName returns the collection name for a user.
func GetUserCollectionName(userID string) string {
	if userID == "" || userID == "default" {
		return DefaultCollectionName
	}
	return userID + "_" + DefaultCollectionName
}

// IsLegacyMode returns true if running in legacy single-user mode.
func IsLegacyMode(userID string) bool {
	return userID == "" || userID == "default"
}

// HasOwnership returns true if the user owns the session (or in legacy mode).
// Parameters:
//   - userID: The requesting user ID (empty = legacy mode)
//   - sessionUserID: The session owner's user ID (empty = unowned/legacy)
func HasOwnership(userID, sessionUserID string) bool {
	return userID == "" || sessionUserID == "" || sessionUserID == userID
}

// generateUserID generates a user ID from a name.
func generateUserID(name string) string {
	// Simple slugify: lowercase, replace spaces with underscores
	id := name
	id = filepath.ToSlash(id)

	// For Chinese names, use pinyin or keep as-is with timestamp
	if len([]rune(id)) > 20 {
		// Truncate long names
		runes := []rune(id)
		id = string(runes[:20])
	}

	return id
}

// generateAPIToken generates a random API token.
func generateAPIToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based
		return fmt.Sprintf("sk_%d", os.Getpid())
	}
	return "sk_" + hex.EncodeToString(bytes)
}

// generateRandomString generates a random string.
func generateRandomString(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)[:n]
}
