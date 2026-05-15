package engine

import (
	"context"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/memory"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// runSimple runs simple streaming without tools.
func (e *Engine) runSimple(ctx context.Context, eventsCh chan<- events.Event, requestID string) {
	messages := e.buildMessages(ctx)

	ch, err := e.client.Stream(ctx, &llm.Request{
		Messages: messages,
		Thinking: e.config.Thinking,
	})
	if err != nil {
		eventsCh <- events.NewEvent(events.EventTypeError, err.Error(), requestID)
		return
	}

	var response string
	for chunk := range ch {
		// Check for cancellation during streaming
		select {
		case <-ctx.Done():
			eventsCh <- events.NewEvent(
				events.EventTypeResponse,
				constants.MessageInterrupted,
				requestID,
			)
			return
		default:
		}

		if chunk.Content != "" {
			response += chunk.Content
			eventsCh <- events.NewEvent(events.EventTypeResponseChunk, chunk.Content, requestID)
		}
		if chunk.Done {
			break
		}
	}

	// Send complete response event
	eventsCh <- events.NewEvent(events.EventTypeResponse, response, requestID)

	e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAssistant)
}

// Chat sends a simple chat message without streaming.
func (e *Engine) Chat(ctx context.Context, input string) (string, error) {
	e.memory.AddWithType(sharedmemory.RoleUser, input, memory.MessageTypeUser)

	messages := e.buildMessages(ctx)

	resp, err := e.client.Complete(ctx, &llm.Request{
		Messages: messages,
		Thinking: e.config.Thinking,
	})
	if err != nil {
		return "", err
	}

	// Extract text from ContentBlocks
	var content string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(memory.TextBlock); ok {
			content = tb.Text
			break
		}
	}
	e.memory.AddWithType(sharedmemory.RoleAssistant, content, memory.MessageTypeAssistant)

	return content, nil
}

// buildMessages builds messages for simple chat.
func (e *Engine) buildMessages(ctx context.Context) []llm.Message {
	messages := make([]llm.Message, 0)

	// Add system prompt (may be augmented by RAG below)
	systemPrompt := e.config.SystemPrompt
	systemPrompt = e.augmentSystemPromptWithRAG(ctx, systemPrompt)
	if systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:          memory.RoleSystem,
			ContentBlocks: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: systemPrompt},
			},
		})
	}

	// Add memory (with summary if summarization is enabled)
	if e.config.EnableSummarization {
		if sm, ok := e.memory.(memory.SummarizingMemory); ok {
			messages = append(messages, sm.GetMessagesWithSummary()...)
		} else {
			messages = append(messages, e.memory.Get()...)
		}
	} else {
		messages = append(messages, e.memory.Get()...)
	}

	return messages
}
