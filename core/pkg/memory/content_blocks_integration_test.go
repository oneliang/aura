// Package memory provides session-based memory implementation.
package memory

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/storage/pkg/jsonl"
	"github.com/oneliang/aura/storage/pkg/message"
)

// TestContentBlocksIntegration tests the complete end-to-end flow of
// ContentBlocks from creation through persistence and loading.
// This test verifies the entire ContentBlocks system works as designed.
func TestContentBlocksIntegration(t *testing.T) {
	// Step 1: Setup - Create temporary directory and JSONL store
	tmpDir, err := os.MkdirTemp("", "content-blocks-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "integration-test-session"
	userID := "test-user"

	// Save session metadata
	err = store.SaveSession(&model.Session{
		ID:     sessionID,
		Name:   "Integration Test Session",
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create SessionMemory with config
	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
		Source:    SourceCLI,
	})
	if err != nil {
		t.Fatalf("Failed to create SessionMemory: %v", err)
	}

	// Step 2: Add user message (simple text)
	t.Log("Step 2: Adding user message with simple text")
	userBlocks1 := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{
			Type: sharedmemory.BlockTypeText,
			Text: "What is the weather in San Francisco?",
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleUser, userBlocks1, sharedmemory.MessageTypeUser)

	// Step 3: Add assistant message with ThinkingBlock + ToolUseBlock
	t.Log("Step 3: Adding assistant message with ThinkingBlock + ToolUseBlock")
	toolUseInput := json.RawMessage(`{"location": "San Francisco", "unit": "celsius"}`)
	assistantBlocks1 := []sharedmemory.ContentBlock{
		sharedmemory.ThinkingBlock{
			Type:      sharedmemory.BlockTypeThinking,
			Thinking:  "The user wants to know the weather. I should use the weather tool to get this information.",
			Signature: "sig_abc123",
		},
		sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    "tool_use_001",
			Name:  "get_weather",
			Input: toolUseInput,
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleAssistant, assistantBlocks1, sharedmemory.MessageTypeAssistant)

	// Step 4: Add tool_result message (user role) with ToolResultBlock containing nested TextBlock
	t.Log("Step 4: Adding tool_result message with ToolResultBlock")
	userBlocks2 := []sharedmemory.ContentBlock{
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "tool_use_001",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Weather in San Francisco: 18C, sunny with light clouds",
				},
			},
			IsError: false,
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleUser, userBlocks2, sharedmemory.MessageTypeUser)

	// Step 5: Add final assistant response with TextBlock
	t.Log("Step 5: Adding final assistant response with TextBlock")
	assistantBlocks2 := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{
			Type: sharedmemory.BlockTypeText,
			Text: "Based on the weather data, San Francisco is currently 18C with sunny conditions and light clouds. Its a nice day for outdoor activities!",
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleAssistant, assistantBlocks2, sharedmemory.MessageTypeAssistant)

	// Step 6: Verify in-memory state (messages should be in correct order)
	t.Log("Step 6: Verifying in-memory messages")
	memMsgs := memory.Get()
	if len(memMsgs) != 4 {
		t.Fatalf("Expected 4 messages in memory, got %d", len(memMsgs))
	}

	// Verify in-memory message order and ContentBlocks (exact order)
	verifyMemoryMessage(t, memMsgs[0], sharedmemory.RoleUser, userBlocks1, "In-memory Message 1")
	verifyMemoryMessage(t, memMsgs[1], sharedmemory.RoleAssistant, assistantBlocks1, "In-memory Message 2")
	verifyMemoryMessage(t, memMsgs[2], sharedmemory.RoleUser, userBlocks2, "In-memory Message 3")
	verifyMemoryMessage(t, memMsgs[3], sharedmemory.RoleAssistant, assistantBlocks2, "In-memory Message 4")

	// Wait for async persistence to complete
	time.Sleep(500 * time.Millisecond)

	// Step 7: Verify all 4 messages are persisted to JSONL storage
	t.Log("Step 7: Verifying messages persisted to JSONL storage")
	ctx := context.Background()
	storedMsgs, err := store.MessageStore().Get(ctx, sessionID, 100, userID)
	if err != nil {
		t.Fatalf("Failed to get stored messages: %v", err)
	}

	if len(storedMsgs) != 4 {
		t.Fatalf("Expected 4 messages persisted, got %d", len(storedMsgs))
	}

	// Step 8: Verify ContentBlocks persistence by content matching (async may reorder)
	// Find and verify each expected message by its content signature
	t.Log("Step 8: Verifying ContentBlocks persistence details (content-based matching)")
	findAndVerifyStoredMessage(t, storedMsgs, sharedmemory.RoleUser, userBlocks1, "Stored: User message 1")
	findAndVerifyStoredMessage(t, storedMsgs, sharedmemory.RoleAssistant, assistantBlocks1, "Stored: Assistant message 1")
	findAndVerifyStoredMessage(t, storedMsgs, sharedmemory.RoleUser, userBlocks2, "Stored: Tool result message")
	findAndVerifyStoredMessage(t, storedMsgs, sharedmemory.RoleAssistant, assistantBlocks2, "Stored: Final assistant message")

	// Step 9: Create a NEW SessionMemory instance (simulates restart)
	t.Log("Step 9: Creating new SessionMemory instance to simulate restart")
	memory2, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
		Source:    SourceCLI,
	})
	if err != nil {
		t.Fatalf("Failed to create second SessionMemory: %v", err)
	}

	// Step 10: Verify all 4 messages are loaded from storage
	t.Log("Step 10: Verifying messages loaded from storage after restart")
	loadedMsgs := memory2.Get()
	if len(loadedMsgs) != 4 {
		t.Fatalf("Expected 4 messages loaded, got %d", len(loadedMsgs))
	}

	// Step 11: Verify ContentBlocks are correctly loaded after restart (content-based matching)
	t.Log("Step 11: Verifying ContentBlocks loaded correctly after restart")
	findAndVerifyLoadedMessage(t, loadedMsgs, sharedmemory.RoleUser, userBlocks1, "Loaded: User message 1")
	findAndVerifyLoadedMessage(t, loadedMsgs, sharedmemory.RoleAssistant, assistantBlocks1, "Loaded: Assistant message 1")
	findAndVerifyLoadedMessage(t, loadedMsgs, sharedmemory.RoleUser, userBlocks2, "Loaded: Tool result message")
	findAndVerifyLoadedMessage(t, loadedMsgs, sharedmemory.RoleAssistant, assistantBlocks2, "Loaded: Final assistant message")

	t.Log("Integration test completed successfully - all ContentBlocks persisted and loaded correctly")
}

// Helper function to find and verify a stored message by matching content
func findAndVerifyStoredMessage(t *testing.T, msgs []message.Message, expectedRole string, expectedBlocks []sharedmemory.ContentBlock, label string) {
	t.Helper()
	for _, msg := range msgs {
		if msg.Role != expectedRole {
			continue
		}
		blocks := msg.GetContentBlocks()
		if contentBlocksMatch(blocks, expectedBlocks) {
			// Found matching message, verify details
			verifyContentBlocks(t, blocks, expectedBlocks, label)
			return
		}
	}
	t.Errorf("%s: No matching message found in storage", label)
}

// Helper function to find and verify a loaded message by matching content
func findAndVerifyLoadedMessage(t *testing.T, msgs []sharedmemory.Message, expectedRole string, expectedBlocks []sharedmemory.ContentBlock, label string) {
	t.Helper()
	for _, msg := range msgs {
		if msg.Role != expectedRole {
			continue
		}
		blocks := msg.GetContentBlocks()
		if contentBlocksMatch(blocks, expectedBlocks) {
			// Found matching message, verify details
			verifyContentBlocks(t, blocks, expectedBlocks, label)
			return
		}
	}
	t.Errorf("%s: No matching message found", label)
}

// contentBlocksMatch returns true if blocks match expected blocks (content-based comparison)
func contentBlocksMatch(actual []sharedmemory.ContentBlock, expected []sharedmemory.ContentBlock) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, exp := range expected {
		act := actual[i]
		switch expBlock := exp.(type) {
		case sharedmemory.TextBlock:
			actBlock, ok := act.(sharedmemory.TextBlock)
			if !ok || actBlock.Text != expBlock.Text {
				return false
			}
		case sharedmemory.ThinkingBlock:
			actBlock, ok := act.(sharedmemory.ThinkingBlock)
			if !ok || actBlock.Thinking != expBlock.Thinking || actBlock.Signature != expBlock.Signature {
				return false
			}
		case sharedmemory.ToolUseBlock:
			actBlock, ok := act.(sharedmemory.ToolUseBlock)
			if !ok || actBlock.ID != expBlock.ID || actBlock.Name != expBlock.Name {
				return false
			}
		case sharedmemory.ToolResultBlock:
			actBlock, ok := act.(sharedmemory.ToolResultBlock)
			if !ok || actBlock.ToolUseID != expBlock.ToolUseID || actBlock.IsError != expBlock.IsError {
				return false
			}
			// Check nested content blocks
			if !contentBlocksMatch(actBlock.Content, expBlock.Content) {
				return false
			}
		}
	}
	return true
}

// Helper function to verify in-memory message
func verifyMemoryMessage(t *testing.T, msg sharedmemory.Message, expectedRole string, expectedBlocks []sharedmemory.ContentBlock, label string) {
	t.Helper()

	if msg.Role != expectedRole {
		t.Errorf("%s: Role = %s, want %s", label, msg.Role, expectedRole)
	}

	loadedBlocks := msg.GetContentBlocks()
	verifyContentBlocks(t, loadedBlocks, expectedBlocks, label)
}

// Helper function to verify content blocks match expected blocks
func verifyContentBlocks(t *testing.T, actual []sharedmemory.ContentBlock, expected []sharedmemory.ContentBlock, label string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("%s: Expected %d ContentBlocks, got %d", label, len(expected), len(actual))
		return
	}

	for i, exp := range expected {
		act := actual[i]
		switch expBlock := exp.(type) {
		case sharedmemory.TextBlock:
			actBlock, ok := act.(sharedmemory.TextBlock)
			if !ok {
				t.Errorf("%s Block %d: Expected TextBlock, got %T", label, i, act)
				continue
			}
			if actBlock.Text != expBlock.Text {
				t.Errorf("%s Block %d: Text = %s, want %s", label, i, actBlock.Text, expBlock.Text)
			}
		case sharedmemory.ThinkingBlock:
			actBlock, ok := act.(sharedmemory.ThinkingBlock)
			if !ok {
				t.Errorf("%s Block %d: Expected ThinkingBlock, got %T", label, i, act)
				continue
			}
			if actBlock.Thinking != expBlock.Thinking {
				t.Errorf("%s Block %d: Thinking = %s, want %s", label, i, actBlock.Thinking, expBlock.Thinking)
			}
			if actBlock.Signature != expBlock.Signature {
				t.Errorf("%s Block %d: Signature = %s, want %s", label, i, actBlock.Signature, expBlock.Signature)
			}
		case sharedmemory.ToolUseBlock:
			actBlock, ok := act.(sharedmemory.ToolUseBlock)
			if !ok {
				t.Errorf("%s Block %d: Expected ToolUseBlock, got %T", label, i, act)
				continue
			}
			if actBlock.ID != expBlock.ID {
				t.Errorf("%s Block %d: ID = %s, want %s", label, i, actBlock.ID, expBlock.ID)
			}
			if actBlock.Name != expBlock.Name {
				t.Errorf("%s Block %d: Name = %s, want %s", label, i, actBlock.Name, expBlock.Name)
			}
		case sharedmemory.ToolResultBlock:
			actBlock, ok := act.(sharedmemory.ToolResultBlock)
			if !ok {
				t.Errorf("%s Block %d: Expected ToolResultBlock, got %T", label, i, act)
				continue
			}
			if actBlock.ToolUseID != expBlock.ToolUseID {
				t.Errorf("%s Block %d: ToolUseID = %s, want %s", label, i, actBlock.ToolUseID, expBlock.ToolUseID)
			}
			if actBlock.IsError != expBlock.IsError {
				t.Errorf("%s Block %d: IsError = %v, want %v", label, i, actBlock.IsError, expBlock.IsError)
			}
			// Verify nested content blocks
			verifyContentBlocks(t, actBlock.Content, expBlock.Content, label+" Nested")
		}
	}
}

// TestContentBlocksIntegration_ToolResultBlockWithMultipleNestedBlocks tests
// ToolResultBlock with multiple nested content blocks.
func TestContentBlocksIntegration_ToolResultBlockWithMultipleNestedBlocks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "content-blocks-multi-nested-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "multi-nested-test"
	userID := "test-user"

	err = store.SaveSession(&model.Session{
		ID:     sessionID,
		Name:   "Multi Nested Test",
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create SessionMemory: %v", err)
	}

	// Add tool result with multiple nested blocks (text + additional info)
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "tool_multi",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Primary result text",
				},
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Additional context information",
				},
			},
			IsError: false,
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleUser, blocks, sharedmemory.MessageTypeUser)

	// Verify in-memory state
	memMsgs := memory.Get()
	if len(memMsgs) != 1 {
		t.Fatalf("Expected 1 message in memory, got %d", len(memMsgs))
	}
	verifyMemoryMessage(t, memMsgs[0], sharedmemory.RoleUser, blocks, "In-memory")

	// Wait for async persistence
	time.Sleep(300 * time.Millisecond)

	// Verify persistence
	ctx := context.Background()
	storedMsgs, err := store.MessageStore().Get(ctx, sessionID, 100, userID)
	if err != nil {
		t.Fatalf("Failed to get stored messages: %v", err)
	}
	if len(storedMsgs) != 1 {
		t.Fatalf("Expected 1 stored message, got %d", len(storedMsgs))
	}
	verifyContentBlocks(t, storedMsgs[0].GetContentBlocks(), blocks, "Stored")

	// Simulate restart and verify loading
	memory2, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create second SessionMemory: %v", err)
	}

	loadedMsgs := memory2.Get()
	if len(loadedMsgs) != 1 {
		t.Fatalf("Expected 1 loaded message, got %d", len(loadedMsgs))
	}
	verifyContentBlocks(t, loadedMsgs[0].GetContentBlocks(), blocks, "Loaded")
}

// TestContentBlocksIntegration_ErrorToolResult tests ToolResultBlock with IsError=true.
func TestContentBlocksIntegration_ErrorToolResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "content-blocks-error-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "error-tool-result-test"
	userID := "test-user"

	err = store.SaveSession(&model.Session{
		ID:     sessionID,
		Name:   "Error Tool Result Test",
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create SessionMemory: %v", err)
	}

	// Add error tool result
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "tool_error",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Error: API rate limit exceeded",
				},
			},
			IsError: true,
		},
	}
	memory.AddWithBlocks(sharedmemory.RoleUser, blocks, sharedmemory.MessageTypeUser)

	// Verify in-memory state (IsError should be true)
	memMsgs := memory.Get()
	if len(memMsgs) != 1 {
		t.Fatalf("Expected 1 message in memory, got %d", len(memMsgs))
	}

	memBlocks := memMsgs[0].GetContentBlocks()
	if trb, ok := memBlocks[0].(sharedmemory.ToolResultBlock); ok {
		if trb.IsError != true {
			t.Errorf("In-memory ToolResultBlock.IsError = %v, want true", trb.IsError)
		}
	} else {
		t.Errorf("In-memory: Expected ToolResultBlock, got %T", memBlocks[0])
	}

	// Wait for async persistence
	time.Sleep(300 * time.Millisecond)

	// Verify persistence (IsError should be persisted)
	ctx := context.Background()
	storedMsgs, err := store.MessageStore().Get(ctx, sessionID, 100, userID)
	if err != nil {
		t.Fatalf("Failed to get stored messages: %v", err)
	}
	if len(storedMsgs) != 1 {
		t.Fatalf("Expected 1 stored message, got %d", len(storedMsgs))
	}

	storedBlocks := storedMsgs[0].GetContentBlocks()
	if trb, ok := storedBlocks[0].(sharedmemory.ToolResultBlock); ok {
		if trb.IsError != true {
			t.Errorf("Stored ToolResultBlock.IsError = %v, want true", trb.IsError)
		}
	} else {
		t.Errorf("Stored: Expected ToolResultBlock, got %T", storedBlocks[0])
	}

	// Simulate restart
	memory2, err := NewSessionMemoryWithConfig(sessionID, userID, store.MessageStore(), SessionMemoryConfig{
		MaxLen:    100,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create second SessionMemory: %v", err)
	}

	loadedMsgs := memory2.Get()
	if len(loadedMsgs) != 1 {
		t.Fatalf("Expected 1 loaded message, got %d", len(loadedMsgs))
	}

	loadedBlocks := loadedMsgs[0].GetContentBlocks()
	if trb, ok := loadedBlocks[0].(sharedmemory.ToolResultBlock); ok {
		if trb.IsError != true {
			t.Errorf("Loaded ToolResultBlock.IsError = %v, want true", trb.IsError)
		}
	} else {
		t.Errorf("Loaded: Expected ToolResultBlock, got %T", loadedBlocks[0])
	}
}

// TestContentBlocksIntegration_DirectPersistence tests persistence using direct store access
// to verify ContentBlock serialization without async race conditions.
func TestContentBlocksIntegration_DirectPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "content-blocks-direct-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	msgStore, err := jsonl.NewMessageStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create message store: %v", err)
	}

	sessionID := "direct-persistence-test"
	userID := "test-user"
	ctx := context.Background()

	// Create and persist a message with complex ContentBlocks directly
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.ThinkingBlock{
			Type:      sharedmemory.BlockTypeThinking,
			Thinking:  "Direct persistence test thinking",
			Signature: "sig_direct",
		},
		sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    "tool_direct_001",
			Name:  "test_tool",
			Input: json.RawMessage(`{"param": "value"}`),
		},
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "tool_direct_001",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Nested result text",
				},
				sharedmemory.TextBlock{
					Type: sharedmemory.BlockTypeText,
					Text: "Additional nested info",
				},
			},
			IsError: false,
		},
	}

	msg := message.Message{
		SessionID: sessionID,
		UserID:    userID,
		Type:      sharedmemory.MessageTypeAssistant,
		Role:      sharedmemory.RoleAssistant,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	msg.SetContentBlocks(blocks)

	// Persist directly
	if err := msgStore.Append(ctx, &msg); err != nil {
		t.Fatalf("Failed to append message: %v", err)
	}

	// Load from storage
	storedMsgs, err := msgStore.Get(ctx, sessionID, 100, userID)
	if err != nil {
		t.Fatalf("Failed to get stored messages: %v", err)
	}
	if len(storedMsgs) != 1 {
		t.Fatalf("Expected 1 stored message, got %d", len(storedMsgs))
	}

	// Verify all blocks persisted correctly
	storedBlocks := storedMsgs[0].GetContentBlocks()
	verifyContentBlocks(t, storedBlocks, blocks, "Direct Persistence")
}