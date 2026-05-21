package tui

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ActionType represents the type of action a keybinding triggers.
type ActionType int

const (
	// ActionCommand executes a slash command via the existing command registry.
	ActionCommand ActionType = iota
	// ActionTUI performs a TUI-internal operation (e.g. toggle popup).
	ActionTUI
)

// KeyBinding maps a key combination to an action.
type KeyBinding struct {
	Keys     []string // Key combinations (e.g. ["ctrl+l"])
	Action   string   // Action identifier (e.g. "clear")
	Type     ActionType
	Command  string // For ActionCommand: the slash command to execute (e.g. "/clear")
	HelpText string // Description shown in /help
}

// keyMap is the global keybinding registry.
var keyMap = make(map[string]*KeyBinding)

// RegisterKeyBinding registers a keybinding and all its key combinations.
func RegisterKeyBinding(binding KeyBinding) {
	for _, key := range binding.Keys {
		keyMap[strings.ToLower(key)] = &binding
	}
}

// GetBinding looks up a keybinding by key string (case-insensitive).
func GetBinding(keyStr string) *KeyBinding {
	return keyMap[strings.ToLower(keyStr)]
}

// GetAllBindings returns all unique keybindings sorted by action name.
func GetAllBindings() []KeyBinding {
	seen := make(map[string]bool)
	var result []KeyBinding
	for _, b := range keyMap {
		if !seen[b.Action] {
			seen[b.Action] = true
			result = append(result, *b)
		}
	}
	// Sort by action name for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Action < result[j].Action
	})
	return result
}

// ExecuteBinding executes the action associated with a keybinding.
// For ActionCommand, it delegates to Model.handleCommand.
func ExecuteBinding(m Model, binding *KeyBinding) (tea.Model, tea.Cmd) {
	if binding.Type == ActionCommand {
		return m.handleCommand(binding.Command)
	}
	return m, nil
}

func init() {
	// Register default keyboard shortcuts (all map to existing slash commands)
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+l"},
		Action:   "clear",
		Type:     ActionCommand,
		Command:  "/clear",
		HelpText: "Clear screen",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+h"},
		Action:   "help",
		Type:     ActionCommand,
		Command:  "/help",
		HelpText: "Show help",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+s"},
		Action:   "sessions",
		Type:     ActionCommand,
		Command:  "/sessions",
		HelpText: "Switch session",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+q"},
		Action:   "quit",
		Type:     ActionCommand,
		Command:  "/exit",
		HelpText: "Quit",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+t"},
		Action:   "tools",
		Type:     ActionCommand,
		Command:  "/tools",
		HelpText: "List tools",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+p"},
		Action:   "profile",
		Type:     ActionCommand,
		Command:  "/profile",
		HelpText: "User profile",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+m"},
		Action:   "mcp",
		Type:     ActionCommand,
		Command:  "/mcp",
		HelpText: "MCP servers",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+k"},
		Action:   "skills",
		Type:     ActionCommand,
		Command:  "/skills",
		HelpText: "Skill list",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+d"},
		Action:   "compact",
		Type:     ActionCommand,
		Command:  "/compact",
		HelpText: "Compact memory",
	})
	RegisterKeyBinding(KeyBinding{
		Keys:     []string{"ctrl+e"},
		Action:   "status",
		Type:     ActionCommand,
		Command:  "/status",
		HelpText: "Execution status",
	})
}
