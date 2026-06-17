package prompt

import (
	"testing"
)

func TestPromptCacheManager_New(t *testing.T) {
	m := NewPromptCacheManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
}

func TestPromptCacheManager_SetAndGetLayers(t *testing.T) {
	m := NewPromptCacheManager()

	// Set all layers
	m.SetStaticSystem("static prompt")
	m.SetToolsBlock("tools block")
	m.SetSkillsBlock("skills block")
	m.SetAgentsBlock("agents block")

	// Verify via BuildSystemBlocks
	blocks := m.BuildSystemBlocks()
	if len(blocks) != 4 {
		t.Errorf("Expected 4 blocks, got %d", len(blocks))
	}

	// Check each block has correct content and cache_control
	for i, block := range blocks {
		if block.Type != "text" {
			t.Errorf("Block %d: expected type='text', got '%s'", i, block.Type)
		}
		if block.CacheControl == nil {
			t.Errorf("Block %d: expected cache_control non-nil", i)
		} else if block.CacheControl.Type != "ephemeral" {
			t.Errorf("Block %d: expected cache_control.type='ephemeral', got '%s'", i, block.CacheControl.Type)
		}
	}

	// Check content
	expectedContents := []string{"static prompt", "tools block", "skills block", "agents block"}
	for i, expected := range expectedContents {
		if blocks[i].Text != expected {
			t.Errorf("Block %d: expected text='%s', got '%s'", i, expected, blocks[i].Text)
		}
	}
}

func TestPromptCacheManager_BuildSystemBlocks_EmptyLayers(t *testing.T) {
	m := NewPromptCacheManager()
	// Don't set any layers

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks with empty layers, got %d", len(blocks))
	}
}

func TestPromptCacheManager_BuildSystemBlocks_PartialLayers(t *testing.T) {
	m := NewPromptCacheManager()
	m.SetStaticSystem("static prompt")
	m.SetToolsBlock("tools block")
	// Skip skills and agents

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}
}

func TestPromptCacheManager_BuildOpenAICacheType(t *testing.T) {
	m := NewPromptCacheManager()
	if m.BuildOpenAICacheType() != "ephemeral" {
		t.Errorf("Expected 'ephemeral', got '%s'", m.BuildOpenAICacheType())
	}
}

func TestPromptCacheManager_Invalidate(t *testing.T) {
	m := NewPromptCacheManager()
	m.SetStaticSystem("static")
	m.SetToolsBlock("tools")
	m.SetSkillsBlock("skills")
	m.SetAgentsBlock("agents")

	// Invalidate each layer
	m.InvalidateTools()
	m.InvalidateSkills()
	m.InvalidateAgents()

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block after invalidation, got %d", len(blocks))
	}
	if blocks[0].Text != "static" {
		t.Errorf("Expected static block to remain, got '%s'", blocks[0].Text)
	}
}

func TestPromptCacheManager_ConcurrentAccess(t *testing.T) {
	m := NewPromptCacheManager()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			m.SetStaticSystem("static" + string(rune(idx)))
			m.SetToolsBlock("tools" + string(rune(idx)))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = m.BuildSystemBlocks()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestPromptCacheManager_LayerOrder(t *testing.T) {
	m := NewPromptCacheManager()
	m.SetAgentsBlock("agents")  // Set out of order
	m.SetStaticSystem("static")
	m.SetSkillsBlock("skills")
	m.SetToolsBlock("tools")

	blocks := m.BuildSystemBlocks()
	// Verify order: static, tools, skills, agents (not dependent on set order)
	expectedOrder := []string{"static", "tools", "skills", "agents"}
	for i, expected := range expectedOrder {
		if blocks[i].Text != expected {
			t.Errorf("Block %d: expected '%s', got '%s'", i, expected, blocks[i].Text)
		}
	}
}

func TestPromptCacheManager_HookContextBlock(t *testing.T) {
	m := NewPromptCacheManager()
	m.SetStaticSystem("static")
	m.SetProfileBlock("profile")
	m.SetHookContextBlock("hook context")
	m.SetToolsBlock("tools")

	blocks := m.BuildSystemBlocks()
	// Expected order: static, profile, hook context, tools
	if len(blocks) != 4 {
		t.Fatalf("Expected 4 blocks, got %d", len(blocks))
	}

	expectedOrder := []string{"static", "profile", "hook context", "tools"}
	for i, expected := range expectedOrder {
		if blocks[i].Text != expected {
			t.Errorf("Block %d: expected '%s', got '%s'", i, expected, blocks[i].Text)
		}
	}
}

func TestPromptCacheManager_InvalidateHookContext(t *testing.T) {
	m := NewPromptCacheManager()
	m.SetStaticSystem("static")
	m.SetHookContextBlock("hook context")
	m.SetToolsBlock("tools")

	m.InvalidateHookContext()

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks after invalidation, got %d", len(blocks))
	}
}
