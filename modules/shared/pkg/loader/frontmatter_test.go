package loader

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileSpec tests FileSpec struct.
func TestFileSpec(t *testing.T) {
	spec := FileSpec{FileName: "SKILL.md"}
	if spec.FileName != "SKILL.md" {
		t.Errorf("Expected FileName 'SKILL.md', got '%s'", spec.FileName)
	}
}

// TestResult tests Result struct.
func TestResult(t *testing.T) {
	type TestItem struct {
		Name string
	}

	result := Result[TestItem]{
		Item:     TestItem{Name: "test"},
		FilePath: "/path/to/file.md",
	}

	if result.Item.Name != "test" {
		t.Errorf("Expected Item.Name 'test', got '%s'", result.Item.Name)
	}
	if result.FilePath != "/path/to/file.md" {
		t.Errorf("Expected FilePath '/path/to/file.md', got '%s'", result.FilePath)
	}
}

// TestParseYAMLFrontmatter tests ParseYAMLFrontmatter function.
func TestParseYAMLFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantYAML    string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter",
			content: `---
name: test
---
Body content`,
			wantYAML: "name: test\n",
			wantBody: "Body content",
			wantErr:  false,
		},
		{
			name: "valid frontmatter with multiline body",
			content: `---
name: test
---
Line 1
Line 2
Line 3`,
			wantYAML: "name: test\n",
			wantBody: "Line 1\nLine 2\nLine 3",
			wantErr:  false,
		},
		{
			name:        "missing frontmatter",
			content:     "no frontmatter here",
			wantErr:     true,
			errContains: "missing YAML frontmatter",
		},
		{
			name: "unclosed frontmatter",
			content: `---
name: test
no closing`,
			wantErr:     true,
			errContains: "unclosed YAML frontmatter",
		},
		{
			name: "empty body",
			content: `---
name: test
---`,
			wantYAML: "name: test\n",
			wantBody: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent, body, err := ParseYAMLFrontmatter(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("Error should contain '%s', got '%s'", tt.errContains, err.Error())
					}
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if yamlContent != tt.wantYAML {
				t.Errorf("YAML content = %q, want %q", yamlContent, tt.wantYAML)
			}
			if body != tt.wantBody {
				t.Errorf("Body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

// TestUnmarshalYAMLFrontmatter tests UnmarshalYAMLFrontmatter function.
func TestUnmarshalYAMLFrontmatter(t *testing.T) {
	type TestMeta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	tests := []struct {
		name        string
		content     string
		wantMeta    TestMeta
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter",
			content: `---
name: test-name
description: test description
---
This is the body.`,
			wantMeta: TestMeta{Name: "test-name", Description: "test description"},
			wantBody: "This is the body.",
			wantErr:  false,
		},
		{
			name:        "missing frontmatter",
			content:     "no frontmatter",
			wantErr:     true,
			errContains: "missing YAML frontmatter",
		},
		{
			name: "invalid yaml",
			content: `---
name: [invalid
---
Body`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta TestMeta
			body, err := UnmarshalYAMLFrontmatter(tt.content, &meta)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if meta.Name != tt.wantMeta.Name {
				t.Errorf("Name = %q, want %q", meta.Name, tt.wantMeta.Name)
			}
			if meta.Description != tt.wantMeta.Description {
				t.Errorf("Description = %q, want %q", meta.Description, tt.wantMeta.Description)
			}
			if body != tt.wantBody {
				t.Errorf("Body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

// TestLoadFromDirectories tests LoadFromDirectories function.
func TestLoadFromDirectories(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "item1")
	subDir2 := filepath.Join(tmpDir, "item2")
	os.Mkdir(subDir1, 0755)
	os.Mkdir(subDir2, 0755)

	// Create files
	file1 := filepath.Join(subDir1, "TEST.md")
	file2 := filepath.Join(subDir2, "TEST.md")
	os.WriteFile(file1, []byte(`---
name: item1
---
Body 1`), 0644)
	os.WriteFile(file2, []byte(`---
name: item2
---
Body 2`), 0644)

	type TestItem struct {
		Name string `yaml:"name"`
	}

	parseFn := func(content, filePath string) (TestItem, string, error) {
		var item TestItem
		_, err := UnmarshalYAMLFrontmatter(content, &item)
		return item, "", err
	}

	results, err := LoadFromDirectories([]string{tmpDir}, FileSpec{FileName: "TEST.md"}, parseFn)
	if err != nil {
		t.Fatalf("LoadFromDirectories() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that both items were loaded
	names := make(map[string]bool)
	for _, r := range results {
		names[r.Item.Name] = true
	}
	if !names["item1"] || !names["item2"] {
		t.Errorf("Expected both item1 and item2, got %v", names)
	}
}

// TestLoadFromDirectories_NonExistentDir tests with non-existent directory.
func TestLoadFromDirectories_NonExistentDir(t *testing.T) {
	type TestItem struct {
		Name string `yaml:"name"`
	}

	parseFn := func(content, filePath string) (TestItem, string, error) {
		return TestItem{}, "", nil
	}

	// Non-existent directory should return empty results, not error
	results, err := LoadFromDirectories([]string{"/non/existent/path"}, FileSpec{FileName: "TEST.md"}, parseFn)
	if err != nil {
		t.Errorf("Expected no error for non-existent directory, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-existent directory, got %d", len(results))
	}
}

// TestLoadFromDirectories_EmptyBaseDirs tests with empty base directories.
func TestLoadFromDirectories_EmptyBaseDirs(t *testing.T) {
	type TestItem struct {
		Name string `yaml:"name"`
	}

	parseFn := func(content, filePath string) (TestItem, string, error) {
		return TestItem{}, "", nil
	}

	results, err := LoadFromDirectories([]string{}, FileSpec{FileName: "TEST.md"}, parseFn)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty base dirs, got %d", len(results))
	}
}

// TestLoadFromDirectories_SkipsFiles tests that files in base directory are skipped.
func TestLoadFromDirectories_SkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file (not a directory) in base dir
	fileInBase := filepath.Join(tmpDir, "regular-file.txt")
	os.WriteFile(fileInBase, []byte("content"), 0644)

	// Create a subdirectory with a matching file
	subDir := filepath.Join(tmpDir, "item1")
	os.Mkdir(subDir, 0755)
	fileInSubDir := filepath.Join(subDir, "TEST.md")
	os.WriteFile(fileInSubDir, []byte(`---
name: item1
---`), 0644)

	type TestItem struct {
		Name string `yaml:"name"`
	}

	parseFn := func(content, filePath string) (TestItem, string, error) {
		var item TestItem
		_, err := UnmarshalYAMLFrontmatter(content, &item)
		return item, "", err
	}

	results, err := LoadFromDirectories([]string{tmpDir}, FileSpec{FileName: "TEST.md"}, parseFn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only find the file in subdirectory
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

// TestLoadFromDirectories_SkipsMissingSpecFile tests skipping directories without spec file.
func TestLoadFromDirectories_SkipsMissingSpecFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory without the expected file
	subDir1 := filepath.Join(tmpDir, "no-file")
	os.Mkdir(subDir1, 0755)

	// Create subdirectory with the expected file
	subDir2 := filepath.Join(tmpDir, "has-file")
	os.Mkdir(subDir2, 0755)
	os.WriteFile(filepath.Join(subDir2, "TEST.md"), []byte(`---
name: item
---`), 0644)

	type TestItem struct {
		Name string `yaml:"name"`
	}

	parseFn := func(content, filePath string) (TestItem, string, error) {
		var item TestItem
		_, err := UnmarshalYAMLFrontmatter(content, &item)
		return item, "", err
	}

	results, err := LoadFromDirectories([]string{tmpDir}, FileSpec{FileName: "TEST.md"}, parseFn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Item.Name != "item" {
		t.Errorf("Expected item name 'item', got '%s'", results[0].Item.Name)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
