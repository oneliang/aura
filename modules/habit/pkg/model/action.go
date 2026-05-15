// Package model provides data models for the habit tracking system.
package model

import "time"

// Action represents a recorded user operation.
type Action struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	SessionID   string        `json:"session_id"`
	Timestamp   time.Time     `json:"timestamp"`
	Input       string        `json:"input"`
	ToolsUsed   []string      `json:"tools_used"`
	OutputStyle string        `json:"output_style"`
	Duration    time.Duration `json:"duration"`
	Feedback    string        `json:"feedback,omitempty"`
}
