// Package model provides data models for the habit tracking system.
package model

import "time"

// Preference source constants.
const (
	SourceExplicit = "explicit" // User explicitly stated
	SourceImplicit = "implicit" // Inferred from behavior
)

// Preference represents a user preference learned from behavior.
type Preference struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Category  string    `json:"category"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}
