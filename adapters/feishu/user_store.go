// Package feishu provides user identity mapping storage for the Feishu adapter.
package feishu

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// UserInfo stores user identity information.
type UserInfo struct {
	SessionID   string    `json:"session_id"`
	OpenID      string    `json:"open_id"`
	UserID      string    `json:"user_id,omitempty"`
	UnionID     string    `json:"union_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	IsGroup     bool      `json:"is_group"`
	ChatID      string    `json:"chat_id,omitempty"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// UserStore manages user identity mappings.
// Thread-safe for concurrent access.
type UserStore struct {
	mu      sync.RWMutex
	users   map[string]*UserInfo // session_id -> UserInfo
	dataDir string
}

// NewUserStore creates a new UserStore with the given data directory.
func NewUserStore(dataDir string) (*UserStore, error) {
	store := &UserStore{
		users:   make(map[string]*UserInfo),
		dataDir: dataDir,
	}

	// Create data directory if it doesn't exist
	if err := ffp.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("failed to create user store directory: %w", err)
	}

	// Load existing data
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load user store: %w", err)
	}

	return store, nil
}

// GetBySessionID retrieves user info by session ID.
func (s *UserStore) GetBySessionID(sessionID string) (*UserInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[sessionID]
	if !ok {
		return nil, false
	}
	return user, true
}

// GetOpenID retrieves the OpenID for a given session ID.
func (s *UserStore) GetOpenID(sessionID string) (string, bool) {
	user, ok := s.GetBySessionID(sessionID)
	if !ok {
		return "", false
	}
	return user.OpenID, true
}

// Set creates or updates a user info record.
func (s *UserStore) Set(user *UserInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if existing, ok := s.users[user.SessionID]; ok {
		existing.LastSeenAt = now
		existing.OpenID = user.OpenID
		existing.UserID = user.UserID
		existing.UnionID = user.UnionID
		existing.Name = user.Name
		existing.Avatar = user.Avatar
		existing.IsGroup = user.IsGroup
		existing.ChatID = user.ChatID
	} else {
		user.FirstSeenAt = now
		user.LastSeenAt = now
		s.users[user.SessionID] = user
	}

	// Persist to disk
	return s.save()
}

// Delete removes a user info record by session ID.
func (s *UserStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, sessionID)
	return s.save()
}

// GetAll returns all user info records.
func (s *UserStore) GetAll() []*UserInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*UserInfo, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// GetAllSessions returns all session IDs.
func (s *UserStore) GetAllSessions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]string, 0, len(s.users))
	for sessionID := range s.users {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// Count returns the number of stored users.
func (s *UserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// dataFile returns the path to the data file.
func (s *UserStore) dataFile() string {
	return filepath.Join(s.dataDir, "users.json")
}

// save persists the user store to disk.
func (s *UserStore) save() error {
	return utils.WriteJSONFile(s.dataFile(), s.users)
}

// load loads the user store from disk.
func (s *UserStore) load() error {
	var users map[string]*UserInfo
	err := utils.ReadJSONFile(s.dataFile(), &users)
	if err != nil {
		if err == utils.ErrFileNotFound {
			return nil // No existing data
		}
		return fmt.Errorf("failed to load user store: %w", err)
	}

	if users == nil {
		return nil // Empty file
	}

	s.users = users
	return nil
}
