// Package tracker provides user action tracking for habit learning.
package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/habit/pkg/model"
	"github.com/oneliang/aura/habit/pkg/storage"
)

// Tracker records user operations for habit analysis.
type Tracker struct {
	store *storage.Storage
}

// New creates a new tracker.
func New(store *storage.Storage) *Tracker {
	return &Tracker{store: store}
}

// RecordAction records a user action for later habit analysis.
func (t *Tracker) RecordAction(ctx context.Context, userID string, action *model.Action) error {
	if userID == "" {
		return nil // Skip in legacy mode
	}

	if action == nil {
		return fmt.Errorf("action cannot be nil")
	}

	// Set metadata if not provided
	if action.ID == "" {
		action.ID = uuid.New().String()
	}
	if action.UserID == "" {
		action.UserID = userID
	}
	if action.Timestamp.IsZero() {
		action.Timestamp = time.Now()
	}

	return t.store.AppendAction(ctx, userID, action)
}

// GetActions retrieves recent actions for a user.
func (t *Tracker) GetActions(ctx context.Context, userID string, limit int) ([]*model.Action, error) {
	if limit <= 0 {
		limit = model.DefaultToolActionLimit
	}
	return t.store.GetActions(ctx, userID, limit)
}

// CleanupOldActions removes actions older than the specified duration.
func (t *Tracker) CleanupOldActions(ctx context.Context, userID string, maxAge time.Duration) error {
	return t.store.CleanupOldActions(ctx, userID, maxAge)
}
