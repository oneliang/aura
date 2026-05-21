// Package factory provides factories for creating core components.
package factory

import (
	"context"
	"net/http"
	"os"

	"github.com/oneliang/aura/shared/pkg/httpclient"
	"github.com/oneliang/aura/shared/pkg/logger"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"

	"github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/knowledge/pkg/embedding"
	"github.com/oneliang/aura/knowledge/pkg/knowledgetool"
	knowledgestorage "github.com/oneliang/aura/knowledge/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/user"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/filesystem"
	"github.com/oneliang/aura/tools/pkg/location"
	"github.com/oneliang/aura/tools/pkg/lsp"
	"github.com/oneliang/aura/tools/pkg/search"
	"github.com/oneliang/aura/tools/pkg/ssh"
	"github.com/oneliang/aura/tools/pkg/system"
	"github.com/oneliang/aura/tools/pkg/utility"
	"github.com/oneliang/aura/tools/pkg/web"
)

// ToolRegistry manages tool registration.
type ToolRegistry struct {
	config        *config.ToolsConfig
	permMgr       *permissions.Manager
	webHttpClient *http.Client
	logger        *logger.Logger
}

// ToolRegistryOption is a configuration option for ToolRegistry.
type ToolRegistryOption func(*ToolRegistry)

// WithWebHTTPClient sets the HTTP client for web tools.
func WithWebHTTPClient(client *http.Client) ToolRegistryOption {
	return func(r *ToolRegistry) {
		r.webHttpClient = client
	}
}

// WithToolRegistryLogger sets the logger for the tool registry.
func WithToolRegistryLogger(log *logger.Logger) ToolRegistryOption {
	return func(r *ToolRegistry) { r.logger = log }
}

// trustedPathAdapter adapts permissions.Manager to filesystem trustedpath.Checker interface.
type trustedPathAdapter struct {
	mgr *permissions.Manager
}

// IsTrustedPath checks if a path is within any trusted directory.
func (a *trustedPathAdapter) IsTrustedPath(path string) bool {
	return a.mgr.IsTrustedPath(path)
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(cfg *config.ToolsConfig, permMgr *permissions.Manager, opts ...ToolRegistryOption) *ToolRegistry {
	r := &ToolRegistry{
		config:        cfg,
		permMgr:       permMgr,
		webHttpClient: httpclient.DefaultWebClient(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RegisterAll registers all tools (built-in) to the engine.
func (r *ToolRegistry) RegisterAll(ctx context.Context, ag *engine.Engine, cfg *config.Config) {
	// Register built-in tools
	r.registerBuiltInTools(ctx, ag, cfg)
}

// registerBuiltInTools registers all built-in tools based on config.Enabled list.
func (r *ToolRegistry) registerBuiltInTools(ctx context.Context, ag *engine.Engine, cfg *config.Config) {
	// Create trusted path adapter for file system tools
	var pathAdapter *trustedPathAdapter
	if r.permMgr != nil {
		pathAdapter = &trustedPathAdapter{mgr: r.permMgr}
	}

	// Build tool factory map (name -> factory function)
	toolFactories := r.buildToolFactories(pathAdapter, cfg)

	// If Enabled list is empty, register all available tools (backward compatible)
	enabledList := r.config.Enabled
	if len(enabledList) == 0 {
		enabledList = r.getDefaultEnabledTools()
	}

	// Register tools based on enabled list
	enabledSet := make(map[string]bool, len(enabledList))
	for _, name := range enabledList {
		enabledSet[name] = true
	}

	for name, factory := range toolFactories {
		if enabledSet[name] {
			ag.AddTool(factory())
		}
	}

	// Knowledge tools (if configured and enabled)
	if enabledSet["knowledge_search"] || enabledSet["knowledge_import"] {
		r.registerKnowledgeTools(ctx, ag, cfg)
	}
}

// buildToolFactories creates a map of tool name to factory function.
func (r *ToolRegistry) buildToolFactories(pathAdapter *trustedPathAdapter, cfg *config.Config) map[string]func() tools.Tool {
	// Safe tools (no confirmation required)
	factories := map[string]func() tools.Tool{
		"file_read":   func() tools.Tool { return filesystem.NewReadTool(pathAdapter) },
		"file_search": func() tools.Tool { return filesystem.NewSearchTool(pathAdapter) },
		"file_list":   func() tools.Tool { return filesystem.NewListTool(pathAdapter) },
		"glob":        func() tools.Tool { return search.NewGlobTool(pathAdapter) },
		"grep":        func() tools.Tool { return search.NewGrepTool(pathAdapter) },
		"datetime":    func() tools.Tool { return utility.NewDateTimeTool() },
		"calculator":  func() tools.Tool { return utility.NewCalculatorTool() },
		"text":        func() tools.Tool { return utility.NewTextTool() },
		"location": func() tools.Tool {
			return location.NewLocationTool(
				location.WithHTTPClient(r.webHttpClient),
				location.WithConfig(location.LocationConfig{
					FixedCity:    cfg.Location.FixedCity,
					FixedCountry: cfg.Location.FixedCountry,
					AutoDetect:   cfg.Location.AutoDetect,
				}),
			)
		},
		"web_fetch": func() tools.Tool {
			return web.NewFetchTool(web.WithHTTPClient(r.webHttpClient))
		},
		// Sensitive tools (will require confirmation)
		"file_write": func() tools.Tool { return filesystem.NewWriteTool(pathAdapter) },
		"bash":       func() tools.Tool { return system.NewShellTool() },
		"ssh_exec": func() tools.Tool {
			sshServers := convertSSHServers(cfg.SSH.Servers)
			return ssh.NewExecTool(ssh.WithServers(sshServers))
		},
		"code_navigate": func() tools.Tool {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			return lsp.NewCodeTool(cwd)
		},
	}
	return factories
}

// getDefaultEnabledTools returns the default list of enabled tools.
func (r *ToolRegistry) getDefaultEnabledTools() []string {
	return []string{
		"file_read", "file_write", "file_search", "file_list",
		"bash", "datetime", "calculator", "web_fetch",
		"code_navigate", "text", "glob", "grep", "location",
	}
}

// registerKnowledgeTools registers knowledge base tools if embedding model is configured.
func (r *ToolRegistry) registerKnowledgeTools(ctx context.Context, ag *engine.Engine, cfg *config.Config) {
	if cfg.LLM.EmbeddingModel == "" {
		return
	}

	if r.logger == nil {
		r.logger = logger.RegistryDefault()
	}

	dataDir := ffp.AuraHomePathOrDefault(constants.DirKnowledge)
	if dataDir == "" {
		r.logger.Warn().Msg("cannot determine home directory, knowledge base disabled")
		return
	}

	embFn := embedding.OllamaEmbeddingFunc(cfg.LLM.BaseURL, cfg.LLM.EmbeddingModel)
	col, err := knowledgestorage.NewChromemCollection(ctx, knowledgestorage.ChromemOptions{
		DataDir:       dataDir,
		Name:          user.DefaultCollectionName,
		EmbeddingFunc: embFn,
	})
	if err != nil {
		r.logger.Warn().Err(err).Msg("knowledge base unavailable")
		return
	}
	ag.AddTool(knowledgetool.NewSearchTool(col))
	ag.AddTool(knowledgetool.NewImportTool(col))
}

// convertSSHServers converts config SSH servers to SSH tool server configs.
func convertSSHServers(servers []config.SSHServerConfig) []ssh.ServerConfig {
	result := make([]ssh.ServerConfig, len(servers))
	for i, s := range servers {
		result[i] = ssh.ServerConfig{
			Name:     s.Name,
			Host:     s.Host,
			Port:     s.Port,
			User:     s.User,
			KeyPath:  s.KeyPath,
			Password: s.Password,
		}
	}
	return result
}

// GetToolNames returns the names of all registered tools.
func GetToolNames(ag *engine.Engine) []string {
	tools := ag.GetTools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return names
}
