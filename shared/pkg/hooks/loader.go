package hooks

import (
	"fmt"
	"os"
	stdpath "path/filepath"

	"github.com/oneliang/aura/shared/pkg/constants"
	"gopkg.in/yaml.v3"
)

// DefaultHooksConfigFile is the default filename for hooks configuration.
const DefaultHooksConfigFile = "hooks.yaml"

// LoadHooksConfig loads hooks configuration from the given path.
// If path is empty, uses ~/.aura/hooks.yaml.
// Returns nil config (Enabled=false) if file does not exist.
func LoadHooksConfig(path string) (*HooksConfig, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = stdpath.Join(homeDir, constants.DefaultHomeDir, DefaultHooksConfigFile)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &HooksConfig{Enabled: false}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read hooks config: %w", err)
	}

	var cfg HooksConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse hooks config: %w", err)
	}

	return &cfg, nil
}

// SaveHooksConfig saves hooks configuration to the given path.
// If path is empty, uses ~/.aura/hooks.yaml.
func SaveHooksConfig(cfg *HooksConfig, path string) error {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = stdpath.Join(homeDir, constants.DefaultHomeDir, DefaultHooksConfigFile)
	}

	// Ensure directory exists
	dir := stdpath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal hooks config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write hooks config: %w", err)
	}

	return nil
}
