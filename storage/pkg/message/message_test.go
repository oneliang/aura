// Package message provides tests for the message data structure.
package message

import (
	"encoding/json"
	"testing"

	"github.com/oneliang/aura/shared/pkg/memory"
)

func TestMessage_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{
			name: "basic message with ContentBlocks",
			msg: Message{
				SessionID: "test-session",
				Role:      "user",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello, world!"},
				},
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
		{
			name: "message with source",
			msg: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hi there!"},
				},
				Timestamp: 1234567890000,
				Source:    "cli",
			},
			wantErr: false,
		},
		{
			name: "empty message",
			msg: Message{
				SessionID: "",
				Role:      "",
				Timestamp: 0,
			},
			wantErr: false,
		},
		{
			name: "message with unicode",
			msg: Message{
				SessionID: "test-session",
				Role:      "user",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "你好，世界！🌍"},
				},
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
		{
			name: "message with thinking and text",
			msg: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Let me think..."},
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Response"},
				},
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
		{
			name: "message with metadata fields",
			msg: Message{
				SessionID:     "test-session",
				UserID:        "user-123",
				Role:          "user",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
				},
				Timestamp:     1234567890000,
				UUID:          "msg-uuid-123",
				ParentUUID:    "parent-uuid-456",
				Subtype:       "question",
				IsSidechain:   true,
				CWD:           "/home/user/project",
				GitBranch:     "feature-branch",
			},
			wantErr: false,
		},
		{
			name: "message with usage",
			msg: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Response"},
				},
				Timestamp: 1234567890000,
				Usage: Usage{
					InputTokens:  100,
					OutputTokens: 200,
					TotalTokens:  300,
				},
			},
			wantErr: false,
		},
		{
			name: "message with compact metadata",
			msg: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Response"},
				},
				Timestamp: 1234567890000,
				CompactMetadata: CompactMetadata{
					CompactedAt:     1234567891000,
					CompactionRatio: 0.5,
					Summary:         "Previous conversation summary",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if data == nil {
				t.Error("Message.MarshalJSON() returned nil data")
			}
		})
	}
}

func TestMessage_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Message
		wantErr bool
	}{
		{
			name: "basic message with content_blocks",
			json: `{"session_id":"test-session","role":"user","content_blocks":[{"type":"text","text":"Hello"}],"timestamp":1234567890000}`,
			want: Message{
				SessionID: "test-session",
				Role:      "user",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
				},
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
		{
			name: "message with source",
			json: `{"session_id":"test-session","role":"assistant","content_blocks":[{"type":"text","text":"Hi"}],"timestamp":1234567890000,"source":"feishu"}`,
			want: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hi"},
				},
				Timestamp: 1234567890000,
				Source:    "feishu",
			},
			wantErr: false,
		},
		{
			name:    "empty json",
			json:    `{}`,
			want:    Message{},
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			want:    Message{},
			wantErr: true,
		},
		{
			name:    "not a json object",
			json:    `"not an object"`,
			want:    Message{},
			wantErr: true,
		},
		{
			name: "message with metadata fields",
			json: `{"session_id":"test-session","role":"user","content_blocks":[{"type":"text","text":"Hello"}],"timestamp":1234567890000,"uuid":"msg-uuid-123","parent_uuid":"parent-uuid-456","subtype":"question","is_sidechain":true,"cwd":"/home/user/project","git_branch":"feature-branch"}`,
			want: Message{
				SessionID: "test-session",
				Role:      "user",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
				},
				Timestamp:     1234567890000,
				UUID:          "msg-uuid-123",
				ParentUUID:    "parent-uuid-456",
				Subtype:       "question",
				IsSidechain:   true,
				CWD:           "/home/user/project",
				GitBranch:     "feature-branch",
			},
			wantErr: false,
		},
		{
			name: "message with usage",
			json: `{"session_id":"test-session","role":"assistant","content_blocks":[{"type":"text","text":"Response"}],"timestamp":1234567890000,"usage":{"input_tokens":100,"output_tokens":200,"total_tokens":300}}`,
			want: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Response"},
				},
				Timestamp: 1234567890000,
				Usage: Usage{
					InputTokens:  100,
					OutputTokens: 200,
					TotalTokens:  300,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Message
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.SessionID != tt.want.SessionID {
					t.Errorf("SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
				}
				if got.Role != tt.want.Role {
					t.Errorf("Role = %q, want %q", got.Role, tt.want.Role)
				}
				if got.Timestamp != tt.want.Timestamp {
					t.Errorf("Timestamp = %d, want %d", got.Timestamp, tt.want.Timestamp)
				}
				if got.Source != tt.want.Source {
					t.Errorf("Source = %q, want %q", got.Source, tt.want.Source)
				}
				// Compare ContentBlocks length
				if len(got.ContentBlocks) != len(tt.want.ContentBlocks) {
					t.Errorf("ContentBlocks length = %d, want %d", len(got.ContentBlocks), len(tt.want.ContentBlocks))
				}
			}
		})
	}
}

func TestMessage_RoundTrip(t *testing.T) {
	original := Message{
		SessionID:     "test-session-123",
		Role:          "user",
		ContentBlocks: []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Test message with unicode: 你好世界"},
		},
		Timestamp:     1234567890000,
		Source:        "api",
		UUID:          "msg-uuid-789",
		ParentUUID:    "parent-uuid-012",
		Subtype:       "question",
		IsSidechain:   false,
		CWD:           "/home/user/project",
		GitBranch:     "main",
		Usage: Usage{
			InputTokens:  50,
			OutputTokens: 100,
			TotalTokens:  150,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: %q != %q", decoded.SessionID, original.SessionID)
	}
	if decoded.Role != original.Role {
		t.Errorf("Role mismatch: %q != %q", decoded.Role, original.Role)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: %d != %d", decoded.Timestamp, original.Timestamp)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source mismatch: %q != %q", decoded.Source, original.Source)
	}
	if decoded.UUID != original.UUID {
		t.Errorf("UUID mismatch: %q != %q", decoded.UUID, original.UUID)
	}
	if decoded.ParentUUID != original.ParentUUID {
		t.Errorf("ParentUUID mismatch: %q != %q", decoded.ParentUUID, original.ParentUUID)
	}
	if decoded.Subtype != original.Subtype {
		t.Errorf("Subtype mismatch: %q != %q", decoded.Subtype, original.Subtype)
	}
	if decoded.IsSidechain != original.IsSidechain {
		t.Errorf("IsSidechain mismatch: %v != %v", decoded.IsSidechain, original.IsSidechain)
	}
	if decoded.CWD != original.CWD {
		t.Errorf("CWD mismatch: %q != %q", decoded.CWD, original.CWD)
	}
	if decoded.GitBranch != original.GitBranch {
		t.Errorf("GitBranch mismatch: %q != %q", decoded.GitBranch, original.GitBranch)
	}
	if decoded.Usage != original.Usage {
		t.Errorf("Usage mismatch: %+v != %+v", decoded.Usage, original.Usage)
	}
}

func TestMessage_GetContentBlocks(t *testing.T) {
	tests := []struct {
		name       string
		msg        Message
		wantLen    int
	}{
		{
			name: "message with ContentBlocks set",
			msg: Message{
				SessionID: "test-session",
				Role:      "assistant",
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: "Block content"},
					memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking..."},
				},
				Timestamp: 1234567890000,
			},
			wantLen: 2,
		},
		{
			name: "message with empty ContentBlocks",
			msg: Message{
				SessionID:     "test-session",
				Role:          "user",
				ContentBlocks: nil,
				Timestamp:     1234567890000,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.GetContentBlocks()

			if len(got) != tt.wantLen {
				t.Errorf("GetContentBlocks() returned %d blocks, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestMessage_SetContentBlocks(t *testing.T) {
	msg := Message{
		SessionID: "test-session",
		Role:      "assistant",
		Timestamp: 1234567890000,
	}

	blocks := []memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: "Hello"},
		memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Thinking..."},
	}
	msg.SetContentBlocks(blocks)

	if len(msg.ContentBlocks) != 2 {
		t.Errorf("ContentBlocks length = %d, want 2", len(msg.ContentBlocks))
	}

	got := msg.GetContentBlocks()
	if len(got) != 2 {
		t.Errorf("GetContentBlocks() returned %d blocks, want 2", len(got))
	}
}

func TestMessage_ContentBlocksRoundTrip(t *testing.T) {
	// Test that ContentBlocks are properly serialized and deserialized
	original := Message{
		SessionID:     "test-session",
		Role:          "assistant",
		ContentBlocks: []memory.ContentBlock{
			memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Let me think..."},
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Here is my response"},
			memory.ToolUseBlock{
				Type:  memory.BlockTypeToolUse,
				ID:    "tool-123",
				Name:  "read_file",
				Input: json.RawMessage(`{"path": "/test/file.txt"}`),
			},
		},
		Timestamp: 1234567890000,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify ContentBlocks were preserved
	decodedBlocks := decoded.GetContentBlocks()
	if len(decodedBlocks) != 3 {
		t.Fatalf("GetContentBlocks() returned %d blocks, want 3", len(decodedBlocks))
	}

	// Check first block is ThinkingBlock
	if _, ok := decodedBlocks[0].(memory.ThinkingBlock); !ok {
		t.Errorf("First block is not ThinkingBlock, got %T", decodedBlocks[0])
	}

	// Check second block is TextBlock
	if _, ok := decodedBlocks[1].(memory.TextBlock); !ok {
		t.Errorf("Second block is not TextBlock, got %T", decodedBlocks[1])
	}

	// Check third block is ToolUseBlock
	toolBlock, ok := decodedBlocks[2].(memory.ToolUseBlock)
	if !ok {
		t.Errorf("Third block is not ToolUseBlock, got %T", decodedBlocks[2])
	}
	if toolBlock.ID != "tool-123" {
		t.Errorf("ToolUseBlock.ID = %q, want %q", toolBlock.ID, "tool-123")
	}
	if toolBlock.Name != "read_file" {
		t.Errorf("ToolUseBlock.Name = %q, want %q", toolBlock.Name, "read_file")
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Test that new message type constants exist
	if MessageTypeToolResult != "tool_result" {
		t.Errorf("MessageTypeToolResult = %q, want %q", MessageTypeToolResult, "tool_result")
	}
	if MessageTypeCompact != "compact" {
		t.Errorf("MessageTypeCompact = %q, want %q", MessageTypeCompact, "compact")
	}
}

func TestUsageStruct(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 200,
		TotalTokens:  300,
	}

	if usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", usage.OutputTokens)
	}
	if usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", usage.TotalTokens)
	}
}

func TestCompactMetadataStruct(t *testing.T) {
	meta := CompactMetadata{
		CompactedAt:     1234567890000,
		CompactionRatio: 0.5,
		Summary:         "Previous conversation summary",
	}

	if meta.CompactedAt != 1234567890000 {
		t.Errorf("CompactedAt = %d, want 1234567890000", meta.CompactedAt)
	}
	if meta.CompactionRatio != 0.5 {
		t.Errorf("CompactionRatio = %f, want 0.5", meta.CompactionRatio)
	}
	if meta.Summary != "Previous conversation summary" {
		t.Errorf("Summary = %q, want %q", meta.Summary, "Previous conversation summary")
	}
}

func TestMessage_Type(t *testing.T) {
	// Test that Message.Type works with new types
	msg := Message{
		SessionID:     "test-session",
		Type:          MessageTypeToolResult,
		Role:          "assistant",
		ContentBlocks: []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "Tool result content"},
		},
		Timestamp:     1234567890000,
	}

	if msg.Type != MessageTypeToolResult {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeToolResult)
	}

	// Test marshaling/unmarshaling with new type
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.Type != MessageTypeToolResult {
		t.Errorf("Decoded Type = %q, want %q", decoded.Type, MessageTypeToolResult)
	}
}

func TestMessage_NoContentField(t *testing.T) {
	// Verify that "content" field does NOT exist in marshaled JSON
	msg := Message{
		SessionID:     "test-session",
		Role:          "user",
		ContentBlocks: []memory.ContentBlock{
			memory.TextBlock{Type: memory.BlockTypeText, Text: "test"},
		},
		Timestamp: 1234567890000,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify "content" field does NOT exist (backward compat removed)
	if _, exists := raw["content"]; exists {
		t.Error("Expected 'content' field to NOT exist after backward compat removal")
	}

	// Verify "content_blocks" field exists
	if _, exists := raw["content_blocks"]; !exists {
		t.Error("Expected 'content_blocks' field to exist")
	}
}