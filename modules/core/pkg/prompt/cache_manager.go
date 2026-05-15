// Package prompt provides prompt caching management.
package prompt

import (
	"sync"

	"github.com/oneliang/aura/core/pkg/llm"
)

// CacheLayer represents a cached prompt layer.
type CacheLayer int

const (
	LayerStaticSystem CacheLayer = 0 // Immutable base system prompt
	LayerTools        CacheLayer = 1 // Tool definitions
	LayerSkills       CacheLayer = 2 // Skills metadata
	LayerAgents       CacheLayer = 3 // Agents metadata
)

// PromptCacheManager manages cached prompt layers for provider-agnostic caching.
type PromptCacheManager struct {
	mu sync.RWMutex

	// Cached layers (immutable after initialization)
	staticSystem string
	toolsBlock   string
	skillsBlock  string
	agentsBlock  string

	// Configuration
	enabled bool
}

// NewPromptCacheManager creates a new cache manager.
func NewPromptCacheManager(enabled bool) *PromptCacheManager {
	return &PromptCacheManager{enabled: enabled}
}

// SetStaticSystem sets the immutable base system prompt.
func (m *PromptCacheManager) SetStaticSystem(prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.staticSystem = prompt
}

// SetToolsBlock sets the cached tool definitions block.
func (m *PromptCacheManager) SetToolsBlock(tools string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolsBlock = tools
}

// SetSkillsBlock sets the cached skills metadata block.
func (m *PromptCacheManager) SetSkillsBlock(skills string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skillsBlock = skills
}

// SetAgentsBlock sets the cached agents metadata block.
func (m *PromptCacheManager) SetAgentsBlock(agents string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentsBlock = agents
}

// BuildSystemBlocks builds Anthropic-style system blocks with cache_control.
func (m *PromptCacheManager) BuildSystemBlocks() []llm.SystemBlock {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return nil
	}

	var blocks []llm.SystemBlock

	// Layer 0: Static system prompt (always cached)
	if m.staticSystem != "" {
		blocks = append(blocks, llm.SystemBlock{
			Type:         "text",
			Text:         m.staticSystem,
			CacheControl: &llm.CacheControl{Type: "ephemeral"},
		})
	}

	// Layer 1: Tools block
	if m.toolsBlock != "" {
		blocks = append(blocks, llm.SystemBlock{
			Type:         "text",
			Text:         m.toolsBlock,
			CacheControl: &llm.CacheControl{Type: "ephemeral"},
		})
	}

	// Layer 2: Skills metadata
	if m.skillsBlock != "" {
		blocks = append(blocks, llm.SystemBlock{
			Type:         "text",
			Text:         m.skillsBlock,
			CacheControl: &llm.CacheControl{Type: "ephemeral"},
		})
	}

	// Layer 3: Agents metadata
	if m.agentsBlock != "" {
		blocks = append(blocks, llm.SystemBlock{
			Type:         "text",
			Text:         m.agentsBlock,
			CacheControl: &llm.CacheControl{Type: "ephemeral"},
		})
	}

	return blocks
}

// BuildOpenAICacheType returns the cache type for OpenAI request-level caching.
func (m *PromptCacheManager) BuildOpenAICacheType() string {
	if m.enabled {
		return "ephemeral"
	}
	return ""
}

// InvalidateTools invalidates the tools cache.
func (m *PromptCacheManager) InvalidateTools() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolsBlock = ""
}

// InvalidateSkills invalidates the skills cache.
func (m *PromptCacheManager) InvalidateSkills() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skillsBlock = ""
}

// InvalidateAgents invalidates the agents cache.
func (m *PromptCacheManager) InvalidateAgents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentsBlock = ""
}

// Enabled returns whether caching is enabled.
func (m *PromptCacheManager) Enabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}