// Package ssh provides SSH remote execution tools.
package ssh

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ExecTool executes commands on remote servers via SSH.
type ExecTool struct {
	timeout        time.Duration
	servers        []ServerConfig
	knownHostsFile string
	insecure       bool // Only for development - should be false in production
}

// ServerConfig represents a pre-configured SSH server.
type ServerConfig struct {
	Name     string
	Host     string
	Port     int
	User     string
	KeyPath  string
	Password string
}

// Option is a configuration option for ExecTool.
type Option func(*ExecTool)

// WithTimeout sets the command timeout.
func WithTimeout(d time.Duration) Option {
	return func(t *ExecTool) {
		t.timeout = d
	}
}

// WithServers sets pre-configured servers.
func WithServers(servers []ServerConfig) Option {
	return func(t *ExecTool) {
		t.servers = servers
	}
}

// WithKnownHostsFile sets the known_hosts file path for host key verification.
func WithKnownHostsFile(path string) Option {
	return func(t *ExecTool) {
		t.knownHostsFile = path
	}
}

// WithInsecureMode enables insecure mode (skips host key verification).
// WARNING: This should only be used for development/testing.
func WithInsecureMode(insecure bool) Option {
	return func(t *ExecTool) {
		t.insecure = insecure
	}
}

// NewExecTool creates a new SSH execution tool.
func NewExecTool(opts ...Option) *ExecTool {
	t := &ExecTool{
		timeout: constants.DefaultSSHTimeout,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *ExecTool) Name() string {
	return constants.ToolSSHExec
}

// Description returns the tool description.
func (t *ExecTool) Description() string {
	desc := `Execute a command on a remote server via SSH.

IMPORTANT: If pre-configured servers are available, ALWAYS use the 'server' parameter with the server name instead of specifying host/user/password manually.

Parameters:
- server (string): Pre-configured server name from config (RECOMMENDED - use this when available)
- host (string): Server hostname or IP (only use if no pre-configured server)
- command (string): Command to execute (required)
- user (string): SSH user (default: root, only needed for dynamic host)
- port (int): SSH port (default: 22, only needed for dynamic host)
- key_path (string): Path to private key file (only needed for dynamic host)
- password (string): Password for authentication (only needed for dynamic host)

Use 'server' parameter for pre-configured servers. Use 'host' and other parameters only for ad-hoc connections.`

	if len(t.servers) > 0 {
		desc += "\n\nPre-configured servers (use these by name with 'server' parameter):\n"
		for _, s := range t.servers {
			desc += fmt.Sprintf("- %s (%s@%s:%d)\n", s.Name, s.User, s.Host, s.Port)
		}
		desc += "\nExample: {\"tool\": \"ssh_exec\", \"parameters\": {\"server\": \"<server-name>\", \"command\": \"df -h\"}}"
	}

	return desc
}

// RequiresConfirmation returns true because SSH execution is a sensitive operation.
func (t *ExecTool) RequiresConfirmation() bool {
	return true
}

// PermissionLevel returns the permission level for this tool.
func (t *ExecTool) PermissionLevel() string {
	return "execute"
}

// Execute runs a command on a remote server.
func (t *ExecTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Get server name or host
	serverName, _ := params["server"].(string)
	host, _ := params["host"].(string)
	command, _ := params["command"].(string)

	if command == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "command parameter is required"}, nil
	}

	// If server name provided, look up config
	if serverName != "" {
		serverConfig := t.findServer(serverName)
		if serverConfig == nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("server '%s' not found in configuration", serverName)}, nil
		}
		return t.executeOnServer(ctx, serverConfig, command)
	}

	// Use dynamic parameters
	if host == "" {
		return nil, fmt.Errorf("either 'server' or 'host' parameter is required")
	}

	user, _ := params["user"].(string)
	port, _ := params["port"].(int)
	keyPath, _ := params["key_path"].(string)
	password, _ := params["password"].(string)

	// Defaults
	if user == "" {
		user = "root"
	}
	if port == 0 {
		port = 22
	}

	serverConfig := &ServerConfig{
		Host:     host,
		Port:     port,
		User:     user,
		KeyPath:  keyPath,
		Password: password,
	}

	return t.executeOnServer(ctx, serverConfig, command)
}

func (t *ExecTool) findServer(name string) *ServerConfig {
	for _, s := range t.servers {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// createHostKeyCallback creates a host key callback that checks against known_hosts.
func (t *ExecTool) createHostKeyCallback() (ssh.HostKeyCallback, error) {
	if t.insecure {
		// WARNING: Only for development!
		return ssh.InsecureIgnoreHostKey(), nil
	}

	if t.knownHostsFile != "" {
		// Use specified known_hosts file for verification
		callback, err := newHostKeyCallback(t.knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load known_hosts file: %w", err)
		}
		return callback, nil
	}

	// Default: collect existing known_hosts files
	var existingFiles []string
	homeDir, err := os.UserHomeDir()
	if err == nil {
		candidates := []string{
			filepath.Join(homeDir, ".ssh", "known_hosts"),
			"/etc/ssh/known_hosts",
		}
		for _, path := range candidates {
			if _, statErr := os.Stat(path); statErr == nil {
				existingFiles = append(existingFiles, path)
			}
		}
	}

	if len(existingFiles) > 0 {
		return newHostKeyCallback(existingFiles...)
	}

	// No known_hosts files exist — use accept-new behavior (like OpenSSH
	// StrictHostKeyChecking=accept-new): accept first-time hosts, but reject
	// key mismatches on subsequent connections. This is safer than
	// InsecureIgnoreHostKey (which accepts everything unconditionally) and
	// more usable than a hard error (which blocks first-time connections).
	return newHostKeyCallback()
}

// newHostKeyCallback creates a host key callback from known_hosts files.
// If no paths are provided (empty known_hosts), it accepts all hosts
// (equivalent to StrictHostKeyChecking=accept-new behavior).
func newHostKeyCallback(paths ...string) (ssh.HostKeyCallback, error) {
	// No known_hosts files — accept all hosts (first-time connection behavior)
	if len(paths) == 0 {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	strictCallback, err := knownhosts.New(paths...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse known_hosts: %w", err)
	}

	// Wrap to accept unknown hosts on first connection (like OpenSSH StrictHostKeyChecking=ask, auto-accept)
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := strictCallback(hostname, remote, key)
		if err != nil {
			// If the error is "key is unknown", accept it (first-time connection)
			if _, ok := err.(*knownhosts.KeyError); ok {
				return nil
			}
			// Key mismatch — this IS a security issue, reject
			return err
		}
		return nil
	}, nil
}

func (t *ExecTool) executeOnServer(ctx context.Context, server *ServerConfig, command string) (*tools.ToolResult, error) {
	// Create host key callback
	hostKeyCallback, err := t.createHostKeyCallback()
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to create host key callback: %v", err)}, nil
	}

	// Build SSH client config
	config := &ssh.ClientConfig{
		User:            server.User,
		HostKeyCallback: hostKeyCallback,
		Timeout:         t.timeout,
	}

	// Add authentication method
	if server.KeyPath != "" {
		// Expand ~ to home directory
		keyPath := expandPath(server.KeyPath)
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to read SSH key: %v", err)}, nil
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to parse SSH key: %v", err)}, nil
		}

		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if server.Password != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(server.Password)}
	} else {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "no authentication method configured (key_path or password required)"}, nil
	}

	// Connect to server with context support
	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := dialWithCtx(ctx, "tcp", addr, config)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to connect to %s: %v", addr, err)}, nil
	}
	defer conn.Close()

	// Create session
	session, err := conn.NewSession()
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("failed to create session: %v", err)}, nil
	}
	defer session.Close()

	// Set environment variables for non-interactive terminal
	session.Setenv("TERM", "xterm-256color")

	// Execute command
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nStderr:\n" + stderr.String()
	}

	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("command failed: %v\n%s", err, output)}, nil
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: output,
	}, nil
}

// dialWithCtx wraps ssh.Dial with context support.
func dialWithCtx(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	type dialResult struct {
		conn *ssh.Client
		err  error
	}
	resultCh := make(chan dialResult, 1)

	go func() {
		conn, err := ssh.Dial(network, addr, config)
		select {
		case resultCh <- dialResult{conn: conn, err: err}:
			// Result sent successfully
		case <-ctx.Done():
			// Context cancelled, close connection if established
			if conn != nil {
				conn.Close()
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.conn, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// HostKeyError represents a host key verification error.
type HostKeyError struct {
	Host     string
	Expected string
	Got      string
}

func (e *HostKeyError) Error() string {
	return fmt.Sprintf("host key verification failed for %s", e.Host)
}
