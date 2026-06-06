// Package client provides MCP client functionality using stdio or HTTP/SSE transport.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client wraps an MCP client connection (stdio or HTTP/SSE).
type Client struct {
	// stdio fields
	command string
	args    []string
	env     []string
	// HTTP fields
	url     string
	headers map[string]string
	// shared
	mcp *client.Client
}

// NewStdioClient creates a new stdio transport MCP client.
func NewStdioClient(command string, args []string, env map[string]string) *Client {
	var envSlice []string
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}
	return &Client{
		command: command,
		args:    args,
		env:     envSlice,
	}
}

// NewHTTPClient creates a new HTTP/SSE transport MCP client.
func NewHTTPClient(url string, headers map[string]string) *Client {
	return &Client{
		url:     url,
		headers: headers,
	}
}

// IsHTTP returns true if this client uses HTTP/SSE transport.
func (c *Client) IsHTTP() bool {
	return c.url != ""
}

// Initialize starts the server process and completes MCP initialization.
func (c *Client) Initialize(ctx context.Context) error {
	if c.mcp != nil {
		return nil // already initialized
	}

	var mcpClient *client.Client
	var err error

	if c.IsHTTP() {
		mcpClient, err = client.NewStreamableHttpClient(
			c.url,
			transport.WithHTTPHeaders(c.headers),
			transport.WithHTTPTimeout(30*time.Second),
		)
		if err != nil {
			return fmt.Errorf("create HTTP MCP client: %w", err)
		}
	} else {
		mcpClient, err = client.NewStdioMCPClient(c.command, c.env, c.args...)
		if err != nil {
			return fmt.Errorf("start MCP server: %w", err)
		}
	}

	c.mcp = mcpClient

	// Perform MCP initialization handshake
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "aura",
		Version: "0.1.0",
	}

	// Use bounded timeouts for transport start to prevent indefinite hangs
	startCtx, startCancel := context.WithTimeout(ctx, 10*time.Second)
	defer startCancel()
	if err := c.mcp.Start(startCtx); err != nil {
		c.mcp.Close()
		c.mcp = nil
		return fmt.Errorf("start MCP transport: %w", err)
	}

	// Use a bounded timeout for the handshake to prevent indefinite hangs
	handshakeCtx, handshakeCancel := context.WithTimeout(ctx, 10*time.Second)
	defer handshakeCancel()
	_, err = c.mcp.Initialize(handshakeCtx, initReq)
	if err != nil {
		c.mcp.Close()
		c.mcp = nil
		return fmt.Errorf("MCP initialize handshake: %w", err)
	}

	return nil
}

// ListTools discovers tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if c.mcp == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	req := mcp.ListToolsRequest{}
	resp, err := c.mcp.ListTools(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return resp.Tools, nil
}

// CallTool calls an MCP tool by name and returns the text result.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	if c.mcp == nil {
		return "", fmt.Errorf("client not initialized")
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := c.mcp.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call tool %s: %w", name, err)
	}

	if result.IsError {
		return extractTextContent(result.Content), fmt.Errorf("tool %s returned error", name)
	}

	return extractTextContent(result.Content), nil
}

// Close gracefully shuts down the MCP client and server process.
// Uses timeout to prevent blocking if MCP process is unresponsive.
func (c *Client) Close() error {
	if c.mcp == nil {
		return nil
	}
	// Use channel to wait for Close completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- c.mcp.Close()
	}()
	select {
	case err := <-done:
		c.mcp = nil
		return err
	case <-time.After(3 * time.Second):
		c.mcp = nil
		return fmt.Errorf("MCP close timeout")
	}
}

// CallToolWithTimeout calls a tool with a timeout context.
func (c *Client) CallToolWithTimeout(ctx context.Context, name string, args map[string]any, timeout time.Duration) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.CallTool(callCtx, name, args)
}

// HealthCheck checks if the client is still responsive.
func (c *Client) HealthCheck(ctx context.Context) bool {
	if c.mcp == nil {
		return false
	}
	_, err := c.mcp.ListTools(ctx, mcp.ListToolsRequest{})
	return err == nil
}

// extractTextContent extracts text from MCP content blocks.
func extractTextContent(content []mcp.Content) string {
	var result string
	for _, c := range content {
		if tc, ok := c.(mcp.TextContent); ok {
			result += tc.Text
		}
	}
	if result == "" {
		return "(empty response)"
	}
	return result
}
