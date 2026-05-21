// Package service provides session service operations.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/utils"
	storagemessage "github.com/oneliang/aura/storage/pkg/message"
)

// Service provides session management operations.
type Service struct {
	store  *storage.JSONLStore
	router *manager.Router
}

// NewService creates a new session service.
func NewService(store *storage.JSONLStore) *Service {
	return &Service{
		store:  store,
		router: manager.NewRouter(),
	}
}

// NewServiceFromDataDir creates a new session service from a data directory.
// This is the preferred factory method for creating a session service.
func NewServiceFromDataDir(dataDir string) (*Service, error) {
	store, err := storage.NewJSONLStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}
	return NewService(store), nil
}

// NewServiceWrapper creates a SessionServiceWrapper with the given config.
// This provides a complete session management solution with manager capabilities.
func (s *Service) NewServiceWrapper(cfg *config.Config) *manager.SessionServiceWrapper {
	mgr, err := manager.NewSessionManager(s.store, cfg)
	if err != nil {
		// Return a wrapper with nil manager on error - caller should check
		return manager.NewSessionServiceWrapper(nil, s.store)
	}
	return manager.NewSessionServiceWrapper(mgr, s.store)
}

// GetStore returns the underlying storage (for advanced use cases).
func (s *Service) GetStore() *storage.JSONLStore {
	return s.store
}

// GetMessages retrieves messages for a session with optional limit.
// In multi-user mode, verifies the messages belong to the specified user.
func (s *Service) GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]storagemessage.Message, error) {
	return s.store.GetMessages(ctx, sessionID, limit, userID)
}

// List returns all sessions for the specified user.
// Empty userID returns all sessions (legacy mode).
func (s *Service) List(userID string) ([]*model.Session, error) {
	return s.store.ListSessions(userID)
}

// Get retrieves a session by ID.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *Service) Get(id string, userID string) (*model.Session, error) {
	return s.store.GetSession(id, userID)
}

// Create creates a new session with the given name and system prompt.
func (s *Service) Create(name string, subscriptions []model.Subscription, systemPrompt string) (*model.Session, error) {
	now := time.Now()
	session := &model.Session{
		ID:            fmt.Sprintf("session_%s_%s", now.Format("20060102150405"), utils.MustRandString(6)),
		Name:          name,
		CreatedAt:     now.UnixMilli(),
		UpdatedAt:     now.UnixMilli(),
		Subscriptions: subscriptions,
		SystemPrompt:  systemPrompt,
	}

	// Save to index
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// Update updates a session's configuration.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *Service) Update(id string, systemPrompt *string, subscriptions *[]model.Subscription, userID string) error {
	session, err := s.store.GetSession(id, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if systemPrompt != nil {
		session.SystemPrompt = *systemPrompt
	}
	if subscriptions != nil {
		session.Subscriptions = *subscriptions
	}
	session.UpdatedAt = time.Now().UnixMilli()

	if err := s.store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// Delete deletes a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *Service) Delete(ctx context.Context, id string, userID string) error {
	return s.store.DeleteSession(ctx, id, userID)
}

// AddSubscription adds a subscription to a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *Service) AddSubscription(sessionID string, sub model.Subscription, userID string) error {
	session, err := s.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Subscriptions = append(session.Subscriptions, sub)
	session.UpdatedAt = time.Now().UnixMilli()

	if err := s.store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// RemoveSubscription removes a subscription from a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *Service) RemoveSubscription(sessionID string, subID string, userID string) error {
	session, err := s.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Find and remove subscription
	found := false
	session.Subscriptions = filterSubscriptions(session.Subscriptions, func(s model.Subscription) bool {
		if s.ID == subID {
			found = true
			return false
		}
		return true
	})

	if !found {
		return fmt.Errorf("subscription %s not found", subID)
	}

	session.UpdatedAt = time.Now().UnixMilli()

	if err := s.store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// RouteEvent routes an event to the appropriate session based on subscriptions.
// Returns the session ID (empty string if no match).
// In multi-user mode, only routes within the user's sessions.
func (s *Service) RouteEvent(source, content, userID string) (string, error) {
	sessions, err := s.store.ListSessions(userID)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	matchedID := s.router.MatchSession(sessions, source, content)
	return matchedID, nil
}

// filterSubscriptions filters subscriptions based on a predicate function.
func filterSubscriptions(subs []model.Subscription, keep func(model.Subscription) bool) []model.Subscription {
	var result []model.Subscription
	for _, sub := range subs {
		if keep(sub) {
			result = append(result, sub)
		}
	}
	return result
}
