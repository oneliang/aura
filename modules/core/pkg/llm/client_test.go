// Package llm provides tests for the llm package.
package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to extract text content from message
func getTextContent(msg Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// Helper function to create a message with text content
func newTextMessage(role, text string) Message {
	msg := Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

// MockClient implements the Client interface for testing.
type MockClient struct {
	completeFunc func(ctx context.Context, req *Request) (*Response, error)
	streamFunc   func(ctx context.Context, req *Request) (<-chan Chunk, error)
	embedFunc    func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &Response{
		Message: newTextMessage("assistant", "mock response"),
		Model:   "mock-model",
		Usage:   Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}, nil
}

func (m *MockClient) Stream(ctx context.Context, req *Request) (<-chan Chunk, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	ch := make(chan Chunk, 10)
	go func() {
		defer close(ch)
		ch <- Chunk{Content: "mock", Done: true}
	}()
	return ch, nil
}

func (m *MockClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, texts)
	}
	// Return mock embeddings
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

// TestMessage tests the Message struct.
func TestMessage(t *testing.T) {
	msg := newTextMessage("user", "Hello, world!")

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if getTextContent(msg) != "Hello, world!" {
		t.Errorf("Content = %q, want %q", getTextContent(msg), "Hello, world!")
	}
}

// TestRequest tests the Request struct.
func TestRequest(t *testing.T) {
	req := Request{
		Messages: []Message{
			newTextMessage("system", "You are a helpful assistant."),
			newTextMessage("user", "Hello!"),
		},
		Model:       "qwen3:8b",
		MaxTokens:   1000,
		Temperature: 0.7,
		Stream:      false,
	}

	if len(req.Messages) != 2 {
		t.Errorf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Model != "qwen3:8b" {
		t.Errorf("Model = %q, want %q", req.Model, "qwen3:8b")
	}
	if req.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", req.Temperature)
	}
	if req.Stream != false {
		t.Errorf("Stream = %v, want false", req.Stream)
	}
}

// TestResponse tests the Response struct.
func TestResponse(t *testing.T) {
	resp := Response{
		Message: newTextMessage("assistant", "Hello! How can I help you?"),
		Model:        "qwen3:8b",
		Usage:        Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		FinishReason: "stop",
	}

	if resp.Message.Role != "assistant" {
		t.Errorf("Message.Role = %q, want %q", resp.Message.Role, "assistant")
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Usage.TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

// TestUsage tests the Usage struct.
func TestUsage(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", usage.TotalTokens)
	}
}

// TestChunk tests the Chunk struct.
func TestChunk(t *testing.T) {
	chunk := Chunk{
		Content:      "Hello",
		FinishReason: "stop",
		Done:         true,
	}

	if chunk.Content != "Hello" {
		t.Errorf("Content = %q, want %q", chunk.Content, "Hello")
	}
	if !chunk.Done {
		t.Error("Done should be true")
	}
}

// TestNewLoggingClient tests the NewLoggingClient function.
func TestNewLoggingClient(t *testing.T) {
	mockClient := &MockClient{}
	provider := "ollama"
	model := "qwen3:8b"
	sessionID := "test-session"

	lc := NewLoggingClient(mockClient, provider, model, sessionID)

	if lc == nil {
		t.Fatal("NewLoggingClient() returned nil")
	}
	if lc.client != mockClient {
		t.Error("client not set correctly")
	}
	if lc.provider != provider {
		t.Errorf("provider = %q, want %q", lc.provider, provider)
	}
	if lc.model != model {
		t.Errorf("model = %q, want %q", lc.model, model)
	}
	if lc.sessionID != sessionID {
		t.Errorf("sessionID = %q, want %q", lc.sessionID, sessionID)
	}
}

// TestLoggingClient_Complete tests the Complete method with logging.
func TestLoggingClient_Complete(t *testing.T) {
	mockClient := &MockClient{
		completeFunc: func(ctx context.Context, req *Request) (*Response, error) {
			return &Response{
				Message: newTextMessage("assistant", "test response"),
				Model:   "test-model",
				Usage:   Usage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
			}, nil
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Hello")},
		Model:    "test-model",
	}

	resp, err := lc.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Complete() returned nil response")
	}
	if getTextContent(resp.Message) != "test response" {
		t.Errorf("Content = %q, want %q", getTextContent(resp.Message), "test response")
	}
}

// TestLoggingClient_Complete_WithDelay tests Complete with measurable duration.
func TestLoggingClient_Complete_WithDelay(t *testing.T) {
	mockClient := &MockClient{
		completeFunc: func(ctx context.Context, req *Request) (*Response, error) {
			time.Sleep(10 * time.Millisecond) // Small delay to measure duration
			return &Response{
				Message: newTextMessage("assistant", "delayed response"),
				Model:   "test-model",
			}, nil
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Hello")},
		Model:    "test-model",
	}

	start := time.Now()
	resp, err := lc.Complete(ctx, req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Complete() returned nil response")
	}
	if duration < 10*time.Millisecond {
		t.Error("Duration should be at least 10ms")
	}
}

// TestLoggingClient_Complete_WithError tests Complete with error.
func TestLoggingClient_Complete_WithError(t *testing.T) {
	mockClient := &MockClient{
		completeFunc: func(ctx context.Context, req *Request) (*Response, error) {
			return nil, context.DeadlineExceeded
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Hello")},
		Model:    "test-model",
	}

	resp, err := lc.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should return error")
	}
	if resp != nil {
		t.Error("Complete() should return nil response on error")
	}
}

// TestLoggingClient_Stream tests the Stream method with logging.
func TestLoggingClient_Stream(t *testing.T) {
	mockClient := &MockClient{}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Stream this")},
		Model:    "test-model",
		Stream:   true,
	}

	ch, err := lc.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if ch == nil {
		t.Fatal("Stream() returned nil channel")
	}

	// Read chunks
	var chunks []Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("No chunks received")
	}
}

// TestLoggingClient_Stream_WithError tests Stream with error.
func TestLoggingClient_Stream_WithError(t *testing.T) {
	mockClient := &MockClient{
		streamFunc: func(ctx context.Context, req *Request) (<-chan Chunk, error) {
			return nil, context.DeadlineExceeded
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Stream this")},
		Model:    "test-model",
		Stream:   true,
	}

	ch, err := lc.Stream(ctx, req)
	if err == nil {
		t.Error("Stream() should return error")
	}
	if ch != nil {
		t.Error("Stream() should return nil channel on error")
	}
}

// TestLoggingClient_Embed tests the Embed method with logging.
func TestLoggingClient_Embed(t *testing.T) {
	mockClient := &MockClient{}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	texts := []string{"Hello", "World"}

	embeddings, err := lc.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embeddings) != len(texts) {
		t.Errorf("Embeddings length = %d, want %d", len(embeddings), len(texts))
	}
	if len(embeddings[0]) != 3 {
		t.Errorf("Embedding dimension = %d, want 3", len(embeddings[0]))
	}
}

// TestLoggingClient_Embed_WithEmptyTexts tests Embed with empty texts.
func TestLoggingClient_Embed_WithEmptyTexts(t *testing.T) {
	mockClient := &MockClient{
		embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return [][]float32{}, nil
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	embeddings, err := lc.Embed(ctx, []string{})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if embeddings == nil {
		t.Error("Embed() should return empty slice, not nil")
	}
}

// TestLoggingClient_Embed_WithError tests Embed with error.
func TestLoggingClient_Embed_WithError(t *testing.T) {
	mockClient := &MockClient{
		embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return nil, context.DeadlineExceeded
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	_, err := lc.Embed(ctx, []string{"Hello"})
	if err == nil {
		t.Error("Embed() should return error")
	}
}

// TestCloseLLMLogger tests the CloseLLMLogger function.
func TestCloseLLMLogger(t *testing.T) {
	// Initialize logger first
	_ = logger.GetLLMAuditLogger()

	// Close should not panic
	err := CloseLLMLogger()
	// Error is acceptable if logger was not initialized with valid file
	_ = err
}

// TestLLMLogEntry_Structure tests the logger.LLMLogEntry structure.
func TestLLMLogEntry_Structure(t *testing.T) {
	entry := logger.LLMLogEntry{
		Timestamp:  1234567890,
		RequestID:  "req-123",
		SessionID:  "session-456",
		Method:     "Complete",
		Provider:   "ollama",
		Model:      "qwen3:8b",
		Messages:   []Message{newTextMessage("user", "Hello")},
		DurationMs: 100,
		Error:      "",
		Metadata:   map[string]interface{}{"key": "value"},
	}

	if entry.Timestamp != 1234567890 {
		t.Errorf("Timestamp = %d, want 1234567890", entry.Timestamp)
	}
	if entry.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", entry.RequestID, "req-123")
	}
	if entry.Method != "Complete" {
		t.Errorf("Method = %q, want %q", entry.Method, "Complete")
	}
	if entry.Provider != "ollama" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "ollama")
	}
}

// TestLogResponse_Structure tests the LLM log response structure.
func TestLogResponse_Structure(t *testing.T) {
	resp := map[string]any{
		"content":       "Hello, world!",
		"model":         "qwen3:8b",
		"finish_reason": "stop",
	}

	if resp["content"] != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp["content"], "Hello, world!")
	}
	if resp["finish_reason"] != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp["finish_reason"], "stop")
	}
}

// TestGetLLMAuditLogger_Singleton tests that GetLLMAuditLogger returns singleton.
func TestGetLLMAuditLogger_Singleton(t *testing.T) {
	l1 := logger.GetLLMAuditLogger()
	l2 := logger.GetLLMAuditLogger()

	if l1 != l2 {
		t.Error("GetLLMAuditLogger() should return singleton instance")
	}
}

// TestMockClient_Complete tests MockClient Complete method.
func TestMockClient_Complete(t *testing.T) {
	client := &MockClient{}
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Test")},
		Model:    "test",
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if getTextContent(resp.Message) != "mock response" {
		t.Errorf("Content = %q, want %q", getTextContent(resp.Message), "mock response")
	}
}

// TestMockClient_Stream tests MockClient Stream method.
func TestMockClient_Stream(t *testing.T) {
	client := &MockClient{}
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Test")},
		Model:    "test",
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var chunks []Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("No chunks received")
	}
}

// TestMockClient_Embed tests MockClient Embed method.
func TestMockClient_Embed(t *testing.T) {
	client := &MockClient{}
	ctx := context.Background()

	texts := []string{"Hello", "World"}

	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embeddings) != 2 {
		t.Errorf("Embeddings length = %d, want 2", len(embeddings))
	}
}

// TestMockClient_CustomFunctions tests MockClient with custom functions.
func TestMockClient_CustomFunctions(t *testing.T) {
	customComplete := func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			Message: newTextMessage("assistant", "custom response"),
			Model:   "custom-model",
		}, nil
	}

	client := &MockClient{
		completeFunc: customComplete,
	}

	ctx := context.Background()
	req := &Request{
		Messages: []Message{newTextMessage("user", "Test")},
		Model:    "test",
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if getTextContent(resp.Message) != "custom response" {
		t.Errorf("Content = %q, want %q", getTextContent(resp.Message), "custom response")
	}
	if resp.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "custom-model")
	}
}

// TestMockClient_CustomEmbed tests MockClient with custom embed function.
func TestMockClient_CustomEmbed(t *testing.T) {
	customEmbed := func(ctx context.Context, texts []string) ([][]float32, error) {
		result := make([][]float32, len(texts))
		for i := range texts {
			result[i] = []float32{0.5, 0.6, 0.7}
		}
		return result, nil
	}

	client := &MockClient{
		embedFunc: customEmbed,
	}

	ctx := context.Background()
	texts := []string{"Test1", "Test2"}

	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embeddings) != 2 {
		t.Errorf("Embeddings length = %d, want 2", len(embeddings))
	}
	// Check first embedding values
	for _, v := range embeddings[0] {
		if v != 0.5 && v != 0.6 && v != 0.7 {
			t.Errorf("Embedding value = %f, want 0.5, 0.6, or 0.7", v)
		}
	}
}

// TestLoggingClient_Stream_ChannelClosure tests that Stream channel is properly closed.
func TestLoggingClient_Stream_ChannelClosure(t *testing.T) {
	mockClient := &MockClient{
		streamFunc: func(ctx context.Context, req *Request) (<-chan Chunk, error) {
			ch := make(chan Chunk, 5)
			go func() {
				defer close(ch)
				ch <- Chunk{Content: "chunk1"}
				ch <- Chunk{Content: "chunk2"}
				ch <- Chunk{Content: "final", Done: true}
			}()
			return ch, nil
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{newTextMessage("user", "Stream test")},
		Model:    "test-model",
	}

	ch, err := lc.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// Read all chunks
	var chunks []Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 3 {
		t.Errorf("Got %d chunks, want 3", len(chunks))
	}

	// Verify last chunk has Done=true
	if !chunks[len(chunks)-1].Done {
		t.Error("Last chunk should have Done=true")
	}
}

// TestLoggingClient_Embed_MultipleTexts tests Embed with multiple texts.
func TestLoggingClient_Embed_MultipleTexts(t *testing.T) {
	mockClient := &MockClient{}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	texts := []string{
		"First text for embedding",
		"Second text for embedding",
		"Third text for embedding",
	}

	embeddings, err := lc.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embeddings) != 3 {
		t.Errorf("Embeddings length = %d, want 3", len(embeddings))
	}

	// Verify all embeddings have same dimension
	if len(embeddings) > 0 {
		firstDim := len(embeddings[0])
		for i, emb := range embeddings {
			if len(emb) != firstDim {
				t.Errorf("Embedding %d dimension = %d, want %d", i, len(emb), firstDim)
			}
		}
	}
}

// TestLoggingClient_Complete_EmptyMessages tests Complete with empty messages.
func TestLoggingClient_Complete_EmptyMessages(t *testing.T) {
	mockClient := &MockClient{
		completeFunc: func(ctx context.Context, req *Request) (*Response, error) {
			// Some LLM APIs might reject empty messages
			if len(req.Messages) == 0 {
				return nil, errors.New("empty messages not allowed")
			}
			return &Response{
				Message: newTextMessage("assistant", "response"),
				Model:   "test-model",
			}, nil
		},
	}

	lc := NewLoggingClient(mockClient, "ollama", "test-model", "test-session")
	ctx := context.Background()

	req := &Request{
		Messages: []Message{},
		Model:    "test-model",
	}

	_, err := lc.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should return error for empty messages")
	}
}

// TestMessage_ContentBlocks tests that LLM Message works with ContentBlocks.
// Since Message is a type alias to memory.Message, it should inherit
// GetContentBlocks() and SetContentBlocks() methods.
func TestMessage_ContentBlocks(t *testing.T) {
	// Test 1: Create message with text content and convert to ContentBlocks
	msg := newTextMessage("user", "Hello, world!")

	blocks := msg.GetContentBlocks()
	if len(blocks) != 1 {
		t.Fatalf("GetContentBlocks() returned %d blocks, want 1", len(blocks))
	}

	tb, ok := blocks[0].(memory.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", blocks[0])
	}
	if tb.Text != "Hello, world!" {
		t.Errorf("TextBlock.Text = %q, want %q", tb.Text, "Hello, world!")
	}
}

// TestMessage_SetContentBlocks tests setting ContentBlocks on a Message.
func TestMessage_SetContentBlocks(t *testing.T) {
	// Test setting multiple content blocks
	msg := Message{Role: "assistant"}

	blocks := []memory.ContentBlock{
		memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Let me think..."},
		memory.TextBlock{Type: memory.BlockTypeText, Text: "Here is my answer."},
	}

	msg.SetContentBlocks(blocks)

	// Verify Content field is updated from first TextBlock
	if getTextContent(msg) != "Here is my answer." {
		t.Errorf("Content = %q, want %q", getTextContent(msg), "Here is my answer.")
	}

	// Verify GetContentBlocks returns the set blocks
	gotBlocks := msg.GetContentBlocks()
	if len(gotBlocks) != 2 {
		t.Fatalf("GetContentBlocks() returned %d blocks, want 2", len(gotBlocks))
	}

	// Verify thinking block
	if tb, ok := gotBlocks[0].(memory.ThinkingBlock); !ok {
		t.Errorf("Expected ThinkingBlock at index 0, got %T", gotBlocks[0])
	} else if tb.Thinking != "Let me think..." {
		t.Errorf("Thinking = %q, want %q", tb.Thinking, "Let me think...")
	}

	// Verify text block
	if tb, ok := gotBlocks[1].(memory.TextBlock); !ok {
		t.Errorf("Expected TextBlock at index 1, got %T", gotBlocks[1])
	} else if tb.Text != "Here is my answer." {
		t.Errorf("Text = %q, want %q", tb.Text, "Here is my answer.")
	}
}

// TestMessage_ToolUseBlock tests Message with ToolUseBlock content.
func TestMessage_ToolUseBlock(t *testing.T) {
	msg := Message{Role: "assistant"}

	toolUseBlocks := []memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: "I'll help you with that."},
		memory.ToolUseBlock{
			Type:  memory.BlockTypeToolUse,
			ID:    "tool-123",
			Name:  "get_weather",
			Input: []byte(`{"location": "San Francisco"}`),
		},
	}

	msg.SetContentBlocks(toolUseBlocks)

	// Verify Content field is updated from first TextBlock
	if getTextContent(msg) != "I'll help you with that." {
		t.Errorf("Content = %q, want %q", getTextContent(msg), "I'll help you with that.")
	}

	gotBlocks := msg.GetContentBlocks()
	if len(gotBlocks) != 2 {
		t.Fatalf("GetContentBlocks() returned %d blocks, want 2", len(gotBlocks))
	}

	// Verify tool use block
	tu, ok := gotBlocks[1].(memory.ToolUseBlock)
	if !ok {
		t.Fatalf("Expected ToolUseBlock at index 1, got %T", gotBlocks[1])
	}
	if tu.ID != "tool-123" {
		t.Errorf("ToolUseBlock.ID = %q, want %q", tu.ID, "tool-123")
	}
	if tu.Name != "get_weather" {
		t.Errorf("ToolUseBlock.Name = %q, want %q", tu.Name, "get_weather")
	}
}

// TestMessage_ToolResultBlock tests Message with ToolResultBlock content.
func TestMessage_ToolResultBlock(t *testing.T) {
	msg := Message{Role: "user"}

	resultBlocks := []memory.ContentBlock{
		memory.ToolResultBlock{
			Type:      memory.BlockTypeToolResult,
			ToolUseID: "tool-123",
			Content: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: "Weather: Sunny, 72°F"},
			},
		},
	}

	msg.SetContentBlocks(resultBlocks)

	// Content should be empty since no TextBlock at top level
	if getTextContent(msg) != "" {
		t.Errorf("Content = %q, want empty", getTextContent(msg))
	}

	gotBlocks := msg.GetContentBlocks()
	if len(gotBlocks) != 1 {
		t.Fatalf("GetContentBlocks() returned %d blocks, want 1", len(gotBlocks))
	}

	// Verify tool result block
	tr, ok := gotBlocks[0].(memory.ToolResultBlock)
	if !ok {
		t.Fatalf("Expected ToolResultBlock, got %T", gotBlocks[0])
	}
	if tr.ToolUseID != "tool-123" {
		t.Errorf("ToolResultBlock.ToolUseID = %q, want %q", tr.ToolUseID, "tool-123")
	}
	if len(tr.Content) != 1 {
		t.Fatalf("ToolResultBlock.Content length = %d, want 1", len(tr.Content))
	}
}

// TestMessage_EmptyContentBlocks tests GetContentBlocks with no content.
func TestMessage_EmptyContentBlocks(t *testing.T) {
	msg := Message{Role: "user"}

	blocks := msg.GetContentBlocks()
	if blocks != nil {
		t.Errorf("GetContentBlocks() = %v, want nil for empty message", blocks)
	}
}

// TestRequest_WithContentBlocks tests Request with messages containing ContentBlocks.
func TestRequest_WithContentBlocks(t *testing.T) {
	req := Request{
		Messages: []Message{
			newTextMessage("system", "You are helpful."),
			newTextMessage("user", "What is the weather?"),
			{
				Role: "assistant",
			},
		},
		Model: "test-model",
	}

	// Set ContentBlocks on assistant message
	assistantBlocks := []memory.ContentBlock{
		memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "User wants weather info"},
		memory.TextBlock{Type: memory.BlockTypeText, Text: "I'll check the weather for you."},
	}
	req.Messages[2].SetContentBlocks(assistantBlocks)

	// Verify the message was updated correctly
	if getTextContent(req.Messages[2]) != "I'll check the weather for you." {
		t.Errorf("Assistant Content = %q, want %q", getTextContent(req.Messages[2]), "I'll check the weather for you.")
	}

	blocks := req.Messages[2].GetContentBlocks()
	if len(blocks) != 2 {
		t.Errorf("Assistant GetContentBlocks() returned %d blocks, want 2", len(blocks))
	}
}

// TestResponse_WithContentBlocks tests Response with message containing ContentBlocks.
func TestResponse_WithContentBlocks(t *testing.T) {
	resp := Response{
		Message: Message{Role: "assistant"},
		Model:   "test-model",
	}

	// Simulate an LLM response with thinking and text
	blocks := []memory.ContentBlock{
		memory.ThinkingBlock{Type: memory.BlockTypeThinking, Thinking: "Let me analyze this..."},
		memory.TextBlock{Type: memory.BlockTypeText, Text: "Here's my analysis."},
	}
	resp.Message.SetContentBlocks(blocks)

	// Verify Content is set from first TextBlock
	if getTextContent(resp.Message) != "Here's my analysis." {
		t.Errorf("Response Message.Content = %q, want %q", getTextContent(resp.Message), "Here's my analysis.")
	}

	// Verify blocks are accessible
	gotBlocks := resp.Message.GetContentBlocks()
	if len(gotBlocks) != 2 {
		t.Errorf("Response GetContentBlocks() returned %d blocks, want 2", len(gotBlocks))
	}
}