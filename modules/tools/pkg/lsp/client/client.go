package client

import (
	"context"
)

// Operation represents LSP operation types.
type Operation string

// LSP operation constants.
const (
	OpDefinition  Operation = "definition"
	OpReferences  Operation = "references"
	OpSymbols     Operation = "symbols"
	OpFormat      Operation = "format"
	OpDiagnostics Operation = "diagnostics"
	OpRename      Operation = "rename"
)

// Params holds operation parameters.
type Params struct {
	File    string
	Line    int
	Column  int
	NewName string // for rename operation
}

// Result holds operation result.
type Result struct {
	Content string
	Error   string
}

// Client defines the LSP client interface.
// Each language-specific client implements this interface.
type Client interface {
	// Language returns the language ID (e.g., "go", "rust", "typescript").
	Language() string

	// IsAvailable checks if the LSP server is installed on the system.
	IsAvailable() bool

	// Execute runs an LSP operation on the given file.
	Execute(ctx context.Context, op Operation, params Params) (*Result, error)
}

// BaseClient provides common functionality for LSP clients.
type BaseClient struct {
	language   string
	serverPath string
	rootPath   string
}

// NewBaseClient creates a base client with common fields.
func NewBaseClient(language, serverPath, rootPath string) *BaseClient {
	return &BaseClient{
		language:   language,
		serverPath: serverPath,
		rootPath:   rootPath,
	}
}

// Language returns the language ID.
func (c *BaseClient) Language() string {
	return c.language
}

// ServerPath returns the LSP server command path.
func (c *BaseClient) ServerPath() string {
	return c.serverPath
}

// RootPath returns the workspace root path.
func (c *BaseClient) RootPath() string {
	return c.rootPath
}