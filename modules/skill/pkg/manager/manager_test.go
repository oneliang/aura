package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/skill/pkg/loader"
)

// TestCreateSkillRequest tests CreateSkillRequest struct.
func TestCreateSkillRequest(t *testing.T) {
	req := CreateSkillRequest{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "Skill body content",
	}

	if req.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", req.Name)
	}
	if req.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got '%s'", req.Description)
	}
	if req.Body != "Skill body content" {
		t.Errorf("Expected body 'Skill body content', got '%s'", req.Body)
	}
}

// TestUpdateSkillRequest tests UpdateSkillRequest struct.
func TestUpdateSkillRequest(t *testing.T) {
	desc := "updated description"
	body := "updated body"
	req := UpdateSkillRequest{
		Description: &desc,
		Body:        &body,
	}

	if req.Description == nil || *req.Description != "updated description" {
		t.Error("Description should be 'updated description'")
	}
	if req.Body == nil || *req.Body != "updated body" {
		t.Error("Body should be 'updated body'")
	}
}

// TestNewSkillManager tests NewSkillManager function.
func TestNewSkillManager(t *testing.T) {
	baseDirs := []string{"/tmp/skills"}
	mgr := NewSkillManager(nil, baseDirs)

	if mgr == nil {
		t.Fatal("NewSkillManager() returned nil")
	}
	if len(mgr.baseDirs) != 1 {
		t.Errorf("Expected 1 base dir, got %d", len(mgr.baseDirs))
	}
}

// TestSkillManager_List tests List method.
func TestSkillManager_List(t *testing.T) {
	mgr := NewSkillManager(nil, nil)

	// With nil loader
	result := mgr.List()
	if result != nil {
		t.Errorf("Expected nil list with nil loader, got %v", result)
	}
}

// TestSkillManager_Get tests Get method.
func TestSkillManager_Get(t *testing.T) {
	mgr := NewSkillManager(nil, nil)

	// With nil loader
	result := mgr.Get("nonexistent")
	if result != nil {
		t.Errorf("Expected nil with nil loader, got %v", result)
	}
}

// TestSkillManager_Create_MissingName tests Create with missing name.
func TestSkillManager_Create_MissingName(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	req := &CreateSkillRequest{
		Description: "test",
		Body:        "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

// TestSkillManager_Create_MissingDescription tests Create with missing description.
func TestSkillManager_Create_MissingDescription(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	req := &CreateSkillRequest{
		Name: "test-skill",
		Body: "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing description")
	}
}

// TestSkillManager_Create_MissingBody tests Create with missing body.
func TestSkillManager_Create_MissingBody(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	req := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing body")
	}
}

// TestSkillManager_Create_NoBaseDirs tests Create with no base directories.
func TestSkillManager_Create_NoBaseDirs(t *testing.T) {
	mgr := NewSkillManager(nil, nil)

	req := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "test",
		Body:        "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for no base directories")
	}
}

// TestSkillManager_Create_Success tests successful skill creation.
func TestSkillManager_Create_Success(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillDir, 0755)

	ldr := loader.NewLoader([]string{skillDir})
	mgr := NewSkillManager(ldr, []string{skillDir})

	req := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "This is the skill body.",
	}

	created, err := mgr.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", created.Name)
	}

	// Verify file was created
	expectedPath := filepath.Join(skillDir, "test-skill", "SKILL.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected SKILL.md file to be created")
	}
}

// TestSkillManager_Create_Duplicate tests creating duplicate skill.
func TestSkillManager_Create_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillDir, 0755)

	ldr := loader.NewLoader([]string{skillDir})
	mgr := NewSkillManager(ldr, []string{skillDir})

	req := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "This is the skill body.",
	}

	// Create first skill
	mgr.Create(context.Background(), req)

	// Try to create duplicate
	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for duplicate skill")
	}
}

// TestSkillManager_Update tests Update method.
func TestSkillManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillDir, 0755)

	ldr := loader.NewLoader([]string{skillDir})
	mgr := NewSkillManager(ldr, []string{skillDir})

	// Create a skill first
	createReq := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "Original description",
		Body:        "Original body",
	}
	mgr.Create(context.Background(), createReq)

	// Update the skill
	newDesc := "Updated description"
	updateReq := &UpdateSkillRequest{
		Description: &newDesc,
	}

	updated, err := mgr.Update(context.Background(), "test-skill", updateReq)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}
}

// TestSkillManager_Update_NotFound tests updating non-existent skill.
func TestSkillManager_Update_NotFound(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	desc := "test"
	updateReq := &UpdateSkillRequest{
		Description: &desc,
	}

	_, err := mgr.Update(context.Background(), "nonexistent", updateReq)
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

// TestSkillManager_Update_MissingName tests Update with missing name.
func TestSkillManager_Update_MissingName(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	desc := "test"
	updateReq := &UpdateSkillRequest{
		Description: &desc,
	}

	_, err := mgr.Update(context.Background(), "", updateReq)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestSkillManager_Delete tests Delete method.
func TestSkillManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillDir, 0755)

	ldr := loader.NewLoader([]string{skillDir})
	mgr := NewSkillManager(ldr, []string{skillDir})

	// Create a skill first
	createReq := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "Skill body",
	}
	mgr.Create(context.Background(), createReq)

	// Delete the skill
	err := mgr.Delete(context.Background(), "test-skill")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify skill directory is removed
	expectedPath := filepath.Join(skillDir, "test-skill")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("Expected skill directory to be removed")
	}
}

// TestSkillManager_Delete_NotFound tests deleting non-existent skill.
func TestSkillManager_Delete_NotFound(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	err := mgr.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

// TestSkillManager_Delete_MissingName tests Delete with missing name.
func TestSkillManager_Delete_MissingName(t *testing.T) {
	mgr := NewSkillManager(nil, []string{"/tmp"})

	err := mgr.Delete(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestSkillManager_Reload tests Reload method.
func TestSkillManager_Reload(t *testing.T) {
	mgr := NewSkillManager(nil, nil)

	// With nil loader, should return nil
	err := mgr.Reload(context.Background())
	if err != nil {
		t.Errorf("Expected nil error with nil loader, got %v", err)
	}
}

// TestSkillManager_Get_WithNilLoader tests Get with nil loader.
func TestSkillManager_Get_WithNilLoader(t *testing.T) {
	mgr := NewSkillManager(nil, nil)

	result := mgr.Get("test")
	if result != nil {
		t.Error("Expected nil with nil loader")
	}
}

// TestSkillManager_Get_Existing tests Get with existing skill.
func TestSkillManager_Get_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillDir, 0755)

	ldr := loader.NewLoader([]string{skillDir})
	mgr := NewSkillManager(ldr, []string{skillDir})

	// Create a skill
	createReq := &CreateSkillRequest{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "Body",
	}
	mgr.Create(context.Background(), createReq)

	// Get the skill
	found := mgr.Get("test-skill")
	if found == nil {
		t.Fatal("Expected to find the skill")
	}
	if found.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", found.Name)
	}
}
