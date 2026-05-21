package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultProfile tests default profile creation.
func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()

	if p.BasicInfo.Name != "User" {
		t.Errorf("BasicInfo.Name = %q, want %q", p.BasicInfo.Name, "User")
	}
	if p.Style.Tone != "casual" {
		t.Errorf("Style.Tone = %q, want %q", p.Style.Tone, "casual")
	}
	if p.Style.Vocabulary != "simple" {
		t.Errorf("Style.Vocabulary = %q, want %q", p.Style.Vocabulary, "simple")
	}
	if p.Style.Humor != 0.3 {
		t.Errorf("Style.Humor = %v, want %v", p.Style.Humor, 0.3)
	}
	if p.Style.Verbosity != "concise" {
		t.Errorf("Style.Verbosity = %q, want %q", p.Style.Verbosity, "concise")
	}
}

// TestProfileSaveAndLoad tests saving and loading a profile.
func TestProfileSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "profile.yaml")

	// Create a test profile
	original := &Profile{
		BasicInfo: BasicInfo{
			Name:       "Test User",
			Location:   "Test City",
			Occupation: "Developer",
		},
		Background: "Test background",
		Skills: []Skill{
			{Name: "Go", Level: "expert", Category: "Programming"},
			{Name: "Python", Level: "intermediate", Category: "Programming"},
		},
		Experiences: []Experience{
			{Title: "Senior Developer", Description: "Work description", StartYear: 2020, EndYear: 0},
		},
		Preferences: []Preference{
			{Category: "Communication", Value: "Direct"},
		},
		Style: Style{
			Tone:       "technical",
			Vocabulary: "technical",
			Humor:      0.5,
			Verbosity:  "detailed",
		},
	}

	// Save the profile
	if err := original.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load the profile
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare
	if loaded.BasicInfo.Name != original.BasicInfo.Name {
		t.Errorf("BasicInfo.Name = %q, want %q", loaded.BasicInfo.Name, original.BasicInfo.Name)
	}
	if loaded.BasicInfo.Location != original.BasicInfo.Location {
		t.Errorf("BasicInfo.Location = %q, want %q", loaded.BasicInfo.Location, original.BasicInfo.Location)
	}
	if loaded.BasicInfo.Occupation != original.BasicInfo.Occupation {
		t.Errorf("BasicInfo.Occupation = %q, want %q", loaded.BasicInfo.Occupation, original.BasicInfo.Occupation)
	}
	if len(loaded.Skills) != len(original.Skills) {
		t.Errorf("Skills len = %d, want %d", len(loaded.Skills), len(original.Skills))
	}
	if len(loaded.Experiences) != len(original.Experiences) {
		t.Errorf("Experiences len = %d, want %d", len(loaded.Experiences), len(original.Experiences))
	}
	if loaded.Style.Tone != original.Style.Tone {
		t.Errorf("Style.Tone = %q, want %q", loaded.Style.Tone, original.Style.Tone)
	}
}

// TestLoadNonExistentFile tests loading a non-existent file.
func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/profile.yaml")

	if err == nil {
		t.Error("Load() expected error for non-existent file")
	}
}

// TestProfileSaveToInvalidDirectory tests saving to an invalid directory.
func TestProfileSaveToInvalidDirectory(t *testing.T) {
	p := DefaultProfile()

	// Try to save to a deeply nested non-existent path
	err := p.Save("/tmp/nonexistent/deeply/nested/path/profile.yaml")

	// Should succeed because MkdirAll creates directories
	if err != nil {
		t.Errorf("Save() unexpected error = %v", err)
	}

	// Clean up
	os.RemoveAll("/tmp/nonexistent")
}

// TestProfileToSystemPrompt tests ToSystemPrompt method.
func TestProfileToSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		profile  *Profile
		contains []string
	}{
		{
			name: "full profile",
			profile: &Profile{
				BasicInfo: BasicInfo{
					Name:       "John",
					Occupation: "Engineer",
				},
				Background: "Software development",
				Skills: []Skill{
					{Name: "Go", Level: "expert"},
				},
				Style: Style{
					Tone:       "technical",
					Vocabulary: "technical",
					Verbosity:  "detailed",
				},
			},
			contains: []string{
				"John",
				"Engineer",
				"Software development",
				"Go",
				"expert",
				"technical",
				"detailed",
			},
		},
		{
			name: "minimal profile",
			profile: &Profile{
				BasicInfo: BasicInfo{},
				Style:     Style{},
			},
			contains: []string{"Communication style"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := tt.profile.ToSystemPrompt()

			for _, want := range tt.contains {
				if !containsString(prompt, want) {
					t.Errorf("ToSystemPrompt() missing %q", want)
				}
			}
		})
	}
}

// TestProfileWithEmptySlices tests profile with empty slices.
func TestProfileWithEmptySlices(t *testing.T) {
	p := &Profile{
		BasicInfo:   BasicInfo{Name: "Test"},
		Skills:      []Skill{},
		Experiences: []Experience{},
		Preferences: []Preference{},
		Style:       DefaultProfile().Style,
	}

	prompt := p.ToSystemPrompt()

	// Should not panic and should contain basic info
	if !containsString(prompt, "Test") {
		t.Error("ToSystemPrompt() should contain name")
	}
}

// TestSkillSerialization tests skill YAML serialization.
func TestSkillSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "skill_test.yaml")

	p := &Profile{
		BasicInfo: BasicInfo{Name: "Skill Test"},
		Skills: []Skill{
			{Name: "Rust", Level: "beginner", Category: "Systems"},
			{Name: "JavaScript", Level: "expert"}, // No category
		},
		Style: DefaultProfile().Style,
	}

	// Save and load
	if err := p.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(loaded.Skills))
	}
}

// TestExperienceSerialization tests experience YAML serialization.
func TestExperienceSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "exp_test.yaml")

	p := &Profile{
		BasicInfo: BasicInfo{Name: "Exp Test"},
		Experiences: []Experience{
			{Title: "Current Job", Description: "Working", StartYear: 2023, EndYear: 0},
			{Title: "Previous Job", Description: "Worked", StartYear: 2020, EndYear: 2023},
		},
		Style: DefaultProfile().Style,
	}

	if err := p.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Experiences) != 2 {
		t.Errorf("Expected 2 experiences, got %d", len(loaded.Experiences))
	}
}

// TestPreferenceSerialization tests preference YAML serialization.
func TestPreferenceSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "pref_test.yaml")

	p := &Profile{
		BasicInfo: BasicInfo{Name: "Pref Test"},
		Preferences: []Preference{
			{Category: "Language", Value: "Go"},
			{Category: "Editor", Value: "VS Code"},
		},
		Style: DefaultProfile().Style,
	}

	if err := p.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Preferences) != 2 {
		t.Errorf("Expected 2 preferences, got %d", len(loaded.Preferences))
	}
}

// TestStyleSerialization tests style YAML serialization.
func TestStyleSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "style_test.yaml")

	p := &Profile{
		BasicInfo: BasicInfo{Name: "Style Test"},
		Style: Style{
			Tone:       "formal",
			Vocabulary: "technical",
			Humor:      0.8,
			Verbosity:  "detailed",
		},
	}

	if err := p.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Style.Tone != "formal" {
		t.Errorf("Style.Tone = %q, want %q", loaded.Style.Tone, "formal")
	}
	if loaded.Style.Humor != 0.8 {
		t.Errorf("Style.Humor = %v, want %v", loaded.Style.Humor, 0.8)
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
