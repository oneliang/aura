// Package trigger provides event trigger interfaces and implementations.
package trigger

import "encoding/json"

// Event represents a trigger event.
type Event struct {
	Source  string         `json:"source"`  // feishu/email/cron/api
	Content string         `json:"content"` // Message content
	Meta    map[string]any `json:"meta,omitempty"`
}

// Trigger defines the interface for event triggers.
type Trigger interface {
	// Name returns the trigger name.
	Name() string
	// Parse parses the raw payload into an Event.
	Parse(payload []byte) (*Event, error)
}

// WebhookTrigger parses generic webhook payloads.
type WebhookTrigger struct {
	SourceName string
}

// NewWebhookTrigger creates a new webhook trigger.
func NewWebhookTrigger(sourceName string) *WebhookTrigger {
	return &WebhookTrigger{SourceName: sourceName}
}

// Name returns the trigger name.
func (t *WebhookTrigger) Name() string {
	return t.SourceName
}

// Parse parses a generic webhook payload.
// Expected format: {"content": "message content", "source": "optional_source"}
func (t *WebhookTrigger) Parse(payload []byte) (*Event, error) {
	var data struct {
		Content string         `json:"content"`
		Source  string         `json:"source"`
		Meta    map[string]any `json:"meta"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	source := data.Source
	if source == "" {
		source = t.SourceName
	}

	return &Event{
		Source:  source,
		Content: data.Content,
		Meta:    data.Meta,
	}, nil
}

// FeishuTrigger parses Feishu webhook payloads.
type FeishuTrigger struct{}

// NewFeishuTrigger creates a new Feishu trigger.
func NewFeishuTrigger() *FeishuTrigger {
	return &FeishuTrigger{}
}

// Name returns the trigger name.
func (t *FeishuTrigger) Name() string {
	return "feishu"
}

// Parse parses a Feishu webhook payload.
// Feishu webhook format: {"text": "message content", "msg_type": "text"}
func (t *FeishuTrigger) Parse(payload []byte) (*Event, error) {
	var data struct {
		Text    string         `json:"text"`
		Content string         `json:"content"` // Alternative field
		MsgType string         `json:"msg_type"`
		Meta    map[string]any `json:"meta"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	content := data.Text
	if content == "" {
		content = data.Content
	}

	return &Event{
		Source:  "feishu",
		Content: content,
		Meta:    data.Meta,
	}, nil
}

// CronTrigger parses cron job payloads.
type CronTrigger struct{}

// NewCronTrigger creates a new cron trigger.
func NewCronTrigger() *CronTrigger {
	return &CronTrigger{}
}

// Name returns the trigger name.
func (t *CronTrigger) Name() string {
	return "cron"
}

// Parse parses a cron job payload.
// Expected format: {"task": "task_name", "content": "message content"}
func (t *CronTrigger) Parse(payload []byte) (*Event, error) {
	var data struct {
		Task    string         `json:"task"`
		Content string         `json:"content"`
		Meta    map[string]any `json:"meta"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	content := data.Content
	if content == "" {
		content = "Execute scheduled task: " + data.Task
	}

	meta := data.Meta
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["task"] = data.Task

	return &Event{
		Source:  "cron",
		Content: content,
		Meta:    meta,
	}, nil
}
