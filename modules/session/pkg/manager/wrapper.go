// Package manager provides session management wrapper for external packages.
package manager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// SessionItem is a data transfer object for session information.
// Used by CLI and TUI packages for displaying session lists.
type SessionItem struct {
	id      string
	name    string
	created int64
	updated int64
	subs    int
	role    string
}

// ID returns the session ID.
func (i SessionItem) ID() string { return i.id }

// Name returns the session name.
func (i SessionItem) Name() string { return i.name }

// Created returns the creation time.
func (i SessionItem) Created() int64 { return i.created }

// Updated returns the last update time.
func (i SessionItem) Updated() int64 { return i.updated }

// Subs returns the number of subscriptions.
func (i SessionItem) Subs() int { return i.subs }

// Role returns the role name.
func (i SessionItem) Role() string { return i.role }

// SubscriptionItem is a data transfer object for subscription information.
type SubscriptionItem struct {
	id      string
	trigger string
	source  string
	active  bool
}

// ID returns the subscription ID.
func (i SubscriptionItem) ID() string { return i.id }

// Trigger returns the trigger keyword.
func (i SubscriptionItem) Trigger() string { return i.trigger }

// Source returns the source.
func (i SubscriptionItem) Source() string { return i.source }

// Active returns whether the subscription is active.
func (i SubscriptionItem) Active() bool { return i.active }

// SessionServiceWrapper wraps SessionManager for easier external use.
// It provides convenience methods for common session operations.
type SessionServiceWrapper struct {
	mgr   *SessionManager
	store *storage.JSONLStore
}

// NewSessionServiceWrapper creates a new session service wrapper.
func NewSessionServiceWrapper(mgr *SessionManager, store *storage.JSONLStore) *SessionServiceWrapper {
	return &SessionServiceWrapper{
		mgr:   mgr,
		store: store,
	}
}

// GetStore returns the underlying JSONL store.
func (w *SessionServiceWrapper) GetStore() *storage.JSONLStore {
	return w.store
}

// GetManager returns the underlying session manager.
func (w *SessionServiceWrapper) GetManager() *SessionManager {
	return w.mgr
}

// ListSessions returns all sessions for the specified user.
// Empty userID returns all sessions (legacy mode).
func (w *SessionServiceWrapper) ListSessions(userID string) ([]SessionItem, error) {
	sessions, err := w.store.ListSessions(userID)
	if err != nil {
		return nil, err
	}

	items := make([]SessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = SessionItem{
			id:      s.ID,
			name:    s.Name,
			created: s.CreatedAt,
			updated: s.UpdatedAt,
			subs:    len(s.Subscriptions),
			role:    getRoleFromPrompt(s.SystemPrompt),
		}
	}
	return items, nil
}

// CreateSession creates a new session with the given name and role for the specified user.
func (w *SessionServiceWrapper) CreateSession(name, role string, userID string) (*SessionItem, error) {
	session, err := w.mgr.CreateSession(name, nil, role, userID)
	if err != nil {
		return nil, err
	}

	return &SessionItem{
		id:      session.ID,
		name:    session.Name,
		created: session.CreatedAt,
		updated: session.UpdatedAt,
		subs:    len(session.Subscriptions),
		role:    role,
	}, nil
}

// GetSession retrieves a session by ID.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) GetSession(id string, userID string) (*SessionItem, error) {
	session, err := w.store.GetSession(id, userID)
	if err != nil {
		return nil, err
	}

	return &SessionItem{
		id:      session.ID,
		name:    session.Name,
		created: session.CreatedAt,
		updated: session.UpdatedAt,
		subs:    len(session.Subscriptions),
		role:    getRoleFromPrompt(session.SystemPrompt),
	}, nil
}

// GetMostRecentlyUpdatedSession returns the session with the most recent update time for the specified user.
func (w *SessionServiceWrapper) GetMostRecentlyUpdatedSession(userID string) (*SessionItem, error) {
	items, err := w.ListSessions(userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	mostRecent := items[0]
	for _, item := range items[1:] {
		if item.updated > mostRecent.updated {
			mostRecent = item
		}
	}
	return &mostRecent, nil
}

// DeleteSession deletes a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) DeleteSession(id string, userID string) error {
	return w.mgr.DeleteSession(id, userID)
}

// UpdateSessionRole updates the role for a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) UpdateSessionRole(id, role string, userID string) error {
	return w.mgr.UpdateSession(id, nil, &role, userID)
}

// AddSubscription adds a subscription to a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) AddSubscription(sessionID, trigger, source string, userID string) error {
	session, err := w.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	newSub := model.Subscription{
		ID:      fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Trigger: trigger,
		Source:  source,
		Active:  true,
	}
	session.Subscriptions = append(session.Subscriptions, newSub)
	session.UpdatedAt = time.Now().UnixMilli()

	return w.store.SaveSession(session)
}

// RemoveSubscription removes a subscription from a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) RemoveSubscription(sessionID, subscriptionID string, userID string) error {
	session, err := w.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Find and remove the subscription
	found := false
	newSubs := make([]model.Subscription, 0, len(session.Subscriptions))
	for _, sub := range session.Subscriptions {
		if sub.ID == subscriptionID {
			found = true
			continue
		}
		newSubs = append(newSubs, sub)
	}

	if !found {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	session.Subscriptions = newSubs
	session.UpdatedAt = time.Now().UnixMilli()

	return w.store.SaveSession(session)
}

// ToggleSubscriptionStatus toggles the active status of a subscription.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) ToggleSubscriptionStatus(sessionID, subscriptionID string, userID string) error {
	session, err := w.store.GetSession(sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Find and toggle the subscription
	found := false
	for i, sub := range session.Subscriptions {
		if sub.ID == subscriptionID {
			session.Subscriptions[i].Active = !sub.Active
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	session.UpdatedAt = time.Now().UnixMilli()

	return w.store.SaveSession(session)
}

// GetSubscriptions returns all subscriptions for a session.
// In multi-user mode, verifies the session belongs to the specified user.
func (w *SessionServiceWrapper) GetSubscriptions(sessionID string, userID string) ([]SubscriptionItem, error) {
	session, err := w.store.GetSession(sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	items := make([]SubscriptionItem, len(session.Subscriptions))
	for i, sub := range session.Subscriptions {
		items[i] = SubscriptionItem{
			id:      sub.ID,
			trigger: sub.Trigger,
			source:  sub.Source,
			active:  sub.Active,
		}
	}
	return items, nil
}

// getRoleFromPrompt extracts role name from system prompt (simplified).
func getRoleFromPrompt(prompt string) string {
	if prompt == "" {
		return ""
	}
	// For now, return empty - role detection would require storing role name separately
	// or parsing the prompt content
	return ""
}

// ExportSession exports the session messages to a markdown file.
// In multi-user mode, verifies the session belongs to the specified user.
// Returns the exported file path.
func (w *SessionServiceWrapper) ExportSession(sessionID, sessionName, outputPath string, userID string) (string, error) {
	ctx := context.Background()
	messages, err := w.store.GetMessages(ctx, sessionID, 0, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}

	// Build markdown content
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Session: %s\n\n", sessionName))
	sb.WriteString(fmt.Sprintf("Exported at: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("---\n\n")

	for _, msg := range messages {
		ts := time.UnixMilli(msg.Timestamp).Format("15:04:05")
		// Extract text content from ContentBlocks
		var textContent string
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("**You** [%s]\n\n%s\n\n", ts, textContent))
		case "assistant":
			sb.WriteString(fmt.Sprintf("**Aura** [%s]\n\n%s\n\n", ts, textContent))
		case "system":
			sb.WriteString(fmt.Sprintf("**System** [%s]\n\n%s\n\n", ts, textContent))
		}
		sb.WriteString("---\n\n")
	}

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}
