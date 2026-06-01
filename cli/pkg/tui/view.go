package tui

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/oneliang/aura/shared/pkg/utils"
)

// formatToolParams formats JSON tool parameters into a human-readable string.
// Instead of showing raw JSON like {"action":"create","content":"foo"},
// it extracts the most meaningful fields and returns a concise display string.
func formatToolParams(rawParams string) string {
	var params map[string]any
	if err := json.Unmarshal([]byte(rawParams), &params); err != nil {
		return utils.Truncate(rawParams, MaxParamsPreview)
	}

	// Priority-ordered field extraction for common tool patterns
	actionKeys := []string{"action", "command", "operation", "method"}
	contentKeys := []string{"content", "query", "path", "text", "message", "file", "expression", "url", "notes", "status", "target", "source", "name", "description"}

	var action, content string
	for _, k := range actionKeys {
		if v, ok := params[k].(string); ok && v != "" {
			action = v
			break
		}
	}
	for _, k := range contentKeys {
		if v, ok := params[k].(string); ok && v != "" {
			content = v
			break
		}
	}

	var result string
	if action != "" && content != "" {
		result = fmt.Sprintf("%s: %s", action, content)
	} else if content != "" {
		result = content
	} else if action != "" {
		// No content field, try to find any remaining meaningful string value
		for _, v := range params {
			if s, ok := v.(string); ok && s != "" && s != action {
				result = fmt.Sprintf("%s: %s", action, s)
				break
			}
		}
		if result == "" {
			result = action
		}
	} else {
		// Fallback: pick first meaningful string value
		for k, v := range params {
			if s, ok := v.(string); ok && s != "" {
				result = fmt.Sprintf("%s=%s", k, s)
				break
			}
		}
	}

	if result == "" {
		// Last resort: show truncated raw JSON
		result = utils.Truncate(rawParams, MaxParamsPreview)
	}

	return utils.Truncate(result, MaxParamsPreview)
}

// formatDuration formats a duration as "X.Xs" (seconds with 1 decimal place).
func formatDuration(d time.Duration) string {
	seconds := math.Round(d.Seconds()*10) / 10
	return fmt.Sprintf("%.1fs", seconds)
}

// renderToolHeader renders the tool block header with a green dot and bold tool name.
func renderToolHeader(toolName string, styles UIStyles) string {
	return "  " + styles.ToolBlockHeader.Render("● "+toolName)
}

// renderToolStartBlock renders the tool execution start block: header + IN line with subtle background.
func renderToolStartBlock(toolName string, params string, styles UIStyles) string {
	var b strings.Builder

	// Header
	b.WriteString(renderToolHeader(toolName, styles))
	b.WriteByte('\n')

	// IN line with subtle background
	b.WriteString("  ")
	inLabel := styles.ToolBlockIn.Render("IN ")
	if params != "" {
		formatted := formatToolParams(params)
		b.WriteString(inLabel + styles.ToolBlockInBg.Render(" "+formatted))
	} else {
		b.WriteString(inLabel)
	}

	return b.String()
}

// renderToolEndBlock renders the tool execution end block: OUT line (no background) + Done.
func renderToolEndBlock(result string, duration time.Duration, styles UIStyles) string {
	var b strings.Builder

	// OUT line — no background, just label + plain content
	b.WriteString("  ")
	outContent := utils.Truncate(result, ToolBlockMaxContentWidth)
	if outContent == "" {
		outContent = "(no output)"
	}
	outLabel := styles.ToolBlockOut.Render("OUT ")
	b.WriteString(outLabel + outContent)
	b.WriteByte('\n')

	// Done line
	doneText := fmt.Sprintf("     ✓ Done in %s", formatDuration(duration))
	b.WriteString(styles.ToolBlockDone.Render(doneText))

	return b.String()
}

// renderToolBlockComplete renders a complete tool block by combining start and end blocks.
// Reuses existing renderToolStartBlock and renderToolEndBlock to avoid code duplication.
// Applies left border for visual separation.
func renderToolBlockComplete(toolName string, params string, result string, duration time.Duration, styles UIStyles) string {
	var b strings.Builder

	// Reuse start block (header + IN line)
	b.WriteString(renderToolStartBlock(toolName, params, styles))
	b.WriteByte('\n')

	// Reuse end block (OUT line + Done)
	b.WriteString(renderToolEndBlock(result, duration, styles))

	// Apply left border
	return styles.ToolBlockBorder.Render(b.String())
}

// renderMessage renders a message based on its type.
// This is the single source of truth for message rendering in the TUI.
// View 层核心函数：纯数据 -> 渲染字符串
func renderMessage(msg *Message, renderer *MarkdownRenderer, styles UIStyles, userName string) string {
	timestamp := utils.FormatTimestamp(msg.Timestamp)
	ts := styles.Timestamp.Render("[" + timestamp + "]")

	switch msg.Type {
	case MessageTypeUser:
		name := userName
		if name == "" {
			name = DefaultUserName
		}
		return "\r\033[K" + styles.UserMessage.Render("["+name+"]:") + " " + msg.Content + " " + ts

	case MessageTypeAssistant:
		rendered := msg.Content
		if renderer != nil {
			rendered = renderer.Render(msg.Content)
		}
		// Trim whitespace/newlines from glamour output
		rendered = strings.Trim(rendered, "\n\t ")
		// Format: [Aura]: 消息内容 [时间]
		return styles.AuraMessage.Render(AuraDisplayName) + " " + rendered + " " + ts

	case MessageTypeThinking:
		rendered := msg.Content
		if renderer != nil {
			rendered = renderer.Render(msg.Content)
		}
		rendered = strings.Trim(rendered, "\n\t ")
		// Add thinking icon prefix for visual identification
		return ThinkingIcon + styles.Thinking.Render(rendered) + " " + ts

	case MessageTypeToolStart:
		params := ""
		if msg.Extra != nil {
			if p, ok := msg.Extra["params"].(string); ok {
				params = p
			}
		}
		return renderToolStartBlock(msg.Content, params, styles)

	case MessageTypeToolEnd:
		duration := time.Duration(0)
		if msg.Extra != nil {
			if d, ok := msg.Extra["duration"].(time.Duration); ok {
				duration = d
			}
		}
		return renderToolEndBlock(msg.Content, duration, styles)

	case MessageTypeError:
		return styles.Error.Render(ToolErrorIcon + msg.Content)

	case MessageTypeSystem:
		return msg.Content

	default:
		return msg.Content
	}
}

// View implements tea.Model.
// Fullscreen layout: viewport (chat area) + bottom bar (input + status) + overlays.
func (m Model) View() tea.View {
	if !m.viewportReady {
		return tea.NewView(ViewLoadingText)
	}

	width := m.state.Width()
	height := m.state.Height()
	if width <= 0 {
		width = 80
	}
	if height < MinTerminalHeight {
		return tea.NewView(m.styles.Error.Render("  " + ViewTooSmallText))
	}

	// Calculate dynamic bottom area height: input(including separators) + status + IME
	bottomH := m.input.Height() + 1 + 1

	// Calculate chat area height and ensure viewport is sized correctly
	chatAreaHeight := height - bottomH
	if chatAreaHeight < MinChatAreaHeight {
		chatAreaHeight = MinChatAreaHeight
	}
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(chatAreaHeight)

	// Build chat content and update viewport
	content := m.buildChatContent()
	if content != m.viewportContent {
		savedOffset := m.viewport.YOffset()
		m.viewportContent = content
		m.viewport.SetContent(content)
		if !m.autoScroll && !m.manualScroll {
			// Only restore savedOffset when NOT in manual scroll mode.
			// In manual scroll mode, user is browsing history, keep their position.
			m.viewport.SetYOffset(savedOffset)
		}
	}
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
	// Always sync manualScrollOffset after viewport position is determined
	m.manualScrollOffset = m.viewport.YOffset()

	// Build the screen: chat area + bottom bar
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteByte('\n')
	b.WriteString(m.buildBottomArea(width))

	result := b.String()

	// Overlay popups
	if m.sessionPopup.IsShowing() {
		result = renderOverlay(result, m.sessionPopup.Render(m.styles, width), width, height, m.styles)
	}
	if m.subscriptionPopup.IsShowing() {
		result = renderOverlay(result, m.subscriptionPopup.Render(m.styles, width), width, height, m.styles)
	}
	if m.commandPopup.IsShowing() {
		popupContent := m.commandPopup.Render()
		popupH := m.commandPopup.Height()
		popupTop := height - bottomH - popupH - 1
		if popupTop < 0 {
			popupTop = 0
		}
		result = renderOverlayAt(result, popupContent, popupTop, width, m.styles)
	}

	v := tea.NewView(result)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// buildChatContent concatenates all messages and inline widgets for the viewport.
func (m Model) buildChatContent() string {
	var b strings.Builder

	// Fixed header — not cleared by /clear
	if m.greeting != "" {
		b.WriteString(m.greeting)
		b.WriteByte('\n')
	}
	if m.sessionInfo != "" {
		b.WriteString(m.sessionInfo)
		b.WriteByte('\n')
	}

	msgs := m.messages.GetMessages()
	currentWidth := m.state.Width()

	for _, msg := range msgs {
		if msg.Rendered != "" {
			// Check if re-rendering is needed due to width change
			// Skip re-rendering for merged tool blocks (preserve complete IN/OUT)
			if m.renderer != nil && msg.RenderedWidth > 0 {
				isMerged := msg.Extra != nil && msg.Extra["merged"] == true
				if !isMerged {
					diff := currentWidth - msg.RenderedWidth
					if diff < 0 {
						diff = -diff
					}
					if diff > 10 {
						// Width changed significantly — re-render
						msg.Rendered = renderMessage(msg, m.renderer, m.styles, m.config.UserName)
						msg.RenderedWidth = currentWidth
					}
				}
			}
			b.WriteString(strings.TrimRight(msg.Rendered, "\n"))
			b.WriteByte('\n')
		} else if msg.Type == MessageTypeAssistant && msg.Content != "" {
			// Streaming assistant message
			b.WriteString(m.styles.AuraMessage.Render(AuraDisplayName))
			b.WriteString(" ")
			b.WriteString(msg.Content)
			// Only show cursor if message is not complete (streaming in progress)
			if !msg.Complete {
				b.WriteString(StreamingCursorChar)
			}
			b.WriteByte('\n')
		} else if msg.Type == MessageTypeThinking && msg.Content != "" {
			// Streaming thinking message
			b.WriteString(m.styles.Thinking.Render(msg.Content))
			// Only show cursor if message is not complete (streaming in progress)
			if !msg.Complete {
				b.WriteString(StreamingCursorChar)
			}
			b.WriteByte('\n')
		}
	}

	// Inline widgets when active (current activity, at bottom)
	if m.thinking != nil && m.thinking.IsActive() {
		b.WriteString(m.thinking.Rendered())
		b.WriteByte('\n')
	}
	if m.processing != nil && m.processing.IsActive() {
		b.WriteString(m.processing.Rendered())
		b.WriteByte('\n')
	}
	// Plan and Task widgets shown independently
	if m.plan != nil && m.plan.RenderStyled() != "" {
		b.WriteString(m.plan.RenderStyled())
		b.WriteByte('\n')
	}
	if m.tasks != nil {
		rendered := m.tasks.RenderStyled()
		if rendered != "" {
			b.WriteString(rendered)
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// buildBottomArea renders the fixed bottom bar: input widget (with borders) + status + IME space.
func (m Model) buildBottomArea(width int) string {
	var b strings.Builder

	// Pending messages queue (above input)
	if len(m.pendingMessages) > 0 {
		b.WriteString(m.renderPendingMessages(width))
		b.WriteByte('\n')
	}

	// Input widget (includes top/bottom separators)
	b.WriteString(m.input.Render())
	b.WriteByte('\n')

	// Status line
	b.WriteString(m.statusBar.Render(width))

	// Empty line for IME candidate space (Chinese input)
	b.WriteByte('\n')

	return b.String()
}

// renderPendingMessages renders the pending messages queue.
// Shows original user input without extra formatting.
func (m Model) renderPendingMessages(width int) string {
	var b strings.Builder
	b.WriteString(m.styles.Help.Render("⏳ Pending messages:"))
	for _, msg := range m.pendingMessages {
		b.WriteByte('\n')
		b.WriteString("  ")
		b.WriteString(utils.Truncate(msg.Content, 50))
	}
	return b.String()
}
