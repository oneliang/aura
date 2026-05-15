package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRoleLoader(t *testing.T) {
	loader := NewRoleLoader("")

	if loader == nil {
		t.Fatal("NewRoleLoader() returned nil")
	}

	// Should default to ~/.aura/roles/
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".aura", "roles")
	if loader.baseDir != expectedDir {
		t.Errorf("NewRoleLoader() baseDir = %q, want %q", loader.baseDir, expectedDir)
	}
}

func TestNewRoleLoader_CustomDir(t *testing.T) {
	customDir := "/tmp/test-roles"
	loader := NewRoleLoader(customDir)

	if loader == nil {
		t.Fatal("NewRoleLoader() returned nil")
	}

	if loader.baseDir != customDir {
		t.Errorf("NewRoleLoader() baseDir = %q, want %q", loader.baseDir, customDir)
	}
}

func TestRoleLoader_Load_EmptyRole(t *testing.T) {
	loader := NewRoleLoader("")
	result := loader.Load("")

	if result != "" {
		t.Errorf("Load(\"\") = %q, want empty string", result)
	}
}

func TestRoleLoader_Load_FileNotFound(t *testing.T) {
	// Create loader with non-existent directory
	loader := NewRoleLoader("/non-existent-dir")
	result := loader.Load("test-role")

	if result != "" {
		t.Errorf("Load() with non-existent dir = %q, want empty string", result)
	}
}

func TestRoleLoader_Load_Success(t *testing.T) {
	// Create temp directory and role file
	tmpDir := t.TempDir()
	rolePath := filepath.Join(tmpDir, "test-role.md")
	roleContent := "You are a test assistant."

	err := os.WriteFile(rolePath, []byte(roleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test role file: %v", err)
	}

	loader := NewRoleLoader(tmpDir)
	result := loader.Load("test-role")

	if result != roleContent {
		t.Errorf("Load() = %q, want %q", result, roleContent)
	}
}

func TestRoleLoader_Load_WhitespaceTrimming(t *testing.T) {
	tmpDir := t.TempDir()
	rolePath := filepath.Join(tmpDir, "trimmed-role.md")
	roleContent := "  You are a trimmed assistant.  \n\n"

	err := os.WriteFile(rolePath, []byte(roleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test role file: %v", err)
	}

	loader := NewRoleLoader(tmpDir)
	result := loader.Load("trimmed-role")

	expected := "You are a trimmed assistant."
	if result != expected {
		t.Errorf("Load() = %q, want %q", result, expected)
	}
}

func TestRoleLoader_Available_EmptyRole(t *testing.T) {
	loader := NewRoleLoader("")
	result := loader.Available("")

	if result {
		t.Error("Available(\"\") = true, want false")
	}
}

func TestRoleLoader_Available_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewRoleLoader(tmpDir)
	result := loader.Available("non-existent")

	if result {
		t.Error("Available() with non-existent file = true, want false")
	}
}

func TestRoleLoader_Available_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	rolePath := filepath.Join(tmpDir, "existing-role.md")

	err := os.WriteFile(rolePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test role file: %v", err)
	}

	loader := NewRoleLoader(tmpDir)
	result := loader.Available("existing-role")

	if !result {
		t.Error("Available() with existing file = false, want true")
	}
}

func TestRoleLoader_List(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some role files
	roles := []string{"role1", "role2", "role3"}
	for _, role := range roles {
		rolePath := filepath.Join(tmpDir, role+".md")
		err := os.WriteFile(rolePath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test role file: %v", err)
		}
	}

	// Create a non-md file (should be ignored)
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not a role"), 0644)

	// Create a subdirectory (should be ignored)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	loader := NewRoleLoader(tmpDir)
	result, err := loader.List()

	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("List() returned %d roles, want 3", len(result))
	}

	// Check all roles are present
	roleSet := make(map[string]bool)
	for _, r := range result {
		roleSet[r] = true
	}

	for _, expected := range roles {
		if !roleSet[expected] {
			t.Errorf("List() missing role: %s", expected)
		}
	}
}

func TestRoleLoader_List_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewRoleLoader(tmpDir)
	result, err := loader.List()

	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("List() returned %d roles, want 0", len(result))
	}
}

func TestRoleLoader_List_NonExistentDirectory(t *testing.T) {
	loader := NewRoleLoader("/non-existent-dir")
	result, err := loader.List()

	if err == nil {
		t.Error("List() with non-existent dir should return error")
	}

	if result != nil {
		t.Error("List() with non-existent dir should return nil slice")
	}
}
