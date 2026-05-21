package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// SessionItem implements list.Item for session selection.
type SessionItem struct {
	id      string
	name    string
	created int64
	updated int64
	subs    int
	role    string
}

// Title returns the session title with role tag.
func (i SessionItem) Title() string {
	roleTag := ""
	if i.role != "" {
		roleTag = fmt.Sprintf(" [%s]", i.role)
	}
	return i.name + roleTag
}

// Description returns the session description.
func (i SessionItem) Description() string {
	updatedStr := time.UnixMilli(i.updated).Format("2006-01-02 15:04")
	subsStr := ""
	if i.subs > 0 {
		subsStr = fmt.Sprintf("  %d subscriptions", i.subs)
	}
	return updatedStr + subsStr
}

// FilterValue returns the value used for filtering.
func (i SessionItem) FilterValue() string { return i.name }

// ID returns the session ID.
func (i SessionItem) ID() string { return i.id }

// Name returns the session name.
func (i SessionItem) Name() string { return i.name }

// Updated returns the last update time.
func (i SessionItem) Updated() int64 { return i.updated }

// Role returns the session role.
func (i SessionItem) Role() string { return i.role }

// Subs returns the subscription count.
func (i SessionItem) Subs() int { return i.subs }

// Created returns the creation time.
func (i SessionItem) Created() int64 { return i.created }

// SessionPopupState holds the state for session selection popup.
type SessionPopupState struct {
	base  PopupBase
	list  ListWrapper
	items []SessionItem
}

// NewSessionPopup creates a new session selection popup.
func NewSessionPopup() SessionPopupState {
	return SessionPopupState{
		base:  NewPopupBase(6),
		list:  NewListWrapper(0, 6),
		items: []SessionItem{},
	}
}

// UpdateItems updates the list items and recalculates pagination.
func (s *SessionPopupState) UpdateItems(items []SessionItem) {
	s.items = items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	s.list.SetItems(listItems)
	s.base.UpdatePagination(len(items))
}

// Show displays the session popup.
func (s *SessionPopupState) Show() {
	s.base.Show()
}

// Hide hides the session popup.
func (s *SessionPopupState) Hide() {
	s.base.Hide()
}

// Toggle toggles the popup visibility.
func (s *SessionPopupState) Toggle() {
	s.base.Toggle()
}

// IsShowing returns whether the popup is showing.
func (s *SessionPopupState) IsShowing() bool {
	return s.base.IsShowing()
}

// Selected returns the currently selected session item.
func (s *SessionPopupState) Selected() (SessionItem, bool) {
	if len(s.items) == 0 {
		return SessionItem{}, false
	}
	if item, ok := s.list.GetSelectedItem().(SessionItem); ok {
		return item, true
	}
	return SessionItem{}, false
}

// Render renders the session popup with fixed width.
func (s *SessionPopupState) Render(styles UIStyles, width int) string {
	if !s.base.IsShowing() || len(s.items) == 0 {
		return ""
	}

	startIdx, endIdx := s.base.GetPageRange(len(s.items))
	var lines []string

	// Use fixed width for popup content
	popupWidth := width
	if popupWidth > OverlayMinWidth {
		popupWidth = OverlayMinWidth
	}
	if popupWidth < 20 {
		popupWidth = 20
	}

	// Header
	header := styles.Help.Render("  Select Session") + styles.Help.Render("  ↑/↓ navigate  Enter select  Esc cancel  / filter")
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

// HandleKeyMsg handles keyboard input for the session popup.
func (s *SessionPopupState) HandleKeyMsg(msg tea.KeyPressMsg, styles UIStyles) (tea.Cmd, bool) {
	if !s.base.IsShowing() {
		return nil, false
	}

	// Check if filtering is active
	if s.list.IsFiltering() {
		cmd, _ := s.list.Update(msg)
		return cmd, true
	}

	switch msg.Code {
	case tea.KeyEsc:
		s.base.Hide()
		return nil, true

	case tea.KeyEnter:
		item, ok := s.Selected()
		if ok {
			s.base.Hide()
			return selectSessionCmd(item), true
		}
		return nil, true

	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		s.base.HandlePageNavigation(msg, *s.list.GetList(), len(s.items))
		s.list.Update(msg)
		return nil, true
	}

	return nil, false
}

// Message types for session operations
type selectSessionMsg struct {
	item SessionItem
}

type createSessionMsg struct {
	name string
	role string
}

func selectSessionCmd(item SessionItem) tea.Cmd {
	return func() tea.Msg {
		return selectSessionMsg{item: item}
	}
}

// RenderSessionList renders a session list for display (non-interactive).
func RenderSessionList(sessions []SessionItem, styles UIStyles) string {
	if len(sessions) == 0 {
		return styles.Help.Render("\n  No sessions found. Use /session create to create one.")
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(styles.Help.Render(fmt.Sprintf("Sessions (%d):", len(sessions))) + "\n\n")

	for i, s := range sessions {
		sb.WriteString(fmt.Sprintf("  %d. ", i+1))
		sb.WriteString(styles.Command.Render(s.Title()))
		sb.WriteString(styles.Help.Render("  " + s.Description()))
		sb.WriteString("\n")
	}

	return sb.String()
}
