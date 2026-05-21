// Package manager provides the unified habit management interface.
package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/oneliang/aura/habit/pkg/analyzer"
	"github.com/oneliang/aura/habit/pkg/model"
	"github.com/oneliang/aura/habit/pkg/storage"
	"github.com/oneliang/aura/habit/pkg/tracker"
)

// Manager provides unified habit tracking and analysis.
type Manager struct {
	tracker  *tracker.Tracker
	analyzer *analyzer.Analyzer
	store    *storage.Storage
	config   Config
}

// Config holds manager configuration.
type Config struct {
	MinOccurrences int
	ConfThreshold  float64
	MaxActionAge   time.Duration
	AnalysisLimit  int // Max actions to analyze
}

// DefaultConfig returns default manager config.
func DefaultConfig() Config {
	return Config{
		MinOccurrences: model.DefaultMinOccurrences,
		ConfThreshold:  model.DefaultConfThreshold,
		MaxActionAge:   model.DefaultMaxActionAgeDays * 24 * time.Hour,
		AnalysisLimit:  model.DefaultAnalysisLimit,
	}
}

// New creates a new habit manager.
func New(cfg Config) (*Manager, error) {
	store, err := storage.New("")
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Manager{
		tracker:  tracker.New(store),
		analyzer: analyzer.New(cfg.MinOccurrences, cfg.ConfThreshold),
		store:    store,
		config:   cfg,
	}, nil
}

// RecordAction records a user action for habit analysis.
func (m *Manager) RecordAction(ctx context.Context, userID string, action *model.Action) error {
	return m.tracker.RecordAction(ctx, userID, action)
}

// GetHabits returns all habits for a user.
func (m *Manager) GetHabits(ctx context.Context, userID string) ([]*model.Habit, error) {
	// First try to load existing habits
	habits, err := m.store.GetHabits(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If no habits exist, analyze actions to generate them
	if len(habits) == 0 {
		actions, err := m.tracker.GetActions(ctx, userID, m.config.AnalysisLimit)
		if err != nil {
			return nil, err
		}
		if len(actions) > 0 {
			habits, err = m.analyzer.Analyze(ctx, userID, actions)
			if err != nil {
				return nil, err
			}
			// Persist generated habits
			if err := m.store.SaveHabits(ctx, userID, habits); err != nil {
				return nil, fmt.Errorf("failed to save habits: %w", err)
			}
		}
	}

	return habits, nil
}

// RefreshHabits re-analyzes actions and updates habits for a user.
func (m *Manager) RefreshHabits(ctx context.Context, userID string) ([]*model.Habit, error) {
	actions, err := m.tracker.GetActions(ctx, userID, m.config.AnalysisLimit)
	if err != nil {
		return nil, err
	}

	habits, err := m.analyzer.Analyze(ctx, userID, actions)
	if err != nil {
		return nil, err
	}

	if err := m.store.SaveHabits(ctx, userID, habits); err != nil {
		return nil, fmt.Errorf("failed to save habits: %w", err)
	}

	return habits, nil
}

// GetPreferences returns preferences for a user.
func (m *Manager) GetPreferences(ctx context.Context, userID string) ([]*model.Preference, error) {
	// Try to load existing preferences
	prefs, err := m.store.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If no preferences exist, analyze actions to generate them
	if len(prefs) == 0 {
		actions, err := m.tracker.GetActions(ctx, userID, m.config.AnalysisLimit)
		if err != nil {
			return nil, err
		}
		if len(actions) > 0 {
			prefs, err = m.analyzer.GetPreferences(ctx, userID, actions)
			if err != nil {
				return nil, err
			}
			// Persist generated preferences
			if err := m.store.SavePreferences(ctx, userID, prefs); err != nil {
				return nil, fmt.Errorf("failed to save preferences: %w", err)
			}
		}
	}

	return prefs, nil
}

// DeleteHabit deletes a habit by ID.
func (m *Manager) DeleteHabit(ctx context.Context, userID string, habitID string) error {
	habits, err := m.store.GetHabits(ctx, userID)
	if err != nil {
		return err
	}

	var filtered []*model.Habit
	for _, h := range habits {
		if h.ID != habitID {
			filtered = append(filtered, h)
		}
	}

	return m.store.SaveHabits(ctx, userID, filtered)
}

// Cleanup removes old action data.
func (m *Manager) Cleanup(ctx context.Context, userID string) error {
	return m.tracker.CleanupOldActions(ctx, userID, m.config.MaxActionAge)
}
