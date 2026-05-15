// Package utility provides utility tools.
package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
)

// DateTimeTool provides date and time information.
type DateTimeTool struct {
	timezone string
}

// DateTimeOption is a configuration option for DateTimeTool.
type DateTimeOption func(*DateTimeTool)

// WithTimezone sets the timezone.
func WithTimezone(tz string) DateTimeOption {
	return func(t *DateTimeTool) {
		t.timezone = tz
	}
}

// NewDateTimeTool creates a new datetime tool.
func NewDateTimeTool(opts ...DateTimeOption) *DateTimeTool {
	t := &DateTimeTool{
		timezone: "Local",
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *DateTimeTool) Name() string {
	return constants.ToolDateTime
}

// Description returns the tool description.
func (t *DateTimeTool) Description() string {
	return "Get current date and time information. Parameters: format (string, optional), timezone (string, optional)"
}

// Execute returns datetime information.
func (t *DateTimeTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	format, _ := params["format"].(string)
	tz, _ := params["timezone"].(string)

	if tz == "" {
		tz = t.timezone
	}

	now := time.Now()

	// Load timezone if specified
	if tz != "" && tz != "Local" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid timezone: %s", tz)}, nil
		}
		now = now.In(loc)
	}

	if format != "" {
		// Translate common non-Go format strings to Go reference-time format
		goFmt := translateTimeFormat(format)
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: now.Format(goFmt),
		}, nil
	}

	// Return comprehensive datetime info
	return &tools.ToolResult{
		Status: tools.ToolStatusSuccess,
		Content: fmt.Sprintf(`Current Date and Time:
  Date: %s
  Time: %s
  Day: %s
  Week: %d
  Year: %d
  Timestamp: %d
  Timezone: %s`,
			now.Format("2006-01-02"),
			now.Format("15:04:05"),
			now.Weekday().String(),
			getWeekNumber(now),
			now.Year(),
			now.Unix(),
			now.Location().String(),
		),
	}, nil
}

// translateTimeFormat converts common format strings (Python/strftime style) to Go's reference-time format.
func translateTimeFormat(f string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"yyyy", "2006",
		"YY", "06",
		"MM", "01",
		"DD", "02",
		"dd", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"SS", "05",
		"%Y", "2006",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%M", "04",
		"%S", "05",
	)
	return replacer.Replace(f)
}

func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// TextTool provides text manipulation utilities.
type TextTool struct{}

// NewTextTool creates a new text tool.
func NewTextTool() *TextTool {
	return &TextTool{}
}

// Name returns the tool name.
func (t *TextTool) Name() string {
	return constants.ToolText
}

// Description returns the tool description.
func (t *TextTool) Description() string {
	return "Text manipulation. Parameters: operation (string, required: count_words, count_chars, uppercase, lowercase, reverse, trim), text (string, required)"
}

// Execute performs text operations.
func (t *TextTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "operation parameter is required"}, nil
	}

	text, ok := params["text"].(string)
	if !ok {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "text parameter is required"}, nil
	}

	switch operation {
	case "count_words":
		words := len(strings.Fields(text))
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Word count: %d", words)}, nil
	case "count_chars":
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Character count: %d", len(text))}, nil
	case "uppercase":
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: strings.ToUpper(text)}, nil
	case "lowercase":
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: strings.ToLower(text)}, nil
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: string(runes)}, nil
	case "trim":
		return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: strings.TrimSpace(text)}, nil
	default:
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("unknown operation: %s", operation)}, nil
	}
}
