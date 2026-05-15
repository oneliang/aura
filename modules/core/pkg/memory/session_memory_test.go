package memory

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// extractTextFromMsgBlocks extracts the text content from message ContentBlocks for testing.
func extractTextFromMsgBlocks(blocks []sharedmemory.ContentBlock) string {
	for _, block := range blocks {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// setupTestSessionMemory creates a temporary SessionMemory for testing.
func setupTestSessionMemory(t *testing.T) (*SessionMemory, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "session-memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-session"

	// Save session metadata
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test Session",
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to save session: %v", err)
	}

	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen: 10,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create memory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return memory, cleanup
}

func TestNewSessionMemoryWithConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-new-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-new-session"

	// Save session first
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen: 10,
	})
	if err != nil {
		t.Fatalf("NewSessionMemoryWithConfig() error = %v", err)
	}

	if memory == nil {
		t.Error("Expected memory to be created")
	}
	if memory.sessionID != sessionID {
		t.Errorf("sessionID = %v, want %v", memory.sessionID, sessionID)
	}
	if memory.maxLen != 10 {
		t.Errorf("maxLen = %v, want 10", memory.maxLen)
	}
}

func TestSessionMemory_Add(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	memory.AddWithType(sharedmemory.RoleUser, "Hello", sharedmemory.MessageTypeUser)

	got := memory.Get()
	if len(got) != 1 {
		t.Errorf("Expected 1 message, got %d", len(got))
	}
	if got[0].Role != "user" {
		t.Errorf("Role = %v, want user", got[0].Role)
	}
	if extractTextFromMsgBlocks(got[0].GetContentBlocks()) != "Hello" {
		t.Errorf("Content = %v, want Hello", extractTextFromMsgBlocks(got[0].GetContentBlocks()))
	}
}

func TestSessionMemory_Add_MaxLength(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Add more messages than maxLen
	for i := 0; i < 15; i++ {
		memory.AddWithType(sharedmemory.RoleUser, string(rune('A'+i)), sharedmemory.MessageTypeUser)
	}

	got := memory.Get()
	if len(got) != 10 {
		t.Errorf("Expected 10 messages (maxLen), got %d", len(got))
	}

	// Verify last 10 messages (F through O)
	for i, msg := range got {
		expected := rune('F' + i)
		if extractTextFromMsgBlocks(msg.GetContentBlocks()) != string(expected) {
			t.Errorf("Message %d: Content = %v, want %v", i, extractTextFromMsgBlocks(msg.GetContentBlocks()), string(expected))
		}
	}
}

func TestSessionMemory_Get(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Add messages
	memory.AddWithType(sharedmemory.RoleUser, "Hello", sharedmemory.MessageTypeUser)
	memory.AddWithType(sharedmemory.RoleAssistant, "Hi there", sharedmemory.MessageTypeAssistant)
	memory.AddWithType(sharedmemory.RoleUser, "How are you?", sharedmemory.MessageTypeUser)

	got := memory.Get()
	if len(got) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(got))
	}

	expected := []struct {
		role    string
		content string
	}{
		{"user", "Hello"},
		{"assistant", "Hi there"},
		{"user", "How are you?"},
	}

	for i, exp := range expected {
		if got[i].Role != exp.role {
			t.Errorf("Message %d: Role = %v, want %v", i, got[i].Role, exp.role)
		}
		if extractTextFromMsgBlocks(got[i].GetContentBlocks()) != exp.content {
			t.Errorf("Message %d: Content = %v, want %v", i, extractTextFromMsgBlocks(got[i].GetContentBlocks()), exp.content)
		}
	}
}

func TestSessionMemory_Clear(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Add messages
	memory.AddWithType(sharedmemory.RoleUser, "Hello", sharedmemory.MessageTypeUser)
	memory.AddWithType(sharedmemory.RoleAssistant, "Hi there", sharedmemory.MessageTypeAssistant)

	// Clear
	memory.Clear()

	got := memory.Get()
	if len(got) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(got))
	}
}

func TestSessionMemory_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-persist-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-persist"

	// Save session
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Add messages synchronously by using store directly
	msg1 := &model.Message{
		SessionID: sessionID,
		Role:      "user",
		ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Persistent message"}},
	}
	msg2 := &model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Response"}},
	}

	ctx := context.Background()
	if err := store.AppendMessage(ctx, msg1); err != nil {
		t.Fatalf("Failed to append message 1: %v", err)
	}
	if err := store.AppendMessage(ctx, msg2); err != nil {
		t.Fatalf("Failed to append message 2: %v", err)
	}

	// Create new memory instance (simulating restart)
	// This should load messages from storage
	memory2, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create memory2: %v", err)
	}

	got := memory2.Get()
	// Messages should be loaded from JSONL storage
	if len(got) != 2 {
		t.Errorf("Expected 2 messages to be loaded from storage, got %d", len(got))
	}
}

func TestSessionMemory_ConcurrentAccess(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			memory.AddWithType(sharedmemory.RoleUser, string(rune('A'+n)), sharedmemory.MessageTypeUser)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	got := memory.Get()
	if len(got) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(got))
	}
}

// TestSessionMemory_SetSummarizer tests the SetSummarizer method.
func TestSessionMemory_SetSummarizer(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	summarizer := &Summarizer{
		config: DefaultSummarizerConfig(),
	}

	memory.SetSummarizer(summarizer)

	if memory.summarizer != summarizer {
		t.Error("Expected summarizer to be set")
	}
}

// TestSessionMemory_GetSummary tests the GetSummary method.
func TestSessionMemory_GetSummary(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Initially empty
	if memory.GetSummary() != "" {
		t.Errorf("Expected empty summary, got %q", memory.GetSummary())
	}

	// Set summary manually (for testing)
	memory.mu.Lock()
	memory.summaryText = "Test summary"
	memory.mu.Unlock()

	if memory.GetSummary() != "Test summary" {
		t.Errorf("Expected 'Test summary', got %q", memory.GetSummary())
	}
}

// TestSessionMemory_ClearSummary tests the ClearSummary method.
func TestSessionMemory_ClearSummary(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Set summary manually
	memory.mu.Lock()
	memory.summaryText = "Test summary"
	memory.lastSummaryAt = 5
	memory.mu.Unlock()

	memory.ClearSummary()

	if memory.GetSummary() != "" {
		t.Errorf("Expected empty summary after clear, got %q", memory.GetSummary())
	}
	if memory.lastSummaryAt != 0 {
		t.Errorf("Expected lastSummaryAt to be 0, got %d", memory.lastSummaryAt)
	}
}

// TestSessionMemory_GetTokenCount tests the GetTokenCount method.
func TestSessionMemory_GetTokenCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-token-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-token"
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create memory with tokenizer
	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen:    10,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Initially zero
	if memory.GetTokenCount() != 0 {
		t.Errorf("Expected 0 tokens, got %d", memory.GetTokenCount())
	}

	// Add messages
	memory.AddWithType(sharedmemory.RoleUser, "Hello world", sharedmemory.MessageTypeUser)
	memory.AddWithType(sharedmemory.RoleAssistant, "Hi there", sharedmemory.MessageTypeAssistant)

	count := memory.GetTokenCount()
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}
}

// TestSessionMemory_trimByTokens tests the trimByTokens method.
func TestSessionMemory_trimByTokens(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-trim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-trim"
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create memory with token limit
	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxTokens: 500, // Increased to allow some messages
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Add many messages to exceed token limit
	for i := 0; i < 50; i++ {
		memory.AddWithType(sharedmemory.RoleUser, "This is a test message", sharedmemory.MessageTypeUser)
		memory.AddWithType(sharedmemory.RoleAssistant, "This is a response", sharedmemory.MessageTypeAssistant)
	}

	// Token count should be around maxTokens (with 5% buffer)
	count := memory.GetTokenCount()
	maxAllowed := int(float64(500) * 1.05) // 5% buffer
	if count > maxAllowed {
		t.Errorf("Token count %d exceeds maxTokens %d (with buffer %d)", count, 500, maxAllowed)
	}
}

// TestSessionMemory_trimByTokens_UserAssistantPair tests that user/assistant pairs are kept together.
func TestSessionMemory_trimByTokens_UserAssistantPair(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-pair-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-pair"
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxTokens: 50,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Add user/assistant pairs
	for i := 0; i < 20; i++ {
		memory.AddWithType(sharedmemory.RoleUser, "Question", sharedmemory.MessageTypeUser)
		memory.AddWithType(sharedmemory.RoleAssistant, "Answer", sharedmemory.MessageTypeAssistant)
	}

	// Verify messages are trimmed but pairs are kept
	msgs := memory.Get()
	if len(msgs)%2 != 0 {
		t.Errorf("Expected even number of messages (user/assistant pairs), got %d", len(msgs))
	}
}

// TestSessionMemory_GetMessagesWithSummary tests the GetMessagesWithSummary method.
func TestSessionMemory_GetMessagesWithSummary(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Add messages
	memory.AddWithType(sharedmemory.RoleUser, "Hello", sharedmemory.MessageTypeUser)
	memory.AddWithType(sharedmemory.RoleAssistant, "Hi", sharedmemory.MessageTypeAssistant)

	// No summary - should return just messages
	msgs := memory.GetMessagesWithSummary()
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}

	// Set summary manually
	memory.mu.Lock()
	memory.summaryText = "Previous summary"
	memory.mu.Unlock()

	// With summary - should prepend system message
	msgs = memory.GetMessagesWithSummary()
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages (1 summary + 2 messages), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("Expected first message to be system, got %s", msgs[0].Role)
	}
}

// TestSessionMemory_MaybeSummarize tests the MaybeSummarize method.
func TestSessionMemory_MaybeSummarize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-summarize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-summarize"
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create mock summarizer
	summarizer := &Summarizer{
		config: SummarizerConfig{
			Threshold: 5,
			Window:    3,
		},
	}

	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen:     10,
		Summarizer: summarizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Not enough messages - should fail
	ctx := context.Background()
	err = memory.MaybeSummarize(ctx)
	if err == nil {
		t.Error("Expected error when not enough messages")
	}
}

// TestSessionMemory_AddWithBlocks tests the AddWithBlocks method.
func TestSessionMemory_AddWithBlocks(t *testing.T) {
	memory, cleanup := setupTestSessionMemory(t)
	defer cleanup()

	// Create content blocks
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"},
	}

	memory.AddWithBlocks(sharedmemory.RoleUser, blocks, sharedmemory.MessageTypeUser)

	// Wait for async persistence
	time.Sleep(100 * time.Millisecond)

	got := memory.Get()
	if len(got) != 1 {
		t.Errorf("Expected 1 message, got %d", len(got))
	}
	if got[0].Role != "user" {
		t.Errorf("Role = %v, want user", got[0].Role)
	}
	if extractTextFromMsgBlocks(got[0].GetContentBlocks()) != "Hello" {
		t.Errorf("Content = %v, want Hello", extractTextFromMsgBlocks(got[0].GetContentBlocks()))
	}

	// Verify ContentBlocks are set
	contentBlocks := got[0].GetContentBlocks()
	if len(contentBlocks) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(contentBlocks))
	}
	if tb, ok := contentBlocks[0].(sharedmemory.TextBlock); ok {
		if tb.Text != "Hello" {
			t.Errorf("TextBlock.Text = %v, want Hello", tb.Text)
		}
	} else {
		t.Errorf("Expected TextBlock, got %T", contentBlocks[0])
	}
}

// TestSessionMemory_AddWithBlocks_Persistence tests that AddWithBlocks persists content blocks to storage.
func TestSessionMemory_AddWithBlocks_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-blocks-persist-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-blocks-persist"

	// Save session
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create memory with tokenizer
	tokenizer := NewSimpleEstimator()
	memory, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen:    10,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Create content blocks with text and tool use
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Please analyze this"},
		sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    "tool_123",
			Name:  "analyze",
			Input: []byte(`{"target": "data"}`),
		},
	}

	memory.AddWithBlocks(sharedmemory.RoleAssistant, blocks, sharedmemory.MessageTypeAssistant)

	// Wait for async persistence
	time.Sleep(200 * time.Millisecond)

	// Load messages from storage directly to verify persistence
	ctx := context.Background()
	storedMsgs, err := store.MessageStore().Get(ctx, sessionID, 10, "")
	if err != nil {
		t.Fatalf("Failed to get stored messages: %v", err)
	}

	if len(storedMsgs) != 1 {
		t.Fatalf("Expected 1 stored message, got %d", len(storedMsgs))
	}

	// Verify ContentBlocks are persisted
	storedBlocks := storedMsgs[0].GetContentBlocks()
	if len(storedBlocks) != 2 {
		t.Errorf("Expected 2 content blocks in storage, got %d", len(storedBlocks))
	}

	// Verify text block
	if tb, ok := storedBlocks[0].(sharedmemory.TextBlock); ok {
		if tb.Text != "Please analyze this" {
			t.Errorf("Stored TextBlock.Text = %v, want 'Please analyze this'", tb.Text)
		}
	} else {
		t.Errorf("Expected first block to be TextBlock, got %T", storedBlocks[0])
	}

	// Verify tool use block
	if tub, ok := storedBlocks[1].(sharedmemory.ToolUseBlock); ok {
		if tub.ID != "tool_123" {
			t.Errorf("Stored ToolUseBlock.ID = %v, want 'tool_123'", tub.ID)
		}
		if tub.Name != "analyze" {
			t.Errorf("Stored ToolUseBlock.Name = %v, want 'analyze'", tub.Name)
		}
	} else {
		t.Errorf("Expected second block to be ToolUseBlock, got %T", storedBlocks[1])
	}
}

// TestSessionMemory_AddWithBlocks_LoadFromStore tests that ContentBlocks are loaded from storage.
func TestSessionMemory_AddWithBlocks_LoadFromStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-memory-blocks-load-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sessionID := "test-blocks-load"

	// Save session
	err = store.SaveSession(&model.Session{
		ID:   sessionID,
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create first memory instance and add message with content blocks
	tokenizer := NewSimpleEstimator()
	memory1, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen:    10,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory1: %v", err)
	}

	// Add message with multiple content blocks
	blocks := []sharedmemory.ContentBlock{
		sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "User request"},
		sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    "tool_001",
			Name:  "search",
			Input: []byte(`{"query": "test"}`),
		},
		sharedmemory.ToolResultBlock{
			Type:      sharedmemory.BlockTypeToolResult,
			ToolUseID: "tool_001",
			Content: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Search results"},
			},
		},
	}

	memory1.AddWithBlocks(sharedmemory.RoleAssistant, blocks, sharedmemory.MessageTypeAssistant)

	// Wait for async persistence
	time.Sleep(200 * time.Millisecond)

	// Create second memory instance (simulating restart) - should load ContentBlocks
	memory2, err := NewSessionMemoryWithConfig(sessionID, "", store.MessageStore(), SessionMemoryConfig{
		MaxLen:    10,
		Tokenizer: tokenizer,
	})
	if err != nil {
		t.Fatalf("Failed to create memory2: %v", err)
	}

	// Verify ContentBlocks are loaded
	got := memory2.Get()
	if len(got) != 1 {
		t.Fatalf("Expected 1 message loaded from storage, got %d", len(got))
	}

	loadedBlocks := got[0].GetContentBlocks()
	if len(loadedBlocks) != 3 {
		t.Errorf("Expected 3 content blocks loaded, got %d", len(loadedBlocks))
		for i, b := range loadedBlocks {
			t.Logf("Block %d: type=%s", i, b.BlockType())
		}
	}

	// Verify each block type
	// First: TextBlock
	if tb, ok := loadedBlocks[0].(sharedmemory.TextBlock); ok {
		if tb.Text != "User request" {
			t.Errorf("Loaded TextBlock.Text = %v, want 'User request'", tb.Text)
		}
	} else {
		t.Errorf("Expected first loaded block to be TextBlock, got %T", loadedBlocks[0])
	}

	// Second: ToolUseBlock
	if tub, ok := loadedBlocks[1].(sharedmemory.ToolUseBlock); ok {
		if tub.ID != "tool_001" {
			t.Errorf("Loaded ToolUseBlock.ID = %v, want 'tool_001'", tub.ID)
		}
		if tub.Name != "search" {
			t.Errorf("Loaded ToolUseBlock.Name = %v, want 'search'", tub.Name)
		}
	} else {
		t.Errorf("Expected second loaded block to be ToolUseBlock, got %T", loadedBlocks[1])
	}

	// Third: ToolResultBlock
	if trb, ok := loadedBlocks[2].(sharedmemory.ToolResultBlock); ok {
		if trb.ToolUseID != "tool_001" {
			t.Errorf("Loaded ToolResultBlock.ToolUseID = %v, want 'tool_001'", trb.ToolUseID)
		}
		if len(trb.Content) != 1 {
			t.Errorf("Loaded ToolResultBlock.Content length = %d, want 1", len(trb.Content))
		}
	} else {
		t.Errorf("Expected third loaded block to be ToolResultBlock, got %T", loadedBlocks[2])
	}
}
