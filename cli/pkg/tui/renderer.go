package tui

import (
	"strings"

	"charm.land/glamour/v2"
	lipglossv2 "charm.land/lipgloss/v2"
)

// MarkdownRenderer wraps glamour v2 for consistent markdown rendering.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer(width int) *MarkdownRenderer {
	r := &MarkdownRenderer{
		width: width,
	}
	r.initRenderer()
	return r
}

// initRenderer initializes the glamour renderer.
func (r *MarkdownRenderer) initRenderer() {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("notty"),
		glamour.WithWordWrap(0), // Disable WordWrap to avoid padding issues
	)
	if err != nil {
		r.renderer = nil
		return
	}
	r.renderer = renderer
}

// Render renders markdown content to ANSI styled string.
func (r *MarkdownRenderer) Render(content string) string {
	if r.renderer == nil {
		return content
	}

	rendered, err := r.renderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimRight(rendered, "\n")
}

// UpdateWidth updates the renderer width and reinitializes if needed.
func (r *MarkdownRenderer) UpdateWidth(width int) {
	if width != r.width && width > 0 {
		r.width = width
		r.initRenderer()
	}
}

// DefaultStyles returns the default UI styles.
func DefaultStyles() UIStyles {
	return UIStyles{
		UserMessage:         lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorUserMessage)),
		AuraMessage:         lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorAuraMessage)),
		Thinking:            lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorThinking)).Italic(true),
		Action:              lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorAction)),
		Result:              lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorResult)),
		Error:               lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorError)),
		Help:                lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorHelp)),
		Processing:          lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorProcessing)),
		Timestamp:           lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorTimestamp)),
		Separator:           lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorSeparator)),
		Command:             lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorCommand)).Bold(true),
		CommandItemSelected: lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorCommandFg)).Bold(true).Background(lipglossv2.Color(ColorCommandBg)),
		CommandItem:         lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorCommandItem)),
		CommandDesc:         lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorCommandDesc)),
		// Tool block styles
		ToolBlockHeader:  lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolBlockHeader)).Bold(true),
		ToolBlockIn:      lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolBlockIn)).Bold(true),
		ToolBlockInBg:    lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolBlockInFg)).Background(lipglossv2.Color(ColorToolBlockInBg)),
		ToolBlockOut:     lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolBlockOut)).Bold(true),
		ToolBlockDone:    lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolDone)),
		ToolBlockBorder:  lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorToolBlockBorder)).Border(lipglossv2.Border{Left: "│"}).BorderLeft(true).PaddingLeft(1),
		// Task widget styles
		TaskWidgetTitle:  lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorTaskWidgetTitle)).Bold(true),
		TaskWidgetBorder: lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorTaskWidgetBorder)),
		TaskInProgress:   lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorTaskInProgress)).Bold(true),
		// Plan widget styles
		PlanWidgetTitle:  lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorPlanWidgetTitle)).Bold(true),
		PlanWidgetBorder: lipglossv2.NewStyle().Foreground(lipglossv2.Color(ColorPlanWidgetBorder)),
	}
}
