package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

func TestGrepTool_Name(t *testing.T) {
	tool := NewGrepTool(trustedpath.NopChecker())
	if tool.Name() != "grep" {
		t.Errorf("expected name 'grep', got '%s'", tool.Name())
	}
}

func TestGrepTool_Description(t *testing.T) {
	tool := NewGrepTool(trustedpath.NopChecker())
	desc := tool.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestGrepTool_Execute_MissingPattern(t *testing.T) {
	tool := NewGrepTool(trustedpath.NopChecker())
	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error return: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Error("expected error status for missing pattern")
	}
}

func TestGrepTool_Execute_InvalidRegex(t *testing.T) {
	tool := NewGrepTool(trustedpath.NopChecker())
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "[invalid(regex", // Invalid regex
	})
	if err != nil {
		t.Fatalf("unexpected error return: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Error("expected error status for invalid regex pattern")
	}
}

func TestGrepTool_Execute_ValidPattern(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with content
	testFiles := map[string]string{
		"test.go":   "package main\nfunc main() {\n\t// TODO: implement\n}",
		"test.txt":  "hello world\nfoo bar\nTODO: fix this",
		"empty.txt": "nothing here",
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create checker that trusts the temp directory
	checker := trustedpath.NewChecker([]string{tmpDir})
	tool := NewGrepTool(checker)

	// Test searching for "TODO"
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "TODO",
		"path":    tmpDir,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find matches in test.go and test.txt
	if result.Content == "" {
		t.Error("expected non-empty result")
	}
}

func TestGrepTool_Execute_NoMatches(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grep-nomatch-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create checker that trusts the temp directory
	checker := trustedpath.NewChecker([]string{tmpDir})
	tool := NewGrepTool(checker)

	// Test searching for pattern that doesn't exist
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "NONEXISTENT_PATTERN_XYZ123",
		"path":    tmpDir,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should report no matches
	if result.Content == "" {
		t.Error("expected non-empty result")
	}
}
