package constants

// Engine messages
const (
	MessageInterrupted = "\n\n[Interrupted by user]"
)

// Engine configuration defaults
const (
	InputQueueBufferSize     = 10
	DoneEventSendTimeout     = 5000 // milliseconds
	ToolAbandonmentTimeout   = 5000 // milliseconds
	MaxParallelTools         = 5
	ComplexTaskWordThreshold = 20
)

// Action parsing prefixes
const (
	ActionPrefix     = "action:"
	ParametersPrefix = "parameters:"
)

// Tool event display
const (
	ToolEndEventTruncateLen = 500
	ToolEndResultPreviewLen = 200
)

// Observation format strings
const (
	ObservationFormat        = "[Observation: %s]: %s"
	ObservationErrorFormat   = "[Observation: %s]: Error: %v"
	ObservationWithDataFmt   = "[Observation: %s]: %s\n\nStructured Data: %s"
	ObservationDataImageHint = "DataURI: data:image/"
)

// Tool execution error messages
const (
	MsgToolNotFound         = "tool not found: %s"
	MsgToolNotFoundShort    = "tool not found"
	MsgInvalidParams        = "Invalid parameters: %s"
	MsgMissingRequiredParam = "missing required parameter: %s"
	MsgParamTypeMismatch    = "parameter %q expected type %s, got %T"
	MsgDeniedByUser         = "Tool execution denied by user."
	MsgHookBlocked          = "Tool execution blocked by hook: %s"
	MsgHookBlockedReason    = "hook blocked execution"
	MsgNilToolResult        = "tool %s returned nil result"
)
