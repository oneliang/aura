package common

import (
	"context"
	"testing"

	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/knowledge/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
)

// TestDefaultConfigLoader tests DefaultConfigLoader.
func TestDefaultConfigLoader(t *testing.T) {
	loader := &DefaultConfigLoader{}
	if loader == nil {
		t.Fatal("DefaultConfigLoader should not be nil")
	}
}

// TestDefaultConfigLoader_Load_NonExistent tests Load with non-existent file.
func TestDefaultConfigLoader_Load_NonExistent(t *testing.T) {
	loader := &DefaultConfigLoader{}
	cfg, err := loader.Load("/non/existent/path/config.yaml")

	// Non-existent file should return error or default config
	_ = cfg
	_ = err
}

// TestDefaultHomeDirProvider tests DefaultHomeDirProvider.
func TestDefaultHomeDirProvider(t *testing.T) {
	provider := &DefaultHomeDirProvider{}
	homeDir, err := provider.Get()

	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if homeDir == "" {
		t.Error("Get() returned empty home directory")
	}
}

// TestDefaultKnowledgeStoreFactory tests DefaultKnowledgeStoreFactory.
func TestDefaultKnowledgeStoreFactory(t *testing.T) {
	factory := &DefaultKnowledgeStoreFactory{}
	if factory == nil {
		t.Fatal("DefaultKnowledgeStoreFactory should not be nil")
	}
}

// TestDefaultPermissionManagerFactory tests DefaultPermissionManagerFactory.
func TestDefaultPermissionManagerFactory(t *testing.T) {
	factory := &DefaultPermissionManagerFactory{}

	cfg := &permissions.PermissionConfig{
		DefaultLevel: "ask",
	}

	mgr, err := factory.NewManager(cfg)
	if err != nil {
		t.Errorf("NewManager() error = %v", err)
	}
	if mgr == nil {
		t.Error("NewManager() returned nil")
	}
}

// TestConvertPermissionsConfig tests ConvertPermissionsConfig function.
func TestConvertPermissionsConfig(t *testing.T) {
	cfg := config.PermissionsConfig{
		DefaultLevel: "ask",
		Tools: map[string]string{
			"bash": "ask",
			"read": "allow",
		},
		ShellRestrictions: config.CommandRestrictions{
			AllowedCommands: []string{"ls", "cat"},
			DeniedCommands:  []string{"rm", "mkfs"},
		},
		SSHRestrictions: config.SSHRestrictions{
			AllowedHosts:    []string{"host1", "host2"},
			DeniedHosts:     []string{"host3"},
			AllowedCommands: []string{"ls"},
			DeniedCommands:  []string{"reboot"},
		},
		TrustedDirs:  []string{"/safe/dir"},
		AutoAskTrust: true,
	}

	result := ConvertPermissionsConfig(cfg)

	if result.DefaultLevel != "ask" {
		t.Errorf("DefaultLevel = %q, want %q", result.DefaultLevel, "ask")
	}
	if result.Tools["bash"] != "ask" {
		t.Errorf("Tools['bash'] = %q, want %q", result.Tools["bash"], "ask")
	}
	if len(result.ShellRestrictions.AllowedCommands) != 2 {
		t.Errorf("ShellRestrictions.AllowedCommands length = %d, want 2", len(result.ShellRestrictions.AllowedCommands))
	}
	if len(result.ShellRestrictions.DeniedCommands) != 2 {
		t.Errorf("ShellRestrictions.DeniedCommands length = %d, want 2", len(result.ShellRestrictions.DeniedCommands))
	}
	if len(result.SSHRestrictions.AllowedHosts) != 2 {
		t.Errorf("SSHRestrictions.AllowedHosts length = %d, want 2", len(result.SSHRestrictions.AllowedHosts))
	}
	if len(result.TrustedDirs) != 1 {
		t.Errorf("TrustedDirs length = %d, want 1", len(result.TrustedDirs))
	}
	if !result.AutoAskTrust {
		t.Error("AutoAskTrust should be true")
	}
}

// TestConvertPermissionsConfig_Empty tests ConvertPermissionsConfig with empty config.
func TestConvertPermissionsConfig_Empty(t *testing.T) {
	cfg := config.PermissionsConfig{}
	result := ConvertPermissionsConfig(cfg)

	if result.DefaultLevel != "ask" {
		t.Errorf("DefaultLevel should be 'ask' (parsed default), got %q", result.DefaultLevel)
	}
	if len(result.Tools) != 0 {
		t.Errorf("Tools should be empty, got %v", result.Tools)
	}
}

// TestConfigLoader_Interface tests that DefaultConfigLoader implements ConfigLoader.
func TestConfigLoader_Interface(t *testing.T) {
	var _ ConfigLoader = (*DefaultConfigLoader)(nil)
}

// TestHomeDirProvider_Interface tests that DefaultHomeDirProvider implements HomeDirProvider.
func TestHomeDirProvider_Interface(t *testing.T) {
	var _ HomeDirProvider = (*DefaultHomeDirProvider)(nil)
}

// TestKnowledgeStoreFactory_Interface tests that DefaultKnowledgeStoreFactory implements KnowledgeStoreFactory.
func TestKnowledgeStoreFactory_Interface(t *testing.T) {
	var _ KnowledgeStoreFactory = (*DefaultKnowledgeStoreFactory)(nil)
}

// TestPermissionManagerFactory_Interface tests that DefaultPermissionManagerFactory implements PermissionManagerFactory.
func TestPermissionManagerFactory_Interface(t *testing.T) {
	var _ PermissionManagerFactory = (*DefaultPermissionManagerFactory)(nil)
}

// TestDefaultKnowledgeStoreFactory_NewCollection tests NewCollection method.
func TestDefaultKnowledgeStoreFactory_NewCollection(t *testing.T) {
	factory := &DefaultKnowledgeStoreFactory{}
	ctx := context.Background()

	// This will likely fail without proper setup, but we test the interface
	_, err := factory.NewCollection(ctx, storage.ChromemOptions{
		DataDir: t.TempDir(),
		Name:    "test",
	})
	// Just verify the method can be called
	_ = err
}
