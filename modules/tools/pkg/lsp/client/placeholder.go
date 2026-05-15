package client

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// PlaceholderClient is a placeholder for unimplemented LSP clients.
// It returns errors for all operations.
type PlaceholderClient struct {
	*BaseClient
}

// NewPlaceholderClient creates a placeholder client for a language.
func NewPlaceholderClient(language, serverPath, rootPath string) *PlaceholderClient {
	return &PlaceholderClient{
		BaseClient: NewBaseClient(language, serverPath, rootPath),
	}
}

// IsAvailable always returns false for placeholder clients.
func (c *PlaceholderClient) IsAvailable() bool {
	return false
}

// Execute returns an error indicating the client is not implemented.
func (c *PlaceholderClient) Execute(ctx context.Context, op Operation, params Params) (*Result, error) {
	return nil, fmt.Errorf("LSP client for %s is not yet implemented", c.language)
}

// NewRustAnalyzerClient creates a placeholder Rust client (to be implemented).
func NewRustAnalyzerClient(rootPath string) *PlaceholderClient {
	return NewPlaceholderClient(constants.LanguageRust, constants.LSPServerRustAnalyzer, rootPath)
}

// NewTSServerClient creates a placeholder TypeScript client (to be implemented).
func NewTSServerClient(rootPath string) *PlaceholderClient {
	return NewPlaceholderClient(constants.LanguageTypeScript, constants.LSPServerTSServer, rootPath)
}

// NewPylspClient creates a placeholder Python client (to be implemented).
func NewPylspClient(rootPath string) *PlaceholderClient {
	return NewPlaceholderClient(constants.LanguagePython, constants.LSPServerPylsp, rootPath)
}