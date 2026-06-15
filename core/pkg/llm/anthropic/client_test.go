package anthropic

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
)

func TestConvertMessages_NoSystemMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "hello"}}},
		{Role: "assistant", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "hi"}}},
	}

	systems, chatMsgs := convertMessages(msgs)

	if len(systems) != 0 {
		t.Errorf("expected 0 system messages, got %d", len(systems))
	}
	if len(chatMsgs) != 2 {
		t.Errorf("expected 2 chat messages, got %d", len(chatMsgs))
	}
}

func TestConvertMessages_SingleSystemMessage(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "you are helpful"}}},
		{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "hello"}}},
	}

	systems, chatMsgs := convertMessages(msgs)

	if len(systems) != 1 {
		t.Fatalf("expected 1 system message, got %d", len(systems))
	}
	if systems[0] != "you are helpful" {
		t.Errorf("expected %q, got %q", "you are helpful", systems[0])
	}
	if len(chatMsgs) != 1 {
		t.Errorf("expected 1 chat message, got %d", len(chatMsgs))
	}
}

func TestConvertMessages_MultipleSystemMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "RAG knowledge"}}},
		{Role: "system", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "conversation summary"}}},
		{Role: "system", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "skill body"}}},
		{Role: "user", ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "hello"}}},
	}

	systems, chatMsgs := convertMessages(msgs)

	if len(systems) != 3 {
		t.Fatalf("expected 3 system messages, got %d", len(systems))
	}
	expected := []string{"RAG knowledge", "conversation summary", "skill body"}
	for i, exp := range expected {
		if systems[i] != exp {
			t.Errorf("systems[%d] = %q, want %q", i, systems[i], exp)
		}
	}
	if len(chatMsgs) != 1 {
		t.Errorf("expected 1 chat message, got %d", len(chatMsgs))
	}
}

func TestBuildSystemValue_NoCache(t *testing.T) {
	systems := []string{"system prompt", "RAG knowledge", "summary"}
	req := &llm.Request{}

	result := buildSystemValue(systems, req)

	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	expected := "system prompt\n\nRAG knowledge\n\nsummary"
	if str != expected {
		t.Errorf("expected %q, got %q", expected, str)
	}
}

func TestBuildSystemValue_NoCache_Empty(t *testing.T) {
	req := &llm.Request{}

	result := buildSystemValue(nil, req)

	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if str != "" {
		t.Errorf("expected empty string, got %q", str)
	}
}

func TestBuildSystemValue_WithCache_StaticOnly(t *testing.T) {
	req := &llm.Request{
		PromptCache: &llm.PromptCacheConfig{
			Enabled: true,
			SystemBlocks: []llm.SystemBlock{
				{Type: "text", Text: "base prompt", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
				{Type: "text", Text: "user profile", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
			},
		},
	}

	result := buildSystemValue(nil, req)

	blocks, ok := result.([]systemBlock)
	if !ok {
		t.Fatalf("expected []systemBlock, got %T", result)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Text != "base prompt" {
		t.Errorf("blocks[0].Text = %q, want %q", blocks[0].Text, "base prompt")
	}
	if blocks[0].CacheControl == nil || blocks[0].CacheControl.Type != "ephemeral" {
		t.Error("blocks[0] should have cache_control ephemeral")
	}
	if blocks[1].Text != "user profile" {
		t.Errorf("blocks[1].Text = %q, want %q", blocks[1].Text, "user profile")
	}
}

func TestBuildSystemValue_WithCache_Merged(t *testing.T) {
	req := &llm.Request{
		PromptCache: &llm.PromptCacheConfig{
			Enabled: true,
			SystemBlocks: []llm.SystemBlock{
				{Type: "text", Text: "base prompt", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
				{Type: "text", Text: "user profile", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
			},
		},
	}
	systems := []string{"RAG knowledge", "skill body"}

	result := buildSystemValue(systems, req)

	blocks, ok := result.([]systemBlock)
	if !ok {
		t.Fatalf("expected []systemBlock, got %T", result)
	}
	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks (2 static + 2 dynamic), got %d", len(blocks))
	}

	// Static blocks should have cache_control
	for i := 0; i < 2; i++ {
		if blocks[i].CacheControl == nil || blocks[i].CacheControl.Type != "ephemeral" {
			t.Errorf("static blocks[%d] should have cache_control ephemeral", i)
		}
	}

	// Dynamic blocks should NOT have cache_control
	for i := 2; i < 4; i++ {
		if blocks[i].CacheControl != nil {
			t.Errorf("dynamic blocks[%d] should NOT have cache_control", i)
		}
	}

	// Verify content order
	expected := []string{"base prompt", "user profile", "RAG knowledge", "skill body"}
	for i, exp := range expected {
		if blocks[i].Text != exp {
			t.Errorf("blocks[%d].Text = %q, want %q", i, blocks[i].Text, exp)
		}
	}
}

func TestBuildSystemValue_WithCache_EmptyDynamic(t *testing.T) {
	req := &llm.Request{
		PromptCache: &llm.PromptCacheConfig{
			Enabled: true,
			SystemBlocks: []llm.SystemBlock{
				{Type: "text", Text: "base prompt", CacheControl: &llm.CacheControl{Type: "ephemeral"}},
			},
		},
	}
	systems := []string{"", "valid content", ""}

	result := buildSystemValue(systems, req)

	blocks, ok := result.([]systemBlock)
	if !ok {
		t.Fatalf("expected []systemBlock, got %T", result)
	}
	// Empty strings should be skipped
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (1 static + 1 non-empty dynamic), got %d", len(blocks))
	}
	if blocks[1].Text != "valid content" {
		t.Errorf("blocks[1].Text = %q, want %q", blocks[1].Text, "valid content")
	}
}
