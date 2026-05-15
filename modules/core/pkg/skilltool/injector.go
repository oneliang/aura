package skilltool

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/core/pkg/llm"
	skill "github.com/oneliang/aura/skill/pkg/skill"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// SkillInjector manages skill body injection for cache-aware message building.
// This implements the second stage of Progressive Disclosure:
// - Inject skill body and store for dynamic message retrieval
// - Deduplication: same skill won't be injected twice
type SkillInjector struct {
	injectedSkills map[string]bool   // Track which skills have been injected
	skillBodies    map[string]string // Store activated skill bodies for dynamic message
}

// NewSkillInjector creates a new SkillInjector.
func NewSkillInjector() *SkillInjector {
	return &SkillInjector{
		injectedSkills: make(map[string]bool),
		skillBodies:    make(map[string]string),
	}
}

// InjectSkill creates a system message with the skill body and stores it for retrieval.
// Deduplication is handled internally - already injected skills are skipped.
// Returns the message content for immediate use in ToolResult.
func (i *SkillInjector) InjectSkill(sk skill.Skill) llm.Message {
	// Deduplication: skip if already injected
	if i.injectedSkills[sk.Name] {
		return llm.Message{
			Role:          "system",
			ContentBlocks: []sharedmemory.ContentBlock{
				sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: i.skillBodies[sk.Name]},
			},
		}
	}

	i.injectedSkills[sk.Name] = true

	// Replace {skill_dir} placeholder with actual skill directory path
	skillDir := filepath.Dir(sk.FilePath)
	bodyWithPaths := strings.ReplaceAll(sk.Body, "{skill_dir}", skillDir)

	content := fmt.Sprintf(`## Skill: %s

**重要**：这是一个专业技能。请根据用户需求，使用你的工具（如 bash）执行 skill 中描述的操作。

%s`, sk.Name, bodyWithPaths)

	// Store for retrieval in buildReActMessages (cache-aware path)
	i.skillBodies[sk.Name] = content

	return llm.Message{
		Role:          "system",
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
	}
}

// GetInjectedBodies returns all activated skill bodies.
// Used by Engine to add skill bodies as dynamic messages (not cached).
func (i *SkillInjector) GetInjectedBodies() []string {
	bodies := make([]string, 0, len(i.skillBodies))
	for _, body := range i.skillBodies {
		bodies = append(bodies, body)
	}
	return bodies
}

// Reset clears the injection history and stored bodies.
// Call this when starting a new session/conversation.
func (i *SkillInjector) Reset() {
	i.injectedSkills = make(map[string]bool)
	i.skillBodies = make(map[string]string)
}
