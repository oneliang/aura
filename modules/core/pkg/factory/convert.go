// Package factory provides factories for creating core components.
package factory

import (
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/config"
)

// PermissionManagerFactory creates permission manager instances.
type PermissionManagerFactory struct{}

// NewPermissionManagerFactory creates a new permission manager factory.
func NewPermissionManagerFactory() *PermissionManagerFactory {
	return &PermissionManagerFactory{}
}

// Create creates a permission manager from configuration.
func (f *PermissionManagerFactory) Create(cfg *config.PermissionsConfig) (*permissions.Manager, error) {
	permCfg := permissions.NewPermissionConfig(
		cfg.DefaultLevel,
		cfg.Tools,
		permissions.CommandRestrictions{
			AllowedCommands: cfg.ShellRestrictions.AllowedCommands,
			DeniedCommands:  cfg.ShellRestrictions.DeniedCommands,
		},
		permissions.SSHRestrictions{
			AllowedHosts:    cfg.SSHRestrictions.AllowedHosts,
			DeniedHosts:     cfg.SSHRestrictions.DeniedHosts,
			AllowedCommands: cfg.SSHRestrictions.AllowedCommands,
			DeniedCommands:  cfg.SSHRestrictions.DeniedCommands,
		},
		cfg.TrustedDirs,
		cfg.AutoAskTrust,
	)
	return permissions.NewManager(permCfg)
}
