package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIsolator(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, err := NewIsolator(tmpDir)
	if err != nil {
		t.Fatalf("NewIsolator() error = %v", err)
	}

	if isolator.BaseDir() != tmpDir {
		t.Errorf("BaseDir() = %s, want %s", isolator.BaseDir(), tmpDir)
	}

	expectedDocsDir := filepath.Join(tmpDir, "docs")
	if isolator.SharedDocsDir() != expectedDocsDir {
		t.Errorf("SharedDocsDir() = %s, want %s", isolator.SharedDocsDir(), expectedDocsDir)
	}
}

func TestNewIsolator_EmptyBaseDir(t *testing.T) {
	_, err := NewIsolator("")
	if err == nil {
		t.Error("NewIsolator(\"\") error = nil, want error")
	}
}

func TestIsolator_Create(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	ws, err := isolator.Create("agent-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if ws.ID != "agent-1" {
		t.Errorf("Workspace.ID = %s, want agent-1", ws.ID)
	}

	expectedDir := filepath.Join(tmpDir, "agent-1")
	if ws.Dir != expectedDir {
		t.Errorf("Workspace.Dir = %s, want %s", ws.Dir, expectedDir)
	}

	// Verify directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("Workspace directory was not created")
	}
}

func TestIsolator_Create_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	ws1, err := isolator.Create("agent-1")
	if err != nil {
		t.Fatalf("Create() first call error = %v", err)
	}

	ws2, err := isolator.Create("agent-1")
	if err != nil {
		t.Fatalf("Create() second call error = %v", err)
	}

	if ws1.Dir != ws2.Dir {
		t.Error("Create() returned different workspaces for same agent ID")
	}
}

func TestIsolator_Create_EmptyAgentID(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	_, err := isolator.Create("")
	if err == nil {
		t.Error("Create(\"\") error = nil, want error")
	}
}

func TestIsolator_Get(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	// Get non-existent
	ws := isolator.Get("agent-1")
	if ws != nil {
		t.Error("Get(non-existent) = non-nil, want nil")
	}

	// Create and get
	_, _ = isolator.Create("agent-1")
	ws = isolator.Get("agent-1")
	if ws == nil {
		t.Error("Get(existing) = nil, want workspace")
	}
}

func TestIsolator_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	_, _ = isolator.Create("agent-1")
	agentDir := filepath.Join(tmpDir, "agent-1")

	// Verify exists
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		t.Fatal("Workspace directory should exist before cleanup")
	}

	err := isolator.Cleanup("agent-1")
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Verify removed
	if _, err := os.Stat(agentDir); !os.IsNotExist(err) {
		t.Error("Workspace directory should be removed after cleanup")
	}

	// Get should return nil after cleanup
	ws := isolator.Get("agent-1")
	if ws != nil {
		t.Error("Get() after Cleanup() = non-nil, want nil")
	}
}

func TestIsolator_Cleanup_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	_ = isolator.Cleanup("non-existent-agent")
	_ = isolator.Cleanup("non-existent-agent")

	// Should not error
}

func TestIsolator_CleanupAll(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	_, _ = isolator.Create("agent-1")
	_, _ = isolator.Create("agent-2")

	docsDir := isolator.SharedDocsDir()

	// Create something in docs
	_ = os.MkdirAll(docsDir, 0755)
	testFile := filepath.Join(docsDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("test"), 0644)

	err := isolator.CleanupAll()
	if err != nil {
		t.Fatalf("CleanupAll() error = %v", err)
	}

	// Verify all workspaces removed
	if len(isolator.ListWorkspaces()) != 0 {
		t.Error("ListWorkspaces() after CleanupAll() should be empty")
	}
}

func TestIsolator_ListWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()

	isolator, _ := NewIsolator(tmpDir)

	// Empty
	dirs := isolator.ListWorkspaces()
	if len(dirs) != 0 {
		t.Errorf("ListWorkspaces() empty = %v, want []", dirs)
	}

	// Create some
	_, _ = isolator.Create("agent-1")
	_, _ = isolator.Create("agent-2")
	_, _ = isolator.Create("agent-3")

	dirs = isolator.ListWorkspaces()
	if len(dirs) != 3 {
		t.Errorf("ListWorkspaces() len = %d, want 3", len(dirs))
	}
}
