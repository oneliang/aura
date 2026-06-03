// Package events provides unified event definitions for the Aura system.
// This package is the single source of truth for all event types,
// used by engine, runtime, commands, and application layers (TUI/API/Adapters).
package events

import "time"

// EventType represents the type of an event.
type EventType string

// Engine execution events - emitted during ReAct loop execution.
const (
	// ThinkingStart event is emitted when the agent starts reasoning.
	EventTypeThinkingStart EventType = "thinking_start"
	// ThinkingChunk event is emitted for each chunk of thinking/reasoning content.
	EventTypeThinkingChunk EventType = "thinking_chunk"
	// ThinkingEnd event is emitted when the agent finishes reasoning.
	EventTypeThinkingEnd EventType = "thinking_end"
	// Action event is emitted when the agent decides to take an action.
	EventTypeAction EventType = "action"
	// Result event is emitted when a tool execution result is available.
	EventTypeResult EventType = "result"
	// Response event is emitted when the agent provides a final response.
	EventTypeResponse EventType = "response"
	// ResponseStart event is emitted when the agent starts streaming response content.
	EventTypeResponseStart EventType = "response_start"
	// ResponseChunk event is emitted for each chunk of a streaming response.
	EventTypeResponseChunk EventType = "response_chunk"
	// ResponseEnd event is emitted when the agent finishes streaming response content.
	EventTypeResponseEnd EventType = "response_end"
	// ThinkingContent event is emitted when LLM thinking/reasoning content is captured.
	// Deprecated: Use EventTypeThinkingChunk instead.
	EventTypeThinkingContent EventType = "thinking_content"
	// Error event is emitted when an error occurs.
	EventTypeError EventType = "error"
	// Step event is emitted to report current step progress.
	EventTypeStep EventType = "step"
	// ToolStart event is emitted when a tool starts executing.
	EventTypeToolStart EventType = "tool_start"
	// ToolEnd event is emitted when a tool finishes executing.
	EventTypeToolEnd EventType = "tool_end"
)

// Planning events - emitted during explicit planning mode.
const (
	// PlanCreated event is emitted when a plan is created.
	EventTypePlanCreated EventType = "plan_created"
	// PlanReviewStart event is emitted when entering review phase.
	EventTypePlanReviewStart EventType = "plan_review_start"
	// PlanReviewFiles event is emitted with critical files to review.
	EventTypePlanReviewFiles EventType = "plan_review_files"
	// PlanStep event is emitted when a plan step is executed.
	EventTypePlanStep EventType = "plan_step"
	// PlanComplete event is emitted when a plan is completed.
	EventTypePlanComplete EventType = "plan_complete"
	// PlanModeExit event is emitted when exiting plan mode (entering execution phase).
	EventTypePlanModeExit EventType = "plan_mode_exit"
	// PlanVerifyStart event is emitted when starting verification phase.
	EventTypePlanVerifyStart EventType = "plan_verify_start"
	// PlanVerifyEnd event is emitted when verification phase completes.
	EventTypePlanVerifyEnd EventType = "plan_verify_end"
	// PlanVerifyResult event is emitted with verification results.
	EventTypePlanVerifyResult EventType = "plan_verify_result"
	// EnterPlanMode event is emitted when entering plan mode.
	EventTypeEnterPlanMode EventType = "enter_plan_mode"
	// SnapshotCreated event is emitted when a rollback snapshot is created.
	EventTypeSnapshotCreated EventType = "snapshot_created"
	// RollbackOffer event is emitted when offering rollback to user.
	EventTypeRollbackOffer EventType = "rollback_offer"
	// RollbackComplete event is emitted when rollback is completed.
	EventTypeRollbackComplete EventType = "rollback_complete"
)

// MaxStepsExceeded event is emitted when ReAct loop exceeds max steps limit.
// Triggers best-effort answer generation instead of reflection/validation.
const (
	EventTypeMaxStepsExceeded EventType = "max_steps_exceeded"
)

// Runtime lifecycle events.
const (
	// Done event is emitted when processing is complete.
	EventTypeDone EventType = "done"
	// ConfirmationRequest event is emitted when user confirmation is needed.
	EventTypeConfirmationRequest EventType = "confirmation_request"

	// ===== IN事件类型（输入到Agent）=====

	// UserInput event is emitted when user sends text input.
	EventTypeUserInput EventType = "user_input"
	// UserMessage event is emitted when user sends a message with metadata.
	EventTypeUserMessage EventType = "user_message"
	// InteractionResponse event is emitted when user responds to an interaction request.
	EventTypeInteractionResponse EventType = "interaction_response"
	// SystemCommand event is emitted for system-level commands.
	EventTypeSystemCommand EventType = "system_command"
	// ContextUpdate event is emitted when context needs to be updated.
	EventTypeContextUpdate EventType = "context_update"

	// ===== Agent状态事件 =====

	// AgentStart event is emitted when agent starts running.
	EventTypeAgentStart EventType = "agent_start"
	// AgentStop event is emitted when agent stops running.
	EventTypeAgentStop EventType = "agent_stop"

	// ===== OUT交互请求事件 =====

	// InteractionRequest event is emitted when agent requests user interaction.
	EventTypeInteractionRequest EventType = "interaction_request"
)

// InteractionType represents the type of interaction request.
type InteractionType string

const (
	InteractionTypeToolConfirmation  InteractionType = "tool_confirmation"
	InteractionTypePlanReview        InteractionType = "plan_review"
	InteractionTypeAskUserQuestion   InteractionType = "ask_user_question"
	InteractionTypeRollbackConfirm   InteractionType = "rollback_confirm"
)

// Confirmation types for ConfirmationRequest.
const (
	ConfirmationTypeSensitiveTool = "sensitive_tool"
	ConfirmationTypePlanReview    = "plan_review"
)

// Memory events - for memory operations.
const (
	// MemoryClearRequest event is sent to request memory clearing.
	EventTypeMemoryClearRequest EventType = "memory_clear_request"
	// MemoryCleared event is emitted after memory is cleared.
	EventTypeMemoryCleared EventType = "memory_cleared"
	// MemoryStatsRequest event is sent to request memory statistics.
	EventTypeMemoryStatsRequest EventType = "memory_stats_request"
	// MemoryStats event is emitted with memory statistics.
	EventTypeMemoryStats EventType = "memory_stats"
	// MemoryCompacted event is emitted after memory is compacted/summarized.
	EventTypeMemoryCompacted EventType = "memory_compacted"
)

// Session events - for session lifecycle.
const (
	// SessionCreated event is emitted when a new session is created.
	EventTypeSessionCreated EventType = "session_created"
	// SessionSwitched event is emitted when switching to a different session.
	EventTypeSessionSwitched EventType = "session_switched"
	// SessionDeleted event is emitted when a session is deleted.
	EventTypeSessionDeleted EventType = "session_deleted"
)

// Command events - for command request/response pattern.
// This enables event-driven communication between CommandProvider and Runtime/TUI.
const (
	// CommandRequest event is emitted when a command needs to be executed.
	EventTypeCommandRequest EventType = "command_request"
	// CommandResponse event is emitted when a command has been executed.
	EventTypeCommandResponse EventType = "command_response"
	// CommandMatched event is emitted when an intent is recognized from natural language.
	EventTypeCommandMatched EventType = "command_matched"
	// CommandResult event is emitted when an intent-based command execution completes.
	EventTypeCommandResult EventType = "command_result"
)

// Command types for CommandRequest events.
const (
	CommandTypeMemoryClear   = "memory_clear"
	CommandTypeMemoryCompact = "memory_compact"
	CommandTypeMemoryStats   = "memory_stats"
	CommandTypeEngineStatus  = "engine_status"
	CommandTypeToolHistory   = "tool_history"
	CommandTypeLLMConfig     = "llm_config"
	CommandTypeSessionRole   = "session_role"
)

// Task tracking events - for LLM-driven task management.
const (
	EventTypeTaskCreate EventType = "task_create"
	EventTypeTaskUpdate EventType = "task_update"
	EventTypeTaskList   EventType = "task_list"
)

// Event is the interface for all events in the system.
// It provides a common contract for event handling across layers.
type Event interface {
	// Type returns the event type.
	Type() EventType
	// Content returns the event content/message.
	Content() string
	// Extra returns additional event data.
	Extra() map[string]any
	// Timestamp returns when the event was created.
	Timestamp() time.Time
	// RequestID returns the request ID for event grouping.
	RequestID() string
	// InteractionType returns the interaction type for interaction events.
	// Returns empty string for non-interaction events.
	InteractionType() InteractionType
}

// BaseEvent is a basic implementation of the Event interface.
type BaseEvent struct {
	eventType EventType
	content   string
	extra     map[string]any
	timestamp time.Time
	requestID string // Unique ID for each user request (tracks event grouping)

	// InteractionType for EventTypeInteractionRequest/Response
	interactionType InteractionType
}

// NewEvent creates a new BaseEvent with the given type, content, and request ID.
// RequestID is required for all events to enable full-chain tracing.
func NewEvent(typ EventType, content string, requestID string) *BaseEvent {
	return &BaseEvent{
		eventType: typ,
		content:   content,
		requestID: requestID,
		timestamp: time.Now(),
	}
}

// NewEventWithExtra creates a new BaseEvent with extra data and request ID.
// RequestID is required for all events to enable full-chain tracing.
func NewEventWithExtra(typ EventType, content string, extra map[string]any, requestID string) *BaseEvent {
	return &BaseEvent{
		eventType: typ,
		content:   content,
		extra:     extra,
		requestID: requestID,
		timestamp: time.Now(),
	}
}

// NewInteractionEvent creates a new BaseEvent for interaction requests/responses.
func NewInteractionEvent(typ EventType, interactionType InteractionType, requestID string, extra map[string]any) *BaseEvent {
	return &BaseEvent{
		eventType:       typ,
		requestID:       requestID,
		extra:           extra,
		interactionType: interactionType,
		timestamp:       time.Now(),
	}
}

// Type implements Event interface.
func (e *BaseEvent) Type() EventType {
	return e.eventType
}

// Content implements Event interface.
func (e *BaseEvent) Content() string {
	return e.content
}

// Extra implements Event interface.
func (e *BaseEvent) Extra() map[string]any {
	return e.extra
}

// Timestamp implements Event interface.
func (e *BaseEvent) Timestamp() time.Time {
	return e.timestamp
}

// RequestID implements Event interface.
func (e *BaseEvent) RequestID() string {
	return e.requestID
}

// InteractionType returns the interaction type for interaction events.
func (e *BaseEvent) InteractionType() InteractionType {
	return e.interactionType
}

// SetInteractionType sets the interaction type.
func (e *BaseEvent) SetInteractionType(typ InteractionType) {
	e.interactionType = typ
}

// ConfirmationRequest represents a request for user confirmation.
// Used when a sensitive operation requires user approval.
// Supports multiple confirmation types via the Type field.
type ConfirmationRequest struct {
	Type       string // "sensitive_tool", "plan_review", or "question"
	ToolName   string
	Params     map[string]any
	PlanGoal   string   // Plan review: the plan goal
	PlanSteps  []string // Plan review: step descriptions
	ResponseCh chan bool

	// AskUserQuestion fields
	Question       string         // The question text to display
	QuestionType   string         // "text", "choice", or "multi_choice"
	Options        []QuestionOption // Available options for choice/multi_choice
	DefaultAnswer  string         // Optional default answer
	QuestionRespCh chan QuestionResponse // Response channel for questions (separate from bool channel)
}

// QuestionOption represents an option in a choice question.
type QuestionOption struct {
	Label       string // Display label
	Description string // Optional description
	Value       string // Value to return if selected
}

// QuestionResponse represents the response to an AskUserQuestion request.
type QuestionResponse struct {
	Answer    string   // Selected answer (text or option value)
	Answers   []string // For multi_choice: all selected values
	Cancelled bool     // User cancelled the question
}

// InteractionResponse represents the response to an interaction request.
// Used for EventTypeInteractionResponse events.
type InteractionResponse struct {
	RequestID  string          // Matches the request ID
	Type       InteractionType // Matches the request type
	Approved   bool            // For tool/plan/rollback confirmation
	Cancelled  bool            // User cancelled
	AnswerText string          // For ask_user_question (text answer)
	Selection  string          // For ask_user_question (single choice)
	Selections []string        // For ask_user_question (multi choice)
	Error      error           // Processing error
}

// InteractionRequest represents a request for user interaction.
// Used internally by Agent for request-response matching.
type InteractionRequest struct {
	ID              string
	Type            InteractionType
	Timeout         time.Duration
	ToolName        string
	ToolParams      map[string]any
	PermissionReason string
	PlanGoal        string
	PlanSteps       []string
	PlanFiles       []string
	Question        string
	QuestionType    string
	Options         []QuestionOption
	DefaultAnswer   string
	RollbackReason  string
	RollbackTarget  string
}

// MemoryStats contains memory statistics.
type MemoryStats struct {
	MessageCount int
	TokenCount   int
	MaxTokens    int
}

// CommandRequest represents a request to execute a command.
// Used for event-driven communication between CommandProvider and Runtime/TUI.
type CommandRequest struct {
	Command    string               // Command type (e.g., CommandTypeMemoryClear)
	Params     map[string]any       // Optional parameters
	ResponseCh chan CommandResponse // Channel for the response
}

// CommandResponse represents the response to a command request.
type CommandResponse struct {
	Success bool   // Whether the command succeeded
	Result  string // Result message or error message
	Error   error  // Error if any
}
