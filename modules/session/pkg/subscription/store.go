// Package subscription provides subscription management for scheduled notifications.
package subscription

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// Store manages subscription persistence.
type Store struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription // id -> Subscription
	sessionIndex  map[string][]string      // session_id -> [subscription_ids]
	dataDir       string
}

// NewStore creates a new subscription store.
func NewStore(dataDir string) (*Store, error) {
	store := &Store{
		subscriptions: make(map[string]*Subscription),
		sessionIndex:  make(map[string][]string),
		dataDir:       dataDir,
	}

	// Create data directory
	if err := ffp.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("failed to create subscription store directory: %w", err)
	}

	// Load existing data
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load subscription store: %w", err)
	}

	return store, nil
}

// Create adds a new subscription.
func (s *Store) Create(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub.ID == "" {
		return fmt.Errorf("subscription ID is required")
	}

	// Check for duplicates
	if _, exists := s.subscriptions[sub.ID]; exists {
		return fmt.Errorf("subscription %s already exists", sub.ID)
	}

	s.subscriptions[sub.ID] = sub
	s.sessionIndex[sub.SessionID] = append(s.sessionIndex[sub.SessionID], sub.ID)

	return s.save()
}

// Get retrieves a subscription by ID.
func (s *Store) Get(id string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subscriptions[id]
	return sub, ok
}

// Update modifies an existing subscription.
func (s *Store) Update(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[sub.ID]; !exists {
		return fmt.Errorf("subscription %s not found", sub.ID)
	}

	sub.UpdatedAt = time.Now()
	s.subscriptions[sub.ID] = sub

	return s.save()
}

// Delete removes a subscription.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	// Remove from session index
	delete(s.subscriptions, id)
	s.removeSessionIndex(sub.SessionID, id)

	return s.save()
}

// GetBySessionID retrieves all subscriptions for a session.
func (s *Store) GetBySessionID(sessionID string) []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.sessionIndex[sessionID]
	if !ok {
		return nil
	}

	subs := make([]*Subscription, 0, len(ids))
	for _, id := range ids {
		if sub, exists := s.subscriptions[id]; exists {
			subs = append(subs, sub)
		}
	}
	return subs
}

// GetAll retrieves all subscriptions.
func (s *Store) GetAll() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subs := make([]*Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// GetActive retrieves all active subscriptions.
func (s *Store) GetActive() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subs := make([]*Subscription, 0)
	for _, sub := range s.subscriptions {
		if sub.IsActive {
			subs = append(subs, sub)
		}
	}
	return subs
}

// Count returns the total number of subscriptions.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscriptions)
}

func (s *Store) removeSessionIndex(sessionID, subID string) {
	ids, ok := s.sessionIndex[sessionID]
	if !ok {
		return
	}

	for i, id := range ids {
		if id == subID {
			s.sessionIndex[sessionID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}

func (s *Store) dataFile() string {
	return filepath.Join(s.dataDir, "subscriptions.json")
}

func (s *Store) save() error {
	return utils.WriteJSONFile(s.dataFile(), s.subscriptions)
}

func (s *Store) load() error {
	var subs map[string]*Subscription
	err := utils.ReadJSONFile(s.dataFile(), &subs)
	if err != nil {
		if err == utils.ErrFileNotFound {
			return nil
		}
		return fmt.Errorf("failed to load subscriptions: %w", err)
	}

	if subs == nil {
		return nil
	}

	s.subscriptions = subs

	// Rebuild session index
	s.sessionIndex = make(map[string][]string)
	for id, sub := range subs {
		s.sessionIndex[sub.SessionID] = append(s.sessionIndex[sub.SessionID], id)
	}

	return nil
}
