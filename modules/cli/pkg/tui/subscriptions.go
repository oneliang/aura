package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// SubscriptionItem implements list.Item for subscription management.
type SubscriptionItem struct {
	id      string
	trigger string
	source  string
	active  bool
}

// Title returns the subscription title with status indicator.
func (i SubscriptionItem) Title() string {
	status := "✓"
	if !i.active {
		status = "✗"
	}
	return fmt.Sprintf("[%s] %s", status, i.trigger)
}

// Description returns the subscription description.
func (i SubscriptionItem) Description() string {
	return fmt.Sprintf("source: %s", i.source)
}

// FilterValue returns the value used for filtering.
func (i SubscriptionItem) FilterValue() string { return i.trigger }

// IsActive returns whether the subscription is active.
func (i SubscriptionItem) IsActive() bool { return i.active }

// Trigger returns the trigger keyword.
func (i SubscriptionItem) Trigger() string { return i.trigger }

// Source returns the source.
func (i SubscriptionItem) Source() string { return i.source }

// ID returns the subscription ID.
func (i SubscriptionItem) ID() string { return i.id }

// SubscriptionPopupState holds the state for subscription management popup.
type SubscriptionPopupState struct {
	base       PopupBase
	list       ListWrapper
	items      []SubscriptionItem
	inputMode  bool
	inputField textarea.Model
	inputStep  int    // 0: trigger input, 1: source input
	triggerVal string // stored trigger value during two-step input
}

// NewSubscriptionPopup creates a new subscription management popup.
func NewSubscriptionPopup() SubscriptionPopupState {
	ta := textarea.New()
	ta.Placeholder = "Enter trigger keyword..."
	ta.Focus()
	ta.CharLimit = 50
	ta.SetWidth(40)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false

	return SubscriptionPopupState{
		base:       NewPopupBase(5),
		list:       NewListWrapper(0, 5),
		items:      []SubscriptionItem{},
		inputField: ta,
		inputStep:  0,
	}
}

// UpdateItems updates the list items and recalculates pagination.
func (s *SubscriptionPopupState) UpdateItems(items []SubscriptionItem) {
	s.items = items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	s.list.SetItems(listItems)
	s.base.UpdatePagination(len(items))
}

// Show displays the subscription popup.
func (s *SubscriptionPopupState) Show() {
	s.base.Show()
}

// Hide hides the subscription popup.
func (s *SubscriptionPopupState) Hide() {
	s.base.Hide()
	s.inputMode = false
	s.inputStep = 0
}

// Toggle toggles the popup visibility.
func (s *SubscriptionPopupState) Toggle() {
	s.base.Toggle()
}

// IsShowing returns whether the popup is showing.
func (s *SubscriptionPopupState) IsShowing() bool {
	return s.base.IsShowing()
}

// StartAddInput enters input mode for adding new subscription.
func (s *SubscriptionPopupState) StartAddInput() {
	s.inputMode = true
	s.inputStep = 0
	s.triggerVal = ""
	s.inputField.SetValue("")
	s.inputField.Placeholder = "Enter trigger keyword..."
	s.inputField.Focus()
}

// StopInput exits input mode.
func (s *SubscriptionPopupState) StopInput() {
	s.inputMode = false
	s.inputStep = 0
	s.inputField.Blur()
}

// Selected returns the currently selected subscription item.
func (s *SubscriptionPopupState) Selected() (SubscriptionItem, bool) {
	if len(s.items) == 0 {
		return SubscriptionItem{}, false
	}
	if item, ok := s.list.GetSelectedItem().(SubscriptionItem); ok {
		return item, true
	}
	return SubscriptionItem{}, false
}

// Render renders the subscription popup with fixed width.
func (s *SubscriptionPopupState) Render(styles UIStyles, width int) string {
	if !s.base.IsShowing() {
		return ""
	}

	// If in input mode, render input field
	if s.inputMode {
		return s.renderInputMode(styles, width)
	}

	// If no items, show empty message
	if len(s.items) == 0 {
		return s.renderEmptyList(styles, width)
	}

	return s.renderList(styles, width)
}

func (s *SubscriptionPopupState) renderInputMode(styles UIStyles, width int) string {
	var lines []string

	stepLabel := "Step 1: Enter Trigger Keyword"
	if s.inputStep == 1 {
		stepLabel = "Step 2: Enter Source (default: *)"
	}

	popupWidth := width
	if popupWidth > OverlayMinWidth {
		popupWidth = OverlayMinWidth
	}
	if popupWidth < 20 {
		popupWidth = 20
	}

	header := styles.Help.Render("  Add Subscription") +
		styles.Help.Render(fmt.Sprintf("  %s  Enter confirm  Esc cancel", stepLabel))
	lines = append(lines, header)
	lines = append(lines, styles.Help.Render(strings.Repeat("─", popupWidth)))

	inputArea := s.inputField.View()
	lines = append(lines, "", inputArea, "")

	return strings.Join(lines, "\n")
}

func (s *SubscriptionPopupState) renderEmptyList(styles UIStyles, width int) string {
	var lines []string
	popupWidth := width
	if popupWidth > OverlayMinWidth {
		popupWidth = OverlayMinWidth
	}
	if popupWidth < 20 {
		popupWidth = 20
	}
	header := styles.Help.Render("  Subscriptions") +
		styles.Help.Render("  a add  Esc close")
	lines = append(lines, header)
	lines = append(lines, styles.Help.Render(strings.Repeat("─", popupWidth)))
	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("  No subscriptions. Press 'a' to add or use CLI:"))
	lines = append(lines, styles.Help.Render("    aura session subscribe <id> --trigger=\"keyword\""))
	return strings.Join(lines, "\n")
}

func (s *SubscriptionPopupState) renderList(styles UIStyles, width int) string {
	startIdx, endIdx := s.base.GetPageRange(len(s.items))

	var lines []string

	popupWidth := width
	if popupWidth > OverlayMinWidth {
		popupWidth = OverlayMinWidth
	}
	if popupWidth < 20 {
		popupWidth = 20
	}

	// Header with keyboard shortcuts
	header := styles.Help.Render("  Subscriptions") +
		styles.Help.Render("  ↑/↓ navigate  a add  d delete  t toggle  Esc close  / filter")
	lines = append(lines, header)
	lines = append(lines, styles.Help.Render(strings.Repeat("─", popupWidth)))

	// Items
	for i := startIdx; i < endIdx; i++ {
		item := s.items[i]
		line := s.base.RenderItemLine(styles, i, s.list.GetIndex(), item.Title(), item.Description())
		lines = append(lines, line)
	}

	// Pager
	if pager := s.base.RenderPager(styles); pager != "" {
		lines = append(lines, pager)
	}

	return strings.Join(lines, "\n")
}

// HandleKeyMsg handles keyboard input for the subscription popup.
func (s *SubscriptionPopupState) HandleKeyMsg(msg tea.KeyPressMsg, styles UIStyles) (tea.Cmd, bool) {
	if !s.base.IsShowing() {
		return nil, false
	}

	// Handle input mode
	if s.inputMode {
		return s.handleInputMode(msg)
	}

	// Check if filtering is active
	if s.list.IsFiltering() {
		cmd, _ := s.list.Update(msg)
		return cmd, true
	}

	return s.handleNavigationMode(msg)
}

func (s *SubscriptionPopupState) handleInputMode(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.Code {
	case tea.KeyEsc:
		s.StopInput()
		return nil, true

	case tea.KeyEnter:
		if s.inputStep == 0 {
			// Store trigger and move to source input
			s.triggerVal = s.inputField.Value()
			if s.triggerVal == "" {
				return nil, true // Don't proceed with empty trigger
			}
			s.inputStep = 1
			s.inputField.SetValue("")
			s.inputField.Placeholder = "Enter source (default: *)"
		} else {
			// Complete input - signal to add subscription
			trigger := s.triggerVal
			source := s.inputField.Value()
			if source == "" {
				source = "*"
			}
			s.StopInput()
			return addSubscriptionCmd(trigger, source), true
		}
		return nil, true

	case tea.KeyTab:
		// Toggle between trigger and source input
		if s.inputStep == 0 {
			s.triggerVal = s.inputField.Value()
			s.inputStep = 1
			s.inputField.SetValue("")
			s.inputField.Placeholder = "Enter source (default: *)"
		} else {
			s.inputField.SetValue(s.triggerVal)
			s.inputStep = 0
			s.inputField.Placeholder = "Enter trigger keyword..."
			s.triggerVal = ""
		}
		return nil, true
	}

	// Let textarea handle other input
	var cmd tea.Cmd
	s.inputField, cmd = s.inputField.Update(msg)
	return cmd, true
}

func (s *SubscriptionPopupState) handleNavigationMode(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.Code {
	case tea.KeyEsc:
		s.base.Hide()
		return nil, true

	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		s.base.HandlePageNavigation(msg, *s.list.GetList(), len(s.items))
		s.list.Update(msg)
		return nil, true
	}

	// Handle character keys
	switch msg.Text {
	case "a":
		s.StartAddInput()
		return nil, true

	case "d":
		item, ok := s.Selected()
		if ok {
			return deleteSubscriptionCmd(item), true
		}
		return nil, true

	case "t":
		item, ok := s.Selected()
		if ok {
			return toggleSubscriptionCmd(item), true
		}
		return nil, true
	}

	return nil, false
}

// Message types for subscription operations
type addSubscriptionMsg struct {
	trigger string
	source  string
}

type deleteSubscriptionMsg struct {
	item SubscriptionItem
}

type toggleSubscriptionMsg struct {
	item SubscriptionItem
}

func addSubscriptionCmd(trigger, source string) tea.Cmd {
	return func() tea.Msg {
		return addSubscriptionMsg{trigger: trigger, source: source}
	}
}

func deleteSubscriptionCmd(item SubscriptionItem) tea.Cmd {
	return func() tea.Msg {
		return deleteSubscriptionMsg{item: item}
	}
}

func toggleSubscriptionCmd(item SubscriptionItem) tea.Cmd {
	return func() tea.Msg {
		return toggleSubscriptionMsg{item: item}
	}
}

// RenderSubscriptionList renders a subscription list for display (non-interactive).
func RenderSubscriptionList(subs []SubscriptionItem, styles UIStyles) string {
	if len(subs) == 0 {
		return styles.Help.Render("\n  No subscriptions. Use /subscription add to create one.")
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(styles.Help.Render(fmt.Sprintf("Subscriptions (%d):", len(subs))) + "\n\n")

	for _, s := range subs {
		status := "active"
		if !s.active {
			status = "inactive"
		}
		sb.WriteString(fmt.Sprintf("  • %s (source: %s, %s)\n", s.trigger, s.source, status))
	}

	return sb.String()
}
