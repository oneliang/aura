package trigger

import (
	"encoding/json"
	"testing"
)

func TestFeishuTrigger_Parse(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    *Event
		wantErr bool
	}{
		{
			name:    "valid text message",
			payload: []byte(`{"text": "Hello from Feishu", "msg_type": "text"}`),
			want: &Event{
				Source:  "feishu",
				Content: "Hello from Feishu",
			},
			wantErr: false,
		},
		{
			name:    "valid content field",
			payload: []byte(`{"content": "Hello from Feishu content field"}`),
			want: &Event{
				Source:  "feishu",
				Content: "Hello from Feishu content field",
			},
			wantErr: false,
		},
		{
			name:    "text takes precedence over content",
			payload: []byte(`{"text": "Text message", "content": "Content message"}`),
			want: &Event{
				Source:  "feishu",
				Content: "Text message",
			},
			wantErr: false,
		},
		{
			name:    "with meta field",
			payload: []byte(`{"text": "Test", "meta": {"user_id": "123", "channel": "general"}}`),
			want: &Event{
				Source:  "feishu",
				Content: "Test",
				Meta:    map[string]any{"user_id": "123", "channel": "general"},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: []byte(`{invalid json}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty payload",
			payload: []byte(`{}`),
			want: &Event{
				Source:  "feishu",
				Content: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := NewFeishuTrigger()
			got, err := trigger.Parse(tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("FeishuTrigger.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Source != tt.want.Source {
					t.Errorf("Source = %v, want %v", got.Source, tt.want.Source)
				}
				if got.Content != tt.want.Content {
					t.Errorf("Content = %v, want %v", got.Content, tt.want.Content)
				}
				if !jsonEqual(got.Meta, tt.want.Meta) {
					t.Errorf("Meta = %v, want %v", got.Meta, tt.want.Meta)
				}
			}
		})
	}
}

func TestCronTrigger_Parse(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    *Event
		wantErr bool
	}{
		{
			name:    "valid task with content",
			payload: []byte(`{"task": "daily_report", "content": "Generate daily report"}`),
			want: &Event{
				Source:  "cron",
				Content: "Generate daily report",
				Meta:    map[string]any{"task": "daily_report"},
			},
			wantErr: false,
		},
		{
			name:    "task without content",
			payload: []byte(`{"task": "cleanup"}`),
			want: &Event{
				Source:  "cron",
				Content: "Execute scheduled task: cleanup",
				Meta:    map[string]any{"task": "cleanup"},
			},
			wantErr: false,
		},
		{
			name:    "with meta field",
			payload: []byte(`{"task": "backup", "content": "Run backup", "meta": {"priority": "high"}}`),
			want: &Event{
				Source:  "cron",
				Content: "Run backup",
				Meta:    map[string]any{"task": "backup", "priority": "high"},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := NewCronTrigger()
			got, err := trigger.Parse(tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("CronTrigger.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Source != tt.want.Source {
					t.Errorf("Source = %v, want %v", got.Source, tt.want.Source)
				}
				if got.Content != tt.want.Content {
					t.Errorf("Content = %v, want %v", got.Content, tt.want.Content)
				}
				if !jsonEqual(got.Meta, tt.want.Meta) {
					t.Errorf("Meta = %v, want %v", got.Meta, tt.want.Meta)
				}
			}
		})
	}
}

func TestWebhookTrigger_Parse(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		payload    []byte
		want       *Event
		wantErr    bool
	}{
		{
			name:       "valid webhook with content",
			sourceName: "slack",
			payload:    []byte(`{"content": "Hello from Slack"}`),
			want: &Event{
				Source:  "slack",
				Content: "Hello from Slack",
			},
			wantErr: false,
		},
		{
			name:       "valid webhook with explicit source",
			sourceName: "slack",
			payload:    []byte(`{"content": "Hello", "source": "custom_source"}`),
			want: &Event{
				Source:  "custom_source",
				Content: "Hello",
			},
			wantErr: false,
		},
		{
			name:       "empty source uses default",
			sourceName: "email",
			payload:    []byte(`{"content": "Email content", "source": ""}`),
			want: &Event{
				Source:  "email",
				Content: "Email content",
			},
			wantErr: false,
		},
		{
			name:       "with meta field",
			sourceName: "teams",
			payload:    []byte(`{"content": "Teams message", "meta": {"sender": "bot"}}`),
			want: &Event{
				Source:  "teams",
				Content: "Teams message",
				Meta:    map[string]any{"sender": "bot"},
			},
			wantErr: false,
		},
		{
			name:       "invalid JSON",
			sourceName: "test",
			payload:    []byte(`{bad json}`),
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := NewWebhookTrigger(tt.sourceName)
			got, err := trigger.Parse(tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("WebhookTrigger.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Source != tt.want.Source {
					t.Errorf("Source = %v, want %v", got.Source, tt.want.Source)
				}
				if got.Content != tt.want.Content {
					t.Errorf("Content = %v, want %v", got.Content, tt.want.Content)
				}
				if !jsonEqual(got.Meta, tt.want.Meta) {
					t.Errorf("Meta = %v, want %v", got.Meta, tt.want.Meta)
				}
			}
		})
	}
}

func TestTrigger_Name(t *testing.T) {
	tests := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{"FeishuTrigger", NewFeishuTrigger(), "feishu"},
		{"CronTrigger", NewCronTrigger(), "cron"},
		{"WebhookTrigger", NewWebhookTrigger("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trigger.Name(); got != tt.want {
				t.Errorf("Trigger.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

// jsonEqual compares two JSON objects for equality.
func jsonEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}

	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)

	var aMap, bMap map[string]any
	json.Unmarshal(aBytes, &aMap)
	json.Unmarshal(bBytes, &bMap)

	if len(aMap) != len(bMap) {
		return false
	}

	for k, v := range aMap {
		if bMap[k] != v {
			return false
		}
	}

	return true
}
