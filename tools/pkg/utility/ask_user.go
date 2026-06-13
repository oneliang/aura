// Package utility provides utility tools.
package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	tools "github.com/oneliang/aura/tools/pkg"
)

// AskUserQuestionTool allows the agent to proactively ask the user questions
// to clarify requirements or get decisions.
type AskUserQuestionTool struct {
	askFn func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error)
}

// NewAskUserQuestionTool creates a new ask_user_question tool.
func NewAskUserQuestionTool(askFn func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error)) *AskUserQuestionTool {
	return &AskUserQuestionTool{
		askFn: askFn,
	}
}

// Name returns the tool name.
func (t *AskUserQuestionTool) Name() string {
	return constants.ToolAskUserQuestion
}

// Description returns the tool description.
func (t *AskUserQuestionTool) Description() string {
	return `Proactively ask the user questions to clarify requirements or get decisions.

When to use:
- Requirements are unclear or have multiple interpretations
- Multiple viable solutions exist and user needs to choose
- Key decision points (architecture, tech stack, UX)
- You are uncertain about the user's true intent

When NOT to use:
- Task is clear and you understand it correctly
- Details where you can make a reasonable judgment
- Continuation of a previous discussion
- Trivial implementation details

Parameters:
- question: The question text (required)
- type: Question type - "text" (free input), "choice" (single select), "multi_choice" (multi select)
- options: Option list, only for choice/multi_choice types, each option is a string

IMPORTANT: NEVER include "Other", "其它", "其他" in your options list. An "Other" option with custom text input is AUTOMATICALLY appended by the system.

Example 1 (single choice):
{
  "question": "Which aspect of performance do you want to optimize?",
  "type": "choice",
  "options": ["Page load speed", "Database queries", "API latency"]
}

Example 2 (text input):
{
  "question": "Please describe the user experience flow you expect",
  "type": "text"
}

Example 3 (multi choice):
{
  "question": "Which databases do you want to support?",
  "type": "multi_choice",
  "options": ["PostgreSQL", "MySQL", "MongoDB", "Redis"]
}`
}

// Timeout returns 0 to indicate no timeout — this tool waits for user input indefinitely.
func (t *AskUserQuestionTool) Timeout() time.Duration {
	return 0
}

// Execute asks the user a question and returns their response.
func (t *AskUserQuestionTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Parse question parameter
	question, ok := params["question"].(string)
	if !ok || question == "" {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  "question parameter is required",
		}, nil
	}

	// Parse type parameter (default to "text")
	questionType, ok := params["type"].(string)
	if !ok || questionType == "" {
		questionType = "text"
	}

	// Validate question type
	if questionType != "text" && questionType != "choice" && questionType != "multi_choice" {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("invalid question type: %s, must be text, choice, or multi_choice", questionType),
		}, nil
	}

	// Parse options parameter
	var options []events.QuestionOption
	if opts, ok := params["options"].([]any); ok {
		for _, opt := range opts {
			if s, ok := opt.(string); ok {
				options = append(options, events.QuestionOption{
					Label: s,
					Value: s,
				})
			}
		}
	}

	// Validate that choice/multi_choice types have options
	if (questionType == "choice" || questionType == "multi_choice") && len(options) == 0 {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  "options parameter is required for choice and multi_choice question types",
		}, nil
	}

	// Always append "Other" option for choice/multi_choice, allowing custom text input.
	// The LLM is instructed NOT to include "Other" — this is Aura-controlled.
	if questionType == "choice" || questionType == "multi_choice" {
		options = append(options, events.QuestionOption{
			Label: i18n.T("tui.question.other"),
			Value: constants.OtherOptionValue,
		})
	}

	// Call the ask function
	if t.askFn == nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  "ask user question handler not configured",
		}, nil
	}

	response, err := t.askFn(ctx, question, options, questionType)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to ask question: %v", err),
		}, nil
	}

	// Handle cancelled response
	if response.Cancelled {
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: i18n.T("tool.ask_user.cancelled"),
		}, nil
	}

	// Format the response based on question type
	var answer string
	if questionType == "multi_choice" && len(response.Answers) > 0 {
		answer = strings.Join(response.Answers, ", ")
	} else if response.Answer != "" {
		answer = response.Answer
	} else {
		answer = i18n.T("tool.ask_user.no_answer")
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf(i18n.T("tool.ask_user.answer"), answer),
	}, nil
}
