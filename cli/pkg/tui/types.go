package tui

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	"github.com/oneliang/aura/core/pkg/sdk"
)

// DefaultSessionNameFormat is the format string for auto-generated session names.
const DefaultSessionNameFormat = "session_%s"

// Event type constants — re-exported from SDK for TUI convenience.
const (
	EventTypeThinkingStart       = sdk.EventTypeThinkingStart
	EventTypeThinkingChunk       = sdk.EventTypeThinkingChunk  // NEW: thinking stream chunks
	EventTypeThinkingEnd         = sdk.EventTypeThinkingEnd
	EventTypeAction              = sdk.EventTypeAction
	EventTypeResult              = sdk.EventTypeResult
	EventTypeResponse            = sdk.EventTypeResponse
	EventTypeResponseStart       = sdk.EventTypeResponseStart  // NEW: response stream start
	EventTypeResponseChunk       = sdk.EventTypeResponseChunk
	EventTypeResponseEnd         = sdk.EventTypeResponseEnd    // NEW: response stream end
	EventTypeThinkingContent     = sdk.EventTypeThinkingContent // Deprecated: use ThinkingChunk
	EventTypeError               = sdk.EventTypeError
	EventTypeDone                = sdk.EventTypeDone
	EventTypeStep                = sdk.EventTypeStep
	EventTypeToolStart           = sdk.EventTypeToolStart
	EventTypeToolEnd             = sdk.EventTypeToolEnd
	EventTypeConfirmationRequest = sdk.EventTypeConfirmationRequest
	EventTypeCommandMatched      = sdk.EventTypeCommandMatched
	EventTypeCommandResult       = sdk.EventTypeCommandResult
	EventTypeTaskCreate          = sdk.EventTypeTaskCreate
	EventTypeTaskUpdate          = sdk.EventTypeTaskUpdate
	EventTypeTaskList            = sdk.EventTypeTaskList
	EventTypePlanCreated         = sdk.EventTypePlanCreated
	EventTypePlanReviewStart     = sdk.EventTypePlanReviewStart
	EventTypePlanReviewFiles     = sdk.EventTypePlanReviewFiles
	EventTypePlanStep            = sdk.EventTypePlanStep
	EventTypePlanComplete        = sdk.EventTypePlanComplete
	EventTypePlanModeExit        = sdk.EventTypePlanModeExit
	EventTypeEnterPlanMode       = sdk.EventTypeEnterPlanMode
	EventTypePlanVerifyStart     = sdk.EventTypePlanVerifyStart
	EventTypePlanVerifyResult    = sdk.EventTypePlanVerifyResult
	EventTypePlanVerifyEnd       = sdk.EventTypePlanVerifyEnd
	EventTypeSnapshotCreated     = sdk.EventTypeSnapshotCreated
	EventTypeRollbackOffer       = sdk.EventTypeRollbackOffer
	EventTypeRollbackComplete    = sdk.EventTypeRollbackComplete
	EventTypeMaxStepsExceeded    = sdk.EventTypeMaxStepsExceeded
)

// CommandHandler handles a slash command.
type CommandHandler func(m Model, input string) (tea.Model, tea.Cmd)

// Command represents a slash command.
type Command struct {
	Name        string         // Primary name (e.g., "/help")
	Description string         // Short description
	Handler     CommandHandler // Handler function
	Aliases     []string       // Aliases (e.g., ["/quit", "/q"] for "/exit")
}

// commandRegistry holds all registered commands.
var commandRegistry = make(map[string]*Command)

// RegisterCommand registers a command and its aliases.
func RegisterCommand(cmd *Command) {
	commandRegistry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		commandRegistry[alias] = cmd
	}
}

// GetCommand returns a command by name or alias.
func GetCommand(name string) *Command {
	return commandRegistry[name]
}

// GetAllCommands returns all unique commands sorted by name (stable order).
func GetAllCommands() []*Command {
	seen := make(map[string]bool)
	var result []*Command
	for name, cmd := range commandRegistry {
		if name == cmd.Name && !seen[cmd.Name] {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}
	// Sort by name for stable order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// CommandInfo represents a slash command for display.
type CommandInfo struct {
	Name        string
	Description string
}

// GetAvailableCommands returns commands for completion display.
func GetAvailableCommands() []CommandInfo {
	cmds := GetAllCommands()
	result := make([]CommandInfo, 0, len(cmds))
	for _, cmd := range cmds {
		result = append(result, CommandInfo{
			Name:        cmd.Name,
			Description: cmd.Description,
		})
	}
	return result
}

// MessageType represents the type of a chat message.
type MessageType int

const (
	MessageTypeUser MessageType = iota
	MessageTypeAssistant
	MessageTypeThinking
	MessageTypeToolStart
	MessageTypeToolEnd
	MessageTypeError
	MessageTypeSystem
)

// Message represents a rendered message in the chat history.
type Message struct {
	ID            string
	SourceID      string // Source identifier (reserved for future AgentID support)
	Type          MessageType
	Content       string
	Rendered      string
	RenderedWidth int    // Width at which content was last rendered (for cache invalidation)
	Timestamp     time.Time
	Extra         map[string]any
	Complete      bool   // Marked complete (streaming ended), no cursor displayed
}

// ChatEvent represents a single event from the agent.
type ChatEvent struct {
	Type       sdk.EventType
	Content    string
	Extra      map[string]any
	ResponseCh chan bool // Direct response channel for confirmations
	RequestID  string    // Unique ID for each user request (tracks event grouping)
}

// RunFunc is the function signature for running the agent.
type RunFunc func(ctx context.Context, input string) (<-chan ChatEvent, error)

// Config holds TUI configuration.
type Config struct {
	Mode             string
	UserName         string
	Tools            []string
	ShowTokens       bool
	TokenMax         int
	DebugMode        bool
	SessionID        string
	SkillDirectories []string          // Directories to load skills from
	Keybindings      map[string]string // User-defined keybindings (key combination → command)
	EnableReview     bool              // Enable plan review before execution
}

// UIStyles holds all styles for the TUI.
type UIStyles struct {
	UserMessage         lipglossv2.Style
	AuraMessage         lipglossv2.Style
	Thinking            lipglossv2.Style
	Action              lipglossv2.Style
	Result              lipglossv2.Style
	Error               lipglossv2.Style
	Help                lipglossv2.Style
	Processing          lipglossv2.Style
	Timestamp           lipglossv2.Style
	Separator           lipglossv2.Style
	Command             lipglossv2.Style
	CommandItemSelected lipglossv2.Style
	CommandItem         lipglossv2.Style
	CommandDesc         lipglossv2.Style
	// Tool block styles
	ToolBlockHeader  lipglossv2.Style // Header (bold, green)
	ToolBlockIn      lipglossv2.Style // IN label style (bold, green)
	ToolBlockInBg    lipglossv2.Style // IN content with subtle background
	ToolBlockOut     lipglossv2.Style // OUT label style
	ToolBlockDone    lipglossv2.Style // Done text style
	ToolBlockBorder  lipglossv2.Style // Left border for complete tool block
	Duration         lipglossv2.Style // Execution duration style
	// Task widget styles
	TaskWidgetTitle  lipglossv2.Style
	TaskWidgetBorder lipglossv2.Style
	TaskInProgress   lipglossv2.Style // Style for in_progress tasks (highlight)
	// Plan widget styles
	PlanWidgetTitle  lipglossv2.Style
	PlanWidgetBorder lipglossv2.Style
}

// ConfirmationType represents the type of confirmation request.
type ConfirmationType string

const (
	ConfirmationSensitiveTool ConfirmationType = "sensitive_tool"
	ConfirmationPlanReview    ConfirmationType = "plan_review"
	ConfirmationQuestion      ConfirmationType = "question"
	ConfirmationRollback      ConfirmationType = "rollback" // Rollback after execution/verification failure
)

// QuestionType represents the type of question.
type QuestionType string

const (
	QuestionTypeText       QuestionType = "text"
	QuestionTypeChoice     QuestionType = "choice"
	QuestionTypeMultiChoice QuestionType = "multi_choice"
)

// QuestionOption represents an option in a choice question.
type QuestionOption struct {
	Label       string
	Description string
	Value       string
}

// ConfirmationRequest represents a Y/N confirmation request or a question.
// Supports sensitive tool, plan review, and question types.
type ConfirmationRequest struct {
	Type       ConfirmationType
	ToolName   string
	Params     map[string]any
	Message    string
	PlanGoal   string   // Plan review: the plan goal
	PlanSteps  []string // Plan review: step descriptions
	ResponseCh chan bool

	// AskUserQuestion fields
	Question       string           // The question text to display
	QuestionType   QuestionType     // "text", "choice", or "multi_choice"
	Options        []QuestionOption // Available options for choice/multi_choice
	DefaultAnswer  string           // Optional default answer
	QuestionRespCh chan QuestionResponse // Response channel for questions
}

// QuestionResponse represents the response to an AskUserQuestion request.
type QuestionResponse struct {
	Answer    string   // Selected answer (text or option value)
	Answers   []string // For multi_choice: all selected values
	Cancelled bool     // User cancelled the question
}

// ConfirmState represents the state when waiting for user confirmation or question.
type ConfirmState struct {
	Waiting  bool
	Request  *ConfirmationRequest
	Selected int            // 0 = Yes/First option, 1 = No/Second option, etc.
	SelectedOptions []int   // For multi_choice: indices of selected options
	TextInput string        // For text questions: the user's input
}

// ModelProvider holds a reference to the current Model.
type ModelProvider struct {
	model *Model
}

// Get returns the current model.
func (p *ModelProvider) Get() *Model {
	return p.model
}

// Set sets the current model.
func (p *ModelProvider) Set(m *Model) {
	p.model = m
}
