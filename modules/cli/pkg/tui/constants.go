package tui

import (
	"time"

	"github.com/oneliang/aura/shared/pkg/i18n"
)

// Constants for TUI configuration and limits.
const (
	// Message content limits
	MaxParamsPreview  = 80  // Maximum length for tool parameters display
	MaxContentPreview = 200 // Maximum length for tool result content display
	TruncateSuffix    = "..."

	// Display limits
	MaxVisibleCommands = 6 // Maximum commands to show in completion popup
	MaxCompletedTools  = 5 // Maximum recently completed tools to display
	MaxMessages        = 500

	// Token configuration
	DefaultTokenMax   = 8000
	DefaultTokenRatio = 0.7

	// Skill caching
	SkillCacheTTL = 5 * time.Minute // Cache TTL for loaded skills

	// Tool block
	ToolBlockMaxContentWidth = 400 // Maximum width for tool block content

	// Tool display strings (non-translatable)
	ToolErrorIcon = "  ✗ "

	// Input configuration
	InputCharLimit = 4096
	InputWidth     = 80
	InputHeight    = 1
	InputMaxHeight = 3 // Maximum lines for textarea (dynamic height)

	// Popup configuration
	PopupPageSize      = 6
	PopupDefaultWidth  = 40
	PopupDefaultHeight = 1
)

// Layout constants for fullscreen mode.
const (
	// BottomAreaLineCount is the number of terminal lines reserved for the bottom bar:
	// separator(1) + input(1) + separator(1) + status(1) + IME space(1) = 5
	// Note: With InputWidget dynamic height, actual height = input.Height() + 1(status) + 1(IME)
	BottomAreaLineCount = 5

	// MinChatAreaHeight is the minimum height for the chat/viewport area.
	MinChatAreaHeight = 4

	// MinTerminalHeight is the minimum terminal height below which we refuse to render.
	MinTerminalHeight = 8
)

// Viewport constants.
const (
	ViewportMouseWheelDelta = 3 // lines per mouse wheel scroll
)

// Widget animation intervals.
var (
	ThinkingWidgetTickInterval   = 100 * time.Millisecond
	ProcessingWidgetTickInterval = 150 * time.Millisecond
)

// Overlay / Popup constants.
const (
	OverlayPadding    = 2  // minimum padding from terminal edges
	OverlayMinWidth   = 40 // minimum popup width
	OverlayMinHeight  = 6  // minimum popup height
	OverlayBorderChar = "─"
)

// Scroll behavior.
const (
	AutoScrollDefault = true // default to auto-scroll to bottom
	ScrollLineDelta   = 5    // lines per fn+up/down scroll
)

// Streaming indicator.
const (
	StreamingCursorChar = "▌" // cursor indicator for streaming output
)

// Task widget.
const (
	TaskWidgetBorderWidth = 60 // width of the task widget border line
)

// UI color palette (ANSI 256-color codes).
const (
	// Semantic colors — messages
	ColorUserMessage = "86"  // cyan
	ColorAuraMessage = "174" // warm pink
	ColorThinking    = "172" // medium orange
	ColorAction      = "99"  // purple
	ColorResult      = "70"  // green
	ColorError       = "196" // red
	ColorProcessing  = "214" // orange

	// Structural colors — UI elements
	ColorHelp        = "241" // dim gray
	ColorTimestamp   = "241"
	ColorSeparator   = "238" // dark gray
	ColorCommand     = "114" // bright green
	ColorCommandFg   = "0"   // black (for selected command text)
	ColorCommandBg   = "114"
	ColorCommandItem = "252" // light white
	ColorCommandDesc = "241"
	ColorToolDone    = "70" // green

	// Tool block colors
	ColorToolBlockHeader = "114"
	ColorToolBlockIn     = "114"
	ColorToolBlockInFg   = "252" // light white
	ColorToolBlockInBg   = "236" // dark background
	ColorToolBlockOut    = "214" // orange
	ColorToolBlockBorder = "238" // dark gray border

	// Task widget colors
	ColorTaskWidgetTitle  = "114"
	ColorTaskWidgetBorder = "238"
	ColorTaskInProgress   = "226" // bright yellow for in_progress highlight

	// Plan widget colors
	ColorPlanWidgetTitle  = "141" // purple
	ColorPlanWidgetBorder = "238"
)

// Display strings - initialized from i18n in init()
var (
	DefaultUserName          string
	AuraDisplayName          string
	IdleStatusText           string
	ThinkingStatusText       string
	ProcessingStatusText     string
	ThinkingIcon             string // Icon prefix for thinking content
	ConfirmEnterHint         string
	ConfirmEscHint           string
	ConfirmYesLabel          string
	ConfirmNoLabel           string
	SubscriptionAddHint      string
	SessionSelectHint        string
	ViewLoadingText          string
	ViewTooSmallText         string
	InputPlaceholder         string
	InputWaitingPlaceholder  string
	CommandNoMatching        string
	CommandCountFormat       string
	SessionPopupHelp         string
	SubscriptionAddHelp      string
)

func init() {
	DefaultUserName = i18n.T("tui.default_user")
	AuraDisplayName = "[Aura]:"
	IdleStatusText = "/help for commands • Ctrl+C to exit"
	ThinkingStatusText = i18n.T("tui.status.thinking")
	ProcessingStatusText = i18n.T("tui.status.processing")
	ThinkingIcon = "💭 "
	ConfirmEnterHint = i18n.T("tui.confirm.enter")
	ConfirmEscHint = i18n.T("tui.confirm.cancel")
	ConfirmYesLabel = "[" + i18n.T("tui.confirm.yes") + "]"
	ConfirmNoLabel = "[" + i18n.T("tui.confirm.no") + "]"
	SubscriptionAddHint = i18n.T("tui.navigate.subscription")
	SessionSelectHint = i18n.T("tui.navigate.session")
	ViewLoadingText = i18n.T("tui.status.loading")
	ViewTooSmallText = i18n.T("tui.status.terminal_small")
	InputPlaceholder = i18n.T("tui.placeholder.input")
	InputWaitingPlaceholder = i18n.T("tui.placeholder.waiting")
	CommandNoMatching = i18n.T("tui.command.no_match")
	CommandCountFormat = "  %d/%d " + i18n.T("tui.default_user") // commands pattern
	SessionPopupHelp = i18n.T("tui.navigate.session")
	SubscriptionAddHelp = i18n.T("tui.navigate.subscription")
}