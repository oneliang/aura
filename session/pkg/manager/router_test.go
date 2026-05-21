package manager

import (
	"testing"

	"github.com/oneliang/aura/session/pkg/model"
)

func TestRouter_MatchSession(t *testing.T) {
	tests := []struct {
		name      string
		sessions  []*model.Session
		source    string
		content   string
		wantMatch bool
		wantID    string
	}{
		{
			name:      "no sessions",
			sessions:  []*model.Session{},
			source:    "cli",
			content:   "hello",
			wantMatch: false,
		},
		{
			name: "match by source only",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Feishu Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "feishu", Trigger: "", Active: true},
					},
				},
			},
			source:    "feishu",
			content:   "any message",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "match by trigger keyword only",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Monitor Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "", Trigger: "监控", Active: true},
					},
				},
			},
			source:    "api",
			content:   "系统监控告警",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "match by both source and trigger",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Feishu Monitor",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "feishu", Trigger: "告警", Active: true},
					},
				},
			},
			source:    "feishu",
			content:   "收到告警：CPU 使用率过高",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "source wildcard matches all",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Universal Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "*", Trigger: "帮助", Active: true},
					},
				},
			},
			source:    "email",
			content:   "请帮助我",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "inactive subscription should not match",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Inactive Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "feishu", Trigger: "测试", Active: false},
					},
				},
			},
			source:    "feishu",
			content:   "测试消息",
			wantMatch: false,
		},
		{
			name: "case insensitive trigger match",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Test Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "", Trigger: "TEST", Active: true},
					},
				},
			},
			source:    "cli",
			content:   "this is a test message",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "multiple sessions - first match wins",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "First Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "", Trigger: "通用", Active: true},
					},
				},
				{
					ID:   "session2",
					Name: "Second Session",
					Subscriptions: []model.Subscription{
						{ID: "sub2", Source: "", Trigger: "通用", Active: true},
					},
				},
			},
			source:    "api",
			content:   "通用消息",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "no match - source mismatch",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Feishu Only",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "feishu", Trigger: "", Active: true},
					},
				},
			},
			source:    "email",
			content:   "hello",
			wantMatch: false,
		},
		{
			name: "no match - trigger mismatch",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Alert Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "", Trigger: "告警", Active: true},
					},
				},
			},
			source:    "cli",
			content:   "普通消息，一切正常",
			wantMatch: false,
		},
		{
			name: "empty source in subscription matches any source",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Any Source Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "", Trigger: "测试", Active: true},
					},
				},
			},
			source:    "any_source",
			content:   "测试",
			wantMatch: true,
			wantID:    "session1",
		},
		{
			name: "multiple subscriptions in one session - any matches",
			sessions: []*model.Session{
				{
					ID:   "session1",
					Name: "Multi Sub Session",
					Subscriptions: []model.Subscription{
						{ID: "sub1", Source: "feishu", Trigger: "关键词 1", Active: true},
						{ID: "sub2", Source: "email", Trigger: "关键词 2", Active: true},
					},
				},
			},
			source:    "email",
			content:   "包含关键词 2 的消息",
			wantMatch: true,
			wantID:    "session1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			got := router.MatchSession(tt.sessions, tt.source, tt.content)

			if tt.wantMatch {
				if got == "" {
					t.Errorf("Router.MatchSession() = %v, want match for %q", got, tt.content)
				}
				if tt.wantID != "" && got != tt.wantID {
					t.Errorf("Router.MatchSession() = %v, want %v", got, tt.wantID)
				}
			} else {
				if got != "" {
					t.Errorf("Router.MatchSession() = %v, want no match", got)
				}
			}
		})
	}
}

func TestRouter_MatchSession_SourcePriority(t *testing.T) {
	// Test that source matching works correctly with trigger matching
	sessions := []*model.Session{
		{
			ID:   "session1",
			Name: "Feishu Specific",
			Subscriptions: []model.Subscription{
				{ID: "sub1", Source: "feishu", Trigger: "告警", Active: true},
			},
		},
		{
			ID:   "session2",
			Name: "Generic",
			Subscriptions: []model.Subscription{
				{ID: "sub2", Source: "", Trigger: "告警", Active: true},
			},
		},
	}

	router := NewRouter()

	// Feishu source should match session1 (more specific)
	got := router.MatchSession(sessions, "feishu", "收到告警")
	if got != "session1" {
		t.Errorf("Expected session1 for feishu source, got %v", got)
	}

	// Email source should match session2 (generic)
	got = router.MatchSession(sessions, "email", "收到告警")
	if got != "session2" {
		t.Errorf("Expected session2 for email source, got %v", got)
	}
}
