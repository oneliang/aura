package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// mockLLMClientWithFunc is a mock implementation of llm.Client for testing.
type mockLLMClientWithFunc struct {
	completeFunc func(ctx context.Context, req *llm.Request) (*llm.Response, error)
}

func (m *mockLLMClientWithFunc) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &llm.Response{
		Message: llm.Message{
			Role:    "assistant",
			ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Test summary"}},
		},
	}, nil
}

func (m *mockLLMClientWithFunc) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	return nil, nil
}

func (m *mockLLMClientWithFunc) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestNewSummarizer(t *testing.T) {
	client := &mockLLMClientWithFunc{}

	t.Run("valid config", func(t *testing.T) {
		config := SummarizerConfig{
			Threshold:   10,
			Window:      5,
			MaxTokens:   100,
			Temperature: 0.5,
		}
		s := NewSummarizer(client, config)

		if s == nil {
			t.Fatal("Expected summarizer to be created")
		}
		if s.config.Threshold != 10 {
			t.Errorf("Threshold = %d, want 10", s.config.Threshold)
		}
		if s.config.Window != 5 {
			t.Errorf("Window = %d, want 5", s.config.Window)
		}
		if s.config.MaxTokens != 100 {
			t.Errorf("MaxTokens = %d, want 100", s.config.MaxTokens)
		}
	})

	t.Run("zero threshold uses default", func(t *testing.T) {
		config := SummarizerConfig{Threshold: 0}
		s := NewSummarizer(client, config)

		if s.config.Threshold != 20 {
			t.Errorf("Threshold = %d, want 20 (default)", s.config.Threshold)
		}
	})

	t.Run("zero window uses default", func(t *testing.T) {
		config := SummarizerConfig{Threshold: 10, Window: 0}
		s := NewSummarizer(client, config)

		if s.config.Window <= 0 {
			t.Errorf("Window should be set to default, got %d", s.config.Window)
		}
	})

	t.Run("zero maxTokens uses default", func(t *testing.T) {
		config := SummarizerConfig{Threshold: 10, MaxTokens: 0}
		s := NewSummarizer(client, config)

		if s.config.MaxTokens <= 0 {
			t.Errorf("MaxTokens should be set to default, got %d", s.config.MaxTokens)
		}
	})
}

func TestDefaultSummarizerConfig(t *testing.T) {
	config := DefaultSummarizerConfig()

	if config.Threshold != 20 {
		t.Errorf("Threshold = %d, want 20", config.Threshold)
	}
	if config.Window <= 0 {
		t.Errorf("Window should be positive, got %d", config.Window)
	}
	if config.MaxTokens <= 0 {
		t.Errorf("MaxTokens should be positive, got %d", config.MaxTokens)
	}
}

func TestSummarizer_ShouldSummarize(t *testing.T) {
	client := &mockLLMClientWithFunc{}
	config := SummarizerConfig{Threshold: 10}
	s := NewSummarizer(client, config)

	tests := []struct {
		name         string
		messageCount int
		want         bool
	}{
		{"below threshold", 5, false},
		{"at threshold", 10, true},
		{"above threshold", 15, true},
		{"zero messages", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.ShouldSummarize(tt.messageCount)
			if got != tt.want {
				t.Errorf("ShouldSummarize(%d) = %v, want %v", tt.messageCount, got, tt.want)
			}
		})
	}
}

func TestSummarizer_GenerateSummary(t *testing.T) {
	t.Run("empty messages returns empty string", func(t *testing.T) {
		client := &mockLLMClientWithFunc{}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		result, err := s.GenerateSummary(context.Background(), []llm.Message{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("successful summary", func(t *testing.T) {
		expectedSummary := "This is a test summary"
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				return &llm.Response{
					Message: llm.Message{
						Role:    "assistant",
						ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: expectedSummary}},
					},
				}, nil
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		messages := []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
			{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hi there"}}},
		}

		result, err := s.GenerateSummary(context.Background(), messages)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != expectedSummary {
			t.Errorf("Summary = %q, want %q", result, expectedSummary)
		}
	})

	t.Run("LLM error handling", func(t *testing.T) {
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				return nil, errors.New("LLM unavailable")
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		messages := []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
		}

		_, err := s.GenerateSummary(context.Background(), messages)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("assistant role renamed to Aura", func(t *testing.T) {
		var capturedPrompt string
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				capturedPrompt = extractTextFromBlocks(req.Messages[0].GetContentBlocks())
				return &llm.Response{
					Message: llm.Message{
						ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Summary"}},
					},
				}, nil
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		messages := []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
			{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hi"}}},
		}

		_, _ = s.GenerateSummary(context.Background(), messages)

		if capturedPrompt != "" {
			if expected := "[Aura]: Hi"; expected != "" && !contains(capturedPrompt, expected) {
				t.Errorf("Prompt should contain '[Aura]: Hi', got %q", capturedPrompt)
			}
		}
	})

	t.Run("trims whitespace from summary", func(t *testing.T) {
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				return &llm.Response{
					Message: llm.Message{
						ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "  Trimmed summary  \n"}},
					},
				}, nil
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		result, err := s.GenerateSummary(context.Background(), []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != "Trimmed summary" {
			t.Errorf("Summary = %q, want %q", result, "Trimmed summary")
		}
	})
}

func TestSummarizer_GenerateSummaryWithPrevious(t *testing.T) {
	t.Run("empty messages and previous summary returns empty", func(t *testing.T) {
		client := &mockLLMClientWithFunc{}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		result, err := s.GenerateSummaryWithPrevious(context.Background(), "", []llm.Message{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("empty new messages returns previous summary", func(t *testing.T) {
		client := &mockLLMClientWithFunc{}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		previous := "Previous summary content"
		result, err := s.GenerateSummaryWithPrevious(context.Background(), previous, []llm.Message{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != previous {
			t.Errorf("Summary = %q, want %q", result, previous)
		}
	})

	t.Run("combines previous summary with new messages", func(t *testing.T) {
		var capturedPrompt string
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				capturedPrompt = extractTextFromBlocks(req.Messages[0].GetContentBlocks())
				return &llm.Response{
					Message: llm.Message{
						ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Updated summary"}},
					},
				}, nil
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		previous := "Old summary"
		newMessages := []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "New message"}}},
		}

		_, _ = s.GenerateSummaryWithPrevious(context.Background(), previous, newMessages)

		if capturedPrompt != "" {
			if !contains(capturedPrompt, "Previous summary:") {
				t.Error("Prompt should contain 'Previous summary:'")
			}
			if !contains(capturedPrompt, previous) {
				t.Errorf("Prompt should contain previous summary %q", previous)
			}
			if !contains(capturedPrompt, "New conversation:") {
				t.Error("Prompt should contain 'New conversation:'")
			}
		}
	})

	t.Run("successful update with previous", func(t *testing.T) {
		expectedSummary := "Updated summary"
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				return &llm.Response{
					Message: llm.Message{
						ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: expectedSummary}},
					},
				}, nil
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		result, err := s.GenerateSummaryWithPrevious(context.Background(), "Old", []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "New"}}},
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != expectedSummary {
			t.Errorf("Summary = %q, want %q", result, expectedSummary)
		}
	})

	t.Run("LLM error handling", func(t *testing.T) {
		client := &mockLLMClientWithFunc{
			completeFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
				return nil, errors.New("LLM error")
			},
		}
		s := NewSummarizer(client, DefaultSummarizerConfig())

		_, err := s.GenerateSummaryWithPrevious(context.Background(), "Old", []llm.Message{
			{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "New"}}},
		})
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestSummarizer_GetConfig(t *testing.T) {
	client := &mockLLMClientWithFunc{}
	config := SummarizerConfig{
		Threshold:   15,
		Window:      8,
		MaxTokens:   200,
		Temperature: 0.7,
	}
	s := NewSummarizer(client, config)

	got := s.GetConfig()
	if got.Threshold != config.Threshold {
		t.Errorf("Threshold = %d, want %d", got.Threshold, config.Threshold)
	}
	if got.Window != config.Window {
		t.Errorf("Window = %d, want %d", got.Window, config.Window)
	}
	if got.MaxTokens != config.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", got.MaxTokens, config.MaxTokens)
	}
	if got.Temperature != config.Temperature {
		t.Errorf("Temperature = %f, want %f", got.Temperature, config.Temperature)
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
