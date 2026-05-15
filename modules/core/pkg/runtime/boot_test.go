package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestLoadProjectAuraMd(t *testing.T) {
	// Create temporary directory with AURA.md
	tmpDir := t.TempDir()
	auraMdPath := filepath.Join(tmpDir, "AURA.md")
	testContent := "# Test Project\n\nBuild: make build\n\nNotes: Test notes"
	
	if err := os.WriteFile(auraMdPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write AURA.md: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create minimal runtime to test
	cfg := &config.Config{}
	rt := &AgentRuntime{
		config: &RuntimeConfig{
			Config: cfg,
		},
	}

	// Test loadProjectAuraMd
	result := rt.loadProjectAuraMd()
	
	// Verify content is loaded
	if result == "" {
		t.Error("Expected AURA.md content to be loaded, got empty string")
	}
	if !contains(result, "Test Project") {
		t.Errorf("Expected 'Test Project' in result, got: %s", result)
	}
	if !contains(result, "Project Instructions (AURA.md)") {
		t.Errorf("Expected section header in result, got: %s", result)
	}
}

func TestLoadProjectAuraMdNoFile(t *testing.T) {
	// Create temporary directory without AURA.md
	tmpDir := t.TempDir()

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create minimal runtime to test
	cfg := &config.Config{}
	rt := &AgentRuntime{
		config: &RuntimeConfig{
			Config: cfg,
		},
	}

	// Test loadProjectAuraMd when no file exists
	result := rt.loadProjectAuraMd()
	
	// Should return empty string when no file
	if result != "" {
		t.Errorf("Expected empty string when no AURA.md, got: %s", result)
	}
}

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
