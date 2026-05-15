// Package model provides data models for session management.
package model

import (
	"time"

	"github.com/oneliang/aura/storage/pkg/message"
)

// Session represents a conversation session.
type Session struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	UserID        string         `json:"user_id"`    // User ID for multi-user isolation (empty = legacy mode)
	CreatedAt     int64          `json:"created_at"` // Unix timestamp in milliseconds
	UpdatedAt     int64          `json:"updated_at"`
	LastActive    int64          `json:"last_active,omitempty"`  // Last activity timestamp (for staleness detection)
	ExpiresAt     int64          `json:"expires_at,omitempty"`   // Optional expiration timestamp
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
	SystemPrompt  string         `json:"system_prompt,omitempty"`
}

// Subscription defines rules for routing events to a session.
type Subscription struct {
	ID      string `json:"id"`
	Trigger string `json:"trigger"` // Keyword to match in message content
	Source  string `json:"source"`  // Trigger source: feishu/email/cron/cli/api/*
	Active  bool   `json:"active"`
}

// Message is an alias for storage/pkg/message.Message for backward compatibility.
type Message = message.Message

// IsStale checks if the session is stale based on a threshold duration (in milliseconds).
// A session is considered stale if:
// 1. LastActive is zero (never active) and CreatedAt is older than threshold
// 2. LastActive is set and older than threshold
func (s *Session) IsStale(thresholdMs int64) bool {
	if thresholdMs <= 0 {
		return false // No threshold means never stale
	}

	now := getCurrentTimeMs()

	// Check expiration first
	if s.ExpiresAt > 0 && now > s.ExpiresAt {
		return true
	}

	// Check last active
	if s.LastActive > 0 {
		return now - s.LastActive > thresholdMs
	}

	// Fallback to created time if last active is not set
	return now - s.CreatedAt > thresholdMs
}

// UpdateLastActive sets the LastActive timestamp to current time.
func (s *Session) UpdateLastActive() {
	s.LastActive = getCurrentTimeMs()
}

// getCurrentTimeMs returns current Unix timestamp in milliseconds.
// This is a helper function that can be mocked in tests.
var getCurrentTimeMs = func() int64 {
	return UnixMs()
}

// UnixMs returns current Unix timestamp in milliseconds.
func UnixMs() int64 {
	return time.Now().UnixMilli()
}
