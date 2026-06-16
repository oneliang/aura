package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// QuestionPopup displays a question dialog for text input, single choice, or multi-choice.
type QuestionPopup struct {
	PopupBase
	question     string
	questionType QuestionType
	options      []QuestionOption
	textarea     textarea.Model
	selected     int       // For choice
	selectedOpts map[int]bool // For multi_choice
	otherText    string    // Custom text input when "Other" option is selected
	width        int
	height       int
	onSubmit     func(answer string, selections []string)
	onCancel     func()
}

// NewQuestionPopup creates a new question popup.
func NewQuestionPopup(width, height int) *QuestionPopup {
	ta := textarea.New()
	ta.SetWidth(width - 4)
	ta.SetHeight(5)
	ta.ShowLineNumbers = false
	ta.Placeholder = "Type your answer..."
	ta.Focus()

	// Strip default textarea chrome to match popup theme and prevent background flicker.
	// The default CursorLine (inverted bg), Prompt ("▐ "), and Base frame cause visual
	// artifacts when rendered inside the popup overlay.
	s := ta.Styles()
	noChrome := lipglossv2.NewStyle().Padding(0).Margin(0)
	s.Focused.Base = noChrome
	s.Blurred.Base = noChrome
	s.Focused.CursorLine = lipglossv2.NewStyle()
	s.Blurred.CursorLine = lipglossv2.NewStyle()
	s.Focused.Prompt = lipglossv2.NewStyle()
	s.Blurred.Prompt = lipglossv2.NewStyle()
	s.Focused.Text = s.Focused.Text.Foreground(lipglossv2.Color(ColorCommandItem))
	s.Blurred.Text = s.Blurred.Text.Foreground(lipglossv2.Color(ColorCommandItem))
	s.Focused.Placeholder = s.Focused.Placeholder.Foreground(lipglossv2.Color(ColorHelp))
	s.Cursor.Blink = false
	ta.SetStyles(s)
	ta.Prompt = ""

	return &QuestionPopup{
		PopupBase:    NewPopupBase(10),
		textarea:     ta,
		width:        width,
		height:       height,
		selectedOpts: make(map[int]bool),
	}
}

// Show displays the question popup with the given question.
func (p *QuestionPopup) Show(question string, qType QuestionType, options []QuestionOption, onSubmit func(string, []string), onCancel func()) {
	p.question = question
	p.questionType = qType
	p.options = p.deduplicateOther(options)
	p.onSubmit = onSubmit
	p.onCancel = onCancel
	p.selected = 0
	p.selectedOpts = make(map[int]bool)
	p.otherText = ""

	if qType == QuestionTypeText {
		p.textarea.SetValue("")
		p.textarea.Focus()
	}

	p.PopupBase.Show()
}

// deduplicateOther ensures any LLM-generated "Other"-like option uses the sentinel value,
// so it's properly detected by isOtherOption(). This handles the case where the LLM
// ignores the instruction not to include "Other" and adds its own variant.
func (p *QuestionPopup) deduplicateOther(options []QuestionOption) []QuestionOption {
	for i := range options {
		if options[i].Value != constants.OtherOptionValue && isOtherLabel(options[i].Label) {
			options[i].Value = constants.OtherOptionValue
		}
	}
	return options
}

// isOtherLabel checks if a label matches known "Other" translations.
func isOtherLabel(label string) bool {
	known := []string{
		"Other",
		"其它",
		"其他",
		"Others",
		"其它选项",
		"其他选项",
	}
	// Also match the localized "Other" label
	localized := i18n.T("tui.question.other")
	for _, k := range known {
		if label == k || label == localized {
			return true
		}
	}
	return false
}

// Hide hides the question popup.
func (p *QuestionPopup) Hide() {
	p.PopupBase.Hide()
}

// Update handles messages for the question popup.
func (p *QuestionPopup) Update(msg tea.Msg) tea.Cmd {
	if !p.IsShowing() {
		return nil
	}

	if p.questionType == QuestionTypeText {
		var cmd tea.Cmd
		p.textarea, cmd = p.textarea.Update(msg)
		return cmd
	}

	return nil
}

// isOtherOption returns true if the option at index i is the "Other" sentinel.
func (p *QuestionPopup) isOtherOption(i int) bool {
	return i >= 0 && i < len(p.options) && p.options[i].Value == constants.OtherOptionValue
}

// HandleKey handles key presses for the question popup.
// Returns: submitted, answer, selections, cancelled
func (p *QuestionPopup) HandleKey(msg tea.KeyPressMsg) (bool, string, []string, bool) {
	if !p.IsShowing() {
		return false, "", nil, false
	}

	// When "Other" is selected in choice mode, handle text input
	if p.questionType == QuestionTypeChoice && p.isOtherOption(p.selected) {
		switch msg.Code {
		case tea.KeyEsc:
			p.Hide()
			if p.onCancel != nil {
				p.onCancel()
			}
			return false, "", nil, true
		case tea.KeyEnter:
			if msg.Mod == tea.ModShift {
				p.otherText += "\n"
				return false, "", nil, false
			}
			answer := strings.TrimSpace(p.otherText)
			p.Hide()
			if p.onSubmit != nil {
				p.onSubmit(answer, nil)
			}
			return true, answer, nil, false
		case tea.KeyBackspace:
			if len(p.otherText) > 0 {
				p.otherText = p.otherText[:len(p.otherText)-1]
			}
			return false, "", nil, false
		case tea.KeyUp, tea.KeyDown:
			// Allow navigating away from "Other"
		default:
			// Append typed character
			if msg.Text != "" {
				p.otherText += msg.Text
			}
			return false, "", nil, false
		}
	}

	// When "Other" is checked in multi_choice mode, handle text input
	if p.questionType == QuestionTypeMultiChoice && p.isOtherOption(p.selected) && p.selectedOpts[p.selected] {
		switch msg.Code {
		case tea.KeyEsc:
			p.Hide()
			if p.onCancel != nil {
				p.onCancel()
			}
			return false, "", nil, true
		case tea.KeyEnter:
			if msg.Mod == tea.ModShift {
				p.otherText += "\n"
				return false, "", nil, false
			}
			var selections []string
			for i, opt := range p.options {
				if p.selectedOpts[i] {
					if p.isOtherOption(i) {
						if t := strings.TrimSpace(p.otherText); t != "" {
							selections = append(selections, t)
						}
					} else {
						selections = append(selections, opt.Value)
					}
				}
			}
			p.Hide()
			if p.onSubmit != nil {
				p.onSubmit("", selections)
			}
			return true, "", selections, false
		case tea.KeyBackspace:
			if len(p.otherText) > 0 {
				p.otherText = p.otherText[:len(p.otherText)-1]
			}
			return false, "", nil, false
		case tea.KeySpace:
			// Uncheck "Other"
			p.selectedOpts[p.selected] = false
			return false, "", nil, false
		case tea.KeyUp, tea.KeyDown:
			// Allow navigating away
		default:
			if msg.Text != "" {
				p.otherText += msg.Text
			}
			return false, "", nil, false
		}
	}

	switch msg.Code {
	case tea.KeyEsc:
		p.Hide()
		if p.onCancel != nil {
			p.onCancel()
		}
		return false, "", nil, true

	case tea.KeyEnter:
		switch p.questionType {
		case QuestionTypeText:
			if msg.Mod == tea.ModShift {
				// Let textarea handle Shift+Enter as newline insertion
				return false, "", nil, false
			}
			answer := strings.TrimSpace(p.textarea.Value())
			p.Hide()
			if p.onSubmit != nil {
				p.onSubmit(answer, nil)
			}
			return true, answer, nil, false

		case QuestionTypeChoice:
			if p.selected >= 0 && p.selected < len(p.options) {
				answer := p.options[p.selected].Value
				p.Hide()
				if p.onSubmit != nil {
					p.onSubmit(answer, nil)
				}
				return true, answer, nil, false
			}

		case QuestionTypeMultiChoice:
			var selections []string
			for i, opt := range p.options {
				if p.selectedOpts[i] {
					if p.isOtherOption(i) {
						if t := strings.TrimSpace(p.otherText); t != "" {
							selections = append(selections, t)
						}
					} else {
						selections = append(selections, opt.Value)
					}
				}
			}
			p.Hide()
			if p.onSubmit != nil {
				p.onSubmit("", selections)
			}
			return true, "", selections, false
		}

	case tea.KeyUp:
		if p.questionType == QuestionTypeChoice || p.questionType == QuestionTypeMultiChoice {
			if p.selected > 0 {
				p.selected--
			}
		}

	case tea.KeyDown:
		if p.questionType == QuestionTypeChoice || p.questionType == QuestionTypeMultiChoice {
			if p.selected < len(p.options)-1 {
				p.selected++
			}
		}

	case tea.KeySpace:
		if p.questionType == QuestionTypeMultiChoice {
			p.selectedOpts[p.selected] = !p.selectedOpts[p.selected]
		}
	}

	return false, "", nil, false
}

// Render renders the question popup with top/bottom border lines.
// Layout: border → title → separator → content → help text → border.
func (p *QuestionPopup) Render(styles UIStyles) string {
	if !p.IsShowing() {
		return ""
	}

	var lines []string
	width := p.width
	if width > 80 {
		width = 80
	}
	contentWidth := width - 4 // inner content width (with 2-space indent on each side)

	// Title — use actual question text (truncated to fit)
	titleText := p.question
	if len(titleText) > contentWidth {
		titleText = titleText[:contentWidth-3] + "..."
	}
	if titleText == "" {
		titleText = "Question"
	}
	lines = append(lines, styles.Command.Render("  "+titleText))

	// Separator
	lines = append(lines, styles.Help.Render("  "+strings.Repeat("─", contentWidth)))

	// Blank line
	lines = append(lines, "")

	// Input area based on question type
	switch p.questionType {
	case QuestionTypeText:
		lines = append(lines, p.renderTextInput(styles, contentWidth)...)
	case QuestionTypeChoice:
		lines = append(lines, p.renderChoice(styles, contentWidth)...)
	case QuestionTypeMultiChoice:
		lines = append(lines, p.renderMultiChoice(styles, contentWidth)...)
	}

	// Blank line before help
	lines = append(lines, "")

	// Help text
	lines = append(lines, styles.Help.Render("  "+p.getHelpText()))

	// Calculate border width: max of content lines and minimum width
	popupW := 0
	for _, line := range lines {
		if w := len(line); w > popupW {
			popupW = w
		}
	}
	if popupW < OverlayMinWidth {
		popupW = OverlayMinWidth
	}

	// Add top/bottom borders
	borderLine := styles.Separator.Render(strings.Repeat(OverlayBorderChar, popupW))
	allLines := append([]string{borderLine}, lines...)
	allLines = append(allLines, borderLine)

	return strings.Join(allLines, "\n")
}

// Height returns the total height of the rendered popup including borders.
// Computed from internal state to avoid double-rendering.
func (p *QuestionPopup) Height() int {
	if !p.IsShowing() {
		return 0
	}
	// Fixed overhead: top border + title + separator + blank + blank + help + bottom border = 7
	h := 7
	switch p.questionType {
	case QuestionTypeText:
		h += 5 // textarea lines
	case QuestionTypeChoice:
		h += len(p.options)
		if p.isOtherOption(p.selected) {
			h++ // Other input line
		}
	case QuestionTypeMultiChoice:
		h += len(p.options)
		for i := range p.options {
			if p.isOtherOption(i) && p.selectedOpts[i] {
				h++ // Other input line
			}
		}
	}
	return h
}

func (p *QuestionPopup) renderTextInput(styles UIStyles, contentWidth int) []string {
	var lines []string

	// Textarea content (5 lines)
	taView := p.textarea.View()
	taLines := strings.Split(taView, "\n")
	for i := 0; i < 5; i++ {
		line := ""
		if i < len(taLines) {
			line = taLines[i]
		}
		if len(line) > contentWidth {
			line = line[:contentWidth]
		}
		lines = append(lines, "  "+line)
	}

	return lines
}

func (p *QuestionPopup) renderChoice(styles UIStyles, contentWidth int) []string {
	var lines []string

	for i, opt := range p.options {
		marker := "( )"
		if i == p.selected {
			marker = "(•)"
		}
		line := fmt.Sprintf("  %s %s", marker, opt.Label)
		if len(line) > contentWidth+2 {
			line = line[:contentWidth+2]
		}

		if i == p.selected {
			lines = append(lines, styles.Command.Render(line))
		} else {
			lines = append(lines, styles.CommandItem.Render(line))
		}

		// Show text input below when "Other" is selected
		if p.isOtherOption(i) && i == p.selected {
			lines = append(lines, p.renderOtherInput(styles, contentWidth)...)
		}
	}

	return lines
}

func (p *QuestionPopup) renderMultiChoice(styles UIStyles, contentWidth int) []string {
	var lines []string

	for i, opt := range p.options {
		marker := "[ ]"
		if p.selectedOpts[i] {
			marker = "[•]"
		}
		line := fmt.Sprintf("  %s %s", marker, opt.Label)
		if len(line) > contentWidth+2 {
			line = line[:contentWidth+2]
		}

		if i == p.selected {
			lines = append(lines, styles.Command.Render(line))
		} else {
			lines = append(lines, styles.CommandItem.Render(line))
		}

		// Show text input below when "Other" is checked
		if p.isOtherOption(i) && p.selectedOpts[i] {
			lines = append(lines, p.renderOtherInput(styles, contentWidth)...)
		}
	}

	return lines
}

// renderOtherInput renders the text input line for the "Other" option with
// a visible cursor and placeholder to make it obvious that input is expected.
func (p *QuestionPopup) renderOtherInput(styles UIStyles, contentWidth int) []string {
	var display string
	if p.otherText == "" {
		// Show placeholder + cursor when empty
		display = fmt.Sprintf("      > %s ▏", i18n.T("tui.question.other_placeholder"))
	} else {
		// Show typed text + cursor
		display = fmt.Sprintf("      > %s▏", p.otherText)
	}
	if len(display) > contentWidth+2 {
		display = display[:contentWidth+2]
	}
	// Use Help style (dim) for placeholder, Command style for active input
	if p.otherText == "" {
		return []string{styles.Help.Render(display)}
	}
	return []string{styles.Command.Render(display)}
}

func (p *QuestionPopup) getHelpText() string {
	switch p.questionType {
	case QuestionTypeText:
		return "  Enter submit  Esc cancel  "
	case QuestionTypeChoice:
		return "  ↑↓ select  Enter confirm  Esc cancel  "
	case QuestionTypeMultiChoice:
		return "  ↑↓ navigate  Space toggle  Enter submit  Esc cancel  "
	default:
		return "  Enter submit  Esc cancel  "
	}
}

