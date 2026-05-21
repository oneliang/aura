// Package lsp provides LSP-based code intelligence tools.
package lsp

import (
	"context"
	"fmt"
	"sync"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/lsp/client"
	"github.com/oneliang/aura/tools/pkg/lsp/manager"
)

// CodeTool provides multi-language code intelligence using LSP.
type CodeTool struct {
	manager *manager.Manager
	mu      sync.Mutex
}

// NewCodeTool creates a new code tool.
func NewCodeTool(rootPath string) *CodeTool {
	return &CodeTool{
		manager: manager.NewManager(rootPath),
	}
}

// Name returns the tool name.
func (t *CodeTool) Name() string {
	return constants.ToolCodeNavigate
}

// Description returns the tool description.
func (t *CodeTool) Description() string {
	langs := t.manager.AvailableLanguages()
	supported := "go"
	if len(langs) > 0 {
		supported = fmt.Sprintf("%v", langs)
	}
	return fmt.Sprintf("Navigate and analyze code using LSP. Supported languages: %s. Operations: definition, references, symbols, format, diagnostics, rename. Requires file path and optionally line/column.", supported)
}

// Execute executes the tool with given parameters.
func (t *CodeTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, file, line, col, newName := t.extractParams(params)
	if err := t.validateParams(operation, file); err != nil {
		return err, nil
	}

	return t.executeWithClient(ctx, operation, file, line, col, newName)
}

// extractParams extracts parameters from the input map.
func (t *CodeTool) extractParams(params map[string]any) (string, string, int, int, string) {
	operation, _ := params["operation"].(string)
	file, _ := params["file"].(string)
	line, _ := params["line"].(float64)
	col, _ := params["column"].(float64)
	newName, _ := params["newName"].(string)
	return operation, file, int(line), int(col), newName
}

// validateParams validates required parameters.
func (t *CodeTool) validateParams(operation, file string) *tools.ToolResult {
	if operation == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "operation is required"}
	}
	if file == "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: "file path is required"}
	}
	return nil
}

// executeWithClient gets the client and executes the operation.
func (t *CodeTool) executeWithClient(ctx context.Context, operation, file string, line, col int, newName string) (*tools.ToolResult, error) {
	c, err := t.manager.GetClientForFile(file)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}

	if !c.IsAvailable() {
		serverName := t.manager.Detector().GetServerForLanguage(c.Language())
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("LSP server '%s' is not installed", serverName),
		}, nil
	}

	clientParams := client.Params{File: file, Line: line, Column: col, NewName: newName}
	if err := t.validateOperationParams(operation, clientParams); err != nil {
		return err, nil
	}

	result, err := c.Execute(ctx, client.Operation(operation), clientParams)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}

	if result.Error != "" {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: result.Error}, nil
	}

	// Build structured data for engine consumption
	data := map[string]any{
		"operation": operation,
		"file":      file,
		"content":   result.Content,
	}
	if line > 0 {
		data["line"] = line
	}
	if col > 0 {
		data["column"] = col
	}
	if newName != "" {
		data["renamed_to"] = newName
	}

	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: result.Content, Data: data}, nil
}

// validateOperationParams validates operation-specific parameters.
func (t *CodeTool) validateOperationParams(operation string, params client.Params) *tools.ToolResult {
	switch client.Operation(operation) {
	case client.OpDefinition, client.OpReferences:
		if params.Line == 0 || params.Column == 0 {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: "line and column are required"}
		}
	case client.OpRename:
		if params.Line == 0 || params.Column == 0 || params.NewName == "" {
			return &tools.ToolResult{Status: tools.ToolStatusError, Error: "line, column, and newName are required"}
		}
	}
	return nil
}

// Manager returns the underlying LSP manager (for testing).
func (t *CodeTool) Manager() *manager.Manager {
	return t.manager
}

// OutputSchema returns the JSON schema for structured output.
// Only includes fields that are actually populated in Execute() Data field.
func (t *CodeTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation":  map[string]any{"type": "string", "enum": []string{"definition", "references", "symbols", "format", "diagnostics", "rename"}},
			"file":       map[string]any{"type": "string"},
			"line":       map[string]any{"type": "integer"},
			"column":     map[string]any{"type": "integer"},
			"content":    map[string]any{"type": "string"},
			"renamed_to": map[string]any{"type": "string"},
		},
	}
}