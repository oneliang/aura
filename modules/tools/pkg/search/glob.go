// Package search provides file search tools (glob and grep).
package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/search/internal"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

const (
	// DefaultMaxGlobResults limits the maximum number of glob results.
	DefaultMaxGlobResults = 1000
)

// GlobTool performs file glob pattern matching.
type GlobTool struct {
	checker     trustedpath.Checker
	maxResults  int
	fileTracker *internal.ToolDetector
}

// NewGlobTool creates a new GlobTool.
func NewGlobTool(checker trustedpath.Checker) *GlobTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &GlobTool{
		checker:     checker,
		maxResults:  DefaultMaxGlobResults,
		fileTracker: internal.NewToolDetector(),
	}
}

// Name returns the tool name.
func (t *GlobTool) Name() string {
	return constants.ToolGlob
}

// Description returns the tool description.
func (t *GlobTool) Description() string {
	return "Fast file pattern matching using glob patterns. Parameters: pattern (string, required, e.g., '*.go' or '**/*.py'), path (string, optional, default '.') " + t.getBackendInfo()
}

// OutputSchema returns the JSON schema for structured output.
func (t *GlobTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string"},
			"path":    map[string]any{"type": "string"},
			"count":   map[string]any{"type": "integer"},
			"files":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"backend": map[string]any{"type": "string"},
		},
	}
}

// getBackendInfo returns information about the backend being used.
func (t *GlobTool) getBackendInfo() string {
	backend := t.fileTracker.GetBestGlob()
	if backend == "find" {
		return "(using find command)"
	}
	return "(using Go filepath.Glob)"
}

// PermissionLevel returns the permission level for this tool.
func (t *GlobTool) PermissionLevel() string {
	return "read"
}

// RequiresConfirmation returns false as glob is a read-only operation.
func (t *GlobTool) RequiresConfirmation() bool {
	return false
}

// Execute performs glob pattern matching.
func (t *GlobTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Get pattern parameter
	pattern, err := internal.ValidateStringParam(params, "pattern")
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("pattern parameter is required: %v", err)}, nil
	}

	// Validate pattern
	if err := internal.ValidatePattern(pattern, false); err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid pattern: %v", err)}, nil
	}

	// Get path parameter (optional, default ".")
	path, ok := params["path"].(string)
	if !ok || path == "" {
		path = "."
	}

	// Validate path
	if err := internal.ValidatePath(path); err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	// Check if path is trusted
	if !t.checker.IsTrustedPath(path) {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("path not in trusted directories: %s", path)}, nil
	}

	// Ensure path exists
	if err := internal.PathExists(path); err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}

	// Execute search based on available backend
	startTime := time.Now()
	var results []string
	backend := t.fileTracker.GetBestGlob()

	switch backend {
	case "find":
		results, err = internal.FindSearch(ctx, pattern, path, t.maxResults)
	default:
		// Go native fallback
		// Handle ** (recursive) patterns specially
		if strings.Contains(pattern, "**") {
			results, err = internal.GoWalkGlob(path, strings.ReplaceAll(pattern, "**/", ""), t.maxResults)
		} else {
			results, err = internal.GoGlob(pattern, path, t.maxResults)
		}
	}

	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("glob search failed: %v", err),
			Data: map[string]any{
				"pattern": pattern,
				"path":    path,
				"count":   0,
				"files":   []string{},
				"error":   err.Error(),
			},
		}, nil
	}

	// Sort results by modification time would be ideal, but requires additional stat calls
	// For now, sort alphabetically for consistent output
	sort.Strings(results)

	duration := time.Since(startTime)

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s) matching '%s' in %s (backend: %s, %v):\n",
		len(results), pattern, path, backend, duration.Round(time.Millisecond)))

	if len(results) == 0 {
		output.WriteString("No matches found.")
		return &tools.ToolResult{
			Status:  tools.ToolStatusSuccess,
			Content: output.String(),
			Data: map[string]any{
				"pattern": pattern,
				"path":    path,
				"count":   0,
				"files":   []string{},
				"backend": backend,
			},
		}, nil
	}

	for i, r := range results {
		if i >= t.maxResults {
			output.WriteString(fmt.Sprintf("\n... and more (limited to %d results)", t.maxResults))
			break
		}
		output.WriteString(fmt.Sprintf("  - %s\n", r))
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: output.String(),
		Data: map[string]any{
			"pattern": pattern,
			"path":    path,
			"count":   len(results),
			"files":   results,
			"backend": backend,
		},
	}, nil
}
