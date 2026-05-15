// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// SubscriptionHandler handles subscription commands.
type SubscriptionHandler struct {
	sessionMgr *manager.SessionManager
	userID     string
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(sessionMgr *manager.SessionManager, userID string) *SubscriptionHandler {
	return &SubscriptionHandler{
		sessionMgr: sessionMgr,
		userID:     userID,
	}
}

// ExecuteCommand executes a subscription command.
// Commands: show, add, delete
func (h *SubscriptionHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	switch cmd {
	case "show":
		sessionID, _ := params["session_id"].(string)
		return h.showSubscriptions(sessionID)
	case "add":
		sessionID, _ := params["session_id"].(string)
		trigger, _ := params["trigger"].(string)
		source, _ := params["source"].(string)
		return h.addSubscription(ctx, sessionID, trigger, source)
	case "delete":
		sessionID, _ := params["session_id"].(string)
		subID, _ := params["subscription_id"].(string)
		return h.deleteSubscription(ctx, sessionID, subID)
	default:
		return "", fmt.Errorf("%s: %s", i18n.T("error.subscription.unknown"), cmd)
	}
}

// showSubscriptions shows all subscriptions for a session.
func (h *SubscriptionHandler) showSubscriptions(sessionID string) (string, error) {
	if sessionID == "" {
		// List all sessions with their subscriptions
		sessions, err := h.sessionMgr.ListSessions(h.userID)
		if err != nil {
			return "", fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			return i18n.T("command.no_sessions_found"), nil
		}

		result := i18n.T("command.sessions_subscriptions_title")
		for _, s := range sessions {
			result += fmt.Sprintf("Session: %s [%s]\n", s.Name, s.ID)
			if len(s.Subscriptions) == 0 {
				result += i18n.T("command.session_no_subscriptions")
			} else {
				for _, sub := range s.Subscriptions {
					status := i18n.T("command.session_subscription_active")
					if !sub.Active {
						status = i18n.T("command.session_subscription_inactive")
					}
					result += fmt.Sprintf("  - [%s] %s: %s\n", status, sub.Source, sub.Trigger)
				}
			}
			result += "\n"
		}
		return result, nil
	}

	// Show subscriptions for specific session
	session, err := h.sessionMgr.GetSession(sessionID, h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	if len(session.Subscriptions) == 0 {
		return fmt.Sprintf(i18n.T("command.no_subscriptions_session"), sessionID), nil
	}

	result := fmt.Sprintf(i18n.T("command.subscriptions_for_session"), session.Name, session.ID)
	for _, sub := range session.Subscriptions {
		status := i18n.T("command.session_subscription_active")
		if !sub.Active {
			status = i18n.T("command.session_subscription_inactive")
		}
		result += fmt.Sprintf("  - [%s] ID: %s, Source: %s, Trigger: %s\n",
			status, sub.ID, sub.Source, sub.Trigger)
	}
	return result, nil
}

// addSubscription adds a new subscription to a session.
func (h *SubscriptionHandler) addSubscription(ctx context.Context, sessionID, trigger, source string) (string, error) {
	if sessionID == "" {
		return "", errors.New(i18n.T("error.param.session_id_required"))
	}
	if trigger == "" {
		return "", errors.New(i18n.T("error.param.trigger_required"))
	}

	session, err := h.sessionMgr.GetSession(sessionID, h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Create new subscription
	newSub := model.Subscription{
		ID:      fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Trigger: trigger,
		Source:  source,
		Active:  true,
	}

	session.Subscriptions = append(session.Subscriptions, newSub)
	session.UpdatedAt = time.Now().UnixMilli()

	// Save updated session
	if err := h.sessionMgr.SaveSession(session); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	return fmt.Sprintf(i18n.T("command.subscription_added_to_session"), sessionID, trigger, source), nil
}

// deleteSubscription deletes a subscription from a session.
func (h *SubscriptionHandler) deleteSubscription(ctx context.Context, sessionID, subID string) (string, error) {
	if sessionID == "" {
		return "", errors.New(i18n.T("error.param.session_id_required"))
	}
	if subID == "" {
		return "", errors.New(i18n.T("error.param.subscription_id_required"))
	}

	session, err := h.sessionMgr.GetSession(sessionID, h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Find and remove subscription
	found := false
	newSubs := make([]model.Subscription, 0, len(session.Subscriptions))
	for _, sub := range session.Subscriptions {
		if sub.ID == subID {
			found = true
			continue
		}
		newSubs = append(newSubs, sub)
	}

	if !found {
		return "", fmt.Errorf("%s: %s", i18n.T("error.subscription.not_found"), subID)
	}

	session.Subscriptions = newSubs
	session.UpdatedAt = time.Now().UnixMilli()

	// Save updated session
	if err := h.sessionMgr.SaveSession(session); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	return fmt.Sprintf(i18n.T("command.subscription_deleted_from_session"), subID, sessionID), nil
}
