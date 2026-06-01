package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/tasks"
)

// handleChatEvent processes chat events from the agent.
func (m Model) handleChatEvent(msg ChatEvent) (tea.Model, tea.Cmd) {
	log.Debug().Str("type", string(msg.Type)).Str("content", msg.Content).Msg("handleChatEvent: received event")

	// Log confirmation request for debugging
	if msg.Type == EventTypeConfirmationRequest {
		toolName, _ := msg.Extra["toolName"].(string)
		log.Debug().Str("tool", toolName).Msg("handleChatEvent: received confirmation request")
		model, cmd := m.handleEventConfirmationRequest(msg)
		return model, cmd
	}

	var model tea.Model
	var cmd tea.Cmd

	switch msg.Type {
	case EventTypeDone:
		model, cmd = m.handleEventDone(msg)

	case EventTypeThinkingStart:
		model, cmd = m.handleEventThinkingStart(msg)

	case EventTypeThinkingChunk:
		model, cmd = m.handleEventThinkingChunk(msg)

	case EventTypeThinkingEnd:
		model, cmd = m.handleEventThinkingEnd(msg)

	case EventTypeResponseStart:
		model, cmd = m.handleEventResponseStart(msg)

	case EventTypeResponseChunk:
		model, cmd = m.handleEventStreamChunk(msg)

	case EventTypeResponseEnd:
		model, cmd = m.handleEventResponseEnd(msg)

	case EventTypeResponse:
		model, cmd = m.handleEventResponse(msg)

	case EventTypeThinkingContent:
		model, cmd = m.handleEventThinkingContent(msg)

	case EventTypeAction:
		model, cmd = m.handleEventAction(msg)

	case EventTypeResult:
		model, cmd = m.handleEventResult(msg)

	case EventTypeError:
		model, cmd = m.handleEventError(msg)

	case EventTypeStep:
		model, cmd = m.handleEventStep(msg)

	case EventTypeToolStart:
		model, cmd = m.handleEventToolStart(msg)

	case EventTypeToolEnd:
		model, cmd = m.handleEventToolEnd(msg)

	case EventTypeCommandMatched:
		model, cmd = m.handleEventCommandMatched(msg)

	case EventTypeCommandResult:
		model, cmd = m.handleEventCommandResult(msg)

	case EventTypeTaskCreate:
		model, cmd = m.handleEventTaskCreate(msg)

	case EventTypeTaskUpdate:
		model, cmd = m.handleEventTaskUpdate(msg)

	case EventTypeTaskList:
		model, cmd = m.handleEventTaskList(msg)

	case EventTypePlanCreated:
		model, cmd = m.handleEventPlanCreated(msg)

	case EventTypePlanReviewStart:
		model, cmd = m.handleEventPlanReviewStart(msg)

	case EventTypePlanReviewFiles:
		model, cmd = m.handleEventPlanReviewFiles(msg)

	case EventTypePlanStep:
		model, cmd = m.handleEventPlanStep(msg)

	case EventTypePlanComplete:
		model, cmd = m.handleEventPlanComplete(msg)

	case EventTypePlanModeExit:
		model, cmd = m.handleEventPlanModeExit(msg)

	case EventTypeEnterPlanMode:
		model, cmd = m.handleEventEnterPlanMode(msg)

	case EventTypePlanVerifyStart:
		model, cmd = m.handleEventPlanVerifyStart(msg)

	case EventTypePlanVerifyResult:
		model, cmd = m.handleEventPlanVerifyResult(msg)

	case EventTypePlanVerifyEnd:
		model, cmd = m.handleEventPlanVerifyEnd(msg)

	case EventTypeSnapshotCreated:
		model, cmd = m.handleEventSnapshotCreated(msg)

	case EventTypeRollbackOffer:
		model, cmd = m.handleEventRollbackOffer(msg)

	case EventTypeRollbackComplete:
		model, cmd = m.handleEventRollbackComplete(msg)

	case EventTypeMaxStepsExceeded:
		model, cmd = m.handleEventMaxStepsExceeded(msg)

	default:
		return m, m.processEvents()
	}

	return model, cmd
}

// handleEventDone handles the done event.
// Response content is already accumulated in MessageStore.
// Note: For non-streaming responses (e.g., intent commands, direct Response events),
// the message is already displayed by its handler. Only render here for streaming
// responses that accumulated chunks without a separate Response event.
func (m Model) handleEventDone(msg ChatEvent) (tea.Model, tea.Cmd) {
	log.Debug().Str("requestID", msg.RequestID).Msg("handleEventDone: processing done event")

	// Check if the last assistant message was already displayed.
	// Messages added via Add() have .Rendered set, meaning they were already rendered in viewport.
	// Only messages accumulated via AppendToLast (streaming chunks) have .Rendered == "".
	lastMsg := m.messages.GetLastAssistantMessage()
	if lastMsg != nil && lastMsg.Rendered != "" {
		log.Debug().Msg("handleEventDone: skipping - last assistant message already displayed")
	} else {
		// Render the last assistant message (accumulated from chunks)
		m.messages.RenderLast(m.renderer, m.styles, renderMessage)
		m.updateTokenUsage()
	}

	// Check for pending messages - auto-send after done
	if len(m.pendingMessages) > 0 {
		// Combine all pending messages
		combined := combinePendingMessages(m.pendingMessages)
		m.pendingMessages = nil
		log.Debug().Str("combined", combined).Msg("handleEventDone: sending pending messages")

		// Add user message to store for the pending messages
		m.messages.Add(MessageTypeUser, combined, nil, renderMessage, m.renderer, m.styles)

		// IMMEDIATE thinking transition - skip old "思考了X秒" display
		// Reset and start new thinking right away (same as handleSubmit)
		m.thinking.Reset()
		m.processing.Reset()
		m.tasks.Reset()
		m.plan.Reset()

		// Set UI state and start thinking IMMEDIATELY
		m.state.SetWaiting(true)
		m.state.SetStartTime(time.Now())
		m.state.SetDisplayState(DisplayThinking)
		_, thinkingCmd := m.thinking.StartAndRender()
		m.autoScroll = true

		// Use tea.Batch for concurrent execution (same as handleSubmit)
		return m, tea.Batch(
			m.sendMessage(combined),
			thinkingCmd,
			m.scrollToBottom(),
			m.processEvents(),
		)
	}

	// No pending messages - complete current interaction
	m.stopWidgets()
	m.tasks.Reset()
	m.plan.Reset()
	m.state.ResetForNewInteraction()

	// Re-enable input and scroll to bottom
	return m, tea.Sequence(
		m.input.EnableAndFocus(),
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// stopWidgets stops the thinking and processing widgets.
func (m *Model) stopWidgets() {
	if m.thinking != nil {
		m.thinking.Complete()
	}
	if m.processing != nil {
		m.processing.Stop()
	}
}

// handleEventThinkingStart handles the thinking start event.
// Creates an empty Thinking message for streaming chunks to accumulate into.
func (m Model) handleEventThinkingStart(msg ChatEvent) (tea.Model, tea.Cmd) {
	if m.thinking == nil {
		return m, m.processEvents()
	}
	// Skip if thinking already started by handleSubmit (avoids restart on late engine event)
	if m.thinking.IsActive() {
		return m, m.processEvents()
	}

	// Create empty Thinking message for streaming chunks
	m.messages.AddEmpty(MessageTypeThinking)

	_, tickCmd := m.thinking.StartAndRender()
	// Thinking widget renders inline in chat area via buildChatContent()
	// IMPORTANT: tickCmd must execute BEFORE processEvents to start the tick chain
	// processEvents is blocking - it waits for next event from channel
	return m, tea.Sequence(tickCmd, m.processEvents())
}

// handleEventThinkingEnd handles the thinking end event.
// Renders any accumulated thinking content.
func (m Model) handleEventThinkingEnd(msg ChatEvent) (tea.Model, tea.Cmd) {
	if m.thinking == nil {
		return m, m.processEvents()
	}
	// Complete thinking widget — stops tick animation chain
	m.thinking.Complete()

	// Render accumulated thinking content and mark as complete (no cursor)
	m.messages.RenderLastWithTypeAndComplete(MessageTypeThinking, m.renderer, m.styles, renderMessage)

	// Continue listening for events
	return m, m.processEvents()
}

// handleEventThinkingChunk handles the thinking_chunk event.
// Accumulates thinking content chunks into a single MessageTypeThinking message.
func (m Model) handleEventThinkingChunk(msg ChatEvent) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(msg.Content) == "" {
		return m, m.processEvents()
	}
	// Accumulate into a single thinking message
	m.messages.AppendToLastTyped(msg.Content, MessageTypeThinking)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventResponseStart handles the response_start event.
// Creates an empty assistant message for streaming chunks to accumulate into.
func (m Model) handleEventResponseStart(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Create empty assistant message for streaming
	m.messages.AddEmpty(MessageTypeAssistant)
	return m, m.processEvents()
}

// handleEventResponseEnd handles the response_end event.
// Marks the last assistant message as complete (no cursor displayed).
func (m Model) handleEventResponseEnd(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Mark last assistant message as complete
	m.messages.MarkLastAssistantComplete()
	// Render the complete message
	m.messages.RenderLast(m.renderer, m.styles, renderMessage)
	m.updateTokenUsage()
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventResponse handles the response event.
// For non-streaming responses: glamour render and add to viewport.
// For streaming responses: the response content already matches accumulated chunks,
// so skip if we already have content (prevents duplicate).
func (m Model) handleEventResponse(msg ChatEvent) (tea.Model, tea.Cmd) {
	log.Debug().Str("content", msg.Content).Msg("handleEventResponse: processing response")

	// Skip empty responses
	if strings.TrimSpace(msg.Content) == "" {
		log.Debug().Msg("handleEventResponse: empty content, skipping")
		return m, m.processEvents()
	}

	// If we already have an assistant message (from accumulated streaming chunks),
	// the response content is already included — skip to prevent duplicate.
	lastMsg := m.messages.GetLastAssistantMessage()
	if lastMsg != nil && lastMsg.Content != "" {
		log.Debug().Msg("handleEventResponse: skipping — content already accumulated from streaming")
		return m, m.processEvents()
	}

	// Non-streaming: add new message with response (glamour rendered)
	m.messages.Add(MessageTypeAssistant, msg.Content, nil, renderMessage, m.renderer, m.styles)

	// Update token usage after response
	m.updateTokenUsage()

	log.Debug().Str("content", msg.Content).Msg("handleEventResponse: added response to viewport")

	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventThinkingContent handles the thinking_content event.
// Accumulates thinking content chunks into a single MessageTypeThinking message.
// Displayed in muted gray italic without [Aura]: prefix.
func (m Model) handleEventThinkingContent(msg ChatEvent) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(msg.Content) == "" {
		return m, m.processEvents()
	}
	// Accumulate into a single thinking message
	m.messages.AppendToLastTyped(msg.Content, MessageTypeThinking)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventStreamChunk handles streaming response chunks.
// Accumulates chunks in MessageStore for final glamour rendering.
// Does not output anything - Done event will render the accumulated content.
func (m Model) handleEventStreamChunk(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Log chunk content
	log.Debug().Str("content", msg.Content).Msg("streamChunk")

	// Accumulate chunk in MessageStore (no rendering)
	m.messages.AppendToLast(msg.Content)

	// Continue listening for next event (including Done)
	return m, m.processEvents()
}

// handleEventAction handles the action event.
func (m Model) handleEventAction(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Parse the action content as JSON to extract tool name for display
	toolName := parseToolNameFromAction(msg.Content)
	if toolName != "" {
		m.messages.Add(MessageTypeToolStart, toolName, msg.Extra, renderMessage, m.renderer, m.styles)
		return m, tea.Sequence(
			m.scrollToBottom(),
			m.processEvents(),
		)
	}
	// Fallback: display raw content if parsing fails
	m.messages.Add(MessageTypeToolStart, msg.Content, msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// parseToolNameFromAction extracts the tool name from action JSON content.
func parseToolNameFromAction(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	var action struct {
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal([]byte(content), &action); err == nil && action.Tool != "" {
		return action.Tool
	}
	return ""
}

// handleEventResult handles the result event.
func (m Model) handleEventResult(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.messages.Add(MessageTypeToolEnd, msg.Content, msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventError handles the error event.
func (m Model) handleEventError(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.messages.Add(MessageTypeError, msg.Content, nil, renderMessage, m.renderer, m.styles)
	// Reset state and enable input on error to prevent input lock
	m.stopWidgets()
	m.state.ResetForNewInteraction()
	return m, tea.Sequence(
		m.input.EnableAndFocus(),
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventStep handles the step event.
func (m Model) handleEventStep(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.messages.Add(MessageTypeSystem, msg.Content, nil, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanCreated handles the plan_created event.
func (m Model) handleEventPlanCreated(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.state.SetPlanModePhase(PlanModePhaseDesign)

	goal, _ := msg.Extra["goal"].(string)

	var steps []string
	if raw, ok := msg.Extra["steps"].([]string); ok {
		steps = raw
	}

	if len(steps) > 0 {
		m.plan.HandleCreate(goal, steps)
		m.messages.Add(MessageTypeSystem, fmt.Sprintf(i18n.T("plan.created"), len(steps)), msg.Extra, renderMessage, m.renderer, m.styles)
	}

	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanStep handles the plan_step event.
// PlanWidget shows checkbox progress — when step N starts, steps 0..N-1 are marked done.
func (m Model) handleEventPlanStep(msg ChatEvent) (tea.Model, tea.Cmd) {
	if msg.Extra == nil {
		return m, tea.Sequence(m.scrollToBottom(), m.processEvents())
	}
	stepNum := 0
	if n, ok := msg.Extra["step_num"].(int); ok {
		stepNum = n
	} else if f, ok := msg.Extra["step_num"].(float64); ok {
		stepNum = int(f)
	}
	stepDesc, _ := msg.Extra["step_desc"].(string)
	totalSteps := 0
	if t, ok := msg.Extra["total_steps"].(int); ok {
		totalSteps = t
	} else if f, ok := msg.Extra["total_steps"].(float64); ok {
		totalSteps = int(f)
	}

	// Mark current step as in-progress and prior steps as completed
	// (stepNum is 1-based from engine, convert to 0-based for widget)
	stepIndex := stepNum - 1
	if stepIndex >= 0 && stepIndex < len(m.plan.steps) {
		// Set current step as executing
		m.plan.SetCurrentStep(stepIndex)
		// Mark all prior steps as completed
		for i := 0; i < stepIndex; i++ {
			m.plan.MarkStepCompleted(i)
		}
	}

	// Show step progress message with visual indicator
	progressText := fmt.Sprintf(i18n.T("plan.step_progress"), stepNum, totalSteps, stepDesc)
	m.messages.Add(MessageTypeSystem, progressText, msg.Extra, renderMessage, m.renderer, m.styles)

	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanComplete handles the plan_complete event.
// Shows completion message and marks all steps as done.
func (m Model) handleEventPlanComplete(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Mark all steps as completed and clear current step
	for i := 0; i < len(m.plan.steps); i++ {
		m.plan.MarkStepCompleted(i)
	}
	m.plan.SetCurrentStep(-1)

	m.messages.Add(MessageTypeSystem, i18n.T("plan.completed"), msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanReviewStart handles the plan_review_start event.
// Shows review phase notification to user.
func (m Model) handleEventPlanReviewStart(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.state.SetPlanModePhase(PlanModePhaseReview)
	m.messages.Add(MessageTypeSystem, i18n.T("plan.review_phase"), msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanReviewFiles handles the plan_review_files event.
// Shows critical files that will be modified.
func (m Model) handleEventPlanReviewFiles(msg ChatEvent) (tea.Model, tea.Cmd) {
	files, ok := msg.Extra["files"].([]string)
	if ok && len(files) > 0 {
		content := i18n.T("plan.files_to_review") + strings.Join(files, "\n")
		m.messages.Add(MessageTypeSystem, content, nil, renderMessage, m.renderer, m.styles)
	}
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanModeExit handles the plan_mode_exit event.
// Shows notification that plan mode is exiting and execution begins.
func (m Model) handleEventPlanModeExit(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.state.SetPlanModePhase(PlanModePhaseExecute)
	m.messages.Add(MessageTypeSystem, i18n.T("plan.exiting_mode"), msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventToolStart handles the tool start event.
// Adds a ToolStart message with executionID for precise matching when tool ends.
// Note: "task" tool events are handled by handleEventTaskCreate/Update — skip to avoid duplicate output.
func (m Model) handleEventToolStart(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Get params from Extra
	params := ""
	if msg.Extra != nil {
		if p, ok := msg.Extra["params"].(string); ok {
			params = p
		}
	}

	// Skip task tool display — TaskWidget already renders it
	if msg.Content == "task" {
		m.state.StartTool(msg.Content, params)
		if m.processing != nil {
			m.processing.UpdateTool(msg.Content)
		}
		return m, m.processEvents()
	}

	// Add ToolStart message with executionID (message appears in flow order)
	m.messages.Add(MessageTypeToolStart, msg.Content, msg.Extra, renderMessage, m.renderer, m.styles)

	// Update ProcessingWidget for spinner display
	if m.processing != nil && m.processing.IsActive() {
		m.state.StartTool(msg.Content, params)
		m.processing.AddTool(msg.Content, params)
		return m, tea.Sequence(
			m.scrollToBottom(),
			m.processEvents(),
		)
	}

	// First tool — start the processing widget
	m.state.StartTool(msg.Content, params)

	var tickCmd tea.Cmd
	if m.processing != nil {
		_, tickCmd = m.processing.Start(msg.Content)
	}

	return m, tea.Sequence(
		m.scrollToBottom(),
		tickCmd,
		m.processEvents(),
	)
}

// handleEventToolEnd handles the tool end event.
// Merges with corresponding ToolStart message using executionID for precise matching.
// Note: "task" tool events are handled by handleEventTaskCreate/Update — skip to avoid duplicate output.
func (m Model) handleEventToolEnd(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Extract executionID and toolName from Extra
	executionID := ""
	toolName := ""
	if msg.Extra != nil {
		if id, ok := msg.Extra["execution_id"].(string); ok {
			executionID = id
		}
		if t, ok := msg.Extra["tool"].(string); ok {
			toolName = t
		}
	}

	// Extract duration - compatible with multiple types (JSON serialization converts to float64)
	var duration time.Duration
	if msg.Extra != nil {
		switch d := msg.Extra["duration"].(type) {
		case time.Duration:
			duration = d
		case int64:
			duration = time.Duration(d)
		case float64:
			duration = time.Duration(int64(d))
		case int:
			duration = time.Duration(d)
		}
	}

	// Skip task tool display — TaskWidget already renders it
	if toolName == "task" {
		if toolName != "" {
			m.state.EndTool(toolName, msg.Content)
		}
		return m, m.processEvents()
	}

	// Merge with ToolStart message using executionID (appears in flow order)
	if executionID != "" {
		m.messages.MergeToolBlockByExecID(executionID, toolName, msg.Content, duration, m.styles)
	}

	// Update state and ProcessingWidget
	if toolName != "" {
		m.state.EndTool(toolName, msg.Content)
	}

	if m.processing != nil && toolName != "" {
		m.processing.RemoveTool(toolName)
		if !m.processing.IsActive() {
			m.processing.Stop()
		}
	}

	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventCommandMatched handles the command matched event.
func (m Model) handleEventCommandMatched(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Execute UI operations based on command type
	switch msg.Content {
	case "command_clear":
		m.clearUIState()
	}

	// Show command matched notification
	m.messages.Add(MessageTypeSystem, fmt.Sprintf(i18n.T("command.matched"), msg.Content), nil, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventCommandResult handles the command result event.
func (m Model) handleEventCommandResult(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Check if this is an exit command
	if msg.Content == "exit" {
		m.messages.Add(MessageTypeSystem, i18n.T("command.exit"), nil, renderMessage, m.renderer, m.styles)
		return m, tea.Sequence(
			m.scrollToBottom(),
			tea.Quit,
		)
	}

	// For other commands, show the result
	m.messages.Add(MessageTypeAssistant, msg.Content, nil, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventConfirmationRequest handles confirmation request event.
// Shows a confirmation dialog for sensitive tool execution, plan review, or questions.
func (m Model) handleEventConfirmationRequest(msg ChatEvent) (tea.Model, tea.Cmd) {
	log.Debug().Msg("handleEventConfirmationRequest: called")

	// Extract confirmation details
	confType := ""
	toolName := ""
	params := map[string]any{}
	planGoal := ""
	var planSteps []string
	question := ""
	questionType := ""
	var questionOptions []QuestionOption
	defaultAnswer := ""

	if msg.Extra != nil {
		if t, ok := msg.Extra["confirmType"].(string); ok {
			confType = t
		}
		if t, ok := msg.Extra["toolName"].(string); ok {
			toolName = t
		}
		if p, ok := msg.Extra["params"].(map[string]any); ok {
			params = p
		}
		if g, ok := msg.Extra["planGoal"].(string); ok {
			planGoal = g
		}
		if s, ok := msg.Extra["planSteps"].([]string); ok {
			planSteps = s
		}
		// Question fields
		if q, ok := msg.Extra["question"].(string); ok {
			question = q
		}
		if qt, ok := msg.Extra["questionType"].(string); ok {
			questionType = qt
		}
		if opts, ok := msg.Extra["options"].([]events.QuestionOption); ok {
			questionOptions = make([]QuestionOption, len(opts))
			for i, opt := range opts {
				questionOptions[i] = QuestionOption{
					Label:       opt.Label,
					Description: opt.Description,
					Value:       opt.Value,
				}
			}
		}
		// Fallback: handle []any for options (from JSON deserialization)
		if opts, ok := msg.Extra["options"].([]any); ok && questionOptions == nil {
			questionOptions = make([]QuestionOption, 0, len(opts))
			for _, opt := range opts {
				if optMap, ok := opt.(map[string]any); ok {
					questionOptions = append(questionOptions, QuestionOption{
						Label:       getStringFromMap(optMap, "label"),
						Description: getStringFromMap(optMap, "description"),
						Value:       getStringFromMap(optMap, "value"),
					})
				}
			}
		}
		if da, ok := msg.Extra["defaultAnswer"].(string); ok {
			defaultAnswer = da
		}
	}

	// Use ResponseCh field directly
	responseCh := msg.ResponseCh
	// Get question response channel if present
	questionRespCh := getQuestionRespCh(msg.Extra)

	log.Debug().Str("type", confType).Str("tool", toolName).Bool("has_response_ch", responseCh != nil).Bool("has_question_resp_ch", questionRespCh != nil).Msg("handleEventConfirmationRequest: extracted details")

	// Clear thinking widget so user attention is drawn to the confirmation dialog
	if m.thinking != nil {
		m.thinking.Clear()
	}

	// For plan review, populate PlanWidget so the plan is visible in chat area
	if confType == events.ConfirmationTypePlanReview {
		m.plan.HandleCreate(planGoal, planSteps)
	}

	// Set display state to confirm
	m.state.SetDisplayState(DisplayConfirm)
	log.Debug().Msg("handleEventConfirmationRequest: cleared thinking, set DisplayConfirm")

	// Show confirmation dialog
	m.confirmState = ConfirmState{
		Waiting:  true,
		Selected: 0, // Default to Yes/First option
		Request: &ConfirmationRequest{
			Type:           ConfirmationType(confType),
			ToolName:       toolName,
			Params:         params,
			Message:        msg.Content,
			PlanGoal:       planGoal,
			PlanSteps:      planSteps,
			ResponseCh:     responseCh,
			Question:       question,
			QuestionType:   QuestionType(questionType),
			Options:        questionOptions,
			DefaultAnswer:  defaultAnswer,
			QuestionRespCh: questionRespCh,
		},
	}

	// Initialize text input for text questions
	if confType == "question" && questionType == "text" && defaultAnswer != "" {
		m.confirmState.TextInput = defaultAnswer
	}

	log.Debug().Bool("waiting", m.confirmState.Waiting).Str("type", confType).Msg("handleEventConfirmationRequest: confirmState set")

	// Return nil command — Bubble Tea v2 calls View() after Update() returns,
	// so the confirmation dialog will be rendered immediately.
	return m, nil
}

// getQuestionRespCh extracts the question response channel from extra.
func getQuestionRespCh(extra map[string]any) chan QuestionResponse {
	if extra == nil {
		return nil
	}
	// Try direct channel type
	if ch, ok := extra["questionRespCh"].(chan QuestionResponse); ok {
		return ch
	}
	// Try events.QuestionOption channel (from engine)
	if ch, ok := extra["questionRespCh"].(chan events.QuestionResponse); ok {
		// Convert to TUI QuestionResponse channel
		// Note: This requires adapter logic - the engine sends events.QuestionResponse
		// but TUI expects tui.QuestionResponse. We'll handle this in the response handler.
		return adaptQuestionRespCh(ch)
	}
	return nil
}

// adaptQuestionRespCh adapts an events.QuestionResponse channel to TUI channel.
func adaptQuestionRespCh(src chan events.QuestionResponse) chan QuestionResponse {
	dst := make(chan QuestionResponse, 1)
	go func() {
		if resp := <-src; !resp.Cancelled {
			dst <- QuestionResponse{
				Answer:    resp.Answer,
				Answers:   resp.Answers,
				Cancelled: resp.Cancelled,
			}
		} else {
			dst <- QuestionResponse{Cancelled: true}
		}
	}()
	return dst
}

// getStringFromMap safely extracts a string from a map.
func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractTaskID extracts task_id from event extra, handling int and float64.
func extractTaskID(extra map[string]any) int {
	if id, ok := extra["task_id"].(int); ok {
		return id
	}
	if f, ok := extra["task_id"].(float64); ok {
		return int(f)
	}
	return 0
}

// handleEventTaskCreate handles task creation events.
func (m Model) handleEventTaskCreate(msg ChatEvent) (tea.Model, tea.Cmd) {
	taskID := extractTaskID(msg.Extra)
	planStepID, _ := msg.Extra["plan_step_id"].(string)

	m.tasks.HandleCreate(taskID, msg.Content, planStepID)

	return m, m.scrollToBottom()
}

// handleEventTaskUpdate handles task update events.
func (m Model) handleEventTaskUpdate(msg ChatEvent) (tea.Model, tea.Cmd) {
	taskID := extractTaskID(msg.Extra)
	status, _ := msg.Extra["status"].(string)
	notes, _ := msg.Extra["notes"].(string)

	m.tasks.HandleUpdate(taskID, status, notes)

	return m, m.scrollToBottom()
}

// handleEventTaskList handles task list update events.
func (m Model) handleEventTaskList(msg ChatEvent) (tea.Model, tea.Cmd) {
	var taskList []tasks.Task

	if rawTasks, ok := msg.Extra["tasks"].([]tasks.Task); ok {
		taskList = rawTasks
	} else if rawTasks, ok := msg.Extra["tasks"].([]interface{}); ok {
		// Fallback for deserialized data (e.g., JSON)
		for _, raw := range rawTasks {
			if t, ok := raw.(tasks.Task); ok {
				taskList = append(taskList, t)
			}
		}
	}

	if len(taskList) > 0 {
		m.tasks.HandleList(taskList)
	}
	return m, m.scrollToBottom()
}

// handleEventEnterPlanMode handles the enter_plan_mode event.
// Shows notification that plan mode is starting with read-only exploration.
func (m Model) handleEventEnterPlanMode(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.state.SetPlanModePhase(PlanModePhaseExplore)
	m.messages.Add(MessageTypeSystem, i18n.T("plan.enter_mode"), msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanVerifyStart handles the plan_verify_start event.
// Shows verification phase notification.
func (m Model) handleEventPlanVerifyStart(msg ChatEvent) (tea.Model, tea.Cmd) {
	m.state.SetPlanModePhase(PlanModePhaseVerify)
	m.messages.Add(MessageTypeSystem, i18n.T("verify.started"), msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanVerifyResult handles the plan_verify_result event.
// Shows individual verification command results.
func (m Model) handleEventPlanVerifyResult(msg ChatEvent) (tea.Model, tea.Cmd) {
	// Extract command and result from extra
	cmd, _ := msg.Extra["command"].(string)
	passed := false
	if p, ok := msg.Extra["passed"].(bool); ok {
		passed = p
	}

	var content string
	if passed {
		content = "  ✓ " + cmd
	} else {
		content = "  ✗ " + cmd + "\n" + msg.Content
	}

	m.messages.Add(MessageTypeSystem, content, msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventPlanVerifyEnd handles the plan_verify_end event.
// Shows final verification summary and resets plan mode state.
func (m Model) handleEventPlanVerifyEnd(msg ChatEvent) (tea.Model, tea.Cmd) {
	passed := false
	if p, ok := msg.Extra["passed"].(bool); ok {
		passed = p
	}

	var content string
	if passed {
		content = i18n.T("verify.passed")
	} else {
		content = fmt.Sprintf(i18n.T("verify.failed"), msg.Content)
	}

	m.messages.Add(MessageTypeSystem, content, msg.Extra, renderMessage, m.renderer, m.styles)

	// Reset state after verification - exit plan mode completely
	m.stopWidgets()
	m.tasks.Reset()
	m.plan.Reset()
	m.state.ResetForNewInteraction()
	m.state.SetPlanModePhase(PlanModePhaseNone) // Exit plan mode

	return m, tea.Sequence(
		m.input.EnableAndFocus(),
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventSnapshotCreated handles the snapshot_created event.
// Shows notification that a rollback snapshot was created before execution.
func (m Model) handleEventSnapshotCreated(msg ChatEvent) (tea.Model, tea.Cmd) {
	snapshotID, _ := msg.Extra["snapshot_id"].(string)
	var files []string
	if rawFiles, ok := msg.Extra["files"].([]string); ok {
		files = rawFiles
	}

	content := fmt.Sprintf(i18n.T("rollback.snapshot_created"), snapshotID, len(files))
	if len(files) > 0 && len(files) <= 5 {
		content += "\n" + strings.Join(files, "\n")
	} else if len(files) > 5 {
		content += "\n" + strings.Join(files[:5], "\n") + fmt.Sprintf("\n... and %d more", len(files)-5)
	}

	m.messages.Add(MessageTypeSystem, content, msg.Extra, renderMessage, m.renderer, m.styles)
	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventRollbackOffer handles the rollback_offer event.
// Shows a confirmation dialog offering rollback after execution/verification failure.
func (m Model) handleEventRollbackOffer(msg ChatEvent) (tea.Model, tea.Cmd) {
	snapshotID, _ := msg.Extra["snapshot_id"].(string)
	var files []string
	if rawFiles, ok := msg.Extra["files"].([]string); ok {
		files = rawFiles
	}
	reason, _ := msg.Extra["reason"].(string)

	// Extract response channel for rollback confirmation
	var responseCh chan bool
	if ch, ok := msg.Extra["response_ch"].(chan bool); ok {
		responseCh = ch
	}

	// Clear thinking widget so user attention is drawn to the rollback dialog
	if m.thinking != nil {
		m.thinking.Clear()
	}

	// Set display state to confirm
	m.state.SetDisplayState(DisplayConfirm)

	// Build rollback confirmation request
	m.confirmState = ConfirmState{
		Waiting:  true,
		Selected: 0, // Default to Yes (rollback)
		Request: &ConfirmationRequest{
			Type:       ConfirmationRollback,
			Message:    fmt.Sprintf(i18n.T("rollback.offer"), reason),
			PlanGoal:   i18n.T("rollback.goal"),
			PlanSteps:  files,
			ResponseCh: responseCh, // Channel to send rollback confirmation
		},
	}

	// Store snapshot ID for rollback action (legacy, now handled via channel)
	m.rollbackSnapshotID = snapshotID

	return m, nil
}

// handleEventRollbackComplete handles the rollback_complete event.
// Shows notification that rollback was completed.
func (m Model) handleEventRollbackComplete(msg ChatEvent) (tea.Model, tea.Cmd) {
	success := false
	if s, ok := msg.Extra["success"].(bool); ok {
		success = s
	}
	files, _ := msg.Extra["files"].([]string)

	var content string
	if success {
		content = fmt.Sprintf(i18n.T("rollback.completed"), strings.Join(files, "\n"))
	} else {
		content = fmt.Sprintf(i18n.T("rollback.failed"), msg.Content)
	}

	m.messages.Add(MessageTypeSystem, content, msg.Extra, renderMessage, m.renderer, m.styles)

	// Reset state after rollback
	m.stopWidgets()
	m.tasks.Reset()
	m.plan.Reset()
	m.state.ResetForNewInteraction()
	m.rollbackSnapshotID = ""

	return m, tea.Sequence(
		m.input.EnableAndFocus(),
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// handleEventMaxStepsExceeded handles the max_steps_exceeded event.
// Shows warning that the ReAct loop hit the step limit and a best-effort answer will be generated.
func (m Model) handleEventMaxStepsExceeded(msg ChatEvent) (tea.Model, tea.Cmd) {
	maxSteps := 0
	if m, ok := msg.Extra["max_steps"].(int); ok {
		maxSteps = m
	} else if f, ok := msg.Extra["max_steps"].(float64); ok {
		maxSteps = int(f)
	}

	content := fmt.Sprintf(i18n.T("engine.max_steps_exceeded"), maxSteps)
	m.messages.Add(MessageTypeSystem, content, msg.Extra, renderMessage, m.renderer, m.styles)

	return m, tea.Sequence(
		m.scrollToBottom(),
		m.processEvents(),
	)
}

// combinePendingMessages combines all pending messages into a single message.
// Preserves original user input without adding extra formatting.
// Format: "content1\ncontent2" (newline-separated if multiple)
func combinePendingMessages(messages []PendingMessage) string {
	var b strings.Builder
	for i, msg := range messages {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(msg.Content)
	}
	return b.String()
}
