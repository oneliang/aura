// Package runtime provides the unified runtime for the agent system.
package runtime

// RuntimeMode represents the runtime execution mode.
type RuntimeMode int

const (
	// RuntimeModeCLI - Command-line interface mode (non-interactive)
	RuntimeModeCLI RuntimeMode = iota
	// RuntimeModeTUI - Terminal UI mode (interactive, Bubble Tea)
	RuntimeModeTUI
	// RuntimeModeAPI - API server mode (HTTP endpoints)
	RuntimeModeAPI
)

// String returns the string representation of the runtime mode.
func (m RuntimeMode) String() string {
	switch m {
	case RuntimeModeCLI:
		return "CLI"
	case RuntimeModeTUI:
		return "TUI"
	case RuntimeModeAPI:
		return "API"
	default:
		return "Unknown"
	}
}
