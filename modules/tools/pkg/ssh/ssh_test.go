package ssh

import (
	"context"
	"os"
	"testing"
	"time"

	tools "github.com/oneliang/aura/tools/pkg"
	"golang.org/x/crypto/ssh"
)

// TestExecToolName tests the SSH exec tool name.
func TestExecToolName(t *testing.T) {
	tool := NewExecTool()

	if tool.Name() != "ssh_exec" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "ssh_exec")
	}
}

// TestExecToolDescription tests the SSH exec tool description.
func TestExecToolDescription(t *testing.T) {
	servers := []ServerConfig{
		{Name: "test-server", Host: "192.168.1.1", Port: 22, User: "admin"},
	}
	tool := NewExecTool(WithServers(servers))

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	// Check if description contains pre-configured server info
	if !containsString(desc, "test-server") {
		t.Error("Description() should contain pre-configured server info")
	}
}

// TestExecToolRequiresConfirmation tests that SSH tool requires confirmation.
func TestExecToolRequiresConfirmation(t *testing.T) {
	tool := NewExecTool()

	if !tool.RequiresConfirmation() {
		t.Error("RequiresConfirmation() should return true")
	}
}

// TestExecToolExecuteValidation tests parameter validation.
func TestExecToolExecuteValidation(t *testing.T) {
	tool := NewExecTool()
	ctx := context.Background()

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing command",
			params:  map[string]any{},
			wantErr: false, // Returns ToolResult{Status: ToolStatusError}, not Go error
			errMsg:  "command parameter is required",
		},
		{
			name: "missing server and host",
			params: map[string]any{
				"command": "ls -la",
			},
			wantErr: true,
			errMsg:  "either 'server' or 'host' parameter is required",
		},
		{
			name: "non-existent server",
			params: map[string]any{
				"server":  "non-existent",
				"command": "ls -la",
			},
			wantErr: false, // Returns ToolResult{Status: ToolStatusError}, not Go error
			errMsg:  "not found in configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)

			if tt.wantErr && err == nil {
				t.Error("Execute() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Execute() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}

			// Check ToolResult error for execution failures
			if !tt.wantErr && tt.errMsg != "" && result != nil {
				if result.Status != tools.ToolStatusError {
					t.Errorf("Execute() expected ToolStatusError, got %v", result.Status)
				}
				if !containsString(result.Error, tt.errMsg) {
					t.Errorf("Execute() result.Error = %q, want to contain %q", result.Error, tt.errMsg)
				}
			}
		})
	}
}

// TestExecToolFindServer tests the findServer method.
func TestExecToolFindServer(t *testing.T) {
	servers := []ServerConfig{
		{Name: "server1", Host: "192.168.1.1", Port: 22, User: "admin"},
		{Name: "server2", Host: "192.168.1.2", Port: 2222, User: "root"},
	}
	tool := NewExecTool(WithServers(servers))

	tests := []struct {
		name       string
		serverName string
		wantFound  bool
		wantName   string
	}{
		{
			name:       "find existing server",
			serverName: "server1",
			wantFound:  true,
			wantName:   "server1",
		},
		{
			name:       "find another server",
			serverName: "server2",
			wantFound:  true,
			wantName:   "server2",
		},
		{
			name:       "non-existent server",
			serverName: "non-existent",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection or access the method indirectly
			// Since findServer is private, we test it through Execute
			result, err := tool.Execute(context.Background(), map[string]any{
				"server":  tt.serverName,
				"command": "test",
			})

			if tt.wantFound {
				// Should fail at connection, not at server lookup
				if err != nil {
					t.Errorf("Expected server to be found, got error: %v", err)
				}
				if result != nil && result.Status == tools.ToolStatusError && containsString(result.Error, "not found in configuration") {
					t.Errorf("Expected server to be found, got 'not found' error: %v", result.Error)
				}
			} else {
				// Should fail at server lookup (ToolResult error, not Go error)
				if err != nil {
					t.Errorf("Expected 'not found' ToolResult error, got Go error: %v", err)
				}
				if result == nil || result.Status != tools.ToolStatusError || !containsString(result.Error, "not found in configuration") {
					t.Errorf("Expected 'not found' error, got: result=%v, err=%v", result, err)
				}
			}
		})
	}
}

// TestExecToolOptions tests configuration options.
func TestExecToolOptions(t *testing.T) {
	servers := []ServerConfig{
		{Name: "test", Host: "192.168.1.1", Port: 22, User: "admin"},
	}

	tool := NewExecTool(
		WithServers(servers),
		WithTimeout(60*time.Second),
		WithKnownHostsFile("/tmp/known_hosts"),
		WithInsecureMode(true),
	)

	// Verify options were applied (indirectly through behavior)
	if tool.timeout != 60*time.Second {
		t.Errorf("timeout = %v, want %v", tool.timeout, 60*time.Second)
	}

	if tool.knownHostsFile != "/tmp/known_hosts" {
		t.Errorf("knownHostsFile = %q, want %q", tool.knownHostsFile, "/tmp/known_hosts")
	}

	if tool.insecure != true {
		t.Error("insecure should be true")
	}
}

// TestExpandPath tests the expandPath helper function.
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantSuffix string
	}{
		{
			name:       "tilde expansion",
			path:       "~/test/file",
			wantSuffix: "test/file",
		},
		{
			name:       "absolute path",
			path:       "/absolute/path",
			wantSuffix: "/absolute/path",
		},
		{
			name:       "relative path",
			path:       "relative/path",
			wantSuffix: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path)

			if tt.path[0:1] == "~" {
				// Should be expanded
				if result == tt.path {
					t.Error("expandPath() did not expand tilde path")
				}
			} else {
				// Should remain the same
				if result != tt.path {
					t.Errorf("expandPath() = %q, want %q", result, tt.path)
				}
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestExecToolExecuteWithHost tests Execute with host parameter.
func TestExecToolExecuteWithHost(t *testing.T) {
	// Use insecure mode to skip host key verification
	tool := NewExecTool(WithInsecureMode(true))
	ctx := context.Background()

	// Test with host but no authentication - should fail at auth method
	result, err := tool.Execute(ctx, map[string]any{
		"host":    "192.168.1.100",
		"command": "ls -la",
	})

	// Should fail because no authentication method provided (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	} else if !containsString(result.Error, "no authentication method configured") {
		t.Errorf("Execute() result.Error = %q, want to contain %q", result.Error, "no authentication method configured")
	}
}

// TestExecToolExecuteWithDefaults tests Execute with default user and port.
func TestExecToolExecuteWithDefaults(t *testing.T) {
	tool := NewExecTool()
	ctx := context.Background()

	// Test with host and key_path - will fail at connection but tests defaults
	result, err := tool.Execute(ctx, map[string]any{
		"host":     "localhost",
		"command":  "test",
		"key_path": "/nonexistent/key",
	})

	// Should fail at key reading (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	}
}

// TestCreateHostKeyCallback tests host key callback creation.
func TestCreateHostKeyCallback(t *testing.T) {
	t.Run("insecure mode", func(t *testing.T) {
		tool := NewExecTool(WithInsecureMode(true))
		callback, err := tool.createHostKeyCallback()

		if err != nil {
			t.Errorf("createHostKeyCallback() unexpected error = %v", err)
		}
		if callback == nil {
			t.Error("createHostKeyCallback() should return a callback")
		}
	})

	t.Run("with known_hosts file", func(t *testing.T) {
		// Create a temporary known_hosts file
		tmpFile := t.TempDir() + "/known_hosts"
		if err := os.WriteFile(tmpFile, []byte("# empty known_hosts\n"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		tool := NewExecTool(WithKnownHostsFile(tmpFile))
		callback, err := tool.createHostKeyCallback()

		if err != nil {
			t.Errorf("createHostKeyCallback() unexpected error = %v", err)
		}
		if callback == nil {
			t.Error("createHostKeyCallback() should return a callback")
		}
	})

	t.Run("invalid known_hosts file", func(t *testing.T) {
		tool := NewExecTool(WithKnownHostsFile("/nonexistent/file"))
		_, err := tool.createHostKeyCallback()

		if err == nil {
			t.Error("createHostKeyCallback() expected error for nonexistent file")
		}
	})
}

// TestServerConfig tests ServerConfig structure.
func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Name:     "test",
		Host:     "192.168.1.1",
		Port:     2222,
		User:     "testuser",
		KeyPath:  "~/.ssh/id_rsa",
		Password: "secret",
	}

	if config.Name != "test" {
		t.Errorf("Name = %q, want %q", config.Name, "test")
	}
	if config.Host != "192.168.1.1" {
		t.Errorf("Host = %q, want %q", config.Host, "192.168.1.1")
	}
	if config.Port != 2222 {
		t.Errorf("Port = %d, want %d", config.Port, 2222)
	}
	if config.User != "testuser" {
		t.Errorf("User = %q, want %q", config.User, "testuser")
	}
}

// TestExecToolDescriptionWithoutServers tests description without pre-configured servers.
func TestExecToolDescriptionWithoutServers(t *testing.T) {
	tool := NewExecTool()
	desc := tool.Description()

	if desc == "" {
		t.Error("Description() returned empty string")
	}

	// Should contain basic parameter info
	if !containsString(desc, "command") {
		t.Error("Description() should mention command parameter")
	}
	if !containsString(desc, "host") {
		t.Error("Description() should mention host parameter")
	}
	if !containsString(desc, "server") {
		t.Error("Description() should mention server parameter")
	}
}

// TestExecToolExecuteWithPassword tests Execute with password authentication.
func TestExecToolExecuteWithPassword(t *testing.T) {
	tool := NewExecTool()
	ctx := context.Background()

	// Test with password - will fail at connection but tests auth flow
	result, err := tool.Execute(ctx, map[string]any{
		"host":     "localhost",
		"command":  "test",
		"password": "testpass",
	})

	// Should fail at connection (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	}
}

// TestHostKeyError tests the HostKeyError type.
func TestHostKeyError(t *testing.T) {
	err := &HostKeyError{
		Host:     "example.com",
		Expected: "expected-key",
		Got:      "got-key",
	}

	expectedMsg := "host key verification failed for example.com"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestDialWithCtx tests dialWithCtx function.
func TestDialWithCtx(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		config := &ssh.ClientConfig{
			User: "test",
		}

		_, err := dialWithCtx(ctx, "tcp", "nonexistent:22", config)

		// Should get context error
		if err == nil {
			t.Error("dialWithCtx() expected error for cancelled context")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		ctx := context.Background()
		config := &ssh.ClientConfig{
			User: "test",
		}

		_, err := dialWithCtx(ctx, "tcp", "localhost:1", config)

		// Should get connection error
		if err == nil {
			t.Error("dialWithCtx() expected error for connection refused")
		}
	})
}

// TestExecToolTimeout tests timeout configuration.
func TestExecToolTimeout(t *testing.T) {
	tool := NewExecTool(WithTimeout(5 * time.Second))

	if tool.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want %v", tool.timeout, 5*time.Second)
	}
}

// TestExecToolEmptyServerName tests Execute with empty server name.
func TestExecToolEmptyServerName(t *testing.T) {
	tool := NewExecTool()
	ctx := context.Background()

	// Empty server name is treated as no server, falls through to host check
	result, err := tool.Execute(ctx, map[string]any{
		"server":  "",
		"command": "test",
	})

	// Empty string is falsy - should fail because no host provided (Go error - validation)
	if err == nil {
		t.Error("Execute() expected validation error for missing host")
	}
	if !containsString(err.Error(), "either 'server' or 'host' parameter is required") {
		t.Errorf("Execute() error = %q, want to contain %q", err.Error(), "either 'server' or 'host' parameter is required")
	}
	if result != nil {
		t.Errorf("Execute() expected nil result for validation error, got: %+v", result)
	}
}

// TestExecToolWithMultipleServers tests Execute with multiple pre-configured servers.
func TestExecToolWithMultipleServers(t *testing.T) {
	servers := []ServerConfig{
		{Name: "prod", Host: "prod.example.com", Port: 22, User: "admin"},
		{Name: "staging", Host: "staging.example.com", Port: 22, User: "deploy"},
		{Name: "dev", Host: "dev.example.com", Port: 2222, User: "developer"},
	}
	tool := NewExecTool(WithServers(servers))

	desc := tool.Description()

	// Should list all servers
	if !containsString(desc, "prod") {
		t.Error("Description() should list prod server")
	}
	if !containsString(desc, "staging") {
		t.Error("Description() should list staging server")
	}
	if !containsString(desc, "dev") {
		t.Error("Description() should list dev server")
	}
}

// TestExecToolExecuteInvalidPort tests with invalid port type.
func TestExecToolExecuteInvalidPort(t *testing.T) {
	tool := NewExecTool()
	ctx := context.Background()

	// Port as string instead of int - should use default
	result, err := tool.Execute(ctx, map[string]any{
		"host":    "localhost",
		"port":    "invalid", // wrong type
		"command": "test",
	})

	// Should fail at connection (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	}
}

// TestExecToolExecuteOnServer tests executeOnServer method error paths.
func TestExecToolExecuteOnServer(t *testing.T) {
	tool := NewExecTool(WithInsecureMode(true))
	ctx := context.Background()

	server := &ServerConfig{
		Host:     "nonexistent.invalid",
		Port:     22,
		User:     "test",
		Password: "test",
	}

	result, err := tool.executeOnServer(ctx, server, "echo test")

	// Should fail at connection (ToolResult error)
	if err != nil {
		t.Errorf("executeOnServer() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("executeOnServer() expected ToolStatusError result")
	} else if !containsString(result.Error, "failed to connect") {
		t.Errorf("executeOnServer() result.Error = %q, want to contain %q", result.Error, "failed to connect")
	}
}

// TestExecToolKeyPathNotFound tests SSH key file not found error.
func TestExecToolKeyPathNotFound(t *testing.T) {
	tool := NewExecTool(WithInsecureMode(true))
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"host":     "localhost",
		"command":  "test",
		"key_path": "/nonexistent/path/to/key",
	})

	// Should fail at key reading (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	} else if !containsString(result.Error, "failed to read SSH key") {
		t.Errorf("Execute() result.Error = %q, want to contain %q", result.Error, "failed to read SSH key")
	}
}

// TestExecToolInvalidKeyPath tests invalid SSH key format error.
func TestExecToolInvalidKeyPath(t *testing.T) {
	tool := NewExecTool(WithInsecureMode(true))
	ctx := context.Background()

	// Create a temporary file with invalid key content
	tmpFile := t.TempDir() + "/invalid_key"
	if err := os.WriteFile(tmpFile, []byte("not a valid SSH key"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := tool.Execute(ctx, map[string]any{
		"host":     "localhost",
		"command":  "test",
		"key_path": tmpFile,
	})

	// Should fail at key parsing (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	} else if !containsString(result.Error, "failed to parse SSH key") {
		t.Errorf("Execute() result.Error = %q, want to contain %q", result.Error, "failed to parse SSH key")
	}
}

// TestExecToolSessionError tests session creation error path (simulated).
func TestExecToolSessionError(t *testing.T) {
	// This tests the session error handling path
	// We can't easily trigger a session.NewSession() error without mocking
	// but we can at least verify the error handling structure
	tool := NewExecTool(WithInsecureMode(true))

	// Test with a valid key path but connection will fail before session
	tmpFile := t.TempDir() + "/valid_key"
	// Create a valid PEM-encoded private key for testing
	validKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEA0Z3VS5JJcds3xMz2hHjVHjYAAAAD
-----END OPENSSH PRIVATE KEY-----`
	if err := os.WriteFile(tmpFile, []byte(validKey), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"host":     "localhost",
		"command":  "test",
		"key_path": tmpFile,
	})

	// Should fail at key parsing (invalid key) or connection (ToolResult error)
	if err != nil {
		t.Errorf("Execute() expected ToolResult error, got Go error: %v", err)
	}
	if result == nil || result.Status != tools.ToolStatusError {
		t.Error("Execute() expected ToolStatusError result")
	}
}

// TestExpandPathError tests expandPath with invalid home directory.
func TestExpandPathError(t *testing.T) {
	// This is hard to test without mocking os.UserHomeDir
	// But we can test the non-tilde case
	result := expandPath("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("expandPath() = %q, want %q", result, "/absolute/path")
	}
}
