// Package manager provides MCP server lifecycle management.
package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/oneliang/aura/mcp/pkg/client"
	"github.com/oneliang/aura/mcp/pkg/config"
	"github.com/oneliang/aura/mcp/pkg/constants"
	"github.com/oneliang/aura/mcp/pkg/loader"
	"github.com/oneliang/aura/mcp/pkg/tool"
	tools "github.com/oneliang/aura/tools/pkg"
)

// Manager manages MCP server lifecycle (start, stop, tool discovery).
type Manager struct {
	loader  *loader.Loader
	clients map[string]*client.Client
	tools   map[string]*tool.MCPTool // fullName → MCPTool
	mu      sync.RWMutex
}

// NewManager creates a new MCP manager.
func NewManager(ldr *loader.Loader) *Manager {
	return &Manager{
		loader:  ldr,
		clients: make(map[string]*client.Client),
		tools:   make(map[string]*tool.MCPTool),
	}
}

// StartAll starts all configured MCP servers in parallel and returns discovered tools.
func (m *Manager) StartAll(ctx context.Context) ([]tools.Tool, error) {
	var servers map[string]config.ServerConfig
	if m.loader != nil {
		servers = m.loader.GetServers()
	}
	if len(servers) == 0 {
		return nil, nil
	}

	g, gCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	var allTools []tools.Tool

	for name, srvCfg := range servers {
		name, srvCfg := name, srvCfg
		g.Go(func() error {
			defer func() { recover() }()
			c, srvTools, err := m.initServer(gCtx, name, srvCfg)
			if err != nil {
				return fmt.Errorf("MCP server %s: %w", name, err)
			}
			m.mu.Lock()
			m.clients[name] = c
			for _, t := range srvTools {
				if mt, ok := t.(*tool.MCPTool); ok {
					m.tools[mt.Name()] = mt
				}
			}
			m.mu.Unlock()
			mu.Lock()
			allTools = append(allTools, srvTools...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return allTools, nil
}

// StartServer starts a specific MCP server and returns its discovered tools.
func (m *Manager) StartServer(ctx context.Context, name string, cfg config.ServerConfig) ([]tools.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing if already running
	if existing, ok := m.clients[name]; ok {
		if err := existing.Close(); err != nil {
			// Non-fatal: existing client may already be closed
		}
		delete(m.clients, name)
		// Remove old tools
		m.removeToolsByServer(name)
	}

	return m.startServerLocked(ctx, name, cfg)
}

// StopServer stops a specific MCP server.
func (m *Manager) StopServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.clients[name]
	if !ok {
		return nil // not running
	}

	if err := c.Close(); err != nil {
		return fmt.Errorf("stop MCP server %s: %w", name, err)
	}

	delete(m.clients, name)
	m.removeToolsByServer(name)
	return nil
}

// StopAll stops all MCP servers gracefully.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	clients := m.clients
	m.clients = make(map[string]*client.Client)
	m.tools = make(map[string]*tool.MCPTool)
	m.mu.Unlock()

	var errs []error
	for name, c := range clients {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("stop MCP server %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		if len(errs) == 1 {
			return errs[0]
		}
		return fmt.Errorf("multiple MCP stop errors: %v", errs)
	}
	return nil
}

// GetTools returns all registered MCP tools.
func (m *Manager) GetTools() []tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]tools.Tool, 0, len(m.tools))
	for _, t := range m.tools {
		result = append(result, t)
	}
	return result
}

// ListServers returns runtime info for all configured MCP servers.
func (m *Manager) ListServers() []config.ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make(map[string]config.ServerConfig)
	if m.loader != nil {
		servers = m.loader.GetServers()
	}
	if len(servers) == 0 {
		return nil
	}

	infos := make([]config.ServerInfo, 0, len(servers))
	for name, srvCfg := range servers {
		info := config.ServerInfo{
			Name: name,
		}
		if srvCfg.IsHTTP() {
			info.Command = "http://"
			info.Args = []string{srvCfg.URL}
		} else {
			info.Command = srvCfg.Command
			info.Args = srvCfg.Args
		}
		if _, ok := m.clients[name]; ok {
			info.Status = "running"
			// Count tools for this server
			for _, t := range m.tools {
				if t.ServerName() == name {
					info.ToolCount++
				}
			}
			info.LastSeen = time.Now()
		} else {
			info.Status = "stopped"
		}
		infos = append(infos, info)
	}
	return infos
}

// AddServer adds a new server to config and starts it.
func (m *Manager) AddServer(ctx context.Context, name string, cfg config.ServerConfig) ([]tools.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update loader config
	loaderCfg := m.loader.GetConfig()
	if loaderCfg == nil {
		loaderCfg = &config.Config{MCPServers: make(map[string]config.ServerConfig)}
	}
	loaderCfg.MCPServers[name] = cfg
	if err := m.loader.Save(loaderCfg); err != nil {
		return nil, fmt.Errorf("save MCP config: %w", err)
	}

	return m.startServerLocked(ctx, name, cfg)
}

// RemoveServer removes a server from config and stops it.
func (m *Manager) RemoveServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop if running
	if c, ok := m.clients[name]; ok {
		if err := c.Close(); err != nil {
			// Non-fatal: client may already be closed
		}
		delete(m.clients, name)
		m.removeToolsByServer(name)
	}

	// Remove from config
	loaderCfg := m.loader.GetConfig()
	if loaderCfg != nil {
		delete(loaderCfg.MCPServers, name)
		if err := m.loader.Save(loaderCfg); err != nil {
			return fmt.Errorf("save MCP config: %w", err)
		}
	}
	return nil
}

// Reload reloads all servers from config.
func (m *Manager) Reload(ctx context.Context) ([]tools.Tool, error) {
	if _, err := m.loader.Load(); err != nil {
		return nil, err
	}
	// Stop all and restart from new config
	if err := m.StopAll(ctx); err != nil {
		// Non-fatal: continue with restart, log warning if available
	}
	return m.StartAll(ctx)
}

// initServer starts a server, initializes client, and returns discovered tools.
// Does NOT acquire the manager lock - caller is responsible for thread safety
// when registering tools. Returns the client and tools for the caller to register.
func (m *Manager) initServer(ctx context.Context, name string, cfg config.ServerConfig) (*client.Client, []tools.Tool, error) {
	timeout := cfg.GetTimeout()

	var c *client.Client
	if cfg.IsHTTP() {
		c = client.NewHTTPClient(cfg.URL, cfg.Headers)
	} else {
		c = client.NewStdioClient(cfg.Command, cfg.Args, cfg.Env)
	}
	if err := c.Initialize(ctx); err != nil {
		return nil, nil, fmt.Errorf("initialize: %w", err)
	}

	mcpTools, err := c.ListTools(ctx)
	if err != nil {
		if closeErr := c.Close(); closeErr != nil {
			// Non-fatal: client may be partially initialized
		}
		return nil, nil, fmt.Errorf("list tools: %w", err)
	}

	var result []tools.Tool
	for _, mt := range mcpTools {
		schemaMap := map[string]any{
			"type":       mt.InputSchema.Type,
			"properties": mt.InputSchema.Properties,
		}
		if len(mt.InputSchema.Required) > 0 {
			schemaMap["required"] = mt.InputSchema.Required
		}

		mcpTool := tool.NewMCPTool(name, mt.Name, mt.Description, schemaMap,
			func(callCtx context.Context, args map[string]any) (string, error) {
				return c.CallToolWithTimeout(callCtx, mt.Name, args, timeout)
			})

		result = append(result, mcpTool)
	}

	return c, result, nil
}

// startServerLocked starts a server and registers its tools (caller must hold lock).
func (m *Manager) startServerLocked(ctx context.Context, name string, cfg config.ServerConfig) ([]tools.Tool, error) {
	c, tools, err := m.initServer(ctx, name, cfg)
	if err != nil {
		return nil, err
	}

	m.clients[name] = c
	for _, t := range tools {
		if mt, ok := t.(*tool.MCPTool); ok {
			m.tools[mt.Name()] = mt
		}
	}
	return tools, nil
}

// removeToolsByServer removes all tools from a specific server (caller must hold lock).
func (m *Manager) removeToolsByServer(name string) {
	for fullName, t := range m.tools {
		if t.ServerName() == name {
			delete(m.tools, fullName)
		}
	}
}

// ServerInfoForName returns runtime info for a specific server.
func (m *Manager) ServerInfoForName(name string) config.ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := config.ServerInfo{
		Name: name,
	}

	var servers map[string]config.ServerConfig
	if m.loader != nil {
		servers = m.loader.GetServers()
	}
	if srvCfg, ok := servers[name]; ok {
		if srvCfg.IsHTTP() {
			info.Command = "http://"
			info.Args = []string{srvCfg.URL}
		} else {
			info.Command = srvCfg.Command
			info.Args = srvCfg.Args
		}
	}

	if _, running := m.clients[name]; running {
		info.Status = "running"
		for _, t := range m.tools {
			if t.ServerName() == name {
				info.ToolCount++
			}
		}
		info.LastSeen = time.Now()
	} else {
		info.Status = "stopped"
	}

	return info
}

// ServerInfoForNameWithTimeout returns runtime info for a specific server,
// with timeout for health check on running servers.
func (m *Manager) ServerInfoForNameWithTimeout(name string, timeout time.Duration) config.ServerInfo {
	if timeout == 0 {
		timeout = constants.DefaultTimeout
	}
	m.mu.RLock()
	c, running := m.clients[name]
	m.mu.RUnlock()

	info := m.ServerInfoForName(name)

	if running && c != nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if !c.HealthCheck(ctx) {
			info.Status = "error"
			info.Error = "health check failed"
		}
	}

	return info
}
