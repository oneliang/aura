package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// handleSessionCreate creates a new session.
func (m Model) handleSessionCreate(input string) (tea.Model, tea.Cmd) {
	// Parse name from command
	parts := strings.Fields(input)
	name := ""
	if len(parts) > 2 {
		name = parts[2]
	}
	if name == "" || strings.HasPrefix(name, "/") {
		name = fmt.Sprintf(DefaultSessionNameFormat, time.Now().Format("20060102150405"))
	}

	if m.commandProvider != nil {
		result, err := m.commandProvider.Execute(m.ctx, commands.CmdNameSessionCreate, map[string]any{"name": name})
		if err != nil {
			m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to create session: %v", err)))
			return m, m.scrollToBottom()
		}
		m.messages.AddRaw(m.styles.Help.Render(result))
		return m, m.scrollToBottom()
	}

	// Fallback to direct session manager
	if m.sessionMgr == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Session manager not initialized"))
		return m, m.scrollToBottom()
	}

	session, err := m.sessionMgr.CreateSession(name, "")
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to create session: %v", err)))
		return m, m.scrollToBottom()
	}

	m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  Session created: %s (%s)", session.Name, session.ID)))
	return m, m.scrollToBottom()
}

// showSessionShow shows current session details.
func (m Model) showSessionShow() string {
	sessionID := ""
	if m.currentSession != nil {
		sessionID = m.currentSession.ID()
	}
	return m.executeCommand(commands.CmdNameSessionShow, map[string]any{"id": sessionID})
}

// showSessionDelete deletes current session.
func (m Model) showSessionDelete() string {
	sessionID := ""
	if m.currentSession != nil {
		sessionID = m.currentSession.ID()
	}
	if sessionID == "" {
		return m.messages.AddRaw(m.styles.Help.Render("  No active session to delete."))
	}
	return m.executeCommand(commands.CmdNameSessionDelete, map[string]any{"id": sessionID})
}

// showSessionExport exports session to file.
func (m Model) showSessionExport() string {
	return m.messages.AddRaw(m.styles.Help.Render("  Use CLI: aura session export <id>"))
}

// handleSubscriptionShow shows subscriptions for current session.
func (m Model) handleSubscriptionShow() (tea.Model, tea.Cmd) {
	if m.sessionMgr == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Session manager not available."))
		return m, m.scrollToBottom()
	}
	if m.currentSession == nil {
		m.messages.AddRaw(m.styles.Help.Render("  No active session. Use /sessions to select a session first."))
		return m, m.scrollToBottom()
	}
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Error getting subscriptions: %v", err)))
		return m, m.scrollToBottom()
	}
	m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	m.subscriptionPopup.Show()
	return m, nil
}

// handleSubscriptionAdd starts the subscription add flow.
func (m *Model) handleSubscriptionAdd() {
	if m.sessionMgr == nil || m.currentSession == nil {
		return
	}
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err == nil {
		m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	}
	m.subscriptionPopup.Show()
	m.subscriptionPopup.StartAddInput()
}

// handleSubscriptionDelete deletes subscription (shows popup for selection).
func (m *Model) handleSubscriptionDelete() string {
	if m.sessionMgr == nil {
		return m.messages.AddRaw(m.styles.Error.Render("  Session manager not available."))
	}
	if m.currentSession == nil {
		return m.messages.AddRaw(m.styles.Help.Render("  No active session. Use /sessions to select a session first."))
	}
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err != nil {
		return m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Error getting subscriptions: %v", err)))
	}
	if len(subs) == 0 {
		return m.messages.AddRaw(m.styles.Help.Render("  No subscriptions to delete."))
	}
	m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	m.subscriptionPopup.Show()
	return m.messages.AddRaw(m.styles.Help.Render("  Select a subscription and press 'd' to delete."))
}

// handleSessions shows session list via popup.
func (m *Model) handleSessions() {
	if m.sessionMgr == nil {
		return
	}
	items, err := m.sessionMgr.ListSessions()
	if err != nil || len(items) == 0 {
		return
	}
	m.sessionItems = sessionInfosToItems(items)
	m.sessionPopup.UpdateItems(m.sessionItems)
	m.sessionPopup.Show()
}

// executeCommand executes a command via CommandProvider and returns rendered string.
// This is the unified method for all command output.
func (m Model) executeCommand(cmdName string, params map[string]any) string {
	if m.commandProvider != nil {
		result, err := m.commandProvider.Execute(m.ctx, cmdName, params)
		if err != nil {
			return m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("Error: %v", err)))
		}
		return m.messages.AddRaw(m.styles.Help.Render(result))
	}
	return m.messages.AddRaw(m.styles.Help.Render("Command not available."))
}

// showConfig shows config info.
func (m Model) showConfig() string {
	return m.executeCommand(commands.CmdNameConfigShow, nil)
}

// showModel shows model info.
func (m Model) showModel() string {
	return m.executeCommand(commands.CmdNameModel, nil)
}

// showProfile shows profile info.
func (m Model) showProfile() string {
	return m.executeCommand(commands.CmdNameProfileShow, nil)
}

// showMemory shows memory usage.
func (m Model) showMemory() string {
	return m.executeCommand(commands.CmdNameMemory, nil)
}

// showSkills shows loaded skills.
func (m Model) showSkills() string {
	return m.executeCommand(commands.CmdNameSkills, nil)
}

// showTools shows available tools.
func (m Model) showTools() string {
	return m.executeCommand(commands.CmdNameTools, nil)
}

// showHistory shows tool execution history.
func (m Model) showHistory() string {
	return m.executeCommand(commands.CmdNameHistory, nil)
}

// showCompact compacts conversation memory.
func (m Model) showCompact() string {
	return m.executeCommand(commands.CmdNameCompact, nil)
}

// showRole shows current session role.
func (m Model) showRole() string {
	return m.executeCommand(commands.CmdNameRole, nil)
}

// showKnowledge shows knowledge base info.
func (m Model) showKnowledge() string {
	return m.executeCommand(commands.CmdNameKnowledgeSearch, nil)
}

// handleSelectSession handles session selection.
func (m Model) handleSelectSession(msg selectSessionMsg) (tea.Model, tea.Cmd) {
	if msg.item.id == "" {
		return m, nil
	}

	// Idempotency: already viewing this session
	if m.currentSession != nil && m.currentSession.ID() == msg.item.id {
		m.sessionPopup.Hide()
		m.messages.AddRaw(m.styles.Help.Render(i18n.T("tui.already_viewing_session", msg.item.name)))
		return m, tea.Sequence(m.input.EnableAndFocus(), m.scrollToBottom())
	}

	// Clear current messages
	m.messages.Clear()
	m.tasks.Reset()

	// Set new current session
	m.currentSession = &msg.item
	m.sessionInfo = fmt.Sprintf("  %s", i18n.T("tui.session_loaded", fmt.Sprintf("%s (%s)", m.currentSession.name, m.currentSession.id)))

	// Load session history into viewport
	m.loadSessionHistory()

	// Build session switch message
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s", i18n.T("tui.session_switched", msg.item.name)))
	for _, msg := range m.messages.GetMessages() {
		if msg.Rendered != "" {
			b.WriteString(msg.Rendered)
			b.WriteByte('\n')
		}
	}
	m.messages.AddRaw(m.styles.Help.Render(b.String()))

	// Update session popup items
	if m.sessionMgr != nil {
		items, err := m.sessionMgr.ListSessions()
		if err == nil {
			m.sessionItems = sessionInfosToItems(items)
			m.sessionPopup.UpdateItems(m.sessionItems)
		}
	}

	return m, tea.Sequence(m.input.EnableAndFocus(), m.scrollToBottom())
}

// handleCreateSession handles create session request.
func (m Model) handleCreateSession(msg createSessionMsg) (tea.Model, tea.Cmd) {
	if m.sessionMgr == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Session manager not initialized"))
		return m, m.scrollToBottom()
	}

	name := msg.name
	if name == "" {
		name = fmt.Sprintf(DefaultSessionNameFormat, time.Now().Format("20060102150405"))
	}

	session, err := m.sessionMgr.CreateSession(name, msg.role)
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to create session: %v", err)))
		return m, m.scrollToBottom()
	}

	// Convert to TUI SessionItem
	tuiSession := sessionInfoToItem(*session)
	m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  Session created: %s (%s)", tuiSession.name, tuiSession.id)))

	// Update session list
	items, err := m.sessionMgr.ListSessions()
	if err == nil {
		m.sessionItems = sessionInfosToItems(items)
		m.sessionPopup.UpdateItems(m.sessionItems)
	}

	// Switch to new session
	m.currentSession = &tuiSession
	m.sessionInfo = fmt.Sprintf("  Session created: %s (%s)", tuiSession.name, tuiSession.id)

	return m, m.scrollToBottom()
}

// handleAddSubscription handles adding a subscription.
func (m Model) handleAddSubscription(msg addSubscriptionMsg) (tea.Model, tea.Cmd) {
	if m.sessionMgr == nil || m.currentSession == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Cannot add subscription: no active session"))
		return m, m.scrollToBottom()
	}

	trigger := msg.trigger
	source := msg.source
	if source == "" {
		source = "*"
	}

	if trigger == "" {
		m.messages.AddRaw(m.styles.Error.Render("  Trigger keyword is required"))
		return m, m.scrollToBottom()
	}

	err := m.sessionMgr.AddSubscription(m.currentSession.ID(), trigger, source)
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to add subscription: %v", err)))
		return m, m.scrollToBottom()
	}

	m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  Subscription added: trigger=%q, source=%s", trigger, source)))

	// Update subscription list
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err == nil {
		m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	}

	return m, m.scrollToBottom()
}

// handleDeleteSubscription handles deleting a subscription.
func (m Model) handleDeleteSubscription(msg deleteSubscriptionMsg) (tea.Model, tea.Cmd) {
	if m.sessionMgr == nil || m.currentSession == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Cannot delete subscription: no active session"))
		return m, m.scrollToBottom()
	}

	err := m.sessionMgr.RemoveSubscription(m.currentSession.ID(), msg.item.ID())
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to delete subscription: %v", err)))
		return m, m.scrollToBottom()
	}

	m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  Subscription deleted: %q", msg.item.Trigger())))

	// Update subscription list
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err == nil {
		m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	}

	return m, m.scrollToBottom()
}

// handleToggleSubscription handles toggling subscription status.
func (m Model) handleToggleSubscription(msg toggleSubscriptionMsg) (tea.Model, tea.Cmd) {
	if m.sessionMgr == nil || m.currentSession == nil {
		m.messages.AddRaw(m.styles.Error.Render("  Cannot toggle subscription: no active session"))
		return m, m.scrollToBottom()
	}

	err := m.sessionMgr.ToggleSubscriptionStatus(m.currentSession.ID(), msg.item.ID())
	if err != nil {
		m.messages.AddRaw(m.styles.Error.Render(fmt.Sprintf("  Failed to toggle subscription: %v", err)))
		return m, m.scrollToBottom()
	}

	newStatus := "active"
	if msg.item.IsActive() {
		newStatus = "inactive"
	}
	m.messages.AddRaw(m.styles.Help.Render(fmt.Sprintf("  Subscription toggled: %q is now %s", msg.item.Trigger(), newStatus)))

	// Update subscription list
	subs, err := m.sessionMgr.GetSubscriptions(m.currentSession.ID())
	if err == nil {
		m.subscriptionPopup.UpdateItems(subscriptionInfosToItems(subs))
	}

	return m, m.scrollToBottom()
}
