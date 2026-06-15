// Package storage provides JSONL-based storage for habit data with per-user isolation.
package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oneliang/aura/habit/pkg/model"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

const (
	// initialScannerBufferSize is the initial buffer size for bufio.Scanner (64 KB).
	initialScannerBufferSize = 64 * 1024
	// maxScannerCapacity is the maximum buffer size for bufio.Scanner (10 MB).
	maxScannerCapacity = 10 * 1024 * 1024
)

// Storage provides per-user isolated habit storage using JSONL files.
type Storage struct {
	mu      sync.RWMutex
	baseDir string // ~/.aura/users/
}

// New creates a new habit storage instance.
func New(baseDir string) (*Storage, error) {
	if baseDir == "" {
		var err error
		baseDir, err = ffp.AuraHomePath("users")
		if err != nil {
			return nil, fmt.Errorf("failed to get base dir: %w", err)
		}
	}
	return &Storage{baseDir: baseDir}, nil
}

// GetHabitsPath returns the habits file path for a user.
func (s *Storage) GetHabitsPath(userID string) string {
	return filepath.Join(s.baseDir, userID, "habits", "habits.jsonl")
}

// GetActionsPath returns the actions file path for a user.
func (s *Storage) GetActionsPath(userID string) string {
	return filepath.Join(s.baseDir, userID, "habits", "actions.jsonl")
}

// GetPreferencesPath returns the preferences file path for a user.
func (s *Storage) GetPreferencesPath(userID string) string {
	return filepath.Join(s.baseDir, userID, "habits", "preferences.jsonl")
}

// AppendAction appends an action record to the user's action log.
func (s *Storage) AppendAction(ctx context.Context, userID string, action *model.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.GetActionsPath(userID)
	if err := s.ensureDir(path); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open actions file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write action: %w", err)
	}

	return nil
}

// GetActions reads actions for a user, limited to the specified count (most recent first).
func (s *Storage) GetActions(ctx context.Context, userID string, limit int) ([]*model.Action, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.GetActionsPath(userID)
	return s.readActions(path, limit)
}

// readActions reads actions from a JSONL file.
func (s *Storage) readActions(path string, limit int) ([]*model.Action, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.Action{}, nil
		}
		return nil, fmt.Errorf("failed to open actions file: %w", err)
	}
	defer f.Close()

	var actions []*model.Action
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, initialScannerBufferSize), maxScannerCapacity)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var action model.Action
		if err := json.Unmarshal([]byte(line), &action); err != nil {
			continue // Skip malformed lines
		}
		actions = append(actions, &action)
	}

	// Return most recent N actions
	if limit > 0 && len(actions) > limit {
		actions = actions[len(actions)-limit:]
	}

	return actions, nil
}

// SaveHabits persists habits for a user (overwrites existing file).
func (s *Storage) SaveHabits(ctx context.Context, userID string, habits []*model.Habit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.GetHabitsPath(userID)
	if err := s.ensureDir(path); err != nil {
		return err
	}

	data, err := json.Marshal(habits)
	if err != nil {
		return fmt.Errorf("failed to marshal habits: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetHabits reads habits for a user.
func (s *Storage) GetHabits(ctx context.Context, userID string) ([]*model.Habit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.GetHabitsPath(userID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.Habit{}, nil
		}
		return nil, fmt.Errorf("failed to open habits file: %w", err)
	}
	defer f.Close()

	var habits []*model.Habit
	if err := json.NewDecoder(f).Decode(&habits); err != nil {
		// File might be empty or malformed
		return []*model.Habit{}, nil
	}

	return habits, nil
}

// SavePreferences persists preferences for a user.
func (s *Storage) SavePreferences(ctx context.Context, userID string, prefs []*model.Preference) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.GetPreferencesPath(userID)
	if err := s.ensureDir(path); err != nil {
		return err
	}

	// Write all preferences
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to open preferences file: %w", err)
	}
	defer f.Close()

	for _, pref := range prefs {
		data, err := json.Marshal(pref)
		if err != nil {
			return fmt.Errorf("failed to marshal preference: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write preference: %w", err)
		}
	}

	return nil
}

// GetPreferences reads preferences for a user.
func (s *Storage) GetPreferences(ctx context.Context, userID string) ([]*model.Preference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.GetPreferencesPath(userID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.Preference{}, nil
		}
		return nil, fmt.Errorf("failed to open preferences file: %w", err)
	}
	defer f.Close()

	var prefs []*model.Preference
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, initialScannerBufferSize), maxScannerCapacity)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var pref model.Preference
		if err := json.Unmarshal([]byte(line), &pref); err != nil {
			continue
		}
		prefs = append(prefs, &pref)
	}

	return prefs, nil
}

// ensureDir creates the directory structure for a file path.
func (s *Storage) ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// CleanupOldActions removes actions older than the specified duration.
func (s *Storage) CleanupOldActions(ctx context.Context, userID string, maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.GetActionsPath(userID)
	actions, err := s.readActions(path, 0)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	var recent []*model.Action
	for _, action := range actions {
		if action.Timestamp.After(cutoff) {
			recent = append(recent, action)
		}
	}

	if len(recent) == len(actions) {
		return nil // No cleanup needed
	}

	// Write back as JSONL (one JSON object per line)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create actions file: %w", err)
	}
	defer f.Close()

	for _, action := range recent {
		data, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write action: %w", err)
		}
	}

	return nil
}
