package memory

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

func TestSimpleEstimator_Estimate(t *testing.T) {
	estimator := NewSimpleEstimator()

	tests := []struct {
		name    string
		text    string
		wantMin int // minimum expected tokens
		wantMax int // maximum expected tokens
	}{
		{
			name:    "empty string",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "short English text",
			text:    "Hello",
			wantMin: 3,
			wantMax: 10,
		},
		{
			name:    "medium English text",
			text:    "Hello, world! This is a test.",
			wantMin: 15,
			wantMax: 30,
		},
		{
			name:    "Chinese text",
			text:    "你好，世界",
			wantMin: 3,
			wantMax: 10,
		},
		{
			name:    "mixed Chinese and English",
			text:    "你好 Hello 世界 World",
			wantMin: 8,
			wantMax: 20,
		},
		{
			name:    "long text",
			text:    strings.Repeat("Hello, world! ", 100),
			wantMin: 900,
			wantMax: 1000,
		},
		{
			name:    "code snippet",
			text:    "func main() { fmt.Println(\"Hello\") }",
			wantMin: 20,
			wantMax: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.Estimate(tt.text)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Estimate(%q) = %v, want [%v, %v]",
					tt.text, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSimpleEstimator_EstimateMessages(t *testing.T) {
	estimator := NewSimpleEstimator()

	tests := []struct {
		name     string
		messages []llm.Message
		wantMin  int
		wantMax  int
	}{
		{
			name:     "empty messages",
			messages: []llm.Message{},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name: "single message",
			messages: []llm.Message{
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
			},
			wantMin: 10, // overhead + content
			wantMax: 25,
		},
		{
			name: "multiple messages",
			messages: []llm.Message{
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hello"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Hi there!"}}},
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "How are you?"}}},
			},
			wantMin: 30,
			wantMax: 80,
		},
		{
			name: "messages with code",
			messages: []llm.Message{
				{Role: "user", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "Write a function"}}},
				{Role: "assistant", ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "```go\nfunc main() {\n\tprintln(\"Hello\")\n}\n```"}}},
			},
			wantMin: 45,
			wantMax: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.EstimateMessages(tt.messages)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateMessages() = %v, want [%v, %v]",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSimpleEstimator_Concurrent(t *testing.T) {
	estimator := NewSimpleEstimator()

	// Test concurrent calls
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_ = estimator.Estimate("test string")
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
