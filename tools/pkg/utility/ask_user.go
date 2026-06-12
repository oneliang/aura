// Package utility provides utility tools.
package utility

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
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
	return `主动向用户提问以澄清需求或获取决策。

使用场景：
- 需求不明确或有多种理解
- 存在多种可行方案需要用户选择
- 涉及关键决策点（架构、技术选型、用户体验）
- 你不确定用户的真实意图

不要使用：
- 任务已经明确且你理解正确
- 可以自己做出合理判断的细节
- 是之前讨论过的延续
- 琐碎的实现细节

参数说明：
- question: 问题文本（必填）
- type: 问题类型 - "text"（文本输入）、"choice"（单选）、"multi_choice"（多选）
- options: 选项列表，仅 choice/multi_choice 类型使用，每个选项是字符串

示例 1（选择题）：
{
  "question": "你想优化哪个方面的性能？",
  "type": "choice",
  "options": ["页面加载速度", "数据库查询", "API 延迟"]
}

示例 2（文本输入）：
{
  "question": "请描述一下你期望的用户体验流程",
  "type": "text"
}

示例 3（多选题）：
{
  "question": "你希望支持哪些数据库？",
  "type": "multi_choice",
  "options": ["PostgreSQL", "MySQL", "MongoDB", "Redis"]
}`
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
			Content: "用户取消了回答",
		}, nil
	}

	// Format the response based on question type
	var answer string
	if questionType == "multi_choice" && len(response.Answers) > 0 {
		answer = strings.Join(response.Answers, ", ")
	} else if response.Answer != "" {
		answer = response.Answer
	} else {
		answer = "(无回答)"
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf("用户回答: %s", answer),
	}, nil
}
