package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultProfile tests default profile creation.
func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()

	if p.Content == "" {
		t.Error("DefaultProfile().Content should not be empty")
	}
	if !strings.Contains(p.Content, "# 关于我") {
		t.Error("DefaultProfile().Content should contain template header")
	}
}

// TestProfileSaveAndLoad tests saving and loading a profile.
func TestProfileSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "profile.md")

	content := "# About Me\n\n- Name: Test User\n- Occupation: Developer\n\n# Skills\n\n- Go (expert)\n- Python (intermediate)\n"
	original := &Profile{Content: content}

	// Save
	if err := original.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare
	if loaded.Content != original.Content {
		t.Errorf("Content mismatch:\ngot:  %q\nwant: %q", loaded.Content, original.Content)
	}
}

// TestLoadNonExistentFile tests loading a non-existent file.
func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/profile.md")
	if err == nil {
		t.Error("Load() expected error for non-existent file")
	}
}

// TestProfileSaveToInvalidDirectory tests saving to a nested path (auto-creates dirs).
func TestProfileSaveToInvalidDirectory(t *testing.T) {
	p := DefaultProfile()
	path := "/tmp/nonexistent/deeply/nested/path/profile.md"

	err := p.Save(path)
	if err != nil {
		t.Errorf("Save() unexpected error = %v", err)
	}

	// Clean up
	os.RemoveAll("/tmp/nonexistent")
}

// TestProfileToSystemPrompt tests ToSystemPrompt method.
func TestProfileToSystemPrompt(t *testing.T) {
	p := &Profile{Content: "# About Me\n\n- Name: John\n- Occupation: Engineer\n"}
	prompt := p.ToSystemPrompt()

	if !strings.Contains(prompt, "User Profile:") {
		t.Error("ToSystemPrompt() should contain 'User Profile:' header")
	}
	if !strings.Contains(prompt, "John") {
		t.Error("ToSystemPrompt() should contain profile content")
	}
}

// TestProfileDisplayName tests DisplayName extraction.
func TestProfileDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Chinese name field",
			content: "# 关于我\n\n- 名字：张三\n- 职业：工程师\n",
			want:    "张三",
		},
		{
			name:    "English name field",
			content: "# About Me\n\n- Name: John Doe\n- Occupation: Engineer\n",
			want:    "John Doe",
		},
		{
			name:    "No name field",
			content: "# About Me\n\nSome text without a name.\n",
			want:    "Aura",
		},
		{
			name:    "Empty content",
			content: "",
			want:    "Aura",
		},
		{
			name:    "Name with colon",
			content: "- Name: Alice\n",
			want:    "Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Profile{Content: tt.content}
			got := p.DisplayName()
			if got != tt.want {
				t.Errorf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestProfileSaveCreatesParentDirs tests that Save auto-creates parent directories.
func TestProfileSaveCreatesParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sub", "dir", "profile.md")

	p := &Profile{Content: "test content"}
	if err := p.Save(tmpFile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("File content = %q, want %q", string(data), "test content")
	}
}
