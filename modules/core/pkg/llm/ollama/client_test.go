package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to create a message with text content
func newTextMessage(role, text string) llm.Message {
	msg := llm.Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

// Helper function to extract text content from message
func getTextContent(msg llm.Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// TestClientCreation tests client creation with various configurations.
func TestClientCreation(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantURL   string
		wantModel string
	}{
		{
			name:      "default",
			opts:      []Option{},
			wantURL:   "http://localhost:11434",
			wantModel: "llama3.2",
		},
		{
			name:      "custom url",
			opts:      []Option{WithBaseURL("http://custom:11434")},
			wantURL:   "http://custom:11434",
			wantModel: "llama3.2",
		},
		{
			name:      "custom model",
			opts:      []Option{WithModel("qwen3:8b")},
			wantURL:   "http://localhost:11434",
			wantModel: "qwen3:8b",
		},
		{
			name:      "custom timeout",
			opts:      []Option{WithTimeout(60 * time.Second)},
			wantURL:   "http://localhost:11434",
			wantModel: "llama3.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.opts...)

			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantURL)
			}

			if client.model != tt.wantModel {
				t.Errorf("model = %q, want %q", client.model, tt.wantModel)
			}
		})
	}
}

// TestComplete tests the Complete method.
func TestComplete(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
				"model": "llama3.2",
				"created_at": "2024-01-01T00:00:00Z",
				"message": {"role": "assistant", "content": "Hello, I am Ollama"},
				"done": true
			}`))
	}))
	defer server.Close()

	client := New(
		WithBaseURL(server.URL),
		WithModel("llama3.2"),
	)

	ctx := context.Background()
	resp, err := client.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
		Model: "llama3.2",
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if getTextContent(resp.Message) != "Hello, I am Ollama" {
		t.Errorf("Content = %q, want %q", getTextContent(resp.Message), "Hello, I am Ollama")
	}

	if resp.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", resp.Model, "llama3.2")
	}
}

// TestCompleteError tests error handling in Complete method.
func TestCompleteError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	_, err := client.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
	})

	if err == nil {
		t.Error("Complete() expected error, got nil")
	}
}

// TestStream tests the Stream method.
func TestStream(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Stream response
		_, _ = w.Write([]byte(`{"model": "llama3.2", "message": {"role": "assistant", "content": "Hello"}, "done": false}`))
		_, _ = w.Write([]byte("\n"))
		_, _ = w.Write([]byte(`{"model": "llama3.2", "message": {"role": "assistant", "content": " there"}, "done": true}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	ch, err := client.Stream(ctx, &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
		Stream: true,
	})

	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var content string
	for chunk := range ch {
		content += chunk.Content
		if chunk.Done {
			break
		}
	}

	expected := "Hello there"
	if content != expected {
		t.Errorf("Streamed content = %q, want %q", content, expected)
	}
}

// TestEmbed tests the Embed method.
func TestEmbed(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3, 0.4, 0.5]}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	embeddings, err := client.Embed(ctx, []string{"Hello world"})

	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embeddings) != 1 {
		t.Errorf("len(embeddings) = %d, want 1", len(embeddings))
	}

	if len(embeddings[0]) != 5 {
		t.Errorf("len(embedding[0]) = %d, want 5", len(embeddings[0]))
	}
}

// TestCompleteCancelledContext tests context cancellation.
func TestCompleteCancelledContext(t *testing.T) {
	// Create test server that delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
	})

	if err == nil {
		t.Error("Complete() with cancelled context expected error, got nil")
	}
}

// TestBuildChatRequest tests buildChatRequest function.
func TestBuildChatRequest(t *testing.T) {
	client := New(WithModel("llama3.2"))

	messages := []llm.Message{
		newTextMessage("user", "Hello"),
		newTextMessage("assistant", "Hi there"),
	}

	body, err := client.buildChatRequest(false, messages)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if len(body) == 0 {
		t.Error("buildChatRequest() should return non-empty body")
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("buildChatRequest() returned invalid JSON: %v", err)
	}

	if data["model"] != "llama3.2" {
		t.Errorf("model = %v, want llama3.2", data["model"])
	}
	if data["stream"] != false {
		t.Errorf("stream = %v, want false", data["stream"])
	}
}

// TestBuildChatRequest_Stream tests buildChatRequest with stream=true.
func TestBuildChatRequest_Stream(t *testing.T) {
	client := New(WithModel("qwen3:8b"))

	messages := []llm.Message{
		newTextMessage("system", "You are helpful"),
	}

	body, err := client.buildChatRequest(true, messages)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("buildChatRequest() returned invalid JSON: %v", err)
	}

	if data["model"] != "qwen3:8b" {
		t.Errorf("model = %v, want qwen3:8b", data["model"])
	}
	if data["stream"] != true {
		t.Errorf("stream = %v, want true", data["stream"])
	}
}

// TestBuildChatRequest_EmptyMessages tests buildChatRequest with empty messages.
func TestBuildChatRequest_EmptyMessages(t *testing.T) {
	client := New(WithModel("llama3.2"))

	body, err := client.buildChatRequest(false, []llm.Message{})
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("buildChatRequest() returned invalid JSON: %v", err)
	}

	// Check if messages key exists
	if _, ok := data["messages"]; !ok {
		t.Fatal("messages key should exist")
	}
	// Messages can be nil (null in JSON) or empty array when no messages
	switch msgs := data["messages"].(type) {
	case []interface{}:
		// Empty array is fine
		if len(msgs) != 0 {
			t.Errorf("messages = %d, want 0", len(msgs))
		}
	case nil:
		// nil is also acceptable (empty messages slice)
	default:
		t.Errorf("messages type = %T, want []interface{} or nil", data["messages"])
	}
}

// TestSendChatRequest tests sendChatRequest function.
func TestSendChatRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	body := []byte(`{"model": "llama3.2", "messages": [], "stream": false}`)
	resp, err := client.sendChatRequest(ctx, body, false)
	if err != nil {
		t.Fatalf("sendChatRequest() error = %v", err)
	}
	if resp == nil {
		t.Fatal("sendChatRequest() returned nil")
	}
	resp.Body.Close()
}

// TestSendChatRequest_ErrorStatus tests sendChatRequest with error status.
func TestSendChatRequest_ErrorStatus(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	body := []byte(`{"model": "llama3.2", "messages": [], "stream": false}`)
	_, err := client.sendChatRequest(ctx, body, false)
	if err == nil {
		t.Error("sendChatRequest() with error status expected error, got nil")
	}
}

// TestSendChatRequest_CancelledContext tests sendChatRequest with cancelled context.
func TestSendChatRequest_CancelledContext(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	body := []byte(`{"model": "llama3.2", "messages": [], "stream": false}`)
	_, err := client.sendChatRequest(ctx, body, false)
	if err == nil {
		t.Error("sendChatRequest() with cancelled context expected error, got nil")
	}
}

// TestEmbedError tests error handling in Embed method.
func TestEmbedError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	_, err := client.Embed(ctx, []string{"Hello"})
	if err == nil {
		t.Error("Embed() with error status expected error, got nil")
	}
}

// TestStreamError tests error handling in Stream method.
func TestStreamError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	_, err := client.Stream(ctx, &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
		Stream: true,
	})
	if err == nil {
		t.Error("Stream() with error status expected error, got nil")
	}
}

// TestEmbedMultipleTexts tests embedding multiple texts.
func TestEmbedMultipleTexts(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	ctx := context.Background()
	embeddings, err := client.Embed(ctx, []string{"text1", "text2", "text3"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embeddings) != 3 {
		t.Errorf("len(embeddings) = %d, want 3", len(embeddings))
	}
}