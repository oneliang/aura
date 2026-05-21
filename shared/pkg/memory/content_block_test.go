package memory_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/oneliang/aura/shared/pkg/i18n"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

func init() {
	// Initialize i18n with English locale for consistent test results
	i18n.Init("", "en")
}

func TestTextBlockSerialization(t *testing.T) {
	block := sharedmemory.TextBlock{
		Type: sharedmemory.BlockTypeText,
		Text: "Hello, world!",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := `{"type":"text","text":"Hello, world!"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestThinkingBlockSerialization(t *testing.T) {
	block := sharedmemory.ThinkingBlock{
		Type:      sharedmemory.BlockTypeThinking,
		Thinking:  "User is asking about...",
		Signature: "abc123",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled sharedmemory.ThinkingBlock
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != sharedmemory.BlockTypeThinking {
		t.Errorf("expected type %s, got %s", sharedmemory.BlockTypeThinking, unmarshaled.Type)
	}
	if unmarshaled.Thinking != "User is asking about..." {
		t.Errorf("expected thinking content, got %s", unmarshaled.Thinking)
	}
}

func TestToolUseBlockSerialization(t *testing.T) {
	input := json.RawMessage(`{"file_path":"/test/file.go"}`)
	block := sharedmemory.ToolUseBlock{
		Type:  sharedmemory.BlockTypeToolUse,
		ID:    "toolu_abc123",
		Name:  "Read",
		Input: input,
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled sharedmemory.ToolUseBlock
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != "Read" {
		t.Errorf("expected name Read, got %s", unmarshaled.Name)
	}
	if unmarshaled.ID != "toolu_abc123" {
		t.Errorf("expected ID toolu_abc123, got %s", unmarshaled.ID)
	}
}

func TestToolResultBlockSerialization(t *testing.T) {
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "toolu_abc123",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "File content here...",
				},
			},
			IsError: false,
		},
	}

	data, err := sharedmemory.MarshalContentBlocks(blocks)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := sharedmemory.UnmarshalContentBlocks(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled) != 1 {
		t.Errorf("expected 1 block, got %d", len(unmarshaled))
	}

	trb, ok := unmarshaled[0].(sharedmemory.ToolResultBlock)
	if !ok {
		t.Fatalf("expected ToolResultBlock, got %T", unmarshaled[0])
	}
	if trb.ToolUseID != "toolu_abc123" {
		t.Errorf("expected ToolUseID toolu_abc123, got %s", trb.ToolUseID)
	}
}

func TestMarshalContentBlocksTextOnly(t *testing.T) {
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{
			Type: sharedmemory.BlockTypeText,
			Text: "Hello",
		},
	}

	data, err := sharedmemory.MarshalContentBlocks(blocks)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := `[{"type":"text","text":"Hello"}]`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	unmarshaled, err := sharedmemory.UnmarshalContentBlocks(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled) != 1 {
		t.Fatalf("expected 1 block, got %d", len(unmarshaled))
	}

	tb, ok := unmarshaled[0].(sharedmemory.TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", unmarshaled[0])
	}
	if tb.Text != "Hello" {
		t.Errorf("expected text Hello, got %s", tb.Text)
	}
}

func TestMarshalContentBlocksMultiple(t *testing.T) {
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{
			Type: sharedmemory.BlockTypeText,
			Text: "First",
		},
		sharedmemory.ThinkingBlock{
			Type:      sharedmemory.BlockTypeThinking,
			Thinking:  "Analyzing...",
			Signature: "sig1",
		},
		sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    "toolu_123",
			Name:  "Read",
			Input: json.RawMessage(`{"path":"/file.go"}`),
		},
	}

	data, err := sharedmemory.MarshalContentBlocks(blocks)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := sharedmemory.UnmarshalContentBlocks(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(unmarshaled))
	}

	// Verify first block is TextBlock
	if _, ok := unmarshaled[0].(sharedmemory.TextBlock); !ok {
		t.Errorf("expected block 0 to be TextBlock, got %T", unmarshaled[0])
	}

	// Verify second block is ThinkingBlock
	tb, ok := unmarshaled[1].(sharedmemory.ThinkingBlock)
	if !ok {
		t.Errorf("expected block 1 to be ThinkingBlock, got %T", unmarshaled[1])
	} else if tb.Signature != "sig1" {
		t.Errorf("expected signature sig1, got %s", tb.Signature)
	}

	// Verify third block is ToolUseBlock
	tu, ok := unmarshaled[2].(sharedmemory.ToolUseBlock)
	if !ok {
		t.Errorf("expected block 2 to be ToolUseBlock, got %T", unmarshaled[2])
	} else if tu.Name != "Read" {
		t.Errorf("expected name Read, got %s", tu.Name)
	}
}

// TestToContentBlockErrors tests that ToContentBlock returns errors for missing required fields.
func TestToContentBlockErrors(t *testing.T) {
	tests := []struct {
		name        string
		raw         sharedmemory.RawContentBlock
		expectedErr string
	}{
		{
			name: "TextBlock missing text",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.BlockTypeText,
			},
			expectedErr: "TextBlock missing required field: text",
		},
		{
			name: "ThinkingBlock missing thinking",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.BlockTypeThinking,
			},
			expectedErr: "ThinkingBlock missing required field: thinking",
		},
		{
			name: "ToolUseBlock missing id",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.BlockTypeToolUse,
				Name: strPtr("Read"),
			},
			expectedErr: "ToolUseBlock missing required field: id",
		},
		{
			name: "ToolUseBlock missing name",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.BlockTypeToolUse,
				ID:   strPtr("toolu_123"),
			},
			expectedErr: "ToolUseBlock missing required field: name",
		},
		{
			name: "ToolResultBlock missing tool_use_id",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.BlockTypeToolResult,
			},
			expectedErr: "ToolResultBlock missing required field: tool_use_id",
		},
		{
			name: "Unknown content block type",
			raw: sharedmemory.RawContentBlock{
				Type: sharedmemory.ContentBlockType("unknown"),
			},
			expectedErr: "unknown content block type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.raw.ToContentBlock()
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.expectedErr)
			} else if !containsString(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestUnmarshalContentBlocksErrors tests that UnmarshalContentBlocks returns errors for malformed JSON.
func TestUnmarshalContentBlocksErrors(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectedErr string
	}{
		{
			name:        "TextBlock missing text",
			json:        `[{"type":"text"}]`,
			expectedErr: "TextBlock missing required field: text",
		},
		{
			name:        "ThinkingBlock missing thinking",
			json:        `[{"type":"thinking"}]`,
			expectedErr: "ThinkingBlock missing required field: thinking",
		},
		{
			name:        "ToolUseBlock missing id",
			json:        `[{"type":"tool_use","name":"Read"}]`,
			expectedErr: "ToolUseBlock missing required field: id",
		},
		{
			name:        "ToolResultBlock missing tool_use_id",
			json:        `[{"type":"tool_result"}]`,
			expectedErr: "ToolResultBlock missing required field: tool_use_id",
		},
		{
			name:        "Unknown block type",
			json:        `[{"type":"unknown"}]`,
			expectedErr: "unknown content block type: unknown",
		},
		{
			name:        "Nested ToolResultBlock with invalid content",
			json:        `[{"type":"tool_result","tool_use_id":"toolu_123","content":[{"type":"text"}]}]`,
			expectedErr: "ToolResultBlock content error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sharedmemory.UnmarshalContentBlocks([]byte(tt.json))
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.expectedErr)
			} else if !containsString(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestContentBlocksToRawPanicOnUnknownType tests that ContentBlocksToRaw panics for unknown block types.
func TestContentBlocksToRawPanicOnUnknownType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unknown content block type, but did not panic")
		} else {
			errMsg := fmt.Sprintf("%v", r)
			if !containsString(errMsg, "unknown content block type") {
				t.Errorf("expected panic message to contain 'unknown content block type', got %q", errMsg)
			}
		}
	}()

	// Create an unknown block type
	unknownBlock := struct {
		sharedmemory.TextBlock // embed to satisfy interface
	}{
		TextBlock: sharedmemory.TextBlock{Type: "unknown_type", Text: "test"},
	}

	// This should panic
	_ = sharedmemory.ContentBlocksToRaw([]sharedmemory.ContentBlock{unknownBlock})
}

// TestValidUnmarshalAfterError tests that valid data can be unmarshaled after fixing errors.
func TestValidUnmarshalAfterError(t *testing.T) {
	// This test verifies that the error messages help identify the issue
	invalidJSON := `[{"type":"text"}]`
	_, err := sharedmemory.UnmarshalContentBlocks([]byte(invalidJSON))
	if err == nil {
		t.Fatal("expected error for missing text field")
	}

	// Fix the JSON
	validJSON := `[{"type":"text","text":"Hello"}]`
	blocks, err := sharedmemory.UnmarshalContentBlocks([]byte(validJSON))
	if err != nil {
		t.Fatalf("expected valid JSON to unmarshal, got error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	tb, ok := blocks[0].(sharedmemory.TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", blocks[0])
	}
	if tb.Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", tb.Text)
	}
}

func strPtr(s string) *string {
	return &s
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}