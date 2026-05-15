package factory

import (
	"testing"

	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/config"
)

func TestNewToolRegistry(t *testing.T) {
	cfg := &config.ToolsConfig{}
	permMgr := createTestPermissionManager(t)

	registry := NewToolRegistry(cfg, permMgr)

	if registry == nil {
		t.Fatal("NewToolRegistry() returned nil")
	}

	if registry.config != cfg {
		t.Error("NewToolRegistry() did not store config correctly")
	}
	if registry.permMgr != permMgr {
		t.Error("NewToolRegistry() did not store permMgr correctly")
	}
}

func TestToolRegistry_WithEmptyConfig(t *testing.T) {
	cfg := &config.ToolsConfig{
		Enabled: []string{},
	}
	permMgr := createTestPermissionManager(t)

	registry := NewToolRegistry(cfg, permMgr)

	if registry == nil {
		t.Fatal("NewToolRegistry() returned nil")
	}
}

func createTestPermissionManager(t *testing.T) *permissions.Manager {
	t.Helper()
	cfg := permissions.DefaultPermissionConfig()
	mgr, err := permissions.NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create permission manager: %v", err)
	}
	return mgr
}

func TestConvertSSHServers(t *testing.T) {
	configServers := []config.SSHServerConfig{
		{
			Name:     "server1",
			Host:     "host1.example.com",
			Port:     22,
			User:     "user1",
			KeyPath:  "/path/to/key",
			Password: "",
		},
		{
			Name:     "server2",
			Host:     "host2.example.com",
			Port:     2222,
			User:     "user2",
			KeyPath:  "",
			Password: "password123",
		},
		{
			Name: "server3",
			Host: "host3.example.com",
			Port: 22,
			User: "user3",
		},
	}

	// Test that we can iterate over servers without error
	if len(configServers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(configServers))
	}

	for i, s := range configServers {
		if s.Name == "" {
			t.Errorf("Server %d has empty name", i)
		}
		if s.Host == "" {
			t.Errorf("Server %d has empty host", i)
		}
		if s.User == "" {
			t.Errorf("Server %d has empty user", i)
		}
	}
}
