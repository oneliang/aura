// Package model provides data models for the habit tracking system.
package model

import "time"

// Habit category constants.
const (
	CategoryToolUsage  = "tool_usage" // Tool usage preferences
	CategoryCommand    = "command"    // Command sequence patterns
	CategoryStyle      = "style"      // Output style (concise/detailed)
	CategoryPreference = "preference" // Explicit preferences
	CategoryWorkflow   = "workflow"   // Workflow patterns
)

// Trend constants.
const (
	TrendIncreasing = "increasing"
	TrendStable     = "stable"
	TrendDecreasing = "decreasing"
)

// Default analysis parameters.
const (
	DefaultMinOccurrences   = 3    // Minimum pattern appearances to form a habit
	DefaultConfThreshold    = 0.3  // Minimum confidence for valid habit
	DefaultMaxActionAgeDays = 30   // Max action age in days
	DefaultAnalysisLimit    = 500  // Max actions to analyze
	DefaultTopKeywords      = 3    // Top context keywords per tool
	DefaultToolActionLimit  = 1000 // Default limit for GetActions
)

// Habit name templates.
const (
	TemplateToolUsageHabit   = "Frequently uses %s tool"
	TemplateOutputStyleHabit = "Prefers %s output style"
	TemplateWorkflowHabit    = "Workflow: %s"
)

// Preference keys and values.
const (
	PrefOutputStyle    = "output_style"
	PrefValuePreferred = "preferred"
)

// Workflow pair separator.
const (
	WorkflowPairSep = " -> "
)

// Habit represents a learned user behavior pattern.
type Habit struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id,omitempty"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Pattern    Pattern   `json:"pattern"`
	Frequency  Frequency `json:"frequency"`
	Confidence float64   `json:"confidence"`
	LastSeen   time.Time `json:"last_seen"`
}

// Pattern represents the trigger pattern for a habit.
type Pattern struct {
	Context    string   `json:"context,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	ToolUsage  []string `json:"tool_usage,omitempty"`
	CommandSeq []string `json:"command_seq,omitempty"`
}

// Frequency represents usage frequency statistics.
type Frequency struct {
	TotalCount int     `json:"total_count"`
	DailyAvg   float64 `json:"daily_avg"`
	WeeklyAvg  float64 `json:"weekly_avg"`
	Trend      string  `json:"trend"` // increasing/stable/decreasing
}
