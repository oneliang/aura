package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Refresh statusBar model pointer — Bubble Tea MVU returns new Model
	// values on each Update, so the pointer captured in New() becomes stale.
	m.statusBar.model = &m

	// Sync viewport content and dimensions BEFORE handling any message.
	// View() does the same but has a value receiver — its modifications are
	// not persisted. Without this, scroll key handlers see a viewport with
	// empty/zero content because SetContent is never called in Update().
	m = m.syncViewport()

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case ChatEvent:
		return m.handleChatEvent(msg)

	case tickMsg:
		return m, m.tick()

	// Scroll to bottom command
	case scrollBottomMsg:
		m.viewport.GotoBottom()
		return m, m.processEvents()

	// Print message - print content to stdout
	case printMsg:
		m.messages.AddRaw(msg.content)
		m.autoScroll = true
		m.manualScroll = false
		m.manualScrollOffset = 0
		return m, func() tea.Msg { return scrollBottomMsg{} }

	// Session events
	case selectSessionMsg:
		return m.handleSelectSession(msg)

	case createSessionMsg:
		return m.handleCreateSession(msg)

	// Subscription events
	case addSubscriptionMsg:
		return m.handleAddSubscription(msg)

	case deleteSubscriptionMsg:
		return m.handleDeleteSubscription(msg)

	case toggleSubscriptionMsg:
		return m.handleToggleSubscription(msg)

	case thinkingTickMsg:
		if m.thinking != nil && m.thinking.IsActive() {
			_, nextCmd := m.thinking.Update(msg)
			if nextCmd != nil {
				return m, tea.Sequence(nextCmd, m.processEvents())
			}
		}
		return m, m.processEvents()

	case processingTickMsg:
		if m.processing != nil && m.processing.IsActive() {
			_, nextCmd := m.processing.Update(msg)
			if nextCmd != nil {
				return m, tea.Sequence(nextCmd, m.processEvents())
			}
		}
		return m, m.processEvents()
	}

	// Delegate to viewport for mouse wheel and keyboard scroll
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	// Update input
	if cmd := m.input.Update(msg); cmd != nil {
		return m, cmd
	}

	return m, m.processEvents()
}

// syncViewport updates the viewport's content and dimensions so that scroll
// operations see the current chat state. View() does the same but has a value
// receiver, so its modifications are not visible to Update().
func (m Model) syncViewport() Model {
	if !m.viewportReady {
		return m
	}

	width := m.state.Width()
	if width <= 0 {
		width = 80
	}
	bottomH := m.input.Height() + 1 + 1
	chatAreaHeight := m.state.Height() - bottomH
	if chatAreaHeight < MinChatAreaHeight {
		chatAreaHeight = MinChatAreaHeight
	}
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(chatAreaHeight)

	content := m.buildChatContent()
	if content != m.viewportContent {
		savedOffset := m.viewport.YOffset()
		m.viewportContent = content
		m.viewport.SetContent(content)
		if m.autoScroll {
			m.viewport.GotoBottom()
		} else if !m.manualScroll {
			// Only restore savedOffset when NOT in manual scroll mode.
			// In manual scroll mode, user is browsing history, keep their position.
			m.viewport.SetYOffset(savedOffset)
		}
		// When manualScroll=true, viewport keeps its current YOffset after SetContent
	}
	// Always sync manualScrollOffset after viewport position is determined
	m.manualScrollOffset = m.viewport.YOffset()

	return m
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 1. Handle session popup
	if cmd, handled := m.sessionPopup.HandleKeyMsg(msg, m.styles); handled {
		return m, cmd
	}

	// 2. Handle subscription popup
	if cmd, handled := m.subscriptionPopup.HandleKeyMsg(msg, m.styles); handled {
		return m, cmd
	}

	// 3. Handle confirmation state
	if m.confirmState.Waiting {
		return m.handleConfirmKey(msg)
	}

	// 4. Handle command completion navigation
	if m.state.ShowCommands() {
		switch msg.Code {
		case tea.KeyUp:
			m.commandPopup.Up()
			return m, nil
		case tea.KeyDown:
			m.commandPopup.Down()
			return m, nil
		case tea.KeyTab:
			// Tab to autocomplete selected command
			if m.commandPopup.HasSelection() {
				m.input.SetValue(m.commandPopup.SelectedName() + " ")
				m.state.SetShowCommands(false)
			}
			return m, nil
		case tea.KeyEnter:
			// Enter to execute selected command
			if m.commandPopup.HasSelection() {
				m.input.SetValue(m.commandPopup.SelectedName())
			}
			m.commandPopup.Hide()
			m.state.SetShowCommands(false)
			return m.handleSubmit()
		case tea.KeyEsc:
			m.commandPopup.Hide()
			m.state.SetShowCommands(false)
			return m, nil
		}
		// Other keys fall through to normal input handling
	}

	// 5. Global shortcuts (idle only, before textarea handling)
	if !m.state.Waiting() && !m.state.ShowCommands() {
		binding := GetBinding(msg.String())
		if binding != nil {
			return ExecuteBinding(m, binding)
		}
	}

	// 6. History navigation with up/down arrows (idle only, single line input)
	if !m.state.Waiting() && !m.state.ShowCommands() && m.input.LineCount() == 1 {
		switch msg.Code {
		case tea.KeyUp:
			m.input.NavigateUp()
			return m, nil
		case tea.KeyDown:
			m.input.NavigateDown()
			return m, nil
		}
	}

	// 7. Shift+Enter for newline insertion
	if msg.Code == tea.KeyEnter && msg.Mod == tea.ModShift {
		m.input.InsertNewline()
		return m, nil
	}

	switch {
	case msg.Code == 'c' && msg.Mod == tea.ModCtrl:
		m.doCancel()
		return m, tea.Quit

	case msg.Code == tea.KeyEnter:
		return m.handleSubmit()
	}

	// Ignore input during processing
	if m.state.Waiting() {
		// Still allow scrolling during processing — fn+up/fn+down (PageUp/PageDown on macOS)
		switch msg.Code {
		case tea.KeyPgUp, tea.KeyPgDown:
			return m.handleViewportScroll(msg)
		}
		return m, nil
	}

	// Intercept scroll keys before textarea handles them
	switch msg.Code {
	case tea.KeyPgUp, tea.KeyPgDown:
		return m.handleViewportScroll(msg)
	}

	// Update input
	cmd := m.input.Update(msg)
	m.updateCommandCompletion()
	return m, cmd
}

// handleViewportScroll handles fn+up/down (PageUp/PageDown) scroll with explicit boundary checks.
func (m Model) handleViewportScroll(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.autoScroll = false
	m.manualScroll = true

	switch msg.Code {
	case tea.KeyPgUp:
		if !m.viewport.AtTop() {
			m.viewport.ScrollUp(ScrollLineDelta)
		}
	case tea.KeyPgDown:
		if !m.viewport.AtBottom() {
			m.viewport.ScrollDown(ScrollLineDelta)
		}
	}
	m.manualScrollOffset = m.viewport.YOffset()
	return m, nil
}

// handleConfirmKey handles Y/N confirmation and question input.
func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	log.Debug().Str("key", msg.String()).Bool("waiting", m.confirmState.Waiting).Msg("handleConfirmKey: called")

	// Check if this is a question type
	if m.confirmState.Request != nil && m.confirmState.Request.Type == ConfirmationQuestion {
		return m.handleQuestionKey(msg)
	}

	// Standard Y/N confirmation handling
	switch msg.Code {
	case tea.KeyLeft, tea.KeyRight, tea.KeyTab:
		// Toggle selection
		if m.confirmState.Selected == 0 {
			m.confirmState.Selected = 1
		} else {
			m.confirmState.Selected = 0
		}
		return m, nil

	case tea.KeyEnter:
		// Confirm selection
		confirmed := m.confirmState.Selected == 0 // 0 = Yes, 1 = No
		log.Debug().Bool("confirmed", confirmed).Bool("has_response_ch", m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil).Msg("handleConfirmKey: Enter pressed")
		if m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil {
			m.confirmState.Request.ResponseCh <- confirmed
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		// Reset display state and continue listening for events
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()

	case tea.KeyEsc:
		// Cancel
		if m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil {
			m.confirmState.Request.ResponseCh <- false
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()
	}

	// Handle Y/N key presses
	switch msg.Text {
	case "y", "Y":
		log.Debug().Bool("has_response_ch", m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil).Msg("handleConfirmKey: Y pressed")
		if m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil {
			m.confirmState.Request.ResponseCh <- true
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()
	case "n", "N":
		if m.confirmState.Request != nil && m.confirmState.Request.ResponseCh != nil {
			m.confirmState.Request.ResponseCh <- false
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()
	}

	return m, nil
}

// handleQuestionKey handles question-type confirmation input.
func (m Model) handleQuestionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	req := m.confirmState.Request
	if req == nil {
		return m, nil
	}

	switch req.QuestionType {
	case QuestionTypeText:
		return m.handleTextQuestionKey(msg)

	case QuestionTypeChoice:
		return m.handleChoiceQuestionKey(msg)

	case QuestionTypeMultiChoice:
		return m.handleMultiChoiceQuestionKey(msg)

	default:
		// Fallback to standard Y/N handling
		return m.handleConfirmKey(msg)
	}
}

// handleTextQuestionKey handles text input questions.
func (m Model) handleTextQuestionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	req := m.confirmState.Request

	switch msg.Code {
	case tea.KeyEnter:
		// Submit answer
		answer := m.confirmState.TextInput
		if answer == "" && req.DefaultAnswer != "" {
			answer = req.DefaultAnswer
		}
		log.Debug().Str("answer", answer).Msg("handleTextQuestionKey: Enter pressed")
		if req.QuestionRespCh != nil {
			req.QuestionRespCh <- QuestionResponse{
				Answer:    answer,
				Cancelled: false,
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.TextInput = ""
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()

	case tea.KeyEsc:
		// Cancel
		if req.QuestionRespCh != nil {
			req.QuestionRespCh <- QuestionResponse{
				Cancelled: true,
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.TextInput = ""
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()

	case tea.KeyBackspace:
		// Delete last character
		if len(m.confirmState.TextInput) > 0 {
			m.confirmState.TextInput = m.confirmState.TextInput[:len(m.confirmState.TextInput)-1]
		}
		return m, nil

	default:
		// Add character to input
		if msg.Text != "" {
			m.confirmState.TextInput += msg.Text
		}
		return m, nil
	}
}

// handleChoiceQuestionKey handles single-choice questions.
func (m Model) handleChoiceQuestionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	req := m.confirmState.Request

	switch msg.Code {
	case tea.KeyLeft, tea.KeyUp:
		// Previous option
		if m.confirmState.Selected > 0 {
			m.confirmState.Selected--
		}
		return m, nil

	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		// Next option
		if m.confirmState.Selected < len(req.Options)-1 {
			m.confirmState.Selected++
		}
		return m, nil

	case tea.KeyEnter:
		// Submit selected option
		if m.confirmState.Selected >= 0 && m.confirmState.Selected < len(req.Options) {
			answer := req.Options[m.confirmState.Selected].Value
			log.Debug().Str("answer", answer).Msg("handleChoiceQuestionKey: Enter pressed")
			if req.QuestionRespCh != nil {
				req.QuestionRespCh <- QuestionResponse{
					Answer:    answer,
					Cancelled: false,
				}
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.Selected = 0
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()

	case tea.KeyEsc:
		// Cancel
		if req.QuestionRespCh != nil {
			req.QuestionRespCh <- QuestionResponse{
				Cancelled: true,
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.Selected = 0
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()
	}

	return m, nil
}

// handleMultiChoiceQuestionKey handles multi-choice questions.
func (m Model) handleMultiChoiceQuestionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	req := m.confirmState.Request

	switch msg.Code {
	case tea.KeyLeft, tea.KeyUp:
		// Previous option
		if m.confirmState.Selected > 0 {
			m.confirmState.Selected--
		}
		return m, nil

	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		// Next option
		if m.confirmState.Selected < len(req.Options)-1 {
			m.confirmState.Selected++
		}
		return m, nil

	case tea.KeySpace:
		// Toggle current option selection
		current := m.confirmState.Selected
		found := false
		for i, idx := range m.confirmState.SelectedOptions {
			if idx == current {
				// Remove from selection
				m.confirmState.SelectedOptions = append(
					m.confirmState.SelectedOptions[:i],
					m.confirmState.SelectedOptions[i+1:]...,
				)
				found = true
				break
			}
		}
		if !found {
			// Add to selection
			m.confirmState.SelectedOptions = append(m.confirmState.SelectedOptions, current)
		}
		return m, nil

	case tea.KeyEnter:
		// Submit all selected options
		answers := make([]string, 0, len(m.confirmState.SelectedOptions))
		for _, idx := range m.confirmState.SelectedOptions {
			if idx >= 0 && idx < len(req.Options) {
				answers = append(answers, req.Options[idx].Value)
			}
		}
		log.Debug().Strs("answers", answers).Msg("handleMultiChoiceQuestionKey: Enter pressed")
		if req.QuestionRespCh != nil {
			req.QuestionRespCh <- QuestionResponse{
				Answers:   answers,
				Cancelled: false,
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.Selected = 0
		m.confirmState.SelectedOptions = nil
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()

	case tea.KeyEsc:
		// Cancel
		if req.QuestionRespCh != nil {
			req.QuestionRespCh <- QuestionResponse{
				Cancelled: true,
			}
		}
		m.confirmState.Waiting = false
		m.confirmState.Request = nil
		m.confirmState.Selected = 0
		m.confirmState.SelectedOptions = nil
		m.state.SetDisplayState(DisplayProcessing)
		return m, m.processEvents()
	}

	return m, nil
}

// handleSubmit handles the Enter key.
// Uses UI-controlled state machine for display order.
func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	log.Debug().Str("input", input).Msg("handleSubmit: called")
	if input == "" {
		return m, nil
	}

	// Hide command list before processing
	m.state.SetShowCommands(false)

	// Add to input history (before clearing)
	m.input.AddToHistory(input)

	// Clear input and disable immediately (prevent ghost text)
	m.input.DisableAndClear()

	// Handle commands
	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}

	// Reset state for new interaction
	m.state.ResetForNewInteraction()

	// Add user message to store
	userRendered := m.messages.Add(MessageTypeUser, input, nil, renderMessage, m.renderer, m.styles)
	log.Debug().Str("rendered", userRendered).Msg("handleSubmit: user message added")

	// Set UI state
	m.state.SetWaiting(true)
	m.state.SetStartTime(time.Now())
	m.state.SetDisplayState(DisplayThinking) // State machine: enter Thinking

	// Reset thinking widget (will be started by EventTypeThinking event from Engine)
	m.thinking.Reset()

	// Reset processing widget
	m.processing.Reset()

	// Reset plan widget for new interaction
	m.plan.Reset()

	// Start thinking immediately — don't wait for engine event
	_, thinkingCmd := m.thinking.StartAndRender()
	m.autoScroll = true
	m.manualScroll = false
	m.manualScrollOffset = 0
	m.lastInputValue = "" // Reset for next input

	// Start processing — user message already in store, no need to print again
	return m, tea.Batch(
		m.sendMessage(input),
		thinkingCmd,
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// printMsg is a message that triggers printing.
type printMsg struct {
	content string
}

// handleResize handles window size changes.
func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.state.SetWidth(msg.Width)
	m.state.SetHeight(msg.Height)
	m.input.SetWidth(msg.Width)
	m.input.updateHeight()
	m.renderer.UpdateWidth(msg.Width)

	// Configure viewport dimensions
	chatAreaHeight := msg.Height - (m.input.Height() + 1 + 1)
	if chatAreaHeight < MinChatAreaHeight {
		chatAreaHeight = MinChatAreaHeight
	}
	m.viewport.SetWidth(msg.Width)
	m.viewport.SetHeight(chatAreaHeight)
	m.viewportReady = true

	return m, nil
}

// updateCommandCompletion updates command completion state based on input.
func (m *Model) updateCommandCompletion() {
	value := m.input.Value()
	if strings.HasPrefix(value, "/") && !m.state.Waiting() {
		m.state.SetShowCommands(true)
		m.state.SetCommandFilter(value)
		m.state.SetCommandSelected(0)
		m.commandPopup.UpdateFilter(value, GetAvailableCommands())
	} else {
		m.commandPopup.Hide()
		m.state.SetShowCommands(false)
	}

	// Only jump to bottom when input goes from empty to non-empty.
	// This prevents interrupting user's scroll when they clear input or modify existing text.
	// Content modifications (a→ab, ab→a) should not trigger scroll jump.
	if value != "" && m.lastInputValue == "" {
		m.autoScroll = true
		m.manualScroll = false
		m.manualScrollOffset = 0
	}
	m.lastInputValue = value
}

// scrollToBottom returns a tea.Cmd that scrolls the viewport to the bottom.
func (m Model) scrollToBottom() tea.Cmd {
	return func() tea.Msg {
		return scrollBottomMsg{}
	}
}

// processEvents returns a cmd that waits for events.
func (m Model) processEvents() tea.Cmd {
	return func() tea.Msg {
		ev := <-m.eventChan
		log.Debug().Str("type", string(ev.Type)).Msg("processEvents: received event")
		return ev
	}
}

// tick returns a periodic tick command.
func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// doCancel cancels the current run context and the root context.
func (m Model) doCancel() {
	if m.currentRunCancel != nil {
		m.currentRunCancel()
	}
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}
