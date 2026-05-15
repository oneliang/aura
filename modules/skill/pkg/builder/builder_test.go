package builder

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/skill/pkg/skill"
)

func TestBuildSystemPromptSection(t *testing.T) {
	skills := []skill.Skill{
		{Name: "skill1", Description: "Description 1"},
		{Name: "skill2", Description: "Description 2"},
	}

	prompt := BuildSystemPromptSection(skills)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "## Skills") {
		t.Error("expected prompt to contain '## Skills'")
	}
	if !strings.Contains(prompt, "skill1") {
		t.Error("expected prompt to contain 'skill1'")
	}
	if !strings.Contains(prompt, "Description 1") {
		t.Error("expected prompt to contain 'Description 1'")
	}
}

func TestBuildSystemPromptSection_Empty(t *testing.T) {
	prompt := BuildSystemPromptSection([]skill.Skill{})
	if prompt != "" {
		t.Errorf("expected empty prompt, got '%s'", prompt)
	}
}

func TestBuildFullPrompt(t *testing.T) {
	sk := skill.Skill{
		Name:        "test-skill",
		Description: "Test description",
		Body:        "## Workflow\n\n1. Step one\n2. Step two",
	}

	prompt := BuildFullPrompt(sk)

	if !strings.Contains(prompt, "Using Skill: test-skill") {
		t.Error("expected prompt to contain skill name")
	}
	if !strings.Contains(prompt, "Test description") {
		t.Error("expected prompt to contain description")
	}
	if !strings.Contains(prompt, "## Workflow") {
		t.Error("expected prompt to contain body")
	}
}

func TestBuildSkillMetadata(t *testing.T) {
	skills := []skill.Skill{
		{Name: "skill1", Description: "Description 1"},
		{Name: "skill2", Description: "Description 2"},
	}

	metadata := BuildSkillMetadata(skills)

	expected := "skill1: Description 1; skill2: Description 2"
	if metadata != expected {
		t.Errorf("expected '%s', got '%s'", expected, metadata)
	}
}

func TestBuildSkillMetadata_Empty(t *testing.T) {
	metadata := BuildSkillMetadata([]skill.Skill{})
	if metadata != "" {
		t.Errorf("expected empty metadata, got '%s'", metadata)
	}
}
