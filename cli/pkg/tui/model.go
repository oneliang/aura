package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/user"
	"github.com/oneliang/aura/shared/pkg/version"
)

// runtimeRegistry is a package-level registry for temporary runtimes.
// Used to route interaction responses to the correct runtime based on RuntimeID.
// Key: RuntimeID (string), Value: *sdk.Runtime
var runtimeRegistry sync.Map

// RegisterRuntime registers a runtime with its RuntimeID for response routing.
func RegisterRuntime(runtimeID string, rt *sdk.Runtime) {
	runtimeRegistry.Store(runtimeID, rt)
}

// UnregisterRuntime removes a runtime from the registry.
func UnregisterRuntime(runtimeID string) {
	runtimeRegistry.Delete(runtimeID)
}

// GetRuntime returns the runtime for a given RuntimeID, or nil if not found.
func GetRuntime(runtimeID string) *sdk.Runtime {
	if v, ok := runtimeRegistry.Load(runtimeID); ok {
		return v.(*sdk.Runtime)
	}
	return nil
}

// Model is the main Bubble Tea model.
type Model struct {
	state    *State
	messages *MessageStore
	input    *InputManager
	styles   UIStyles
	renderer *MarkdownRenderer

	// Widgets - independent UI components
	waiting     *WaitingWidget
	thinking    *ThinkingWidget
	processing  *ProcessingWidget
	tasks       *TaskWidget
	plan        *PlanWidget

	// ===== 新架构：统一事件流 =====

	// Runtime引用 - 黑盒，只有IN/OUT事件流接口
	runtime *sdk.Runtime

	// Temporary runtime registry - maps RuntimeID to runtime for response routing
	// Used in multi-runtime scenarios like /init command
	tempRuntimes map[string]*sdk.Runtime

	// OUT: 发送事件通道（用户输入、交互响应等）
	eventOutCh chan events.Event

	// IN: 接收事件通道（Agent响应、交互请求等）
	eventInCh chan events.Event

	// Shared: 共享事件通道（多 runtime 场景）
	// 所有 runtime 的事件汇集到这里，根据 RuntimeID 区分
	sharedEventCh chan events.Event

	// Internal event processing
	ctx        context.Context
	cancelFunc context.CancelFunc
	eventChan  chan ChatEvent // Internal channel for Bubble Tea event loop

	// Channel lifecycle protection
	closed  bool
	closeMu sync.Mutex

	// Goroutine lifecycle management for tempRuntimes
	tempRuntimeWg sync.WaitGroup

	getSystemPromptFunc GetSystemPromptFunc // Optional: function to get system prompt

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

// NewWithRuntime creates a new TUI model with runtime reference.
// This is the new architecture: TUI holds runtime reference and uses event stream.
// sharedEventCh: externally provided shared event channel for multi-runtime scenarios.
func NewWithRuntime(ctx context.Context, rt *sdk.Runtime, config Config, sessionMgr *sdk.SessionManager, summarizer *sdk.Summarizer, modelProvider *ModelProvider, commandProvider commands.Command, mcpManager *sdk.MCPManager, sharedEventCh chan sdk.Event) *Model {
	startTime := time.Now()
	log.Debug("[DIAG] NewWithRuntime: starting")

	// Initialize i18n constants (must be called after i18n.Init)
	InitI18nConstants()
	log.Debug("[DIAG] NewWithRuntime: InitI18nConstants done", "elapsed", time.Since(startTime))

	ctx, cancel := context.WithCancel(ctx)

	styles := DefaultStyles()
	renderer := NewMarkdownRenderer(80)
	messages := NewMessageStore(500, config.UserName)
	input := NewInputManager(styles)
	log.Debug("[DIAG] NewWithRuntime: styles/renderer/messages/input created", "elapsed", time.Since(startTime))

	state := NewState()
	state.SetDebugMode(config.DebugMode)
	state.SetTokenMax(config.TokenMax)
	state.SetShowTokens(config.ShowTokens)
	log.Debug("[DIAG] NewWithRuntime: state created", "elapsed", time.Since(startTime))

	m := Model{
		state:               state,
		messages:            messages,
		input:               input,
		styles:              styles,
		renderer:            renderer,
		waiting:             NewWaitingWidget(),
		thinking:            NewThinkingWidget(),
		processing:          NewProcessingWidget(),
		tasks:               NewTaskWidget(styles),
		plan:                NewPlanWidget(styles),
		runtime:             rt,  // 新架构：持有runtime引用
		tempRuntimes:        make(map[string]*sdk.Runtime), // Temporary runtime registry for response routing
		getSystemPromptFunc: config.GetSystemPrompt,
		ctx:                 ctx,
		cancelFunc:          cancel,
		eventChan:           make(chan ChatEvent, 100),
		eventOutCh:          make(chan events.Event, 100),  // OUT: 发送事件
		eventInCh:           make(chan events.Event, 100),  // IN: 接收事件（备用）
		sharedEventCh:       sharedEventCh,                  // Shared: 共享事件通道
		config:              config,
		sessionMgr:          sessionMgr,
		summarizer:          summarizer,
		modelProvider:       modelProvider,
		commandProvider:     commandProvider,
		sessionPopup:        NewSessionPopup(),
		subscriptionPopup:   NewSubscriptionPopup(),
		userID:              user.GetDefaultUserID(), // Load current user ID from users.yaml
		mcpManager:          mcpManager,
		viewport:            viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
		autoScroll:        AutoScrollDefault,
		viewportReady:     false,
	}
	log.Debug("[DIAG] NewWithRuntime: Model struct created", "elapsed", time.Since(startTime))

	m.statusBar = NewStatusBarWidget(styles, state, &m)
	m.commandPopup = &CommandPopupWidget{styles: styles}
	log.Debug("[DIAG] NewWithRuntime: statusBar/commandPopup created", "elapsed", time.Since(startTime))

	// Set greeting (fixed header, not cleared by /clear)
	versionInfo := styles.Help.Render(version.FullVersion())
	m.greeting = fmt.Sprintf("%s\n%s",
		lipglossv2.NewStyle().Bold(true).Foreground(lipglossv2.Color(ColorCommand)).Render("✨ Aura — Personal AI Assistant"),
		versionInfo)

	if modelProvider != nil {
		modelProvider.Set(&m)
	}
	log.Debug("[DIAG] NewWithRuntime: greeting/modelProvider done", "elapsed", time.Since(startTime))

	// Load sessions if session manager is available
	if sessionMgr != nil {
		items, err := sessionMgr.ListSessions()
		log.Debug("[DIAG] NewWithRuntime: ListSessions done", "elapsed", time.Since(startTime))
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
		log.Debug("[DIAG] NewWithRuntime: sessionItems set", "elapsed", time.Since(startTime))

		// Load session history if current session exists
		if m.currentSession != nil {
			m.loadSessionHistory()
			m.sessionInfo = fmt.Sprintf("  %s", i18n.T("tui.session_loaded", fmt.Sprintf("%s (%s)", m.currentSession.name, m.currentSession.id)))
		} else {
			m.sessionInfo = ""
		}
		log.Debug("[DIAG] NewWithRuntime: loadSessionHistory done", "elapsed", time.Since(startTime))
	}

	log.Debug("[DIAG] NewWithRuntime: completed", "elapsed", time.Since(startTime))
	return &m
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
		log.Debug("loadSessionHistory: sessionMgr or currentSession is nil")
		return false
	}

	store := m.sessionMgr.GetStore()
	if store == nil {
		log.Debug("loadSessionHistory: store is nil")
		return false
	}

	// Get messages from session
	ctx := context.Background()
	msgs, err := store.GetMessages(ctx, m.currentSession.ID(), 0) // Load all messages
	if err != nil {
		log.Debug("loadSessionHistory: error loading messages", "error", err.Error(), "sessionID", m.currentSession.ID())
		return false
	}
	if len(msgs) == 0 {
		log.Debug("loadSessionHistory: no messages", "sessionID", m.currentSession.ID())
		return false
	}

	log.Debug("loadSessionHistory: loaded messages", "count", len(msgs), "sessionID", m.currentSession.ID())

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

// sendMessage sends a message to the agent via event stream.
// 新架构：通过事件流发送用户输入，不再直接调用runFn
// 注意：requestID在调用前由Update设置到m.currentRequestID
func (m *Model) sendMessage(input string) tea.Cmd {
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	m.currentRequestID = requestID

	return func() tea.Msg {

		userEvent := events.NewEvent(events.EventTypeUserInput, input, requestID)
		select {
		case m.eventOutCh <- userEvent:
		default:
			log.Warn("sendMessage: eventOutCh full, dropping user input")
		}

		return nil
	}
}

// sendMessageWithConfig sends a message with a custom runtime config.
// This is used for special operations like /init that need a different system prompt.
// Events are forwarded to eventChan for UI updates using the event stream pattern.
// Note: AutoApprove is enabled - all confirmations including plan review are auto-approved.
func (m Model) sendMessageWithConfig(input string, cfg *sdk.RuntimeConfig) tea.Cmd {
	return func() tea.Msg {
		m.currentRunCtx, m.currentRunCancel = context.WithCancel(m.ctx)

		// Generate unique RuntimeID for temp runtime
		runtimeID := fmt.Sprintf("init_%d", time.Now().UnixNano())

		// Create temporary runtime with custom config and RuntimeID
		// AutoApprove will auto-approve all confirmations (tools + plan review)
		tempRt, err := sdk.NewRuntime(cfg,
			sdk.WithAutoApprove(),
			sdk.WithLogger(GetLogger()),
			sdk.WithRuntimeID(runtimeID),
		)
		if err != nil {
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone, Reason: DoneReasonErrorRuntimeCreate}:
			default:
			}
			return nil
		}

		// Register temp runtime for response routing
		RegisterRuntime(runtimeID, tempRt)

		// Initialize temporary runtime
		if err := tempRt.Initialize(m.ctx); err != nil {
			UnregisterRuntime(runtimeID)
			tempRt.Shutdown()
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone, Reason: DoneReasonErrorRuntimeInit}:
			default:
			}
			return nil
		}

		// Start temporary runtime with event stream pattern
		if err := tempRt.Start(m.currentRunCtx); err != nil {
			UnregisterRuntime(runtimeID)
			tempRt.Shutdown()
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone, Reason: DoneReasonErrorRuntimeStart}:
			default:
			}
			return nil
		}

		// Run event stream forwarding in background goroutine
		m.tempRuntimeWg.Add(1)
		go func() {
			defer m.tempRuntimeWg.Done()
			adapter := NewAdapter()
			agentEvents := tempRt.Events()

			for {
				select {
				case <-m.currentRunCtx.Done():
					UnregisterRuntime(runtimeID)
					tempRt.Stop(m.currentRunCtx)
					tempRt.Shutdown()
					return
				case ev, ok := <-agentEvents:
					if !ok {
						// Events channel closed, send Done and cleanup
						UnregisterRuntime(runtimeID)
						select {
						case m.eventChan <- ChatEvent{Type: EventTypeDone, Reason: DoneReasonShutdown}:
						default:
						}
						tempRt.Shutdown()
						return
					}
					select {
					case m.eventChan <- adapter.ConvertSDKEvent(ev):
					default:
						// Channel full, skip event
					}
				}
			}
		}()

		// Send user input event to runtime
		requestID := runtimeID // Use same ID for request tracking
		userEvent := events.NewEvent(events.EventTypeUserInput, input, requestID)
		if err := tempRt.SendEvent(m.currentRunCtx, userEvent); err != nil {
			m.currentRunCancel() // Cancel goroutine context before cleanup
			UnregisterRuntime(runtimeID)
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeError, Content: err.Error()}:
			default:
			}
			select {
			case m.eventChan <- ChatEvent{Type: EventTypeDone, Reason: DoneReasonErrorSendEvent}:
			default:
			}
			tempRt.Stop(m.currentRunCtx)
			tempRt.Shutdown()
		}

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

// ===== 新架构：统一事件流接口 =====

// ReceiveEvent 接收事件（IN）
// 统一入口：所有Agent输出事件通过这个方法处理
// 注意：此方法从Orchestrator goroutine调用，通过eventChan传递到Bubble Tea主循环
func (m *Model) ReceiveEvent(event events.Event) {
	chatEvent := m.convertEventToChatEvent(event)
	select {
	case m.eventChan <- chatEvent:
	default:
		log.Warn("ReceiveEvent: eventChan full, dropping event", "type", string(event.Type()))
	}
}

// convertEventToChatEvent 将events.Event转换为ChatEvent
func (m *Model) convertEventToChatEvent(event events.Event) ChatEvent {
	switch event.Type() {
	case events.EventTypeThinkingStart:
		return ChatEvent{Type: EventTypeThinkingStart, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeThinkingChunk:
		return ChatEvent{Type: EventTypeThinkingChunk, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeThinkingEnd:
		return ChatEvent{Type: EventTypeThinkingEnd, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeThinkingContent:
		return ChatEvent{Type: EventTypeThinkingContent, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeResponseStart:
		return ChatEvent{Type: EventTypeResponseStart, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeResponseChunk:
		return ChatEvent{Type: EventTypeResponseChunk, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeResponseEnd:
		return ChatEvent{Type: EventTypeResponseEnd, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeResponse:
		return ChatEvent{Type: EventTypeResponse, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeAction:
		return ChatEvent{Type: EventTypeAction, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeResult:
		return ChatEvent{Type: EventTypeResult, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeError:
		return ChatEvent{Type: EventTypeError, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeStep:
		return ChatEvent{Type: EventTypeStep, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeToolStart:
		return ChatEvent{Type: EventTypeToolStart, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeToolEnd:
		return ChatEvent{Type: EventTypeToolEnd, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeInteractionRequest:
		return ChatEvent{Type: EventTypeConfirmationRequest, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeCommandMatched:
		return ChatEvent{Type: EventTypeCommandMatched, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeCommandResult:
		return ChatEvent{Type: EventTypeCommandResult, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeTaskCreate:
		return ChatEvent{Type: EventTypeTaskCreate, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeTaskUpdate:
		return ChatEvent{Type: EventTypeTaskUpdate, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeTaskList:
		return ChatEvent{Type: EventTypeTaskList, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanCreated:
		return ChatEvent{Type: EventTypePlanCreated, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanReviewStart:
		return ChatEvent{Type: EventTypePlanReviewStart, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanReviewFiles:
		return ChatEvent{Type: EventTypePlanReviewFiles, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanStep:
		return ChatEvent{Type: EventTypePlanStep, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanComplete:
		return ChatEvent{Type: EventTypePlanComplete, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanModeExit:
		return ChatEvent{Type: EventTypePlanModeExit, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeEnterPlanMode:
		return ChatEvent{Type: EventTypeEnterPlanMode, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanVerifyStart:
		return ChatEvent{Type: EventTypePlanVerifyStart, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanVerifyResult:
		return ChatEvent{Type: EventTypePlanVerifyResult, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypePlanVerifyEnd:
		return ChatEvent{Type: EventTypePlanVerifyEnd, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeSnapshotCreated:
		return ChatEvent{Type: EventTypeSnapshotCreated, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeRollbackComplete:
		return ChatEvent{Type: EventTypeRollbackComplete, Content: event.Content(), Extra: event.Extra(), RequestID: event.RequestID()}
	case events.EventTypeAgentStart:
		return ChatEvent{Type: EventTypeAgentStart, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeAgentStop:
		return ChatEvent{Type: EventTypeAgentStop, Content: event.Content(), RequestID: event.RequestID()}
	case events.EventTypeDone:
		return ChatEvent{Type: EventTypeDone, RequestID: event.RequestID(), Reason: DoneReasonNormal}
	default:
		return ChatEvent{Type: EventTypeDone, RequestID: event.RequestID(), Reason: DoneReasonNormal}
	}
}

// Events 获取输出事件流（OUT）
// 返回事件流供Orchestrator转发到Agent
func (m *Model) Events() <-chan events.Event {
	return m.eventOutCh
}

// handleInteractionRequest 处理交互请求
func (m *Model) handleInteractionRequest(event events.Event) {
	reqID := event.RequestID()
	interactionType := event.InteractionType()

	// 设置确认状态
	m.confirmState = ConfirmState{
		Waiting: true,
		Request: &ConfirmationRequest{
			Type:     ConfirmationType(interactionType),
			ToolName: "",
		},
	}

	// 从Extra中提取请求详情
	if extra := event.Extra(); extra != nil {
		if toolName, ok := extra["tool_name"].(string); ok {
			m.confirmState.Request.ToolName = toolName
		}
		if params, ok := extra["tool_params"].(map[string]any); ok {
			m.confirmState.Request.Params = params
		}
		if planGoal, ok := extra["plan_goal"].(string); ok {
			m.confirmState.Request.PlanGoal = planGoal
		}
		if planSteps, ok := extra["plan_steps"].([]string); ok {
			m.confirmState.Request.PlanSteps = planSteps
		}
	}

	// 保存requestID用于响应
	m.confirmState.RequestID = reqID
	m.confirmState.InteractionType = interactionType

	// 设置显示状态为确认对话框
	m.state.SetDisplayState(DisplayConfirm)
}

// sendUserInput 发送用户输入事件
// 将用户文本包装成EventTypeUserInput发送
func (m *Model) sendUserInput(input string) {
	event := events.NewEvent(events.EventTypeUserInput, input, m.currentRequestID)
	select {
	case m.eventOutCh <- event:
	default:
		log.Warn("eventOutCh full, dropping user input event")
	}
}

// sendInteractionResponse sends an interaction response via event stream.
// Additional fields (answer, answers, cancelled) are merged into the event extra.
// Routes response to correct runtime based on RuntimeID.
func (m *Model) sendInteractionResponse(approved bool, extraFields ...map[string]any) {
	if m.confirmState.RequestID == "" {
		return
	}

	extra := map[string]any{
		"approved": approved,
		"type":     m.confirmState.InteractionType,
	}
	for _, fields := range extraFields {
		for k, v := range fields {
			extra[k] = v
		}
	}

	event := events.NewEventWithExtra(
		events.EventTypeInteractionResponse,
		"",
		extra,
		m.confirmState.RequestID,
	)

	// Route response based on RuntimeID
	runtimeID := m.confirmState.RuntimeID
	if runtimeID == "" || runtimeID == "main" {
		// Main runtime: send via eventOutCh (Orchestrator forwards to main runtime)
		select {
		case m.eventOutCh <- event:
		default:
			log.Warn("eventOutCh full, dropping interaction response event")
		}
	} else {
		// Temp runtime: send directly to the registered runtime
		rt := GetRuntime(runtimeID)
		if rt != nil {
			if err := rt.SendEvent(m.ctx, event); err != nil {
				log.Warn("failed to send interaction response to temp runtime", "runtimeID", runtimeID, "error", err)
			}
		} else {
			log.Warn("temp runtime not found in registry", "runtimeID", runtimeID)
			// Fallback: try eventOutCh
			select {
			case m.eventOutCh <- event:
			default:
				log.Warn("eventOutCh full, dropping interaction response event")
			}
		}
	}

	// 重置确认状态
	m.confirmState = ConfirmState{Waiting: false}
	m.state.SetDisplayState(DisplayProcessing)
}

// Cancel cancels the context to unblock goroutines waiting on ctx.Done().
// Must be called BEFORE orchestrator.Stop() to prevent deadlock.
func (m *Model) Cancel() {
	// Cancel main context
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	// Cancel current run context (for tempRuntime goroutines)
	if m.currentRunCancel != nil {
		m.currentRunCancel()
	}
}

// Close 关闭Model的所有channel，释放资源
// 应在TUI退出后调用，确保Orchestrator goroutine能正确退出
func (m *Model) Close() {
	m.closeMu.Lock()
	if m.closed {
		m.closeMu.Unlock()
		return
	}
	m.closed = true
	m.closeMu.Unlock()

	m.cancelFunc()

	// Wait for tempRuntime goroutines to complete (unblocked by ctx.Done())
	m.tempRuntimeWg.Wait()

	// 关闭所有channel，确保接收方能正确退出
	// eventOutCh may already be closed by closeEventOutCh, use recover to handle panic
	func() {
		defer func() { recover() }()
		close(m.eventOutCh)
	}()
	close(m.eventInCh)
	close(m.eventChan)
}

// closeEventOutCh closes only the eventOutCh channel.
// Called before orchestrator.Stop() to unblock TUI→Runtime goroutine.
func (m *Model) closeEventOutCh() {
	m.closeMu.Lock()
	// Use recover to handle case where channel is already closed
	defer func() {
		m.closeMu.Unlock()
		recover()
	}()
	close(m.eventOutCh)
}
