// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/session/pkg/manager"
)

// SessionHandler handles session commands.
type SessionHandler struct {
	sessionMgr        *manager.SessionManager
	userID            string
	currentSessionID  string // Current active session ID for show command
}

// NewSessionHandler creates a new session handler.
func NewSessionHandler(sessionMgr *manager.SessionManager, userID string) *SessionHandler {
	return &SessionHandler{
		sessionMgr:       sessionMgr,
		userID:           userID,
		currentSessionID: "",
	}
}

// SetCurrentSessionID sets the current active session ID.
func (h *SessionHandler) SetCurrentSessionID(sessionID string) {
	h.currentSessionID = sessionID
}

// ExecuteCommand executes a session command.
// Commands: list, create, delete, show, update
func (h *SessionHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	switch cmd {
	case "list":
		return h.listSessions(ctx)
	case "create":
		name, _ := params["name"].(string)
		role, _ := params["role"].(string)
		return h.createSession(ctx, name, role)
	case "delete":
		id, _ := params["id"].(string)
		return h.deleteSession(id)
	case "show":
		id, _ := params["id"].(string)
		return h.showSession(id)
	case "update":
		id, _ := params["id"].(string)
		role, _ := params["role"].(string)
		return h.updateSession(ctx, id, role)
	default:
		return "", fmt.Errorf("unknown session command: %s", cmd)
	}
}

// listSessions lists all sessions for the current user.
func (h *SessionHandler) listSessions(ctx context.Context) (string, error) {
	sessions, err := h.sessionMgr.ListSessions(h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return "No sessions found.", nil
	}

	result := "Sessions:\n"
	for _, s := range sessions {
		result += fmt.Sprintf("  - [%s] %s (created: %d)\n", s.ID, s.Name, s.CreatedAt)
	}
	return result, nil
}

// createSession creates a new session for the current user.
func (h *SessionHandler) createSession(ctx context.Context, name, role string) (string, error) {
	session, err := h.sessionMgr.CreateSession(name, nil, role, h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return fmt.Sprintf("Created session [%s] %s", session.ID, session.Name), nil
}

// deleteSession deletes a session.
func (h *SessionHandler) deleteSession(id string) (string, error) {
	if err := h.sessionMgr.DeleteSession(id, h.userID); err != nil {
		return "", fmt.Errorf("failed to delete session: %w", err)
	}
	return fmt.Sprintf("Deleted session [%s]", id), nil
}

// showSession shows session details.
// If id is empty, uses the current active session.
func (h *SessionHandler) showSession(id string) (string, error) {
	// Use current session if id is empty
	if id == "" {
		if h.currentSessionID == "" {
			return "", fmt.Errorf("no session ID specified and no current session available")
		}
		id = h.currentSessionID
	}

	session, err := h.sessionMgr.GetSession(id, h.userID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	result := fmt.Sprintf("Session: %s\n", session.Name)
	result += fmt.Sprintf("ID: %s\n", session.ID)
	result += fmt.Sprintf("Created: %d\n", session.CreatedAt)
	result += fmt.Sprintf("Updated: %d\n", session.UpdatedAt)
	if session.SystemPrompt != "" {
		result += fmt.Sprintf("System Prompt: %s\n", session.SystemPrompt)
	}
	if len(session.Subscriptions) > 0 {
		result += "Subscriptions:\n"
		for _, sub := range session.Subscriptions {
			result += fmt.Sprintf("  - Trigger: %s, Source: %s\n", sub.Trigger, sub.Source)
		}
	}
	return result, nil
}

// updateSession updates a session's configuration.
func (h *SessionHandler) updateSession(ctx context.Context, id, role string) (string, error) {
	if err := h.sessionMgr.UpdateSession(id, nil, &role, h.userID); err != nil {
		return "", fmt.Errorf("failed to update session: %w", err)
	}
	return fmt.Sprintf("Updated session [%s] with role: %s", id, role), nil
}
