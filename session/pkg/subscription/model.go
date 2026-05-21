// Package subscription provides subscription management for scheduled notifications.
package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a scheduled subscription for notifications.
type Subscription struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`    // User ID for multi-user isolation (empty = legacy mode)
	SessionID   string                 `json:"session_id"` // Target session to notify
	EventType   string                 `json:"event_type"` // e.g., "daily_report", "task_notify"
	CronExpr    string                 `json:"cron_expr"`  // Cron expression, e.g., "0 9 * * *"
	Config      map[string]interface{} `json:"config"`     // Additional configuration
	IsActive    bool                   `json:"is_active"`  // Whether subscription is active
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastTrigger time.Time              `json:"last_trigger,omitempty"`
}

// NewSubscription creates a new subscription with the given parameters.
func NewSubscription(userID, sessionID, eventType, cronExpr string, config map[string]interface{}) *Subscription {
	now := time.Now()
	return &Subscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		EventType: eventType,
		CronExpr:  cronExpr,
		Config:    config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Trigger records that the subscription was triggered.
func (s *Subscription) Trigger() {
	s.LastTrigger = time.Now()
	s.UpdatedAt = time.Now()
}

// Enable activates the subscription.
func (s *Subscription) Enable() {
	s.IsActive = true
	s.UpdatedAt = time.Now()
}

// Disable deactivates the subscription.
func (s *Subscription) Disable() {
	s.IsActive = false
	s.UpdatedAt = time.Now()
}
