package memory

import (
	"encoding/json"
	"testing"
)

func TestMessageContentBlocks(t *testing.T) {
	// Create message with content blocks directly
	msg := Message{
		Role:          "assistant",
		ContentBlocks: []ContentBlock{
			TextBlock{Type: BlockTypeText, Text: "Hello!"},
			ThinkingBlock{Type: BlockTypeThinking, Thinking: "Let me think..."},
		},
	}

	blocks := msg.GetContentBlocks()
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 content blocks, got %d", len(blocks))
	}

	// First block should be TextBlock
	textBlock, ok := blocks[0].(TextBlock)
	if !ok {
		t.Fatalf("Expected first block to be TextBlock, got %T", blocks[0])
	}
	if textBlock.Text != "Hello!" {
		t.Errorf("Expected text 'Hello!', got %q", textBlock.Text)
	}

	// Second block should be ThinkingBlock
	thinkBlock, ok := blocks[1].(ThinkingBlock)
	if !ok {
		t.Fatalf("Expected second block to be ThinkingBlock, got %T", blocks[1])
	}
	if thinkBlock.Thinking != "Let me think..." {
		t.Errorf("Expected thinking 'Let me think...', got %q", thinkBlock.Thinking)
	}
}

func TestMessageSetContentBlocks(t *testing.T) {
	msg := Message{Role: "assistant"}

	blocks := []ContentBlock{
		TextBlock{Type: BlockTypeText, Text: "First text"},
		TextBlock{Type: BlockTypeText, Text: "Second text"},
	}
	msg.SetContentBlocks(blocks)

	// GetContentBlocks should return the same blocks
	gotBlocks := msg.GetContentBlocks()
	if len(gotBlocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(gotBlocks))
	}
}

func TestMessageWithToolUseBlock(t *testing.T) {
	input := json.RawMessage(`{"query":"test"}`)
	msg := Message{
		Role:          "assistant",
		ContentBlocks: []ContentBlock{
			TextBlock{Type: BlockTypeText, Text: "I'll help with that."},
			ToolUseBlock{Type: BlockTypeToolUse, ID: "tool-1", Name: "search", Input: input},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	blocks := decoded.GetContentBlocks()
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	toolBlock, ok := blocks[1].(ToolUseBlock)
	if !ok {
		t.Fatalf("Expected ToolUseBlock, got %T", blocks[1])
	}
	if toolBlock.ID != "tool-1" {
		t.Errorf("Expected ID 'tool-1', got %q", toolBlock.ID)
	}
	if toolBlock.Name != "search" {
		t.Errorf("Expected name 'search', got %q", toolBlock.Name)
	}
}

func TestMessageWithToolResultBlock(t *testing.T) {
	msg := Message{
		Role:          "user",
		ContentBlocks: []ContentBlock{
			ToolResultBlock{
				Type:      BlockTypeToolResult,
				ToolUseID: "tool-1",
				Content: []ContentBlock{
					TextBlock{Type: BlockTypeText, Text: "result data"},
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	blocks := decoded.GetContentBlocks()
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}

	resultBlock, ok := blocks[0].(ToolResultBlock)
	if !ok {
		t.Fatalf("Expected ToolResultBlock, got %T", blocks[0])
	}
	if resultBlock.ToolUseID != "tool-1" {
		t.Errorf("Expected ToolUseID 'tool-1', got %q", resultBlock.ToolUseID)
	}
	if len(resultBlock.Content) != 1 {
		t.Fatalf("Expected 1 nested content block, got %d", len(resultBlock.Content))
	}
}

func TestMessageEmptyContent(t *testing.T) {
	msg := Message{Role: "user"}

	blocks := msg.GetContentBlocks()
	if blocks != nil {
		t.Errorf("Expected nil blocks for empty message, got %v", blocks)
	}

	// Marshal and unmarshal empty message
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ContentBlocks != nil {
		t.Errorf("Expected nil ContentBlocks, got %v", decoded.ContentBlocks)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	// Test that marshaling and unmarshaling preserves all data
	original := Message{
		Role:          "assistant",
		Type:          MessageTypeAssistant,
		ContentBlocks: []ContentBlock{
			TextBlock{Type: BlockTypeText, Text: "Hello!"},
			ThinkingBlock{Type: BlockTypeThinking, Thinking: "Thinking...", Signature: "sig123"},
			ToolUseBlock{Type: BlockTypeToolUse, ID: "t1", Name: "test", Input: json.RawMessage(`{"a":1}`)},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role mismatch: expected %q, got %q", original.Role, decoded.Role)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, decoded.Type)
	}

	originalBlocks := original.GetContentBlocks()
	decodedBlocks := decoded.GetContentBlocks()

	if len(decodedBlocks) != len(originalBlocks) {
		t.Fatalf("Block count mismatch: expected %d, got %d", len(originalBlocks), len(decodedBlocks))
	}

	// Verify TextBlock
	tb1, ok1 := decodedBlocks[0].(TextBlock)
	tb2, ok2 := originalBlocks[0].(TextBlock)
	if !ok1 || !ok2 || tb1.Text != tb2.Text {
		t.Errorf("TextBlock mismatch")
	}

	// Verify ThinkingBlock
	th1, ok1 := decodedBlocks[1].(ThinkingBlock)
	th2, ok2 := originalBlocks[1].(ThinkingBlock)
	if !ok1 || !ok2 || th1.Thinking != th2.Thinking || th1.Signature != th2.Signature {
		t.Errorf("ThinkingBlock mismatch")
	}

	// Verify ToolUseBlock
	tu1, ok1 := decodedBlocks[2].(ToolUseBlock)
	tu2, ok2 := originalBlocks[2].(ToolUseBlock)
	if !ok1 || !ok2 || tu1.ID != tu2.ID || tu1.Name != tu2.Name {
		t.Errorf("ToolUseBlock mismatch")
	}
}

func TestMessageContentBlocksFieldInJSON(t *testing.T) {
	// Verify that JSON field is named "content_blocks"
	msg := Message{
		Role:          "user",
		ContentBlocks: []ContentBlock{
			TextBlock{Type: BlockTypeText, Text: "test"},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check that the JSON contains "content_blocks" array
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	contentBlocks, ok := raw["content_blocks"].([]interface{})
	if !ok {
		t.Errorf("Expected 'content_blocks' to be an array, got %T", raw["content_blocks"])
	}
	if len(contentBlocks) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(contentBlocks))
	}

	// Verify "content" field does NOT exist (backward compat removed)
	if _, exists := raw["content"]; exists {
		t.Error("Expected 'content' field to NOT exist after backward compat removal")
	}
}