// Package storage provides JSONL-based storage for session data.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/shared/pkg/user"
	"github.com/oneliang/aura/shared/pkg/utils"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	"github.com/oneliang/aura/storage/pkg/message"
)

// SessionIndex manages session metadata in memory with JSON persistence.
type SessionIndex struct {
	sessions map[string]*model.Session
	path     string
	mu       sync.RWMutex
}

// NewSessionIndex creates or loads a session index from disk.
func NewSessionIndex(path string) (*SessionIndex, error) {
	idx := &SessionIndex{
		sessions: make(map[string]*model.Session),
		path:     path,
	}

	// Try to load existing index
	if err := idx.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// File doesn't exist, create empty index
		return idx, nil
	}

	return idx, nil
}

// load reads the index from disk.
func (idx *SessionIndex) load() error {
	var sessions []*model.Session
	err := utils.ReadJSONFile(idx.path, &sessions)
	if err != nil {
		if err == utils.ErrFileNotFound {
			return nil
		}
		return fmt.Errorf("failed to read index: %w", err)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.sessions = make(map[string]*model.Session, len(sessions))
	for _, s := range sessions {
		idx.sessions[s.ID] = s
	}

	return nil
}

// save writes the index to disk.
func (idx *SessionIndex) save() error {
	idx.mu.RLock()
	sessions := make([]*model.Session, 0, len(idx.sessions))
	for _, s := range idx.sessions {
		sessions = append(sessions, s)
	}
	idx.mu.RUnlock()

	return utils.WriteJSONFile(idx.path, sessions)
}

// Add adds a session to the index.
func (idx *SessionIndex) Add(session *model.Session) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.sessions[session.ID] = session
}

// Get retrieves a session from the index.
func (idx *SessionIndex) Get(id string) (*model.Session, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	s, ok := idx.sessions[id]
	return s, ok
}

// GetWithOwner retrieves a session and verifies ownership.
// Returns nil if session not found or owner doesn't match (in multi-user mode).
func (idx *SessionIndex) GetWithOwner(id string, userID string) (*model.Session, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	s, ok := idx.sessions[id]
	if !ok {
		return nil, false
	}
	// Verify ownership in multi-user mode
	if !user.HasOwnership(userID, s.UserID) {
		return nil, false
	}
	return s, true
}

// Delete removes a session from the index.
func (idx *SessionIndex) Delete(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.sessions, id)
}

// DeleteWithOwner removes a session from the index after verifying ownership.
// Returns true if deleted, false if session not found or owner doesn't match.
func (idx *SessionIndex) DeleteWithOwner(id string, userID string) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	s, ok := idx.sessions[id]
	if !ok {
		return false
	}
	// Verify ownership in multi-user mode
	if !user.HasOwnership(userID, s.UserID) {
		return false
	}
	delete(idx.sessions, id)
	return true
}

// List returns all sessions in the index.
func (idx *SessionIndex) List() []*model.Session {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sessions := make([]*model.Session, 0, len(idx.sessions))
	for _, s := range idx.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// ListByUser returns sessions filtered by userID.
func (idx *SessionIndex) ListByUser(userID string) []*model.Session {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sessions := make([]*model.Session, 0, len(idx.sessions))
	for _, s := range idx.sessions {
		// Legacy mode: return all sessions
		if user.IsLegacyMode(userID) {
			sessions = append(sessions, s)
			continue
		}
		// Multi-user mode: filter by owner
		if user.HasOwnership(userID, s.UserID) {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// JSONLStore provides JSONL-based storage for session messages.
type JSONLStore struct {
	dataDir      string
	index        *SessionIndex
	messageStore *jsonl.MessageStore
	mu           sync.RWMutex
}

// NewJSONLStore creates a new JSONL store in the specified directory.
func NewJSONLStore(dataDir string) (*JSONLStore, error) {
	// Create data directory if it doesn't exist
	if err := ffp.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create or load session index
	index, err := NewSessionIndex(filepath.Join(dataDir, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create session index: %w", err)
	}

	// Create message store
	msgStore, err := jsonl.NewMessageStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create message store: %w", err)
	}

	return &JSONLStore{
		dataDir:      dataDir,
		index:        index,
		messageStore: msgStore,
	}, nil
}

// MessageStore returns the underlying message store.
func (s *JSONLStore) MessageStore() *jsonl.MessageStore {
	return s.messageStore
}

// getSessionFilePath returns the path to a session's JSONL file.
func (s *JSONLStore) getSessionFilePath(sessionID string) string {
	return filepath.Join(s.dataDir, sessionID+".jsonl")
}

// SaveSession saves session metadata to the index.
func (s *JSONLStore) SaveSession(session *model.Session) error {
	s.index.Add(session)
	return s.index.save()
}

// GetSession retrieves a session from the index.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *JSONLStore) GetSession(id string, userID string) (*model.Session, error) {
	session, ok := s.index.GetWithOwner(id, userID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

// ListSessions returns all sessions for the specified user.
// Empty userID returns all sessions (legacy mode).
func (s *JSONLStore) ListSessions(userID string) ([]*model.Session, error) {
	return s.index.ListByUser(userID), nil
}

// DeleteSession deletes a session and its message file.
// In multi-user mode, verifies the session belongs to the specified user.
func (s *JSONLStore) DeleteSession(ctx context.Context, sessionID string, userID string) error {
	// Verify ownership and delete from index
	if !s.index.DeleteWithOwner(sessionID, userID) {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	if err := s.index.save(); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	// Remove the JSONL file
	sessionFile := s.getSessionFilePath(sessionID)
	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}

	return nil
}

// AppendMessage appends a message to a session's JSONL file.
func (s *JSONLStore) AppendMessage(ctx context.Context, msg *message.Message) error {
	return s.messageStore.Append(ctx, msg)
}

// GetMessages retrieves messages from a session.
// In multi-user mode, verifies the messages belong to the specified user.
func (s *JSONLStore) GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]message.Message, error) {
	return s.messageStore.Get(ctx, sessionID, limit, userID)
}
