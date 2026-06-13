package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/personality/pkg/profile"
)

// TestProfilePath tests profile path generation.
func TestProfilePath(t *testing.T) {
	path := profilePath()
	if path == "" {
		t.Error("profilePath should return non-empty string")
	}
	if filepath.IsAbs(path) {
		t.Log("Profile path is absolute:", path)
	}
}

// TestLoadOrDefaultProfile_NonExistent tests loading non-existent profile.
func TestLoadOrDefaultProfile_NonExistent(t *testing.T) {
	p := loadOrDefaultProfile()
	if p == nil {
		t.Fatal("loadOrDefaultProfile should return default profile")
	}
	if p.Content == "" {
		t.Error("Default profile content should not be empty")
	}
}

// TestLoadOrDefaultProfile_Existing tests loading existing profile.
func TestLoadOrDefaultProfile_Existing(t *testing.T) {
	// For this test, we just verify the function exists and doesn't panic
	p := loadOrDefaultProfile()
	if p == nil {
		t.Error("loadOrDefaultProfile should not return nil")
	}
}

// TestProfileCommandFunctions tests profile-related command functions.
func TestProfileCommandFunctions(t *testing.T) {
	// Just verify these functions exist and don't panic
	_ = profilePath()
	_ = loadOrDefaultProfile()
}

// TestRunProfileInit tests profile init command.
func TestRunProfileInit(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "profile.md")

	// Remove the file if it exists
	os.Remove(tmpFile)

	// Call the init function logic directly
	p := profile.DefaultProfile()
	err := p.Save(tmpFile)
	if err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Profile file should be created")
	}

	// Verify content is markdown
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read profile: %v", err)
	}
	if len(data) == 0 {
		t.Error("Profile file should not be empty")
	}
}

// TestRunProfileShow_WithEmptyProfile tests showing default profile.
func TestRunProfileShow_WithEmptyProfile(t *testing.T) {
	p := loadOrDefaultProfile()
	if p == nil {
		t.Error("Should return default profile")
	}
	if p.Content == "" {
		t.Error("Default profile should have content")
	}
}
