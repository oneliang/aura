// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"gopkg.in/yaml.v3"
)

// ConfigExecutor handles config commands.
type ConfigExecutor struct {
	configPath string
}

// NewConfigExecutor creates a new config executor.
func NewConfigExecutor(configPath string) *ConfigExecutor {
	return &ConfigExecutor{
		configPath: configPath,
	}
}

// ExecuteCommand executes a config command.
// Commands: show, path, get
func (e *ConfigExecutor) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	switch cmd {
	case "show":
		return e.showConfig()
	case "path":
		return e.showConfigPath(), nil
	case "get":
		return e.getConfigValue(params)
	default:
		return "", fmt.Errorf("unknown config command: %s", cmd)
	}
}

// showConfig shows the current configuration.
func (e *ConfigExecutor) showConfig() (string, error) {
	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	// Pretty print YAML
	return string(data), nil
}

// showConfigPath shows the config file path.
func (e *ConfigExecutor) showConfigPath() string {
	if e.configPath == "" {
		e.configPath = ffp.MustAuraHomePath(constants.DefaultConfigFile)
	}
	return fmt.Sprintf("Config file: %s", e.configPath)
}

// getConfigValue gets a specific configuration value by key.
// Supports dot notation: "llm.model", "agent.planning_mode", etc.
func (e *ConfigExecutor) getConfigValue(params map[string]any) (string, error) {
	key, ok := params["key"].(string)
	if !ok || key == "" {
		return "", fmt.Errorf("missing or invalid 'key' parameter")
	}

	// Load config
	config, err := LoadConfig(e.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Navigate dot notation
	value, err := getNestedValue(config, key)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s = %v", key, value), nil
}

// getNestedValue navigates a nested map using dot notation.
func getNestedValue(m map[string]any, key string) (any, error) {
	parts := strings.Split(key, ".")
	current := any(m)

	for i, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, exists := v[part]
			if !exists {
				return nil, fmt.Errorf("key '%s' not found", strings.Join(parts[:i+1], "."))
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot navigate into non-map at '%s'", strings.Join(parts[:i], "."))
		}
	}

	return current, nil
}

// LoadConfig loads configuration from file.
func LoadConfig(path string) (map[string]any, error) {
	if path == "" {
		path = ffp.MustAuraHomePath(constants.DefaultConfigFile)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves configuration to file.
func SaveConfig(path string, config map[string]any) error {
	if path == "" {
		path = ffp.MustAuraHomePath(constants.DefaultConfigFile)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := ffp.EnsureDir(dir); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
