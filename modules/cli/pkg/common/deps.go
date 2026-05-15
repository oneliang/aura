// Package common provides shared dependencies for CLI packages.
package common

import (
	"context"
	"os"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/knowledge/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
)

// ConfigLoader defines the interface for loading configuration.
type ConfigLoader interface {
	Load(path string) (*config.Config, error)
}

// HomeDirProvider defines the interface for getting the user's home directory.
type HomeDirProvider interface {
	Get() (string, error)
}

// KnowledgeStoreFactory defines the interface for creating knowledge collections.
type KnowledgeStoreFactory interface {
	NewCollection(ctx context.Context, opts storage.ChromemOptions) (*storage.ChromemCollection, error)
}

// PermissionManagerFactory defines the interface for creating permission managers.
type PermissionManagerFactory interface {
	NewManager(cfg *sdk.PermissionConfig) (*sdk.PermissionManager, error)
}

// DefaultConfigLoader is the default implementation of ConfigLoader.
type DefaultConfigLoader struct{}

// Load implements ConfigLoader.
func (d *DefaultConfigLoader) Load(path string) (*config.Config, error) {
	return config.Load(path)
}

// DefaultHomeDirProvider is the default implementation of HomeDirProvider.
type DefaultHomeDirProvider struct{}

// Get implements HomeDirProvider.
func (d *DefaultHomeDirProvider) Get() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

// DefaultKnowledgeStoreFactory is the default implementation of KnowledgeStoreFactory.
type DefaultKnowledgeStoreFactory struct{}

// NewCollection implements KnowledgeStoreFactory.
func (d *DefaultKnowledgeStoreFactory) NewCollection(ctx context.Context, opts storage.ChromemOptions) (*storage.ChromemCollection, error) {
	return storage.NewChromemCollection(ctx, opts)
}

// DefaultPermissionManagerFactory is the default implementation of PermissionManagerFactory.
type DefaultPermissionManagerFactory struct{}

// NewManager implements PermissionManagerFactory.
func (d *DefaultPermissionManagerFactory) NewManager(cfg *sdk.PermissionConfig) (*sdk.PermissionManager, error) {
	return sdk.NewPermissionManagerFromConfig(cfg)
}

// ConvertPermissionsConfig converts config.PermissionsConfig to sdk.PermissionConfig.
func ConvertPermissionsConfig(cfg config.PermissionsConfig) *sdk.PermissionConfig {
	return sdk.NewPermissionConfig(
		cfg.DefaultLevel,
		cfg.Tools,
		sdk.CommandRestrictions{
			AllowedCommands: cfg.ShellRestrictions.AllowedCommands,
			DeniedCommands:  cfg.ShellRestrictions.DeniedCommands,
		},
		sdk.SSHRestrictions{
			AllowedHosts:    cfg.SSHRestrictions.AllowedHosts,
			DeniedHosts:     cfg.SSHRestrictions.DeniedHosts,
			AllowedCommands: cfg.SSHRestrictions.AllowedCommands,
			DeniedCommands:  cfg.SSHRestrictions.DeniedCommands,
		},
		cfg.TrustedDirs,
		cfg.AutoAskTrust,
	)
}
