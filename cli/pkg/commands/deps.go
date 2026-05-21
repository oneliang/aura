// Package commands provides CLI commands.
package commands

import (
	"github.com/oneliang/aura/cli/pkg/common"
)

// Re-export types from common package for backward compatibility.
type (
	ConfigLoader             = common.ConfigLoader
	HomeDirProvider          = common.HomeDirProvider
	KnowledgeStoreFactory    = common.KnowledgeStoreFactory
	PermissionManagerFactory = common.PermissionManagerFactory
)

// Re-export implementations from common package.
var (
	NewConfigLoader             = func() ConfigLoader { return &common.DefaultConfigLoader{} }
	NewHomeDirProvider          = func() HomeDirProvider { return &common.DefaultHomeDirProvider{} }
	NewKnowledgeStoreFactory    = func() KnowledgeStoreFactory { return &common.DefaultKnowledgeStoreFactory{} }
	NewPermissionManagerFactory = func() PermissionManagerFactory { return &common.DefaultPermissionManagerFactory{} }
)

// ConvertPermissionsConfig is an alias for common.ConvertPermissionsConfig.
var ConvertPermissionsConfig = common.ConvertPermissionsConfig
