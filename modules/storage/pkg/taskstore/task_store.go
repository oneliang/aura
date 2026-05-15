// Package taskstore provides persistent storage for tasks per session.
package taskstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/oneliang/aura/shared/pkg/tasks"
	"github.com/oneliang/aura/shared/pkg/utils"
)

// TaskStore persists tasks for a session to a JSON file.
type TaskStore struct {
	dataDir   string
	sessionID string
	mu        sync.RWMutex
}

// New creates a TaskStore for the given session.
func New(dataDir, sessionID string) *TaskStore {
	return &TaskStore{
		dataDir:   dataDir,
		sessionID: sessionID,
	}
}

// path returns the file path for this session's tasks.
func (s *TaskStore) path() string {
	return filepath.Join(s.dataDir, s.sessionID+".tasks.json")
}

// Load reads tasks from disk. Returns nil if the file does not exist.
func (s *TaskStore) Load() ([]tasks.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ts []tasks.Task
	err := utils.ReadJSONFile(s.path(), &ts)
	if err != nil {
		if err == utils.ErrFileNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	return ts, nil
}

// Save writes tasks to disk.
func (s *TaskStore) Save(ts []tasks.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("ensure data dir: %w", err)
	}

	if err := utils.WriteJSONFile(s.path(), ts); err != nil {
		return fmt.Errorf("save tasks: %w", err)
	}
	return nil
}
