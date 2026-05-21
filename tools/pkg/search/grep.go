// Package search provides file search tools (glob and grep).
package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/search/internal"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

const (
	// DefaultMaxGrepResults limits the maximum number of grep results.
	DefaultMaxGrepResults = 1000
)

// GrepTool performs regex content search in files.
type GrepTool struct {
	checker     trustedpath.Checker
	maxLines    int
	fileTracker *internal.ToolDetector
}

// NewGrepTool creates a new GrepTool.
func NewGrepTool(checker trustedpath.Checker) *GrepTool {
	if checker == nil {
		checker = trustedpath.NopChecker()
	}
	return &GrepTool{
		checker:     checker,
		maxLines:    DefaultMaxGrepResults,
		fileTracker: internal.NewToolDetector(),
	}
}

// Name returns the tool name.
func (t *GrepTool) Name() string {
	return constants.ToolGrep
}

// Description returns the tool description.
func (t *GrepTool) Description() string {
	return "Search for text patterns in files using regular expressions. Parameters: pattern (string, required, regex), path (string, optional, default '.'), include (string, optional, glob pattern for file filtering). " + t.getBackendInfo()
}

// OutputSchema returns the JSON schema for structured output.
func (t *GrepTool) OutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern":  map[string]any{"type": "string"},
			"path":     map[string]any{"type": "string"},
			"count":    map[string]any{"type": "integer"},
			"matches": map[string]any{
				"type":  "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file":     map[string]any{"type": "string"},
						"line_num": map[string]any{"type": "integer"},
						"content":  map[string]any{"type": "string"},
					},
				},
			},
			"backend": map[string]any{"type": "string"},
		},
	}
}

// getBackendInfo returns information about the backend being used.
func (t *GrepTool) getBackendInfo() string {
	backend := t.fileTracker.GetBestGrep()
	switch backend {
	case "rg":
		return "(using ripgrep)"
	case "grep":
		return "(using grep)"
	default:
		return "(using Go regexp)"
	}
}

// PermissionLevel returns the permission level for this tool.
func (t *GrepTool) PermissionLevel() string {
	return "read"
}

// RequiresConfirmation returns false as grep is a read-only operation.
func (t *GrepTool) RequiresConfirmation() bool {
	return false
}

// Execute performs regex search in files.
func (t *GrepTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	// Get pattern parameter
	pattern, err := internal.ValidateStringParam(params, "pattern")
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("pattern parameter is required: %v", err)}, nil
	}

	// Validate regex pattern
	if err := internal.ValidatePattern(pattern, true); err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf("invalid regex pattern: %v", err)}, nil
	}

	// Get path parameter (optional, default ".")
	path, ok := params["path"].(string)
	if !ok || path == "" {
		path = "."
	}

	// Get include parameter (optional, for file filtering)
	includePattern, _ := params["include"].(string)

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
	var results []internal.RipgrepResult
	backend := t.fileTracker.GetBestGrep()

	switch backend {
	case "rg":
		results, err = internal.RipgrepSearch(ctx, pattern, path, includePattern, t.maxLines)
	case "grep":
		results, err = internal.GrepSearch(ctx, pattern, path, t.maxLines)
	default:
		// Go native fallback
		results, err = t.goNativeSearch(ctx, pattern, path, includePattern, t.maxLines)
	}

	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("grep search failed: %v", err),
			Data: map[string]any{
				"pattern": pattern,
				"path":    path,
				"count":   0,
				"matches": []string{},
				"error":   err.Error(),
			},
		}, nil
	}

	duration := time.Since(startTime)

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d match(es) for pattern '%s' in %s (backend: %s, %v):\n",
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
				"matches": []map[string]any{},
				"backend": backend,
			},
		}, nil
	}

	// Group results by file for better readability
	fileResults := make(map[string][]internal.RipgrepResult)
	for _, r := range results {
		fileResults[r.File] = append(fileResults[r.File], r)
	}

	for file, matches := range fileResults {
		output.WriteString(fmt.Sprintf("\n%s:\n", file))
		for _, m := range matches {
			output.WriteString(fmt.Sprintf("  %4d: %s\n", m.LineNum, m.Content))
		}
	}

	// Build structured matches data
	structuredMatches := make([]map[string]any, len(results))
	for i, r := range results {
		structuredMatches[i] = map[string]any{
			"file":     r.File,
			"line_num": r.LineNum,
			"content":  r.Content,
		}
	}

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: output.String(),
		Data: map[string]any{
			"pattern": pattern,
			"path":    path,
			"count":   len(results),
			"matches": structuredMatches,
			"backend": backend,
		},
	}, nil
}

// goNativeSearch performs grep search using Go's regexp package (fallback).
func (t *GrepTool) goNativeSearch(ctx context.Context, pattern string, path string, includePattern string, maxLines int) ([]internal.RipgrepResult, error) {
	var results []internal.RipgrepResult
	count := 0

	re, err := internal.CompileRegex(pattern)
	if err != nil {
		return nil, err
	}

	err = internal.WalkAndGrep(ctx, path, includePattern, re, func(result internal.RipgrepResult) bool {
		results = append(results, result)
		count++
		return count < maxLines
	})

	return results, err
}
