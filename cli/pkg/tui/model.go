package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/i18n"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/user"
	"github.com/oneliang/aura/shared/pkg/version"
)

// Model is the main Bubble Tea model.
type Model struct {
	state    *State
	messages *MessageStore
	input    *InputManager
	styles   UIStyles
	renderer *MarkdownRenderer

	// Widgets - independent UI components
	thinking   *ThinkingWidget
	processing *ProcessingWidget
	tasks      *TaskWidget
	plan       *PlanWidget

	runFn      RunFunc
	ctx        context.Context
	cancelFunc context.CancelFunc
	eventChan  chan ChatEvent

	// Run state
	currentRunCtx    context.Context
	currentRunCancel context.CancelFunc
	currentRequestID string // Tracks current request ID for event grouping

	// Session
	sessionMgr      *sdk.SessionManager
	summarizer      *sdk.Summarizer
	modelProvider   *ModelProvider
	commandProvider commands.Command // Command execution provider

	// Popup states
	sessionPopup      SessionPopupState
	subscriptionPopup SubscriptionPopupState
	currentSession    *SessionItem
	sessionItems      []SessionItem

	// Widget states
	statusBar    *StatusBarWidget
	commandPopup *CommandPopupWidget

	// Confirmation state
	confirmState ConfirmState

	// Rollback state for plan mode
	rollbackSnapshotID string

	// Init state for /init command
	initPending    bool   // True when /init is running
	initAuraMdPath string // Path to save AURA.md
	initContent    string // Accumulated response content for AURA.md

	config Config

	// Testing flags
	skipSkillLoading bool // Set to true in tests to skip skill file loading

	// Skill caching
	cachedSkills     []CommandInfo
	cachedSkillsTime time.Time
	cachedSkillDefs  []sdk.SkillInfo // Cached skill definitions with Body

	// Token usage cache
	cachedTokenUsage   int
	cachedTokenDisplay string

	// User ID for multi-user isolation (empty = legacy mode)
	userID string

	// MCP server manager
	mcpManager *sdk.MCPManager

	// Viewport for fullscreen chat area
	viewport           viewport.Model
	viewportReady      bool
	viewportContent    string // Tracks last viewport content to avoid redundant SetContent calls
	autoScroll         bool
	manualScroll       bool // Whether manual scrolling has been initiated
	manualScrollOffset int
	lastInputValue     string // Tracks last input value to detect real content changes

	// Greeting and session info — fixed header above chat area, not cleared by /clear
	greeting    string
	sessionInfo string

	// Queue input state
	pendingMessages []PendingMessage // 待发送的排队消息
}

// PendingMessage represents a message waiting to be sent.
type PendingMessage struct {
	Content   string
	Timestamp time.Time
}

// Ensure Model implements tea.Model.
var _ tea.Model = Model{}

// Internal message types for Bubble Tea.
type (
	tickMsg         struct{}
	scrollBottomMsg struct{}
)

// New creates a new TUI model.
func New(ctx context.Context, runFn RunFunc, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager) Model {
	// Initialize i18n constants (must be called after i18n.Init)
	InitI18nConstants()

	ctx, cancel := context.WithCancel(ctx)

	styles := DefaultStyles()
	renderer := NewMarkdownRenderer(80)
	messages := NewMessageStore(500, config.UserName)
	input := NewInputManager(styles)

	state := NewState()
	state.SetDebugMode(config.DebugMode)
	state.SetTokenMax(config.TokenMax)
	state.SetShowTokens(config.ShowTokens)

	m := Model{
		state:            state,
		messages:         messages,
		input:            input,
		styles:           styles,
		renderer:         renderer,
		thinking:         NewThinkingWidget(),
		processing:       NewProcessingWidget(),
		tasks:            NewTaskWidget(styles),
		plan:             NewPlanWidget(styles),
		runFn:            runFn,
		ctx:               ctx,
		cancelFunc:        cancel,
		eventChan:         make(chan ChatEvent, 100),
		config:            config,
		sessionMgr:        sessionMgr,
		summarizer:        summarizer,
		modelProvider:     modelProvider,
		commandProvider:   commandProvider,
		sessionPopup:      NewSessionPopup(),
		subscriptionPopup: NewSubscriptionPopup(),
		userID:            user.GetDefaultUserID(), // Load current user ID from users.yaml
		mcpManager:        mcpManager,
		viewport:          viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
		autoScroll:        AutoScrollDefault,
		viewportReady:     false,
	}

	m.statusBar = NewStatusBarWidget(styles, state, &m)
	m.commandPopup = &CommandPopupWidget{styles: styles}

	// Set greeting (fixed header, not cleared by /clear)
	versionInfo := styles.Help.Render(version.FullVersion())
	m.greeting = fmt.Sprintf("%s\n%s",
		lipglossv2.NewStyle().Bold(true).Foreground(lipglossv2.Color(ColorCommand)).Render("✨ Aura — Personal AI Assistant"),
		versionInfo)

	if modelProvider != nil {
		modelProvider.Set(&m)
	}

	// Load sessions if session manager is available
	if sessionMgr != nil {
		items, err := sessionMgr.ListSessions()
		if err == nil {
			m.sessionItems = sessionInfosToItems(items)

			// Set current session based on SessionID
			if config.SessionID != "" {
				for i := range items {
					if items[i].ID == config.SessionID {
						tuiItem := sessionInfoToItem(items[i])
						m.currentSession = &tuiItem
						break
					}
				}
			}
		}

		// Load session history if current session exists
		if m.currentSession != nil {
			m.loadSessionHistory()
			m.sessionInfo = fmt.Sprintf("  %s", i18n.T("tui.session_loaded", fmt.Sprintf("%s (%s)", m.currentSession.name, m.currentSession.id)))
		} else {
			m.sessionInfo = ""
		}
	}

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	// Start event processing
	return tea.Batch(m.input.Focus(), m.tick(), m.eventLoop())
}

// loadSessionHistory loads the current session's message history.
// Returns true if history was loaded.
func (m *Model) loadSessionHistory() bool {
	if m.sessionMgr == nil || m.currentSession == nil {
		log.Debug().Msg("loadSessionHistory: sessionMgr or currentSession is nil")
		return false
	}

	store := m.sessionMgr.GetStore()
	if store == nil {
		log.Debug().Msg("loadSessionHistory: store is nil")
		return false
	}

	// Get messages from session
	ctx := context.Background()
	msgs, err := store.GetMessages(ctx, m.currentSession.ID(), 0) // Load all messages
	if err != nil {
		log.Debug().Err(err).Str("sessionID", m.currentSession.ID()).Msg("loadSessionHistory: error loading messages")
		return false
	}
	if len(msgs) == 0 {
		log.Debug().Str("sessionID", m.currentSession.ID()).Msg("loadSessionHistory: no messages")
		return false
	}

	log.Debug().Int("count", len(msgs)).Str("sessionID", m.currentSession.ID()).Msg("loadSessionHistory: loaded messages")

	// Filter user messages for input history (newest first)
	var userInputs []string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			// Extract text from ContentBlocks
			for _, block := range msgs[i].ContentBlocks {
				if tb, ok := block.(sharedmemory.TextBlock); ok {
					userInputs = append(userInputs, tb.Text)
					break
				}
			}
		}
	}
	m.input.SetUserMessageHistory(userInputs)

	// Add messages to store for viewport rendering
	for _, msg := range msgs {
		// Skip observation messages - they're tool results, not user input
		if msg.Type == sharedmemory.MessageTypeObservation {
			continue
		}
		timestamp := time.UnixMilli(msg.Timestamp)
		// Extract text from ContentBlocks
		var textContent string
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(sharedmemory.TextBlock); ok {
				textContent = tb.Text
				break
			}
		}
		// Skip messages without actual text content (e.g., tool_use, pure thinking)
		if textContent == "" {
			continue
		}
		switch msg.Role {
		case "user":
			m.messages.AddWithTimestamp(MessageTypeUser, textContent, nil, timestamp, renderMessage, m.renderer, m.styles)
		case "assistant":
			m.messages.AddWithTimestamp(MessageTypeAssistant, textContent, nil, timestamp, renderMessage, m.renderer, m.styles)
		}
	}

	// Update token count to reflect loaded messages
	m.updateTokenUsage()

	return true
}

// sendMessage sends a message to the agent.
func (m Model) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		m.currentRunCtx, m.currentRunCancel = context.WithCancel(m.ctx)

		log.Debug().Str("input", input).Msg("sendMessage: starting")

		// Run Process() in a background goroutine so the Bubble Tea main loop
		// can continue processing user input (e.g., confirmation responses).
		go func() {
			events, err := m.runFn(m.currentRunCtx, input)
			if err != nil {
				log.Debug().Err(err).Msg("sendMessage: error")
				select {
				case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
				default:
				}
				// Send Done event to unlock input
				select {
				case m.eventChan <- ChatEvent{Type: EventTypeDone}:
				default:
				}
				return
			}

			log.Debug().Msg("sendMessage: got events channel, starting goroutine")

			count := 0
			for ev := range events {
				count++
				log.Debug().Int("count", count).Str("type", string(ev.Type)).Msg("sendMessage: forwarding event")
				select {
				case m.eventChan <- ev:
				case <-m.currentRunCtx.Done():
					log.Debug().Msg("sendMessage: run cancelled, exiting goroutine")
					return
				}
				log.Debug().Int("count", count).Str("type", string(ev.Type)).Msg("sendMessage: event sent")
			}
			log.Debug().Int("total", count).Msg("sendMessage: events channel closed, sending done")
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone}:
			default:
			}
		}()

		return nil
	}
}

// sendMessageWithConfig sends a message with a custom runtime config.
// This is used for special operations like /init that need a different system prompt.
// Events are still forwarded to eventChan for UI updates.
// Note: AutoApprove is enabled - all confirmations including plan review are auto-approved.
func (m Model) sendMessageWithConfig(input string, cfg *sdk.RuntimeConfig) tea.Cmd {
	return func() tea.Msg {
		m.currentRunCtx, m.currentRunCancel = context.WithCancel(m.ctx)

		log.Debug().Str("input", input).Msg("sendMessageWithConfig: starting")

		// Create temporary runtime with custom config
		// AutoApprove will auto-approve all confirmations (tools + plan review)
		tempRt, err := sdk.NewRuntime(cfg,
			sdk.WithAutoApprove(),
			sdk.WithLogger(GetLogger()),
		)
		if err != nil {
			log.Debug().Err(err).Msg("sendMessageWithConfig: failed to create runtime")
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone}:
			default:
			}
			return nil
		}

		// Initialize temporary runtime
		if err := tempRt.Initialize(m.ctx); err != nil {
			tempRt.Shutdown()
			log.Debug().Err(err).Msg("sendMessageWithConfig: failed to initialize runtime")
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone}:
			default:
			}
			return nil
		}

		// Run Process() in background goroutine
		go func() {
			events, err := tempRt.Process(m.currentRunCtx, input)
			if err != nil {
				log.Debug().Err(err).Msg("sendMessageWithConfig: process error")
				select {
				case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
				default:
				}
				select {
				case m.eventChan <- ChatEvent{Type: EventTypeDone}:
				default:
				}
				tempRt.Shutdown()
				return
			}

			log.Debug().Msg("sendMessageWithConfig: got events channel")

			adapter := NewAdapter()
			count := 0
			for ev := range events {
				count++
				log.Debug().Int("count", count).Str("type", string(ev.Type())).Msg("sendMessageWithConfig: forwarding event")
				select {
				case m.eventChan <- adapter.ConvertSDKEvent(ev):
				case <-m.currentRunCtx.Done():
					log.Debug().Msg("sendMessageWithConfig: run cancelled")
					tempRt.Shutdown()
					return
				}
			}

			log.Debug().Int("total", count).Msg("sendMessageWithConfig: events channel closed")
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone}:
			default:
			}
			tempRt.Shutdown()
		}()

		return nil
	}
}

// showStatus shows current execution status.
func (m Model) showStatus() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("Current Status:") + "\n")

	if m.state.Waiting() {
		b.WriteString(fmt.Sprintf("  State: %s\n", m.styles.Processing.Render("Processing")))
		if m.state.CurrentTool() != "" {
			b.WriteString(fmt.Sprintf("  Tool: %s\n", m.state.CurrentTool()))
		}
	} else {
		b.WriteString(fmt.Sprintf("  State: %s\n", m.styles.Help.Render("Idle")))
	}

	return m.messages.AddRaw(b.String())
}

// clearUIState clears UI layer state (MessageStore, TokenUsage, scroll state, tasks).
// Used by both /clear command and intent-based command_clear to ensure consistent behavior.
func (m *Model) clearUIState() {
	m.messages.Clear()
	m.state.SetTokenUsage(0)
	m.manualScroll = false
	m.manualScrollOffset = 0
	m.autoScroll = true
	m.tasks.Reset()
}
