package skill

import "testing"

func TestSkill(t *testing.T) {
	skill := Skill{
		Name:        "test-skill",
		Description: "A test skill",
		FilePath:    "/path/to/skill.md",
		Content:     "---\nname: test\n---\nBody content",
		Body:        "Body content",
	}

	if skill.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got '%s'", skill.Description)
	}
	if skill.FilePath != "/path/to/skill.md" {
		t.Errorf("expected filepath '/path/to/skill.md', got '%s'", skill.FilePath)
	}
}
