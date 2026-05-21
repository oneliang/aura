package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/mcp/pkg/config"
)

func TestNewLoader_DefaultPath(t *testing.T) {
	ldr := NewLoader("")
	if ldr.configPath == "" {
		t.Error("NewLoader(\"\") should use default config path")
	}
}

func TestLoader_Load_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ldr := NewLoader(filepath.Join(tmpDir, "nonexistent.json"))

	cfg, err := ldr.Load()
	if err != nil {
		t.Fatalf("Load() for missing file should return nil error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() should return empty config, got nil")
	}
	if len(cfg.MCPServers) != 0 {
		t.Errorf("Load() should return empty servers, got %d", len(cfg.MCPServers))
	}
}

func TestLoader_SaveLoad_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "mcp.json")
	ldr := NewLoader(configFile)

	// Save a config
	inputCfg := &config.Config{
		MCPServers: map[string]config.ServerConfig{
			"test-server": {
				Command: "npx",
				Args:    []string{"-y", "@some-mcp-server"},
			},
		},
	}
	if err := ldr.Save(inputCfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load it back
	loadedCfg, err := ldr.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loadedCfg.MCPServers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(loadedCfg.MCPServers))
	}
	srv, ok := loadedCfg.MCPServers["test-server"]
	if !ok {
		t.Fatal("expected test-server, not found")
	}
	if srv.Command != "npx" {
		t.Errorf("expected command npx, got %s", srv.Command)
	}
	if len(srv.Args) != 2 || srv.Args[0] != "-y" {
		t.Errorf("unexpected args: %v", srv.Args)
	}
}

func TestLoader_GetServers_FiltersDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "mcp.json")
	ldr := NewLoader(configFile)

	cfg := &config.Config{
		MCPServers: map[string]config.ServerConfig{
			"active":   {Command: "npx"},
			"disabled": {Command: "node", Disabled: true},
		},
	}
	if err := ldr.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	_, err := ldr.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	servers := ldr.GetServers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 enabled server, got %d", len(servers))
	}
	if _, ok := servers["active"]; !ok {
		t.Error("expected active server, not found")
	}
	if _, ok := servers["disabled"]; ok {
		t.Error("disabled server should not be returned")
	}
}

func TestLoader_GetServers_NilConfig(t *testing.T) {
	ldr := &Loader{}
	servers := ldr.GetServers()
	if servers != nil {
		t.Error("GetServers() on nil config should return nil")
	}
}

func TestLoader_GetServer(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "mcp.json")
	ldr := NewLoader(configFile)

	cfg := &config.Config{
		MCPServers: map[string]config.ServerConfig{
			"my-server": {Command: "npx"},
		},
	}
	if err := ldr.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	_, err := ldr.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	srv, ok := ldr.GetServer("my-server")
	if !ok {
		t.Fatal("expected my-server, not found")
	}
	if srv.Command != "npx" {
		t.Errorf("expected command npx, got %s", srv.Command)
	}

	_, ok = ldr.GetServer("nonexistent")
	if ok {
		t.Error("expected nonexistent server to not be found")
	}
}

func TestLoader_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "mcp.json")

	// Write invalid JSON
	if err := os.WriteFile(configFile, []byte(`{invalid json}`), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	ldr := NewLoader(configFile)
	_, err := ldr.Load()
	if err == nil {
		t.Fatal("Load() should error on invalid JSON")
	}
}
