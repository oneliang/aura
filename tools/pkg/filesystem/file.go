// Package filesystem provides file system tools.
package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/utils"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/filesystem/internal"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

// ReadTool reads file contents.
type ReadTool struct {
	checker trustedpath.Checker
}

// NewReadTool creates a new ReadTool with optional trusted path checker.
// If checker is nil, uses NopChecker which allows all paths.
func NewReadTool(checker trustedpath.Checker) *ReadTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &ReadTool{checker: checker}
}

// Name returns the tool name.
func (t *ReadTool) Name() string {
	return constants.ToolFileRead
}

// Description returns the tool description.
func (t *ReadTool) Description() string {
	return "Read the contents of a file. Parameters: path (string, required)"
}

// OutputSchema returns the JSON schema for structured output.
func (t *ReadTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string"},
			"bytes":  map[string]any{"type": "integer"},
			"is_image": map[string]any{"type": "boolean"},
		},
	}
}

// PermissionLevel returns the permission level for this tool.
// file_read is a read-only operation, matching the default config ControlAllow.
func (t *ReadTool) PermissionLevel() string {
	return "read"
}

// RequiresConfirmation returns true if the path is not in trusted directories.
func (t *ReadTool) RequiresConfirmation() bool {
	// We can't check here without params, so we return false
	// The actual check happens in Execute via ValidatePathWithConfirm
	return false
}

// Execute reads a file.
func (t *ReadTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Get path - trusted check is handled by permission manager before Execute is called
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to read file: %v", err),
			Data: map[string]any{
				"path":     path,
				"bytes":    0,
				"is_image": false,
				"error":    err.Error(),
			},
		}, nil
	}

	// Check if it's an image file
	if utils.IsImageFile(path) {
		// Convert to dataURI for images
		dataURI, err := utils.ImageToDataURI(path, content)
		if err != nil {
			return &tools.ToolResult{
				Status: tools.ToolStatusError,
				Error:  fmt.Sprintf("failed to convert image to dataURI: %v", err),
				Data: map[string]any{
					"path":     path,
					"bytes":    len(content),
					"is_image": true,
					"error":    err.Error(),
				},
			}, nil
		}
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: fmt.Sprintf("Image file: %s\nDataURI: %s", path, dataURI),
			Data: map[string]any{
				"path":     path,
				"bytes":    len(content),
				"is_image": true,
			},
		}, nil
	}

	// Return byte count for non-image files to avoid cluttering the terminal
	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf("Read %d bytes from %s", len(content), path),
		Data: map[string]any{
			"path":     path,
			"bytes":    len(content),
			"is_image": false,
		},
	}, nil
}

// WriteTool writes content to a file.
type WriteTool struct {
	checker trustedpath.Checker
}

// NewWriteTool creates a new WriteTool with optional trusted path checker.
// If checker is nil, uses NopChecker which allows all paths.
func NewWriteTool(checker trustedpath.Checker) *WriteTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &WriteTool{checker: checker}
}

// Name returns the tool name.
func (t *WriteTool) Name() string {
	return constants.ToolFileWrite
}

// Description returns the tool description.
func (t *WriteTool) Description() string {
	return "Write content to a file. Parameters: path (string, required), content (string, required)"
}

// PermissionLevel returns the permission level for this tool.
func (t *WriteTool) PermissionLevel() string {
	return "write"
}

// RequiresConfirmation returns true because writing files is a sensitive operation.
func (t *WriteTool) RequiresConfirmation() bool {
	return true
}

// Execute writes a file.
func (t *WriteTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	path, err := internal.ValidatePath(params, t.checker)
	if err != nil {
		return nil, err
	}

	content, err := internal.ValidateStringParam(params, "content")
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to create directory: %v", err),
			Data: map[string]any{
				"path":          path,
				"bytes_written": 0,
				"success":       false,
				"error":         err.Error(),
			},
		}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to write file: %v", err),
			Data: map[string]any{
				"path":          path,
				"bytes_written": 0,
				"success":       false,
				"error":         err.Error(),
			},
		}, nil
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf("Successfully wrote to %s", path),
		Data: map[string]any{
			"path":        path,
			"bytes_written": len(content),
			"success":     true,
		},
	}, nil
}

// OutputSchema returns the JSON schema for structured output.
func (t *WriteTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":          map[string]any{"type": "string"},
			"bytes_written": map[string]any{"type": "integer"},
			"success":       map[string]any{"type": "boolean"},
		},
	}
}

// SearchTool searches for content in files.
type SearchTool struct {
	checker trustedpath.Checker
}

// NewSearchTool creates a new SearchTool with optional trusted path checker.
// If checker is nil, uses NopChecker which allows all paths.
func NewSearchTool(checker trustedpath.Checker) *SearchTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &SearchTool{checker: checker}
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return constants.ToolFileSearch
}

// Description returns the tool description.
func (t *SearchTool) Description() string {
	return "Search for content in files. Parameters: path (string, required), pattern (string, required)"
}

// OutputSchema returns the JSON schema for structured output.
func (t *SearchTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string"},
			"path":    map[string]any{"type": "string"},
			"count":   map[string]any{"type": "integer"},
			"files":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
}

// Execute searches files.
func (t *SearchTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	path, err := internal.ValidatePath(params, t.checker)
	if err != nil {
		return nil, err
	}

	pattern, err := internal.ValidateStringParam(params, "pattern")
	if err != nil {
		return nil, err
	}

	var results []string

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}

		if strings.Contains(string(content), pattern) {
			results = append(results, filePath)
		}
		return nil
	})

	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("search failed: %v", err),
			Data: map[string]any{
				"path":    path,
				"pattern": pattern,
				"count":   0,
				"files":   []string{},
				"error":   err.Error(),
			},
		}, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d files containing '%s':\n", len(results), pattern))
	for _, r := range results {
		output.WriteString(fmt.Sprintf("- %s\n", r))
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: output.String(),
		Data: map[string]any{
			"pattern": pattern,
			"count":   len(results),
			"files":   results,
		},
	}, nil
}

// ListTool lists directory contents.
type ListTool struct {
	checker trustedpath.Checker
}

// NewListTool creates a new ListTool with optional trusted path checker.
// If checker is nil, uses NopChecker which allows all paths.
func NewListTool(checker trustedpath.Checker) *ListTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &ListTool{checker: checker}
}

// Name returns the tool name.
func (t *ListTool) Name() string {
	return constants.ToolFileList
}

// Description returns the tool description.
func (t *ListTool) Description() string {
	return "List directory contents. Parameters: path (string, required)"
}

// OutputSchema returns the JSON schema for structured output.
func (t *ListTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":       map[string]any{"type": "string"},
			"file_count": map[string]any{"type": "integer"},
			"dir_count":  map[string]any{"type": "integer"},
		},
	}
}

// Execute lists a directory.
func (t *ListTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	path, err := internal.ValidatePath(params, t.checker)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to read directory: %v", err),
			Data: map[string]any{
				"path":       path,
				"file_count": 0,
				"dir_count":  0,
				"error":      err.Error(),
			},
		}, nil
	}

	fileCount := 0
	dirCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			dirCount++
		} else {
			fileCount++
		}
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: fmt.Sprintf("Found %d files and %d directories in %s", fileCount, dirCount, path),
		Data: map[string]any{
			"path":       path,
			"file_count": fileCount,
			"dir_count":  dirCount,
		},
	}, nil
}

// AllTools returns all filesystem tools with no-op checker (for backward compatibility).
func AllTools() []interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error)
} {
	checker := trustedpath.NopChecker()
	return []interface {
		Name() string
		Description() string
		Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error)
	}{
		NewReadTool(checker),
		NewWriteTool(checker),
		NewSearchTool(checker),
		NewListTool(checker),
	}
}
