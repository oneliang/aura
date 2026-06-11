// Package openai provides an OpenAI compatible client.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/constants"
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

// Test Client creation
func TestNewClient(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.baseURL != constants.DefaultOpenAIBaseURL {
		t.Errorf("default baseURL = %v, want '%s'", client.baseURL, constants.DefaultOpenAIBaseURL)
	}
	if client.model != constants.DefaultOpenAIModel {
		t.Errorf("default model = %v, want '%s'", client.model, constants.DefaultOpenAIModel)
	}
	if client.apiKey != "" {
		t.Error("default apiKey should be empty")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	client := New(
		WithBaseURL("http://localhost:11434"),
		WithModel("qwen-plus"),
		WithAPIKey("test-key"),
		WithTimeout(60*time.Second),
	)

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %v, want 'http://localhost:11434'", client.baseURL)
	}
	if client.model != "qwen-plus" {
		t.Errorf("model = %v, want 'qwen-plus'", client.model)
	}
	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %v, want 'test-key'", client.apiKey)
	}
	if client.httpClient.Timeout != 0 {
		t.Errorf("httpClient.Timeout = %v, want 0 (should use ResponseHeaderTimeout instead)", client.httpClient.Timeout)
	}
	if t2, ok := client.httpClient.Transport.(*http.Transport); ok {
		if t2.ResponseHeaderTimeout != 60*time.Second {
			t.Errorf("Transport.ResponseHeaderTimeout = %v, want 60s", t2.ResponseHeaderTimeout)
		}
	} else {
		t.Error("Transport is not *http.Transport")
	}
}

// Test Complete
func TestComplete(t *testing.T) {
	expectedContent := "Hello, I am an AI assistant."

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Unexpected method: %s", r.Method)
		}

		// Send response
		resp := openaiResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openaiMessage `json:"message"`
				Delta        openaiDelta   `json:"delta"`
				FinishReason string        `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: openaiMessage{
						Role:    "assistant",
						Content: expectedContent,
					},
					FinishReason: "stop",
				},
			},
			Usage: &struct {
				PromptTokens            int `json:"prompt_tokens"`
				CompletionTokens        int `json:"completion_tokens"`
				TotalTokens             int `json:"total_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL), WithAPIKey("test-key"))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{
			newTextMessage("user", "Hello"),
		},
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Message.Role != "assistant" {
		t.Errorf("Message.Role = %v, want 'assistant'", resp.Message.Role)
	}
	if getTextContent(resp.Message) != expectedContent {
		t.Errorf("Message.Content = %v, want %v", getTextContent(resp.Message), expectedContent)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Errorf("Model = %v, want 'gpt-4o-mini'", resp.Model)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Usage.TotalTokens = %v, want 30", resp.Usage.TotalTokens)
	}
}

func TestCompleteEmptyMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openaiMessage `json:"message"`
				Delta        openaiDelta   `json:"delta"`
				FinishReason string        `json:"finish_reason"`
			}{
				{Message: openaiMessage{Role: "assistant", Content: "Hi"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{},
	}

	_, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete() with empty messages error = %v", err)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should error on HTTP 500")
	}
	if !strings.Contains(err.Error(), "Internal server error") {
		t.Errorf("Error should contain error message, got: %v", err)
	}
}

func TestCompleteAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL), WithAPIKey("invalid-key"))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should error on 401")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Error should mention 'Invalid API key', got: %v", err)
	}
}

func TestCompleteNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openaiMessage `json:"message"`
				Delta        openaiDelta   `json:"delta"`
				FinishReason string        `json:"finish_reason"`
			}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should error when no choices returned")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("Error should mention 'no choices', got: %v", err)
	}
}

func TestCompleteCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should error with cancelled context")
	}
}

func TestCompleteConnectionError(t *testing.T) {
	client := New(WithBaseURL("http://invalid-host-that-does-not-exist-12345.com"))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should error on connection failure")
	}
}

// Test Stream
func TestStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send SSE stream
		chunks := []string{"Hello", " ", "World", "!"}
		for _, chunk := range chunks {
			resp := openaiResponse{
				Choices: []struct {
					Index        int           `json:"index"`
					Message      openaiMessage `json:"message"`
					Delta        openaiDelta   `json:"delta"`
					FinishReason string        `json:"finish_reason"`
				}{
					{
						Delta: openaiDelta{
							Content: chunk,
						},
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		}

		// Send done signal
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var content strings.Builder
	doneReceived := false

	for chunk := range ch {
		if chunk.Done {
			doneReceived = true
			break
		}
		content.WriteString(chunk.Content)
	}

	if !doneReceived {
		t.Error("Stream should send Done signal")
	}

	expected := "Hello World!"
	if content.String() != expected {
		t.Errorf("Streamed content = %q, want %q", content.String(), expected)
	}
}

func TestStreamHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Stream(ctx, req)
	if err == nil {
		t.Error("Stream() should error on HTTP 500")
	}
}

func TestStreamCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &llm.Request{
		Messages: []llm.Message{newTextMessage("user", "Hello")},
	}

	_, err := client.Stream(ctx, req)
	if err == nil {
		t.Error("Stream() should error with cancelled context")
	}
}

// Test Embed
func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := openaiEmbedResponse{
			Object: "list",
			Model:  "text-embedding-3-small",
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: []float32{0.1, 0.2, 0.3},
				},
				{
					Object:    "embedding",
					Index:     1,
					Embedding: []float32{0.4, 0.5, 0.6},
				},
			},
			Usage: struct {
				PromptTokens int `json:"prompt_tokens"`
				TotalTokens  int `json:"total_tokens"`
			}{
				PromptTokens: 5,
				TotalTokens:  5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL), WithModel("text-embedding-3-small"))
	ctx := context.Background()

	texts := []string{"Hello", "World"}
	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embeddings) != 2 {
		t.Errorf("Embeddings length = %d, want 2", len(embeddings))
	}

	// Check first embedding
	if len(embeddings[0]) != 3 {
		t.Errorf("First embedding length = %d, want 3", len(embeddings[0]))
	}
	if embeddings[0][0] != 0.1 || embeddings[0][1] != 0.2 || embeddings[0][2] != 0.3 {
		t.Errorf("First embedding values incorrect, got: %v", embeddings[0])
	}

	// Check second embedding
	if len(embeddings[1]) != 3 {
		t.Errorf("Second embedding length = %d, want 3", len(embeddings[1]))
	}
	if embeddings[1][0] != 0.4 || embeddings[1][1] != 0.5 || embeddings[1][2] != 0.6 {
		t.Errorf("Second embedding values incorrect, got: %v", embeddings[1])
	}
}

func TestEmbedEmptyTexts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiEmbedResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	texts := []string{}
	embeddings, err := client.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed() with empty texts error = %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("Embeddings length = %d, want 0", len(embeddings))
	}
}

func TestEmbedHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx := context.Background()

	texts := []string{"Hello"}
	_, err := client.Embed(ctx, texts)
	if err == nil {
		t.Error("Embed() should error on HTTP 500")
	}
}

func TestEmbedCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	texts := []string{"Hello"}
	_, err := client.Embed(ctx, texts)
	if err == nil {
		t.Error("Embed() should error with cancelled context")
	}
}

func TestEmbedConnectionError(t *testing.T) {
	client := New(WithBaseURL("http://invalid-host-that-does-not-exist-12345.com"))
	ctx := context.Background()

	texts := []string{"Hello"}
	_, err := client.Embed(ctx, texts)
	if err == nil {
		t.Error("Embed() should error on connection failure")
	}
}