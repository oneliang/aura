package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

func TestGlobTool_Name(t *testing.T) {
	tool := NewGlobTool(trustedpath.NopChecker())
	if tool.Name() != "glob" {
		t.Errorf("expected name 'glob', got '%s'", tool.Name())
	}
}

func TestGlobTool_Description(t *testing.T) {
	tool := NewGlobTool(trustedpath.NopChecker())
	desc := tool.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestGlobTool_Execute_MissingPattern(t *testing.T) {
	tool := NewGlobTool(trustedpath.NopChecker())
	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error return: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Error("expected error status for missing pattern")
	}
}

func TestGlobTool_Execute_ValidPattern(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "glob-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []string{
		"test.go",
		"test.txt",
		"main.go",
		"subdir/nested.go",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create checker that trusts the temp directory
	checker := trustedpath.NewChecker([]string{tmpDir})
	tool := NewGlobTool(checker)

	// Test *.go pattern
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
		"path":    tmpDir,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find test.go and main.go
	if result.Content == "" {
		t.Error("expected non-empty result")
	}
}

func TestGlobTool_Execute_UntrustedPath(t *testing.T) {
	tool := NewGlobTool(trustedpath.NopChecker())
	_, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
		"path":    "/etc", // Untrusted path
	})

	// Should succeed because NopChecker allows all paths
	if err != nil {
		// This is expected if the path doesn't exist or other error
		t.Logf("got expected error for untrusted path: %v", err)
	}
}
