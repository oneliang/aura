package factory

import (
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestPermissionManagerFactory_Create(t *testing.T) {
	factory := NewPermissionManagerFactory()

	if factory == nil {
		t.Fatal("NewPermissionManagerFactory() returned nil")
	}
}

func TestPermissionManagerFactory_Create_AllowDefault(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "allow",
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_AskDefault(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_DenyDefault(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "deny",
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_WithToolPermissions(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
		Tools: map[string]string{
			"file_read":  "allow",
			"file_write": "ask",
			"bash":       "deny",
		},
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_WithShellRestrictions(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
		ShellRestrictions: config.CommandRestrictions{
			DeniedCommands: []string{"rm -rf", "mkfs", "dd"},
		},
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_WithSSHRestrictions(t *testing.T) {
	cfg := &config.PermissionsConfig{
		DefaultLevel: "ask",
		SSHRestrictions: config.SSHRestrictions{
			AllowedHosts:    []string{"trusted.example.com"},
			DeniedHosts:     []string{"untrusted.example.com"},
			AllowedCommands: []string{"ls", "cat", "grep"},
			DeniedCommands:  []string{"rm", "sudo"},
		},
	}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}

func TestPermissionManagerFactory_Create_NilConfig(t *testing.T) {
	factory := NewPermissionManagerFactory()

	// This should panic or return error since config is nil
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior if it panics
			t.Logf("Create() panicked as expected: %v", r)
		}
	}()

	manager, err := factory.Create(nil)

	// If we get here without panic, check the result
	if err == nil && manager == nil {
		t.Error("Create() with nil config should return error or panic")
	}
}

func TestPermissionManagerFactory_Create_EmptyConfig(t *testing.T) {
	cfg := &config.PermissionsConfig{}

	factory := NewPermissionManagerFactory()
	manager, err := factory.Create(cfg)

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if manager == nil {
		t.Fatal("Create() returned nil manager")
	}
}
