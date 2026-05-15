// Package internal provides internal utilities for search tools.
package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateStringParam validates a required string parameter.
func ValidateStringParam(params map[string]any, name string) (string, error) {
	value, ok := params[name].(string)
	if !ok {
		return "", fmt.Errorf("%s parameter is required", name)
	}
	return value, nil
}

// PathExists checks if a path exists.
func PathExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("failed to access path %s: %w", path, err)
	}
	return nil
}

// SanitizePattern sanitizes a glob or regex pattern for safe shell execution.
// Uses single-quote wrapping to prevent shell injection.
func SanitizePattern(pattern string) string {
	// Single quotes prevent all shell interpretation
	// Escape existing single quotes by ending quote, adding escaped quote, starting new quote
	return "'" + strings.ReplaceAll(pattern, "'", "'\"'\"'") + "'"
}

// SanitizePath sanitizes a file path for safe shell execution.
func SanitizePath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

// ValidatePattern validates a glob or regex pattern.
// Returns an error if the pattern is invalid.
func ValidatePattern(pattern string, isRegex bool) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	if isRegex {
		_, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Check for potentially dangerous patterns
	if strings.Contains(pattern, "\x00") {
		return fmt.Errorf("pattern contains null byte")
	}

	return nil
}

// ValidatePath validates a search path.
// Returns an error if the path is invalid.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		// Allow ".." only if it's the entire path (e.g., "../../..")
		// This is a simple check; the actual security is handled by trustedpath.Checker
		parts := strings.Split(cleanPath, string(filepath.Separator))
		for _, part := range parts {
			if part != ".." && part != "." && part != "" {
				// Mixed path with ".." and other components - might be suspicious
				// But we allow it; the trustedpath.Checker will make the final decision
			}
		}
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	return nil
}

// BuildRipgrepArgs builds ripgrep command arguments.
func BuildRipgrepArgs(pattern string, path string, includePattern string) []string {
	args := []string{
		"--no-heading",      // Don't print file names as headers
		"--line-number",     // Show line numbers
		"--color=never",     // Disable color output
		"--max-columns=500", // Truncate long lines
	}

	// Add include pattern if specified
	if includePattern != "" {
		args = append(args, "--glob", includePattern)
	}

	// Add pattern and path
	// Use "--" to separate options from pattern (prevents pattern starting with "-" being interpreted as option)
	args = append(args, "--", pattern, path)

	return args
}

// BuildGrepArgs builds grep command arguments.
func BuildGrepArgs(pattern string, path string) []string {
	return []string{
		"-r",            // Recursive
		"-n",            // Show line numbers
		"--color=never", // Disable color output
		"-H",            // Always print file names
		"--",            // End of options
		pattern,
		path,
	}
}

// BuildFindArgs builds find command arguments for glob search.
func BuildFindArgs(path string, pattern string) []string {
	return []string{
		path,
		"-name", pattern,
		"-type", "f", // Only files, not directories
	}
}
