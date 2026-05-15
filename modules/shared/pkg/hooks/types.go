// Package hooks provides a server-wide hooks mechanism for external script
// integration at key execution points (tool use, session lifecycle, etc.).
package hooks

// HookEventType represents a hook trigger point in the execution lifecycle.
type HookEventType string

const (
	EventSessionStart     HookEventType = "SessionStart"
	EventUserPromptSubmit HookEventType = "UserPromptSubmit"
	EventPreToolUse       HookEventType = "PreToolUse"
	EventPostToolUse      HookEventType = "PostToolUse"
	EventPostToolUseFail  HookEventType = "PostToolUseFailure"
	EventStop             HookEventType = "Stop"
	EventStopFailure      HookEventType = "StopFailure"
	EventSubagentStop     HookEventType = "SubagentStop"
	EventPreCompact       HookEventType = "PreCompact"
	EventPostCompact      HookEventType = "PostCompact"
	EventTaskCreated      HookEventType = "TaskCreated"
	EventTaskCompleted    HookEventType = "TaskCompleted"
	EventFileChanged      HookEventType = "FileChanged"
	EventSessionEnd       HookEventType = "SessionEnd"
	EventPreResponse      HookEventType = "PreResponse" // Before emitting final response (blocking)
	// Plan mode hooks
	EventEnterPlanMode HookEventType = "EnterPlanMode"
	EventExitPlanMode  HookEventType = "ExitPlanMode"
)

// HookExitCode represents special exit codes from hook subprocesses.
const (
	HookExitCodeNormal   = 0
	HookExitCodeBlocking = 2 // Hook signals that the main flow should be blocked
)

// HookResult holds the outcome of a hook execution.
type HookResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Parsed   *HookOutput
}

// HookOutput is the parsed JSON output from a hook's stdout.
// Hooks can influence the main flow by returning these fields.
type HookOutput struct {
	// SystemMessage adds a message to the conversation context.
	SystemMessage string `json:"systemMessage,omitempty"`

	// Continue controls whether the main flow proceeds.
	// nil or true = continue; false = block.
	Continue *bool `json:"continue,omitempty"`

	// StopReason provides a reason when Continue=false.
	StopReason string `json:"stopReason,omitempty"`

	// UpdatedInput replaces the original input for the event.
	UpdatedInput any `json:"updatedInput,omitempty"`

	// AdditionalContext provides extra context for downstream processing.
	AdditionalContext string `json:"additionalContext,omitempty"`

	// PermissionDecision overrides the default permission decision.
	PermissionDecision string `json:"permissionDecision,omitempty"`

	// PermissionReason provides the reason for the permission decision.
	PermissionReason string `json:"permissionDecisionReason,omitempty"`

	// SuppressOutput hides the output from the user.
	SuppressOutput bool `json:"suppressOutput,omitempty"`

	// ShouldRetry signals that the operation should be retried (for PreResponse hooks).
	ShouldRetry bool `json:"shouldRetry,omitempty"`

	// RetryReason provides the reason for retry (for PreResponse hooks).
	RetryReason string `json:"retryReason,omitempty"`

	// ReflectionFeedback provides quality feedback (for PreResponse hooks).
	ReflectionFeedback string `json:"reflectionFeedback,omitempty"`
}
