package skilltool

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/skill/pkg/skill"
)

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

func TestSkillInjector_InjectSkill(t *testing.T) {
	injector := NewSkillInjector()

	testSkill := skill.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "## Instructions\n\nDo something useful.",
	}

	msg := injector.InjectSkill(testSkill)

	assert.Equal(t, "system", msg.Role)
	assert.Contains(t, getTextContent(msg), "## Skill: test-skill")
	assert.Contains(t, getTextContent(msg), "## Instructions")
	assert.Contains(t, getTextContent(msg), "Do something useful")

	// Verify deduplication: second call returns cached body
	msg2 := injector.InjectSkill(testSkill)
	assert.Equal(t, getTextContent(msg), getTextContent(msg2), "duplicate injection should return cached body")
}

func TestSkillInjector_GetInjectedBodies(t *testing.T) {
	injector := NewSkillInjector()

	// Initially empty
	bodies := injector.GetInjectedBodies()
	assert.Len(t, bodies, 0)

	// Inject skills
	testSkill1 := skill.Skill{
		Name:        "postgres",
		Description: "PostgreSQL skill",
		Body:        "PostgreSQL instructions",
	}
	testSkill2 := skill.Skill{
		Name:        "pdf",
		Description: "PDF skill",
		Body:        "PDF instructions",
	}

	injector.InjectSkill(testSkill1)
	injector.InjectSkill(testSkill2)

	// Get bodies
	bodies = injector.GetInjectedBodies()
	assert.Len(t, bodies, 2)

	// Verify bodies contain skill content
	foundPostgres := false
	foundPdf := false
	for _, body := range bodies {
		assert.Contains(t, body, "## Skill:")
		if strings.Contains(body, "postgres") {
			foundPostgres = true
		}
		if strings.Contains(body, "pdf") {
			foundPdf = true
		}
	}
	assert.True(t, foundPostgres, "postgres body should be in GetInjectedBodies")
	assert.True(t, foundPdf, "pdf body should be in GetInjectedBodies")
}

func TestSkillInjector_Reset(t *testing.T) {
	injector := NewSkillInjector()

	// Inject some skills
	testSkill1 := skill.Skill{Name: "postgres", Body: "postgres body"}
	testSkill2 := skill.Skill{Name: "pdf", Body: "pdf body"}
	injector.InjectSkill(testSkill1)
	injector.InjectSkill(testSkill2)

	bodies := injector.GetInjectedBodies()
	assert.Len(t, bodies, 2)

	// Reset
	injector.Reset()
	bodies = injector.GetInjectedBodies()
	assert.Len(t, bodies, 0, "skillBodies should be cleared after Reset")

	// Verify can inject again after reset
	injector.InjectSkill(testSkill1)
	bodies = injector.GetInjectedBodies()
	assert.Len(t, bodies, 1)
}

func TestSkillInjector_InjectSkill_WithPathReplacement(t *testing.T) {
	injector := NewSkillInjector()

	// Test skill with {skill_dir} placeholder
	testSkill := skill.Skill{
		Name:        "postgres",
		Description: "PostgreSQL database skill",
		FilePath:    "/Users/test/skills/postgres/SKILL.md",
		Body:        "Use {skill_dir}/darwin-arm64/postgres --config {skill_dir}/config.json",
	}

	msg := injector.InjectSkill(testSkill)

	assert.Equal(t, "system", msg.Role)
	assert.Contains(t, getTextContent(msg), "## Skill: postgres")
	// Verify {skill_dir} was replaced with actual path
	assert.Contains(t, getTextContent(msg), "/Users/test/skills/postgres/darwin-arm64/postgres")
	assert.Contains(t, getTextContent(msg), "/Users/test/skills/postgres/config.json")
	// Verify placeholder is NOT present
	assert.NotContains(t, getTextContent(msg), "{skill_dir}")
}

func TestSkillInjector_Deduplication(t *testing.T) {
	injector := NewSkillInjector()

	testSkill := skill.Skill{
		Name: "duplicate-test",
		Body: "Original body content",
	}

	// First injection
	msg1 := injector.InjectSkill(testSkill)
	assert.Contains(t, getTextContent(msg1), "Original body content")

	// Modify skill body (simulate different content)
	testSkill.Body = "Modified body content"

	// Second injection - should return cached (original) body, not modified
	msg2 := injector.InjectSkill(testSkill)
	assert.Contains(t, getTextContent(msg2), "Original body content", "should return cached body, not modified")
	assert.NotContains(t, getTextContent(msg2), "Modified body content")

	// Only one body stored
	bodies := injector.GetInjectedBodies()
	assert.Len(t, bodies, 1)
}