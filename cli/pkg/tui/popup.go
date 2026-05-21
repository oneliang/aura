package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// PopupBase provides common pagination and navigation functionality for popups.
type PopupBase struct {
	showing    bool
	page       int
	pageSize   int
	totalPages int
}

// NewPopupBase creates a new popup base with the given page size.
func NewPopupBase(pageSize int) PopupBase {
	return PopupBase{
		pageSize: pageSize,
	}
}

// Show displays the popup.
func (p *PopupBase) Show() {
	p.showing = true
}

// Hide hides the popup.
func (p *PopupBase) Hide() {
	p.showing = false
}

// Toggle toggles the popup visibility.
func (p *PopupBase) Toggle() {
	p.showing = !p.showing
}

// IsShowing returns whether the popup is showing.
func (p *PopupBase) IsShowing() bool {
	return p.showing
}

// UpdatePagination recalculates pagination based on item count.
func (p *PopupBase) UpdatePagination(itemCount int) {
	p.page = 0
	if itemCount > 0 {
		p.totalPages = (itemCount + p.pageSize - 1) / p.pageSize
	} else {
		p.totalPages = 0
	}
}

// GetPageRange returns the start and end indices for the current page.
func (p *PopupBase) GetPageRange(itemCount int) (int, int) {
	startIdx := p.page * p.pageSize
	endIdx := startIdx + p.pageSize
	if endIdx > itemCount {
		endIdx = itemCount
	}
	return startIdx, endIdx
}

// HandlePageNavigation handles page navigation keys and returns whether the key was handled.
func (p *PopupBase) HandlePageNavigation(msg tea.KeyPressMsg, listModel list.Model, itemCount int) bool {
	switch msg.Code {
	case tea.KeyUp:
		if listModel.Index() > 0 {
			if listModel.Index() < p.page*p.pageSize {
				p.page--
			}
		}
		return true

	case tea.KeyDown:
		if listModel.Index() < itemCount-1 {
			if listModel.Index() >= (p.page+1)*p.pageSize && p.page < p.totalPages {
				p.page++
			}
		}
		return true

	case tea.KeyPgUp:
		if p.page > 0 {
			p.page--
			newIndex := p.page * p.pageSize
			if newIndex < itemCount {
				listModel.Select(newIndex)
			}
		}
		return true

	case tea.KeyPgDown:
		if (p.page+1)*p.pageSize < itemCount {
			p.page++
			newIndex := p.page * p.pageSize
			if newIndex < itemCount {
				listModel.Select(newIndex)
			}
		}
		return true
	}
	return false
}

// RenderPager returns the pager display string.
func (p *PopupBase) RenderPager(styles UIStyles) string {
	if p.totalPages <= 1 {
		return ""
	}
	return styles.Help.Render(fmt.Sprintf("  [%d/%d]", p.page+1, p.totalPages))
}

// RenderItemLine renders a single item line with selection styling.
func (p *PopupBase) RenderItemLine(styles UIStyles, index int, selectedIndex int, title string, description string) string {
	var line string
	if index == selectedIndex {
		line = styles.CommandItemSelected.Render(" ❯ "+title) +
			styles.CommandDesc.Render("  "+description)
	} else {
		line = styles.CommandItem.Render("   "+title) +
			styles.CommandDesc.Render("  "+description)
	}
	return line
}

// ListWrapper wraps a bubbles list with common functionality.
type ListWrapper struct {
	list list.Model
}

// NewListWrapper creates a new list wrapper.
func NewListWrapper(width, height int) ListWrapper {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	return ListWrapper{list: l}
}

// SetItems sets the list items.
func (l *ListWrapper) SetItems(items []list.Item) {
	l.list.SetItems(items)
	l.list.ResetSelected()
}

// GetIndex returns the current selected index.
func (l *ListWrapper) GetIndex() int {
	return l.list.Index()
}

// Select selects an item at the given index.
func (l *ListWrapper) Select(index int) {
	l.list.Select(index)
}

// Update updates the list with the given message.
func (l *ListWrapper) Update(msg tea.Msg) (tea.Cmd, bool) {
	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return cmd, l.list.FilterState() == list.Filtering
}

// IsFiltering returns whether the list is in filtering mode.
func (l *ListWrapper) IsFiltering() bool {
	return l.list.FilterState() == list.Filtering
}

// GetSelectedItem returns the currently selected item.
func (l *ListWrapper) GetSelectedItem() list.Item {
	return l.list.SelectedItem()
}

// GetList returns the underlying list model.
func (l *ListWrapper) GetList() *list.Model {
	return &l.list
}

// RenderHeader renders the popup header with title and help text.
func RenderHeader(styles UIStyles, title string, helpText string, width int) []string {
	var lines []string
	header := styles.Help.Render("  "+title) + styles.Help.Render("  "+helpText)
	lines = append(lines, header)
	lines = append(lines, styles.Help.Render(strings.Repeat("─", width)))
	return lines
}
