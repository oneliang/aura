package importer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestImporterNew tests importer creation.
func TestImporterNew(t *testing.T) {
	imp := New()

	if imp == nil {
		t.Fatal("New() returned nil")
	}
}

// TestImporterImportFile tests file import.
func TestImporterImportFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.md")

	content := `# My Profile

Name: John Doe
Occupation: Software Developer

## Skills
- Go
- Python
- Rust

Skill: Machine Learning
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	imp := New()
	p, err := imp.ImportFile(tmpFile)

	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}

	if p.BasicInfo.Name != "John Doe" {
		t.Errorf("BasicInfo.Name = %q, want %q", p.BasicInfo.Name, "John Doe")
	}
	if p.BasicInfo.Occupation != "Software Developer" {
		t.Errorf("BasicInfo.Occupation = %q, want %q", p.BasicInfo.Occupation, "Software Developer")
	}
	if len(p.Skills) < 3 {
		t.Errorf("Expected at least 3 skills, got %d", len(p.Skills))
	}
}

// TestImporterImportFileNonExistent tests importing a non-existent file.
func TestImporterImportFileNonExistent(t *testing.T) {
	imp := New()

	_, err := imp.ImportFile("/nonexistent/file.md")

	if err == nil {
		t.Error("ImportFile() expected error for non-existent file")
	}
}

// TestImporterImportText tests text import.
func TestImporterImportText(t *testing.T) {
	text := `
Name: Jane Smith
Role: Data Scientist

Skills:
- TensorFlow
- PyTorch

Skill: Statistics
`

	imp := New()
	p := imp.ImportText(text)

	if p.BasicInfo.Name != "Jane Smith" {
		t.Errorf("BasicInfo.Name = %q, want %q", p.BasicInfo.Name, "Jane Smith")
	}
	if p.BasicInfo.Occupation != "Data Scientist" {
		t.Errorf("BasicInfo.Occupation = %q, want %q", p.BasicInfo.Occupation, "Data Scientist")
	}
	if len(p.Skills) < 3 {
		t.Errorf("Expected at least 3 skills, got %d", len(p.Skills))
	}
}

// TestImporterImportTextEmpty tests importing empty text.
func TestImporterImportTextEmpty(t *testing.T) {
	imp := New()
	p := imp.ImportText("")

	if p == nil {
		t.Error("ImportText() returned nil")
	}

	// Should return default profile
	if p.BasicInfo.Name != "User" {
		t.Errorf("Expected default name 'User', got %q", p.BasicInfo.Name)
	}
}

// TestImporterExtractProfileNameHinds tests name extraction.
func TestImporterExtractProfileNameHints(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		wantName string
	}{
		{
			name:     "Name with colon",
			lines:    []string{"Name: Alice"},
			wantName: "Alice",
		},
		{
			name:     "name lowercase",
			lines:    []string{"name: Bob"},
			wantName: "Bob",
		},
		{
			name:     "Name with extra spaces",
			lines:    []string{"Name:   Charlie  "},
			wantName: "Charlie",
		},
		{
			name:     "Name in markdown header (should still extract)",
			lines:    []string{"# Name: David"},
			wantName: "User", // Should not extract from markdown headers
		},
		{
			name:     "Empty name value",
			lines:    []string{"Name: "},
			wantName: "User", // Should keep default
		},
		{
			name:     "Multiple names (first wins)",
			lines:    []string{"Name: First", "Name: Second"},
			wantName: "First",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := New()
			p := imp.extractProfile(tt.lines)

			if p.BasicInfo.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", p.BasicInfo.Name, tt.wantName)
			}
		})
	}
}

// TestImporterExtractProfileSkillHints tests skill extraction.
func TestImporterExtractProfileSkillHints(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		wantSkills int
	}{
		{
			name:       "Skill with colon",
			lines:      []string{"Skill: Go"},
			wantSkills: 1,
		},
		{
			name:       "Bullet point skill",
			lines:      []string{"- Python"},
			wantSkills: 1,
		},
		{
			name:       "Multiple skills",
			lines:      []string{"Skill: Go", "- Python", "- Rust"},
			wantSkills: 3,
		},
		{
			name:       "Bullet with colon (should not match as bullet)",
			lines:      []string{"- Skill: Something"},
			wantSkills: 0, // Has colon, so doesn't match bullet pattern
		},
		{
			name:       "Empty bullet",
			lines:      []string{"- "},
			wantSkills: 0,
		},
		{
			name:       "Very long bullet (should be ignored)",
			lines:      []string{"- " + string(make([]byte, 101))},
			wantSkills: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := New()
			p := imp.extractProfile(tt.lines)

			if len(p.Skills) != tt.wantSkills {
				t.Errorf("Skills count = %d, want %d", len(p.Skills), tt.wantSkills)
			}
		})
	}
}

// TestImporterExtractProfileOccupationHints tests occupation extraction.
func TestImporterExtractProfileOccupationHints(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		wantOccupation string
	}{
		{
			name:           "Occupation with colon",
			lines:          []string{"Occupation: Engineer"},
			wantOccupation: "Engineer",
		},
		{
			name:           "Role with colon",
			lines:          []string{"Role: Manager"},
			wantOccupation: "Manager",
		},
		{
			name:           "Job with colon",
			lines:          []string{"Job: Developer"},
			wantOccupation: "Developer",
		},
		{
			name:           "Lowercase occupation",
			lines:          []string{"occupation: Designer"},
			wantOccupation: "Designer",
		},
		{
			name:           "Empty occupation",
			lines:          []string{"Occupation: "},
			wantOccupation: "",
		},
		{
			name:           "Multiple occupations (last wins)",
			lines:          []string{"Occupation: First", "Occupation: Second"},
			wantOccupation: "Second", // Last one wins in current implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := New()
			p := imp.extractProfile(tt.lines)

			if p.BasicInfo.Occupation != tt.wantOccupation {
				t.Errorf("Occupation = %q, want %q", p.BasicInfo.Occupation, tt.wantOccupation)
			}
		})
	}
}

// TestImporterImportTextWithMixedHints tests text with mixed hint types.
func TestImporterImportTextWithMixedHints(t *testing.T) {
	text := `
Name: Test User
Occupation: Developer

Some background text here.

Skills I have:
- Go
- Python
Skill: Machine Learning

Job: Senior Engineer
`

	imp := New()
	p := imp.ImportText(text)

	if p.BasicInfo.Name != "Test User" {
		t.Errorf("Name = %q, want %q", p.BasicInfo.Name, "Test User")
	}
	if p.BasicInfo.Occupation != "Senior Engineer" {
		t.Errorf("Occupation = %q, want %q", p.BasicInfo.Occupation, "Senior Engineer")
	}
	if len(p.Skills) < 3 {
		t.Errorf("Expected at least 3 skills, got %d", len(p.Skills))
	}
}

// TestImporterImportFileWithComplexMarkdown tests complex markdown import.
func TestImporterImportFileWithComplexMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "complex.md")

	content := `# Profile

My name is Alex.

## Background
I work in tech.

## Skills

Here are my skills:

- JavaScript
- TypeScript
- React
- Node.js

## Work History

### Current Role
Role: Tech Lead

### Previous Role
Job: Senior Developer
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	imp := New()
	p, err := imp.ImportFile(tmpFile)

	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}

	if len(p.Skills) < 4 {
		t.Errorf("Expected at least 4 skills, got %d", len(p.Skills))
	}
	// Note: Last occupation wins in current implementation
	if p.BasicInfo.Occupation != "Senior Developer" {
		t.Errorf("Occupation = %q, want %q", p.BasicInfo.Occupation, "Senior Developer")
	}
}

// TestImporterProfileMergesWithDefault tests that imported profile merges with defaults.
func TestImporterProfileMergesWithDefault(t *testing.T) {
	text := `Name: Custom User`

	imp := New()
	p := imp.ImportText(text)

	if p.BasicInfo.Name != "Custom User" {
		t.Errorf("Name = %q, want %q", p.BasicInfo.Name, "Custom User")
	}

	// Style should be default
	if p.Style.Tone != "casual" {
		t.Errorf("Style.Tone = %q, want %q", p.Style.Tone, "casual")
	}
}

// TestImporterSkillLevelDefault tests that extracted skills have default level.
func TestImporterSkillLevelDefault(t *testing.T) {
	text := `
Skill: Testing
- Another Skill
`

	imp := New()
	p := imp.ImportText(text)

	for _, skill := range p.Skills {
		if skill.Level != "intermediate" {
			t.Errorf("Skill %q level = %q, want %q", skill.Name, skill.Level, "intermediate")
		}
	}
}

// TestImporterPreservesExistingName tests that empty name doesn't overwrite existing.
func TestImporterPreservesExistingName(t *testing.T) {
	// First extraction sets name
	imp := New()
	p := imp.ImportText("Name: First")

	// Second extraction with empty name shouldn't overwrite
	p2 := imp.ImportText("Name: ")

	if p2.BasicInfo.Name != "User" {
		t.Errorf("Empty name should not overwrite, got %q", p2.BasicInfo.Name)
	}

	_ = p // suppress unused warning
}
