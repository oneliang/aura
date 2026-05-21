package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/skill/pkg/skill"
)

func TestParseSkill(t *testing.T) {
	content := `---
name: test-skill
description: A test skill for testing
---

## Workflow

1. Step one
2. Step two
`

	sk, _, err := parseSkill(content, "/path/to/skill.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", sk.Name)
	}
	if sk.Description != "A test skill for testing" {
		t.Errorf("expected description 'A test skill for testing', got '%s'", sk.Description)
	}
	if sk.Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestParseSkill_InvalidFormat(t *testing.T) {
	// Missing frontmatter
	_, _, err := parseSkill("No frontmatter", "/path/to/skill.md")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}

	// Missing name
	_, _, err = parseSkill(`---
description: No name
---
Body`, "/path/to/skill.md")
	if err == nil {
		t.Error("expected error for missing name")
	}

	// Missing description
	_, _, err = parseSkill(`---
name: no-desc
---
Body`, "/path/to/skill.md")
	if err == nil {
		t.Error("expected error for missing description")
	}
}

func TestLoader_Load(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "skill-loader-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a skill directory
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	// Write SKILL.md
	skillContent := `---
name: test-skill
description: A test skill
---

## Body

Test content.
`
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Load skills
	loader := NewLoader([]string{tmpDir})
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	sk := skills[0]
	if sk.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", sk.Name)
	}
	if sk.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got '%s'", sk.Description)
	}
}

func TestLoader_Load_NonExistentDirectory(t *testing.T) {
	loader := NewLoader([]string{"/non/existent/dir"})
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestLoader_GetSkills(t *testing.T) {
	loader := NewLoader([]string{})
	expected := []skill.Skill{
		{Name: "skill1", Description: "desc1"},
		{Name: "skill2", Description: "desc2"},
	}
	loader.skills = expected

	skills := loader.GetSkills()
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}
