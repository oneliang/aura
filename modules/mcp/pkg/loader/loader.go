// Package loader provides MCP configuration loading from mcp.json.
package loader

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/oneliang/aura/mcp/pkg/config"
	"github.com/oneliang/aura/mcp/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// Loader loads MCP server configurations from mcp.json.
type Loader struct {
	configPath string
	cfg        *config.Config
}

// NewLoader creates a new MCP config loader.
// If configPath is empty, uses ~/.aura/mcp.json.
func NewLoader(configPath string) *Loader {
	if configPath == "" {
		configPath = ffp.AuraHomePathOrDefault(constants.DefaultConfigFile)
	}
	return &Loader{
		configPath: configPath,
	}
}

// Load loads MCP server configurations from the config file.
// Returns an empty Config (no servers) if the file doesn't exist.
func (l *Loader) Load() (*config.Config, error) {
	cfg := &config.Config{
		MCPServers: make(map[string]config.ServerConfig),
	}

	data, err := os.ReadFile(l.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			l.cfg = cfg
			return cfg, nil
		}
		return nil, fmt.Errorf("read MCP config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse MCP config: %w", err)
	}

	l.cfg = cfg
	return cfg, nil
}

// GetConfig returns the loaded configuration.
func (l *Loader) GetConfig() *config.Config {
	return l.cfg
}

// GetServers returns enabled server configurations.
func (l *Loader) GetServers() map[string]config.ServerConfig {
	if l.cfg == nil {
		return nil
	}
	result := make(map[string]config.ServerConfig)
	for name, srv := range l.cfg.MCPServers {
		if !srv.Disabled {
			result[name] = srv
		}
	}
	return result
}

// GetServer returns a specific server by name.
func (l *Loader) GetServer(name string) (config.ServerConfig, bool) {
	if l.cfg == nil {
		return config.ServerConfig{}, false
	}
	srv, ok := l.cfg.MCPServers[name]
	if !ok || srv.Disabled {
		return config.ServerConfig{}, false
	}
	return srv, true
}

// Save writes the configuration back to the config file.
func (l *Loader) Save(cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal MCP config: %w", err)
	}

	if err := os.WriteFile(l.configPath, data, 0600); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}

	l.cfg = cfg
	return nil
}
