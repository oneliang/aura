package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
)

// InputManager handles all input state and rendering.
// Key design: Uses a disabled flag to cleanly switch between textarea and placeholder,
// avoiding the ghost text bug by never manipulating textarea state during processing.
type InputManager struct {
	textarea    textarea.Model
	disabled    bool
	placeholder string
	styles      UIStyles
	width       int
	maxHeight   int

	// Input history (memory cache, filtered from session messages)
	userMsgHistory []string // User message history (filtered, newest first)
	historyIdx     int      // Current index (-1 = new input)
	savedInput     string   // Saved draft while browsing history
	maxHistory     int      // Max history entries (default 50)
}

// NewInputManager creates a new input manager.
func NewInputManager(styles UIStyles) *InputManager {
	ta := textarea.New()
	ta.Placeholder = InputPlaceholder
	ta.Focus()
	ta.CharLimit = InputCharLimit
	ta.SetWidth(InputWidth)
	ta.SetHeight(InputHeight)
	ta.ShowLineNumbers = false
	ta.Prompt = "> "
	// Add macOS Cmd+Left/Right as LineStart/LineEnd bindings
	km := ta.KeyMap
	km.LineStart.SetKeys("home", "ctrl+a", "super+left", "alt+left")
	km.LineEnd.SetKeys("end", "ctrl+e", "super+right", "alt+right")
	ta.KeyMap = km
	// Note: InsertNewline key map kept enabled; Shift+Enter inserts newline via InsertNewline()

	// Clean styling without borders
	taStyles := ta.Styles()
	taStyles.Focused.Base = lipglossv2.NewStyle()
	taStyles.Focused.CursorLine = lipglossv2.NewStyle()
	taStyles.Focused.Prompt = lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorCommand))
	taStyles.Blurred.Base = lipglossv2.NewStyle()
	taStyles.Blurred.CursorLine = lipglossv2.NewStyle()
	taStyles.Blurred.Prompt = lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorHelp))
	ta.SetStyles(taStyles)

	return &InputManager{
		textarea:        ta,
		disabled:        false,
		placeholder:     InputWaitingPlaceholder,
		styles:          styles,
		width:           InputWidth,
		maxHeight:       InputMaxHeight,
		userMsgHistory:  []string{},
		historyIdx:      -1,
		savedInput:      "",
		maxHistory:      50,
	}
}

// SetDisabled cleanly enables/disables input.
// When disabled, View() returns placeholder instead of textarea.
// This is the KEY to avoiding ghost text - we never touch textarea state during processing.
func (m *InputManager) SetDisabled(disabled bool) {
	m.disabled = disabled
	// Do NOT modify textarea when disabled - this is key to avoiding ghost text
}

// IsDisabled returns whether input is disabled.
func (m *InputManager) IsDisabled() bool {
	return m.disabled
}

// View returns the appropriate view based on state.
func (m *InputManager) View() string {
	if m.disabled {
		return "> " + m.styles.Help.Render(m.placeholder)
	}
	return m.textarea.View()
}

// Value returns the current input value.
func (m *InputManager) Value() string {
	return m.textarea.Value()
}

// SetValue sets the input value.
func (m *InputManager) SetValue(value string) {
	m.textarea.SetValue(value)
}

// Reset clears the input completely.
func (m *InputManager) Reset() {
	m.textarea.Reset()
	m.textarea.SetValue("")
	m.textarea.Blur()
	m.textarea.Focus()
}

// DisableAndClear disables input and clears the text.
func (m *InputManager) DisableAndClear() {
	// Clear textarea first, then disable
	m.textarea.Reset()
	m.textarea.SetValue("")
	m.textarea.Blur()
	m.disabled = true
	m.textarea.SetHeight(1)
}

// EnableAndFocus enables input and focuses it.
func (m *InputManager) EnableAndFocus() tea.Cmd {
	m.disabled = false
	m.textarea.Reset()
	m.textarea.SetValue("")
	m.textarea.SetHeight(1)
	return m.textarea.Focus()
}

// Focus focuses the textarea.
func (m *InputManager) Focus() tea.Cmd {
	return m.textarea.Focus()
}

// Blur blurs the textarea.
func (m *InputManager) Blur() {
	m.textarea.Blur()
}

// Update handles textarea updates - ONLY when not disabled.
func (m *InputManager) Update(msg tea.Msg) tea.Cmd {
	if m.disabled {
		return nil
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

// SetWidth sets the textarea width.
func (m *InputManager) SetWidth(width int) {
	m.textarea.SetWidth(width)
	m.width = width
}

// Render returns the full input block: top separator + textarea/placeholder + bottom separator.
func (m *InputManager) Render() string {
	border := m.styles.Separator.Render(strings.Repeat("─", m.width))
	if m.disabled {
		return border + "\n> " + m.styles.Help.Render(m.placeholder) + "\n" + border
	}
	return border + "\n" + m.textarea.View() + "\n" + border
}

// Height returns the total number of lines this widget occupies, including separators.
func (m *InputManager) Height() int {
	return 2 + m.currentLineCount()
}

// InsertNewline inserts a newline character into the textarea.
func (m *InputManager) InsertNewline() {
	m.textarea.InsertString("\n")
	m.updateHeight()
}

// updateHeight recalculates the textarea height based on content line count.
func (m *InputManager) updateHeight() {
	h := m.currentLineCount()
	if h != m.textarea.Height() {
		m.textarea.SetHeight(h)
	}
}

// currentLineCount returns the current content line count, clamped to [1, maxHeight].
func (m *InputManager) currentLineCount() int {
	if m.disabled {
		return 1
	}
	value := m.textarea.Value()
	if value == "" {
		return 1
	}
	lines := strings.Count(value, "\n") + 1
	if lines < 1 {
		lines = 1
	}
	if lines > m.maxHeight {
		lines = m.maxHeight
	}
	return lines
}

// SetMaxHeight sets the maximum height for the textarea.
func (m *InputManager) SetMaxHeight(h int) {
	if h < 1 {
		h = 1
	}
	m.maxHeight = h
}

// ForceRedraw forces a clean redraw of the textarea.
// This is useful after terminal resize to prevent ghost text.
func (m *InputManager) ForceRedraw(width int) {
	// Store current value
	value := m.textarea.Value()
	// Reset and reconfigure
	m.textarea.Reset()
	m.textarea.SetValue(value)
	m.textarea.SetWidth(width)
	if !m.disabled {
		m.textarea.Focus()
	}
}

// IsEmpty returns true if the input is empty.
func (m *InputManager) IsEmpty() bool {
	return strings.TrimSpace(m.textarea.Value()) == ""
}

// NavigateUp navigates backward in history (older messages).
// Only triggers when LineCount() == 1 (single line input).
func (m *InputManager) NavigateUp() {
	if len(m.userMsgHistory) == 0 {
		return
	}
	if m.historyIdx < len(m.userMsgHistory)-1 {
		if m.historyIdx == -1 {
			m.savedInput = m.textarea.Value()
		}
		m.historyIdx++
		m.textarea.SetValue(m.userMsgHistory[m.historyIdx])
		m.textarea.SetHeight(1)
	}
}

// NavigateDown navigates forward in history (newer messages).
// Only triggers when LineCount() == 1 (single line input).
func (m *InputManager) NavigateDown() {
	if m.historyIdx > 0 {
		m.historyIdx--
		m.textarea.SetValue(m.userMsgHistory[m.historyIdx])
		m.textarea.SetHeight(1)
	} else if m.historyIdx == 0 {
		m.historyIdx = -1
		m.textarea.SetValue(m.savedInput)
		m.textarea.SetHeight(1)
	}
}

// SetUserMessageHistory sets the input history from user messages.
// Messages should be in reverse order (newest first).
func (m *InputManager) SetUserMessageHistory(history []string) {
	m.userMsgHistory = history
	m.historyIdx = -1
	m.savedInput = ""
}

// AddToHistory adds a message to the input history.
// Deduplication: removes older duplicate entries.
func (m *InputManager) AddToHistory(content string) {
	if content == "" {
		return
	}
	// Remove duplicates from existing history
	for i, entry := range m.userMsgHistory {
		if entry == content {
			m.userMsgHistory = append(m.userMsgHistory[:i], m.userMsgHistory[i+1:]...)
			break
		}
	}
	// Add to front (newest first)
	m.userMsgHistory = append([]string{content}, m.userMsgHistory...)
	// Limit history size
	if len(m.userMsgHistory) > m.maxHistory {
		m.userMsgHistory = m.userMsgHistory[:m.maxHistory]
	}
	// Reset navigation state
	m.historyIdx = -1
	m.savedInput = ""
}

// HistoryLen returns the length of input history.
func (m *InputManager) HistoryLen() int {
	return len(m.userMsgHistory)
}

// LineCount returns the current line count of the textarea.
func (m *InputManager) LineCount() int {
	return m.currentLineCount()
}
