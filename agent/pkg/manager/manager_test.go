package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oneliang/aura/agent/pkg/loader"
)

// TestCreateAgentRequest tests CreateAgentRequest struct.
func TestCreateAgentRequest(t *testing.T) {
	req := CreateAgentRequest{
		Name:         "test-agent",
		Description:  "A test agent",
		Body:         "Agent body content",
		LLMModel:     "qwen3:8b",
		Temperature:  0.7,
		DisableTools: []string{"bash"},
	}

	if req.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", req.Name)
	}
	if req.Description != "A test agent" {
		t.Errorf("Expected description 'A test agent', got '%s'", req.Description)
	}
	if req.Body != "Agent body content" {
		t.Errorf("Expected body 'Agent body content', got '%s'", req.Body)
	}
}

// TestUpdateAgentRequest tests UpdateAgentRequest struct.
func TestUpdateAgentRequest(t *testing.T) {
	desc := "updated description"
	body := "updated body"
	req := UpdateAgentRequest{
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

// TestNewAgentManager tests NewAgentManager function.
func TestNewAgentManager(t *testing.T) {
	baseDirs := []string{"/tmp/agents"}
	mgr := NewAgentManager(nil, baseDirs)

	if mgr == nil {
		t.Fatal("NewAgentManager() returned nil")
	}
	if len(mgr.baseDirs) != 1 {
		t.Errorf("Expected 1 base dir, got %d", len(mgr.baseDirs))
	}
}

// TestAgentManager_List tests List method.
func TestAgentManager_List(t *testing.T) {
	mgr := NewAgentManager(nil, nil)

	// With nil loader
	result := mgr.List()
	if result != nil {
		t.Errorf("Expected nil list with nil loader, got %v", result)
	}
}

// TestAgentManager_Get tests Get method.
func TestAgentManager_Get(t *testing.T) {
	mgr := NewAgentManager(nil, nil)

	// With nil loader
	result := mgr.Get("nonexistent")
	if result != nil {
		t.Errorf("Expected nil with nil loader, got %v", result)
	}
}

// TestAgentManager_Create_MissingName tests Create with missing name.
func TestAgentManager_Create_MissingName(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	req := &CreateAgentRequest{
		Description: "test",
		Body:        "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

// TestAgentManager_Create_MissingDescription tests Create with missing description.
func TestAgentManager_Create_MissingDescription(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	req := &CreateAgentRequest{
		Name: "test-agent",
		Body: "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing description")
	}
}

// TestAgentManager_Create_MissingBody tests Create with missing body.
func TestAgentManager_Create_MissingBody(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	req := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing body")
	}
}

// TestAgentManager_Create_NoBaseDirs tests Create with no base directories.
func TestAgentManager_Create_NoBaseDirs(t *testing.T) {
	mgr := NewAgentManager(nil, nil)

	req := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "test",
		Body:        "test",
	}

	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for no base directories")
	}
}

// TestAgentManager_Create_Success tests successful agent creation.
func TestAgentManager_Create_Success(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentDir, 0755)

	ldr := loader.NewLoader([]string{agentDir})
	mgr := NewAgentManager(ldr, []string{agentDir})

	req := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "A test agent",
		Body:        "This is the agent body.",
	}

	created, err := mgr.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", created.Name)
	}

	// Verify file was created
	expectedPath := filepath.Join(agentDir, "test-agent", "AGENT.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected AGENT.md file to be created")
	}
}

// TestAgentManager_Create_Duplicate tests creating duplicate agent.
func TestAgentManager_Create_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentDir, 0755)

	ldr := loader.NewLoader([]string{agentDir})
	mgr := NewAgentManager(ldr, []string{agentDir})

	req := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "A test agent",
		Body:        "This is the agent body.",
	}

	// Create first agent
	mgr.Create(context.Background(), req)

	// Try to create duplicate
	_, err := mgr.Create(context.Background(), req)
	if err == nil {
		t.Error("Expected error for duplicate agent")
	}
}

// TestAgentManager_Update tests Update method.
func TestAgentManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentDir, 0755)

	ldr := loader.NewLoader([]string{agentDir})
	mgr := NewAgentManager(ldr, []string{agentDir})

	// Create an agent first
	createReq := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "Original description",
		Body:        "Original body",
	}
	mgr.Create(context.Background(), createReq)

	// Update the agent
	newDesc := "Updated description"
	updateReq := &UpdateAgentRequest{
		Description: &newDesc,
	}

	updated, err := mgr.Update(context.Background(), "test-agent", updateReq)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}
}

// TestAgentManager_Update_NotFound tests updating non-existent agent.
func TestAgentManager_Update_NotFound(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	desc := "test"
	updateReq := &UpdateAgentRequest{
		Description: &desc,
	}

	_, err := mgr.Update(context.Background(), "nonexistent", updateReq)
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
}

// TestAgentManager_Update_MissingName tests Update with missing name.
func TestAgentManager_Update_MissingName(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	desc := "test"
	updateReq := &UpdateAgentRequest{
		Description: &desc,
	}

	_, err := mgr.Update(context.Background(), "", updateReq)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestAgentManager_Delete tests Delete method.
func TestAgentManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentDir, 0755)

	ldr := loader.NewLoader([]string{agentDir})
	mgr := NewAgentManager(ldr, []string{agentDir})

	// Create an agent first
	createReq := &CreateAgentRequest{
		Name:        "test-agent",
		Description: "A test agent",
		Body:        "Agent body",
	}
	mgr.Create(context.Background(), createReq)

	// Delete the agent
	err := mgr.Delete(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify agent directory is removed
	expectedPath := filepath.Join(agentDir, "test-agent")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("Expected agent directory to be removed")
	}
}

// TestAgentManager_Delete_NotFound tests deleting non-existent agent.
func TestAgentManager_Delete_NotFound(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	err := mgr.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
}

// TestAgentManager_Delete_MissingName tests Delete with missing name.
func TestAgentManager_Delete_MissingName(t *testing.T) {
	mgr := NewAgentManager(nil, []string{"/tmp"})

	err := mgr.Delete(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestAgentManager_Reload tests Reload method.
func TestAgentManager_Reload(t *testing.T) {
	mgr := NewAgentManager(nil, nil)

	// With nil loader, should return nil
	err := mgr.Reload(context.Background())
	if err != nil {
		t.Errorf("Expected nil error with nil loader, got %v", err)
	}
}

// TestAgentManager_Get_WithNilLoader tests Get with nil loader.
func TestAgentManager_Get_WithNilLoader(t *testing.T) {
	mgr := NewAgentManager(nil, nil)

	result := mgr.Get("test")
	if result != nil {
		t.Error("Expected nil with nil loader")
	}
}

// TestAgentManager_CreateAgentContent tests that Create produces correct file content.
func TestAgentManager_CreateAgentContent(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentDir, 0755)

	ldr := loader.NewLoader([]string{agentDir})

	tests := []struct {
		name     string
		req      *CreateAgentRequest
		contains []string
	}{
		{
			name: "basic agent",
			req: &CreateAgentRequest{
				Name:        "test-agent",
				Description: "A test agent",
				Body:        "This is the body.",
			},
			contains: []string{"name: test-agent", "description: A test agent", "This is the body."},
		},
		{
			name: "agent with LLM model",
			req: &CreateAgentRequest{
				Name:        "test-agent-llm",
				Description: "A test agent",
				Body:        "Body.",
				LLMModel:    "qwen3:8b",
			},
			contains: []string{"llm_model: qwen3:8b"},
		},
		{
			name: "agent with temperature",
			req: &CreateAgentRequest{
				Name:        "test-agent-temp",
				Description: "A test agent",
				Body:        "Body.",
				Temperature: 0.7,
			},
			contains: []string{"temperature: 0.70"},
		},
		{
			name: "agent with disabled tools",
			req: &CreateAgentRequest{
				Name:         "test-agent-tools",
				Description:  "A test agent",
				Body:         "Body.",
				DisableTools: []string{"bash", "file_write"},
			},
			contains: []string{"disable_tools:", "- bash", "- file_write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewAgentManager(ldr, []string{agentDir})
			_, err := mgr.Create(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			agentPath := filepath.Join(agentDir, tt.req.Name, "AGENT.md")
			content, err := os.ReadFile(agentPath)
			if err != nil {
				t.Fatalf("Failed to read agent file: %v", err)
			}
			for _, s := range tt.contains {
				if !strings.Contains(string(content), s) {
					t.Errorf("Content should contain '%s', got: %s", s, string(content))
				}
			}
		})
	}
}
