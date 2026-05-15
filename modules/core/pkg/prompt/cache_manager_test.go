package prompt

import (
	"testing"
)

func TestPromptCacheManager_New(t *testing.T) {
	m := NewPromptCacheManager(true)
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if !m.Enabled() {
		t.Error("Expected enabled=true")
	}

	mDisabled := NewPromptCacheManager(false)
	if mDisabled.Enabled() {
		t.Error("Expected enabled=false")
	}
}

func TestPromptCacheManager_SetAndGetLayers(t *testing.T) {
	m := NewPromptCacheManager(true)

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

func TestPromptCacheManager_BuildSystemBlocks_Disabled(t *testing.T) {
	m := NewPromptCacheManager(false)
	m.SetStaticSystem("static prompt")

	blocks := m.BuildSystemBlocks()
	if blocks != nil {
		t.Errorf("Expected nil blocks when disabled, got %d blocks", len(blocks))
	}
}

func TestPromptCacheManager_BuildSystemBlocks_EmptyLayers(t *testing.T) {
	m := NewPromptCacheManager(true)
	// Don't set any layers

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks with empty layers, got %d", len(blocks))
	}
}

func TestPromptCacheManager_BuildSystemBlocks_PartialLayers(t *testing.T) {
	m := NewPromptCacheManager(true)
	m.SetStaticSystem("static prompt")
	m.SetToolsBlock("tools block")
	// Skip skills and agents

	blocks := m.BuildSystemBlocks()
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}
}

func TestPromptCacheManager_BuildOpenAICacheType(t *testing.T) {
	mEnabled := NewPromptCacheManager(true)
	if mEnabled.BuildOpenAICacheType() != "ephemeral" {
		t.Error("Expected 'ephemeral' for enabled manager")
	}

	mDisabled := NewPromptCacheManager(false)
	if mDisabled.BuildOpenAICacheType() != "" {
		t.Errorf("Expected empty string for disabled manager, got '%s'", mDisabled.BuildOpenAICacheType())
	}
}

func TestPromptCacheManager_Invalidate(t *testing.T) {
	m := NewPromptCacheManager(true)
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
	m := NewPromptCacheManager(true)

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
			_ = m.Enabled()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestPromptCacheManager_LayerOrder(t *testing.T) {
	m := NewPromptCacheManager(true)
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