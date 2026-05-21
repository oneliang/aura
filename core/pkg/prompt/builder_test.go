package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestNewPromptBuilder(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	if builder == nil {
		t.Fatal("NewPromptBuilder() returned nil")
	}

	if builder.roleLoader != roleLoader {
		t.Error("NewPromptBuilder() did not store roleLoader correctly")
	}
}

func TestPromptBuilder_BuildBase(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	result := builder.BuildBase()

	if result == "" {
		t.Error("BuildBase() returned empty string")
	}

	// Check for expected content
	expectedSubstrings := []string{
		"Aura",
		"personal AI assistant",
		"helpful",
		"File Reference Verification", // Memory staleness reminder
		"VERIFY FIRST",
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("BuildBase() missing expected substring: %s", substr)
		}
	}
}

func TestPromptBuilder_BuildWithRole_EmptyRole(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	result := builder.BuildWithRole("")

	if result == "" {
		t.Error("BuildWithRole(\"\") returned empty string")
	}
	// Should return base prompt when role is empty
	base := builder.BuildBase()
	if result != base {
		t.Errorf("BuildWithRole(\"\") = %q, want base prompt %q", result, base)
	}
}

func TestPromptBuilder_BuildWithRole_NonExistent(t *testing.T) {
	roleLoader := NewRoleLoader("/non-existent-dir")
	builder := NewPromptBuilder(roleLoader)

	result := builder.BuildWithRole("non-existent")

	// Should return base prompt when role file doesn't exist
	base := builder.BuildBase()
	if result != base {
		t.Errorf("BuildWithRole() with non-existent file = %q, want base prompt %q", result, base)
	}
}

func TestPromptBuilder_BuildWithRole_Success(t *testing.T) {
	tmpDir := t.TempDir()
	rolePath := filepath.Join(tmpDir, "test-role.md")
	roleContent := "You are a test role."

	err := os.WriteFile(rolePath, []byte(roleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test role file: %v", err)
	}

	roleLoader := NewRoleLoader(tmpDir)
	builder := NewPromptBuilder(roleLoader)

	result := builder.BuildWithRole("test-role")

	// Should contain both base and role content
	base := builder.BuildBase()
	if !contains(result, "Aura") {
		t.Error("BuildWithRole() missing base prompt content")
	}
	if !contains(result, roleContent) {
		t.Error("BuildWithRole() missing role content")
	}
	if !contains(result, base) {
		t.Error("BuildWithRole() missing full base prompt")
	}
}

func TestPromptBuilder_BuildWithProfile(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	// Test with non-existent profile - should return base
	result := builder.BuildWithProfile("/non-existent/profile.yaml")
	base := builder.BuildBase()

	if result != base {
		t.Errorf("BuildWithProfile() with non-existent file = %q, want base prompt %q", result, base)
	}
}

func TestPromptBuilder_BuildWithProfile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profile.yaml")

	// Create a minimal profile file
	profileContent := `
tone: friendly
vocabulary: technical
`
	err := os.WriteFile(profilePath, []byte(profileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	result := builder.BuildWithProfile(profilePath)

	if result == "" {
		t.Error("BuildWithProfile() returned empty string")
	}
}

func TestPromptBuilder_BuildWithConfig(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "qwen3:8b",
		},
	}

	result := builder.BuildWithConfig(cfg)

	if result == "" {
		t.Error("BuildWithConfig() returned empty string")
	}

	// Should contain either base prompt content or profile-based content
	// The result should be non-empty and contain assistant-related text
	if len(result) < 10 {
		t.Error("BuildWithConfig() returned suspiciously short result")
	}
}

func TestPromptBuilder_BuildWithConfig_WithSSH(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	cfg := &config.Config{
		SSH: config.SSHConfig{
			Servers: []config.SSHServerConfig{
				{
					Name: "test-server",
					Host: "test.example.com",
					User: "testuser",
					Port: 22,
				},
			},
		},
	}

	result := builder.BuildWithConfig(cfg)

	if result == "" {
		t.Error("BuildWithConfig() returned empty string")
	}

	// Should contain SSH notes
	if !contains(result, "SSH") {
		t.Error("BuildWithConfig() missing SSH notes")
	}
	if !contains(result, "test-server") {
		t.Error("BuildWithConfig() missing server name")
	}
}

func TestPromptBuilder_BuildWithConfig_ProfileAndSSH(t *testing.T) {
	tmpDir := t.TempDir()

	// Create profile
	profilePath := filepath.Join(tmpDir, "profile.yaml")
	profileContent := `
tone: professional
`
	os.WriteFile(profilePath, []byte(profileContent), 0644)

	// Temporarily move home directory profile
	homeDir, _ := os.UserHomeDir()
	originalProfilePath := filepath.Join(homeDir, ".aura", "profile.yaml")

	// Read original if exists
	var originalProfile []byte
	var originalExists bool
	if _, err := os.Stat(originalProfilePath); err == nil {
		originalProfile, _ = os.ReadFile(originalProfilePath)
		originalExists = true
	}

	// Ensure .aura directory exists
	configDir := filepath.Join(homeDir, ".aura")
	os.MkdirAll(configDir, 0755)

	// Copy test profile
	testProfile, _ := os.ReadFile(profilePath)
	os.WriteFile(originalProfilePath, testProfile, 0644)

	// Restore original on cleanup
	defer func() {
		if originalExists {
			os.WriteFile(originalProfilePath, originalProfile, 0644)
		} else {
			os.Remove(originalProfilePath)
		}
	}()

	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	cfg := &config.Config{
		SSH: config.SSHConfig{
			Servers: []config.SSHServerConfig{
				{
					Name: "deploy-server",
					Host: "deploy.example.com",
					User: "deploy",
					Port: 22,
				},
			},
		},
	}

	result := builder.BuildWithConfig(cfg)

	if result == "" {
		t.Error("BuildWithConfig() returned empty string")
	}
}

func TestPromptBuilder_Combine(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	base := "This is the base prompt."
	custom := "This is custom content."

	result := builder.Combine(base, custom)

	expected := base + "\n\n" + custom
	if result != expected {
		t.Errorf("Combine() = %q, want %q", result, expected)
	}
}

func TestPromptBuilder_Combine_EmptyCustom(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	base := "This is the base prompt."

	result := builder.Combine(base, "")

	if result != base {
		t.Errorf("Combine() with empty custom = %q, want %q", result, base)
	}
}

func TestPromptBuilder_Combine_EmptyBase(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	custom := "This is custom content."

	result := builder.Combine("", custom)

	expected := "\n\n" + custom
	if result != expected {
		t.Errorf("Combine() with empty base = %q, want %q", result, expected)
	}
}

func TestPromptBuilder_BuildSSHNote(t *testing.T) {
	roleLoader := NewRoleLoader("")
	builder := NewPromptBuilder(roleLoader)

	servers := []config.SSHServerConfig{
		{
			Name: "server1",
			Host: "host1.example.com",
			User: "user1",
			Port: 22,
		},
		{
			Name: "server2",
			Host: "host2.example.com",
			User: "user2",
			Port: 2222,
		},
	}

	// Use reflection or direct access to test private method
	// Since buildSSHNote is private, we test it through BuildWithConfig
	cfg := &config.Config{
		SSH: config.SSHConfig{
			Servers: servers,
		},
	}

	result := builder.BuildWithConfig(cfg)

	// Verify SSH-related content is present
	if !contains(result, "SSH Remote Access") {
		t.Error("BuildWithConfig() missing SSH Remote Access header")
	}
	if !contains(result, "server1") {
		t.Error("BuildWithConfig() missing server1")
	}
	if !contains(result, "server2") {
		t.Error("BuildWithConfig() missing server2")
	}
	if !contains(result, "ssh_exec") {
		t.Error("BuildWithConfig() missing ssh_exec tool reference")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
