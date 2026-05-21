// Package sdk provides the unified agent runtime and factories.
// This file provides SessionManager — a user-scoped session management API
// that wraps the underlying session service, eliminating cross-layer imports
// for application code (CLI/TUI/API).
package sdk

import (
	"context"
	"fmt"
	"time"

	sessionMgr "github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/service"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	storagemessage "github.com/oneliang/aura/storage/pkg/message"
)

// SessionInfo mirrors sessionMgr.SessionItem for SDK-level consumers.
// All fields are exported (unlike the internal DTO which uses getter methods).
type SessionInfo struct {
	ID      string
	Name    string
	Created int64
	Updated int64
	Subs    int
	Role    string
}

// SubscriptionInfo mirrors sessionMgr.SubscriptionItem for SDK-level consumers.
type SubscriptionInfo struct {
	ID      string
	Trigger string
	Source  string
	Active  bool
}

// SessionManager provides user-scoped session management operations.
// It wraps session.SessionServiceWrapper with a bound userID, so callers
// don't need to pass userID on every method call.
type SessionManager struct {
	wrapper *sessionMgr.SessionServiceWrapper
	userID  string
}

// NewSessionManager creates a SessionManager from a data directory and config.
// This is the primary factory — it handles store creation, wrapper init, and
// binds the userID for user isolation.
func NewSessionManager(dataDir string, userID string, cfg *config.Config) (*SessionManager, error) {
	svc, err := service.NewServiceFromDataDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session service: %w", err)
	}
	wrapper := svc.NewServiceWrapper(cfg)
	if wrapper == nil {
		return nil, fmt.Errorf("failed to create session wrapper")
	}
	return &SessionManager{
		wrapper: wrapper,
		userID:  userID,
	}, nil
}

// ListSessions returns all sessions for the bound user.
func (m *SessionManager) ListSessions() ([]SessionInfo, error) {
	items, err := m.wrapper.ListSessions(m.userID)
	if err != nil {
		return nil, err
	}
	result := make([]SessionInfo, len(items))
	for i, item := range items {
		result[i] = sessionItemToInfo(&item)
	}
	return result, nil
}

// CreateSession creates a new session with the given name and role.
func (m *SessionManager) CreateSession(name, role string) (*SessionInfo, error) {
	item, err := m.wrapper.CreateSession(name, role, m.userID)
	if err != nil {
		return nil, err
	}
	info := sessionItemToInfo(item)
	return &info, nil
}

// GetSession retrieves a session by ID.
func (m *SessionManager) GetSession(id string) (*SessionInfo, error) {
	item, err := m.wrapper.GetSession(id, m.userID)
	if err != nil {
		return nil, err
	}
	info := sessionItemToInfo(item)
	return &info, nil
}

// GetMostRecentSession returns the most recently updated session for the bound user.
func (m *SessionManager) GetMostRecentSession() (*SessionInfo, error) {
	item, err := m.wrapper.GetMostRecentlyUpdatedSession(m.userID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	info := sessionItemToInfo(item)
	return &info, nil
}

// DeleteSession deletes a session by ID.
func (m *SessionManager) DeleteSession(id string) error {
	return m.wrapper.DeleteSession(id, m.userID)
}

// UpdateSessionRole updates the role for a session.
func (m *SessionManager) UpdateSessionRole(id, role string) error {
	return m.wrapper.UpdateSessionRole(id, role, m.userID)
}

// GetSubscriptions returns all subscriptions for a session.
func (m *SessionManager) GetSubscriptions(sessionID string) ([]SubscriptionInfo, error) {
	items, err := m.wrapper.GetSubscriptions(sessionID, m.userID)
	if err != nil {
		return nil, err
	}
	result := make([]SubscriptionInfo, len(items))
	for i, item := range items {
		result[i] = subscriptionItemToInfo(item)
	}
	return result, nil
}

// sessionItemToInfo converts a manager.SessionItem to a SessionInfo.
func sessionItemToInfo(item *sessionMgr.SessionItem) SessionInfo {
	return SessionInfo{
		ID:      item.ID(),
		Name:    item.Name(),
		Created: item.Created(),
		Updated: item.Updated(),
		Subs:    item.Subs(),
		Role:    item.Role(),
	}
}

// subscriptionItemToInfo converts a manager.SubscriptionItem to a SubscriptionInfo.
func subscriptionItemToInfo(item sessionMgr.SubscriptionItem) SubscriptionInfo {
	return SubscriptionInfo{
		ID:      item.ID(),
		Trigger: item.Trigger(),
		Source:  item.Source(),
		Active:  item.Active(),
	}
}

// AddSubscription adds a subscription to a session.
func (m *SessionManager) AddSubscription(sessionID, trigger, source string) error {
	return m.wrapper.AddSubscription(sessionID, trigger, source, m.userID)
}

// RemoveSubscription removes a subscription from a session.
func (m *SessionManager) RemoveSubscription(sessionID, subscriptionID string) error {
	return m.wrapper.RemoveSubscription(sessionID, subscriptionID, m.userID)
}

// ToggleSubscriptionStatus toggles the active status of a subscription.
func (m *SessionManager) ToggleSubscriptionStatus(sessionID, subscriptionID string) error {
	return m.wrapper.ToggleSubscriptionStatus(sessionID, subscriptionID, m.userID)
}

// ExportSession exports the session messages to a markdown file.
func (m *SessionManager) ExportSession(sessionID, sessionName, outputPath string) (string, error) {
	return m.wrapper.ExportSession(sessionID, sessionName, outputPath, m.userID)
}

// GetStore returns the underlying message store for loading history messages.
func (m *SessionManager) GetStore() *MessageStoreWrapper {
	return &MessageStoreWrapper{store: m.wrapper.GetStore(), userID: m.userID}
}

// GetSessionMgr returns the underlying session manager for cross-layer use.
func (m *SessionManager) GetSessionMgr() *sessionMgr.SessionManager {
	return m.wrapper.GetManager()
}

// GetOrCreateSession returns the ID of the most recently updated session,
// or creates a new session if none exists.
func (m *SessionManager) GetOrCreateSession() (string, error) {
	session, err := m.GetMostRecentSession()
	if err != nil {
		return "", err
	}
	if session != nil {
		return session.ID, nil
	}

	// No sessions — create a new one with timestamp-based name
	name := "session_" + time.Now().Format("20060102150405")
	newSession, err := m.CreateSession(name, "")
	if err != nil {
		return "", err
	}
	return newSession.ID, nil
}

// MessageStore is a type alias for the underlying message store.
type MessageStore = jsonl.MessageStore

// MessageStoreWrapper wraps the underlying store for message loading.
type MessageStoreWrapper struct {
	store  *storage.JSONLStore
	userID string
}

// GetMessages retrieves messages for a session.
func (w *MessageStoreWrapper) GetMessages(ctx context.Context, sessionID string, limit int) ([]storagemessage.Message, error) {
	return w.store.GetMessages(ctx, sessionID, limit, w.userID)
}

// MessageStore returns the underlying JSONL store for runtime persistence.
func (w *MessageStoreWrapper) MessageStore() *jsonl.MessageStore {
	return w.store.MessageStore()
}
