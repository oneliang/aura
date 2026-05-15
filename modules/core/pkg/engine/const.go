// Package engine provides the core agent implementation.
package engine

import (
	"time"

	"github.com/oneliang/aura/shared/pkg/i18n"
)

// DefaultSystemPrompt is the default system prompt used when no custom prompt is provided.
// It tries to get a localized version from i18n, falling back to English if not available.
var DefaultSystemPrompt = getDefaultPrompt()

func getDefaultPrompt() string {
	prompt := i18n.T("agent.system_prompt")
	if prompt == "agent.system_prompt" || prompt == "" {
		return "You are a helpful AI assistant."
	}
	return prompt
}

// Tool event truncation limits.
const (
	toolEndEventTruncateLen = 500 // max chars in tool end event content
	toolEndResultPreviewLen = 200 // max chars in tool result preview
)

// Parallel tool execution.
const (
	defaultMaxParallelTools = 5 // default max concurrent tool executions
)

// Action parsing constants.
const (
	actionPrefixLen     = 7  // len("action:")
	parametersPrefixLen = 11 // len("parameters:")
)

// Task complexity heuristics.
const (
	complexTaskWordThreshold = 20 // words to consider task complex
)

// Exploration phase constants.
const (
	explorationMaxSteps     = 3  // max LLM iterations in exploration phase
	planFilenameGoalMax     = 30 // max characters for goal portion of plan filename
	defaultMaxParallelExplore = 3 // default max concurrent exploration agents
)

// Rollback confirmation timeout.
const rollbackConfirmationTimeout = 60 * time.Second // timeout for rollback confirmation wait

// Default read-only tools available during exploration.
var explorationTools = []string{
	"file_read", "file_search", "file_list",
	"glob", "grep", "code_navigate",
}
