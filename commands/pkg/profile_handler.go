// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/personality/pkg/profile"
)

// ProfileHandler handles profile commands.
type ProfileHandler struct {
	profile *profile.Profile
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(profile *profile.Profile) *ProfileHandler {
	return &ProfileHandler{
		profile: profile,
	}
}

// ExecuteCommand executes a profile command.
// Commands: show, update
func (h *ProfileHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	switch cmd {
	case "show":
		return h.showProfile()
	case "update":
		// Profile update would require additional logic
		return h.showProfile()
	default:
		return "", fmt.Errorf("unknown profile command: %s", cmd)
	}
}

// showProfile shows the current profile.
func (h *ProfileHandler) showProfile() (string, error) {
	if h.profile == nil || h.profile.Content == "" {
		return "No profile loaded", nil
	}
	return h.profile.Content, nil
}
