package manager

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/aura/mcp/pkg/config"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(nil)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
	if mgr.clients == nil {
		t.Error("clients map should be initialized")
	}
	if mgr.tools == nil {
		t.Error("tools map should be initialized")
	}
}

func TestManager_GetTools_Empty(t *testing.T) {
	mgr := NewManager(nil)
	tools := mgr.GetTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestManager_StopServer_NotRunning(t *testing.T) {
	mgr := NewManager(nil)
	err := mgr.StopServer(context.Background(), "nonexistent")
	if err != nil {
		t.Errorf("StopServer on non-running server should return nil, got %v", err)
	}
}

func TestManager_StopAll_Empty(t *testing.T) {
	mgr := NewManager(nil)
	err := mgr.StopAll(context.Background())
	if err != nil {
		t.Errorf("StopAll with no servers should return nil, got %v", err)
	}
}

func TestManager_ListServers_NoConfig(t *testing.T) {
	mgr := NewManager(nil)
	infos := mgr.ListServers()
	if infos != nil {
		t.Errorf("ListServers with no loader config should return nil, got %d items", len(infos))
	}
}

func TestManager_ServerInfoForName(t *testing.T) {
	mgr := NewManager(nil)
	info := mgr.ServerInfoForName("test")
	if info.Name != "test" {
		t.Errorf("expected name 'test', got %s", info.Name)
	}
	if info.Status != "stopped" {
		t.Errorf("expected status 'stopped', got %s", info.Status)
	}
}

func TestManager_ServerInfoForNameWithTimeout_NoClient(t *testing.T) {
	mgr := NewManager(nil)
	info := mgr.ServerInfoForNameWithTimeout("test", 1*time.Second)
	if info.Name != "test" {
		t.Errorf("expected name 'test', got %s", info.Name)
	}
}

func TestManager_RemoveToolsByServer(t *testing.T) {
	mgr := NewManager(nil)
	// This is an internal method, but we can test via StartServer/StopServer path
	// Just verify no panic when called on empty maps
	mgr.removeToolsByServer("nonexistent")
}

func TestManager_AddServer_NilLoader(t *testing.T) {
	mgr := NewManager(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil loader causes panic in GetConfig
			t.Logf("AddServer with nil loader panicked as expected: %v", r)
		}
	}()
	_, err := mgr.AddServer(context.Background(), "test", config.ServerConfig{Command: "echo"})
	if err == nil {
		t.Error("AddServer with nil loader should error")
	}
}

func TestManager_Reload_NilLoader(t *testing.T) {
	mgr := NewManager(nil)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Reload with nil loader panicked as expected: %v", r)
		}
	}()
	_, err := mgr.Reload(context.Background())
	if err == nil {
		t.Error("Reload with nil loader should error")
	}
}
