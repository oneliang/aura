// Package internal provides internal utilities for search tools.
package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FindSearch executes a find search for glob patterns.
// Returns a list of matching file paths.
func FindSearch(ctx context.Context, pattern string, path string, maxResults int) ([]string, error) {
	if !HasFind() {
		return nil, fmt.Errorf("find command not found")
	}

	// Convert to absolute path if relative
	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		absPath = path
	}

	args := BuildFindArgs(absPath, pattern)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "find", args...)
	// Don't set cmd.Dir, let find work with absolute paths

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("find failed: %w, stderr: %s", err, stderr.String())
	}

	return parseFindOutput(stdout.Bytes(), maxResults)
}

// parseFindOutput parses find output into a slice of file paths.
func parseFindOutput(output []byte, maxResults int) ([]string, error) {
	var results []string
	count := 0

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if count >= maxResults {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		results = append(results, line)
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading output: %w", err)
	}

	// Sort by modification time is not possible from find output alone
	// The results are sorted alphabetically for consistency
	sort.Strings(results)

	return results, nil
}

// GoGlob performs glob matching using Go's filepath.Glob (fallback).
// Supports basic glob patterns but not ** recursive matching.
func GoGlob(pattern string, basePath string, maxResults int) ([]string, error) {
	// Build full pattern
	fullPattern := filepath.Join(basePath, pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob failed: %w", err)
	}

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	// Sort for consistency
	sort.Strings(matches)

	return matches, nil
}

// GoWalkGlob performs recursive glob matching using filepath.Walk (fallback for ** patterns).
func GoWalkGlob(basePath string, pattern string, maxResults int) ([]string, error) {
	var results []string
	count := 0

	// Convert glob pattern to matching function
	matchFunc := func(path string) bool {
		// Check if the file name matches the pattern
		fileName := filepath.Base(path)
		matched, err := filepath.Match(pattern, fileName)
		if err != nil {
			return false
		}
		return matched
	}

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		if count >= maxResults {
			return filepath.SkipAll
		}

		if info.IsDir() {
			// Skip hidden directories and common non-essential directories
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != ".git" {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if matchFunc(path) {
			results = append(results, path)
			count++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("filepath.Walk failed: %w", err)
	}

	sort.Strings(results)

	return results, nil
}
