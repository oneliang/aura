package tui

import (
	"fmt"
	"strings"

	lipglossv2 "charm.land/lipgloss/v2"
)

// CommandPopupWidget renders a floating command list above the input area.
type CommandPopupWidget struct {
	showing  bool
	commands []CommandInfo // Filtered commands for display
	selected int
	filter   string
	styles   UIStyles
}

// Show makes the popup visible.
func (w *CommandPopupWidget) Show() {
	w.showing = true
}

// Hide hides the popup.
func (w *CommandPopupWidget) Hide() {
	w.showing = false
}

// Toggle toggles popup visibility.
func (w *CommandPopupWidget) Toggle() {
	w.showing = !w.showing
}

// IsShowing returns whether the popup is visible.
func (w *CommandPopupWidget) IsShowing() bool {
	return w.showing
}

// UpdateFilter updates the filter string and recomputes the filtered command list.
func (w *CommandPopupWidget) UpdateFilter(filter string, allCommands []CommandInfo) {
	w.filter = filter
	w.commands = filterCommands(filter, allCommands)
	w.selected = 0
	if len(w.commands) > 0 {
		w.showing = true
	}
}

// Up moves the selection up.
func (w *CommandPopupWidget) Up() {
	if len(w.commands) == 0 {
		return
	}
	w.selected--
	if w.selected < 0 {
		w.selected = len(w.commands) - 1
	}
}

// Down moves the selection down.
func (w *CommandPopupWidget) Down() {
	if len(w.commands) == 0 {
		return
	}
	w.selected++
	if w.selected >= len(w.commands) {
		w.selected = 0
	}
}

// Tab autocompletes the selected command into the filter.
func (w *CommandPopupWidget) Tab() {
	if len(w.commands) > 0 && w.selected >= 0 && w.selected < len(w.commands) {
		w.filter = w.commands[w.selected].Name + " "
	}
}

// SelectedName returns the name of the currently selected command.
func (w *CommandPopupWidget) SelectedName() string {
	if len(w.commands) > 0 && w.selected >= 0 && w.selected < len(w.commands) {
		return w.commands[w.selected].Name
	}
	return ""
}

// HasSelection returns whether there is a selectable command.
func (w *CommandPopupWidget) HasSelection() bool {
	return len(w.commands) > 0 && w.selected >= 0 && w.selected < len(w.commands)
}

// filterCommands filters commands by prefix match.
func filterCommands(filter string, allCommands []CommandInfo) []CommandInfo {
	prefix := strings.TrimPrefix(filter, "/")
	var filtered []CommandInfo
	for _, cmd := range allCommands {
		name := strings.TrimPrefix(cmd.Name, "/")
		if prefix == "" || strings.HasPrefix(name, prefix) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// Render renders the command popup with top/bottom borders.
func (w *CommandPopupWidget) Render() string {
	if len(w.commands) == 0 {
		border := lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorHelp)).Render(strings.Repeat("─", PopupDefaultWidth))
		return border + "\n" + w.styles.Help.Render(CommandNoMatching) + "\n" + border
	}

	maxShow := MaxVisibleCommands
	total := len(w.commands)

	// Calculate scroll window
	start := 0
	if w.selected >= maxShow {
		start = w.selected - maxShow + 1
	}
	if start+maxShow > total {
		start = total - maxShow
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < start+maxShow && i < total; i++ {
		cmd := w.commands[i]
		var line string
		if i == w.selected {
			line = w.styles.CommandItemSelected.Render(" > "+cmd.Name+" ") +
				w.styles.CommandDesc.Render(" "+cmd.Description)
		} else {
			line = w.styles.CommandItem.Render("   "+cmd.Name) +
				w.styles.CommandDesc.Render(" "+cmd.Description)
		}
		lines = append(lines, line)
	}

	if total > maxShow {
		lines = append(lines, w.styles.Help.Render(fmt.Sprintf(CommandCountFormat, w.selected+1, total)))
	}

	border := lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorHelp)).Render(strings.Repeat("─", PopupDefaultWidth))
	allLines := append([]string{border}, lines...)
	allLines = append(allLines, border)

	return strings.Join(allLines, "\n")
}

// Height returns the actual number of lines this widget occupies.
func (w *CommandPopupWidget) Height() int {
	if len(w.commands) == 0 {
		return 3 // top border + no match + bottom border
	}
	maxShow := MaxVisibleCommands
	total := len(w.commands)
	showCount := maxShow
	if total < maxShow {
		showCount = total
	}
	// +2 for top/bottom borders, +1 for count line if total > maxShow
	h := showCount + 2
	if total > maxShow {
		h++
	}
	return h
}
