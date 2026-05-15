package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// GoplsClient implements Client for Go language using gopls CLI.
type GoplsClient struct {
	*BaseClient
	mu sync.Mutex
}

// NewGoplsClient creates a new gopls client.
func NewGoplsClient(rootPath string) *GoplsClient {
	goplsPath, _ := exec.LookPath(constants.LSPServerGopls)
	if goplsPath == "" {
		goplsPath = constants.LSPServerGopls // fallback to PATH lookup at runtime
	}

	// Find workspace root by looking for go.work or go.mod
	workspaceRoot := rootPath
	for {
		if _, err := os.Stat(filepath.Join(workspaceRoot, "go.work")); err == nil {
			break
		}
		if _, err := os.Stat(filepath.Join(workspaceRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(workspaceRoot)
		if parent == workspaceRoot {
			workspaceRoot = rootPath
			break
		}
		workspaceRoot = parent
	}

	return &GoplsClient{
		BaseClient: NewBaseClient(constants.LanguageGo, goplsPath, workspaceRoot),
	}
}

// IsAvailable checks if gopls is installed.
func (c *GoplsClient) IsAvailable() bool {
	_, err := exec.LookPath(c.serverPath)
	return err == nil
}

// Execute runs a gopls operation.
func (c *GoplsClient) Execute(ctx context.Context, op Operation, params Params) (*Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if params.File == "" {
		return nil, fmt.Errorf("file path is required")
	}

	switch op {
	case OpDefinition:
		return c.definition(ctx, params)
	case OpReferences:
		return c.references(ctx, params)
	case OpSymbols:
		return c.symbols(ctx, params)
	case OpFormat:
		return c.format(ctx, params)
	case OpDiagnostics:
		return c.diagnostics(ctx, params)
	case OpRename:
		return c.rename(ctx, params)
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
}

func (c *GoplsClient) definition(ctx context.Context, params Params) (*Result, error) {
	if params.Line == 0 || params.Column == 0 {
		return nil, fmt.Errorf("line and column are required for definition lookup")
	}

	pos := formatPosition(params.File, params.Line, params.Column)
	output, err := c.runGopls(ctx, "definition", pos)
	if err != nil {
		return &Result{Error: fmt.Sprintf("gopls definition failed: %v, output: %s", err, output)}, nil
	}
	return &Result{Content: formatOutput(output, "No definition found")}, nil
}

func (c *GoplsClient) references(ctx context.Context, params Params) (*Result, error) {
	if params.Line == 0 || params.Column == 0 {
		return nil, fmt.Errorf("line and column are required for references lookup")
	}

	pos := formatPosition(params.File, params.Line, params.Column)
	output, err := c.runGopls(ctx, "references", pos)
	if err != nil {
		return &Result{Error: fmt.Sprintf("gopls references failed: %v, output: %s", err, output)}, nil
	}
	return &Result{Content: formatOutput(output, "No references found")}, nil
}

func (c *GoplsClient) symbols(ctx context.Context, params Params) (*Result, error) {
	output, err := c.runGopls(ctx, "symbols", params.File)
	if err != nil {
		return &Result{Error: fmt.Sprintf("gopls symbols failed: %v, output: %s", err, output)}, nil
	}
	return &Result{Content: formatOutput(output, "No symbols found")}, nil
}

func (c *GoplsClient) format(ctx context.Context, params Params) (*Result, error) {
	cmd := exec.CommandContext(ctx, c.serverPath, "format", params.File)
	cmd.Dir = c.rootPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &Result{Error: fmt.Sprintf("gopls format failed: %v, output: %s", err, output)}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return &Result{Content: fmt.Sprintf("Code formatted successfully (%d lines)", len(lines))}, nil
}

func (c *GoplsClient) diagnostics(ctx context.Context, params Params) (*Result, error) {
	output, _ := c.runGopls(ctx, "check", params.File)
	return &Result{Content: formatOutput(output, "No errors or warnings found")}, nil
}

func (c *GoplsClient) rename(ctx context.Context, params Params) (*Result, error) {
	if params.Line == 0 || params.Column == 0 {
		return nil, fmt.Errorf("line and column are required for rename")
	}
	if params.NewName == "" {
		return nil, fmt.Errorf("newName is required for rename")
	}

	pos := formatPosition(params.File, params.Line, params.Column)
	cmd := exec.CommandContext(ctx, c.serverPath, "rename", "-w", pos, params.NewName)
	cmd.Dir = c.rootPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &Result{Error: fmt.Sprintf("gopls rename failed: %v, output: %s", err, output)}, nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return &Result{Content: fmt.Sprintf("Symbol renamed to %s", params.NewName)}, nil
	}
	return &Result{Content: result}, nil
}

// runGopls executes gopls command with given arguments.
func (c *GoplsClient) runGopls(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.serverPath, args...)
	cmd.Dir = c.rootPath
	return cmd.CombinedOutput()
}

// formatPosition creates file:line:col format for gopls.
func formatPosition(file string, line, col int) string {
	return fmt.Sprintf("%s:%d:%d", file, line, col)
}

// formatOutput formats gopls output with fallback message.
func formatOutput(output []byte, fallback string) string {
	result := strings.TrimSpace(string(output))
	if result == "" {
		return fallback
	}
	return result
}