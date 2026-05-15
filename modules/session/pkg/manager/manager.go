// Package manager provides session management functionality.
package manager

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// SessionManager manages all conversation sessions.
// It only handles session data CRUD operations, not agent/runtime lifecycle.
type SessionManager struct {
	store  *storage.JSONLStore
	router *Router
	config *config.Config // Configuration for session settings
	mu     sync.RWMutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager(store *storage.JSONLStore, cfg *config.Config) (*SessionManager, error) {
	return &SessionManager{
		store:  store,
		router: NewRouter(),
		config: cfg,
	}, nil
}

// CreateSession creates a new session with the given name and subscriptions.
func (m *SessionManager) CreateSession(name string, subscriptions []model.Subscription, role string, userID string) (*model.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load system prompt from role file if specified
	systemPrompt := loadRolePrompt(role)

	now := time.Now()
	session := &model.Session{
		ID:            fmt.Sprintf("session_%s_%s", now.Format("20060102150405"), utils.MustRandString(6)),
		Name:          name,
		UserID:        userID,
		CreatedAt:     now.UnixMilli(),
		UpdatedAt:     now.UnixMilli(),
		Subscriptions: subscriptions,
		SystemPrompt:  systemPrompt,
	}

	// Save to index
	if err := m.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID.
// In multi-user mode, it verifies the session belongs to the specified user.
func (m *SessionManager) GetSession(id string, userID string) (*model.Session, error) {
	return m.store.GetSession(id, userID)
}

// ListSessions returns all sessions for the specified user.
// In legacy mode (userID = ""), returns all sessions.
func (m *SessionManager) ListSessions(userID string) ([]*model.Session, error) {
	return m.store.ListSessions(userID)
}

// DeleteSession deletes a session.
// In multi-user mode, it verifies the session belongs to the specified user.
func (m *SessionManager) DeleteSession(id string, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.store.DeleteSession(ctx, id, userID)
}

// RouteEvent routes an event to the appropriate session for the given user.
// Returns the session ID (empty string if should create new session).
func (m *SessionManager) RouteEvent(source, content, userID string) (string, error) {
	sessions, err := m.store.ListSessions(userID)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	matchedID := m.router.MatchSession(sessions, source, content)
	return matchedID, nil
}

// Close cleans up resources (no-op for data-only manager).
func (m *SessionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
}

// loadRolePrompt loads the system prompt from a role MD file.
// Returns empty string if role is empty, file doesn't exist, or file is empty.
func loadRolePrompt(role string) string {
	if role == "" {
		return ""
	}

	rolePath := ffp.MustAuraHomePath(constants.DirRoles, fmt.Sprintf("%s.md", role))

	content, err := os.ReadFile(rolePath)
	if err != nil {
		return ""
	}

	return string(content)
}

// UpdateSession updates a session's configuration.
// In multi-user mode, verifies the session belongs to the specified user.
func (m *SessionManager) UpdateSession(sessionID string, systemPrompt *string, role *string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Update system prompt from role file if specified
	if role != nil {
		session.SystemPrompt = loadRolePrompt(*role)
	} else if systemPrompt != nil {
		session.SystemPrompt = *systemPrompt
	}

	session.UpdatedAt = time.Now().UnixMilli()

	// Save updated session
	if err := m.store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// SaveSession saves an updated session.
func (m *SessionManager) SaveSession(session *model.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

// GetRouter returns the router for external use.
func (m *SessionManager) GetRouter() *Router {
	return m.router
}
