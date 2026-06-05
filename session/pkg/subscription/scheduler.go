// Package subscription provides subscription management for scheduled notifications.
package subscription

import (
	"context"
	"fmt"
	"sync"

	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/robfig/cron/v3"
)

// TriggerFunc is called when a subscription is triggered.
type TriggerFunc func(ctx context.Context, sub *Subscription) error

// Scheduler manages scheduled subscription triggers.
type Scheduler struct {
	mu          sync.RWMutex
	cron        *cron.Cron
	store       *Store
	triggerFunc TriggerFunc
	logger      *logger.Logger
	running     bool
	entryIDs    map[cron.EntryID]string // cron entry ID -> subscription ID
}

// NewScheduler creates a new scheduler.
func NewScheduler(store *Store, triggerFunc TriggerFunc, log *logger.Logger) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		store:       store,
		triggerFunc: triggerFunc,
		logger:      log,
		entryIDs:    make(map[cron.EntryID]string),
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	// Load existing active subscriptions and schedule them
	subs := s.store.GetActive()
	for _, sub := range subs {
		if err := s.scheduleSubscription(sub); err != nil {
			s.logger.Warn("Failed to schedule subscription", "error", err.Error(), "subscription_id", sub.ID, "event_type", sub.EventType)
		}
	}

	s.cron.Start()
	s.running = true
	s.logger.Info("Scheduler started", "module", "subscription", "count", len(s.entryIDs))
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.entryIDs = make(map[cron.EntryID]string)
	s.logger.Info("Scheduler stopped", "module", "subscription")
}

// AddSubscription adds a new subscription and schedules it.
func (s *Scheduler) AddSubscription(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to store
	if err := s.store.Create(sub); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Schedule if active
	if sub.IsActive {
		if err := s.scheduleSubscriptionLocked(sub); err != nil {
			// Rollback
			s.store.Delete(sub.ID)
			return fmt.Errorf("failed to schedule subscription: %w", err)
		}
	}

	s.logger.Info("Subscription added", "module", "subscription", "id", sub.ID, "event_type", sub.EventType, "cron", sub.CronExpr)

	return nil
}

// RemoveSubscription removes a subscription and unschedules it.
func (s *Scheduler) RemoveSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove cron entry
	for entryID, subID := range s.entryIDs {
		if subID == id {
			s.cron.Remove(entryID)
			delete(s.entryIDs, entryID)
			break
		}
	}

	// Remove from store
	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	s.logger.Info("Subscription removed", "module", "subscription", "id", id)

	return nil
}

// TriggerSubscription manually triggers a subscription.
func (s *Scheduler) TriggerSubscription(id string) error {
	s.mu.RLock()
	sub, exists := s.store.Get(id)
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	ctx := context.Background()
	if err := s.triggerFunc(ctx, sub); err != nil {
		s.logger.Error("Manual trigger failed", "error", err.Error(), "module", "subscription", "id", id)
		return fmt.Errorf("trigger failed: %w", err)
	}

	sub.Trigger()
	return s.store.Update(sub)
}

// scheduleSubscription schedules a subscription (locks mutex).
func (s *Scheduler) scheduleSubscriptionLocked(sub *Subscription) error {
	subID := sub.ID

	// Create job function
	job := func() {
		s.logger.Debug("Triggering subscription", "module", "subscription", "id", subID, "event_type", sub.EventType)

		ctx := context.Background()
		if err := s.triggerFunc(ctx, sub); err != nil {
			s.logger.Error("Scheduled trigger failed", "error", err.Error(), "module", "subscription", "id", subID)
		} else {
			// Update last trigger time
			sub.Trigger()
			if err := s.store.Update(sub); err != nil {
				s.logger.Error("Failed to update subscription after trigger", "error", err.Error(), "module", "subscription", "id", subID)
			}
		}
	}

	// Schedule with cron
	entryID, err := s.cron.AddFunc(sub.CronExpr, job)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", sub.CronExpr, err)
	}

	s.entryIDs[entryID] = subID
	return nil
}

// scheduleSubscription schedules a subscription (assumes lock is held by caller).
func (s *Scheduler) scheduleSubscription(sub *Subscription) error {
	return s.scheduleSubscriptionLocked(sub)
}

// ListSubscriptions returns all subscriptions.
func (s *Scheduler) ListSubscriptions() []*Subscription {
	return s.store.GetAll()
}

// GetSubscription returns a subscription by ID.
func (s *Scheduler) GetSubscription(id string) (*Subscription, bool) {
	return s.store.Get(id)
}
