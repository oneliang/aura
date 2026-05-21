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
	if h.profile == nil {
		return "No profile loaded", nil
	}

	result := "=== Personal Profile ===\n"
	result += fmt.Sprintf("Name:       %s\n", h.profile.BasicInfo.Name)
	if h.profile.BasicInfo.Occupation != "" {
		result += fmt.Sprintf("Occupation: %s\n", h.profile.BasicInfo.Occupation)
	}
	if h.profile.BasicInfo.Location != "" {
		result += fmt.Sprintf("Location:   %s\n", h.profile.BasicInfo.Location)
	}
	if h.profile.Background != "" {
		result += fmt.Sprintf("Background: %s\n", h.profile.Background)
	}

	if len(h.profile.Skills) > 0 {
		result += "\nSkills:\n"
		for _, s := range h.profile.Skills {
			cat := ""
			if s.Category != "" {
				cat = " [" + s.Category + "]"
			}
			result += fmt.Sprintf("  - %s (%s)%s\n", s.Name, s.Level, cat)
		}
	}

	result += "\nCommunication Style:\n"
	result += fmt.Sprintf("  Tone:       %s\n", h.profile.Style.Tone)
	result += fmt.Sprintf("  Vocabulary: %s\n", h.profile.Style.Vocabulary)
	result += fmt.Sprintf("  Verbosity:  %s\n", h.profile.Style.Verbosity)
	result += fmt.Sprintf("  Humor:      %.1f\n", h.profile.Style.Humor)

	return result, nil
}
