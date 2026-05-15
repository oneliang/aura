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
	"regexp"
	"strings"
	"time"
)

// RipgrepResult represents a single grep match.
type RipgrepResult struct {
	File    string
	LineNum int
	Content string
}

// RipgrepSearch executes a ripgrep search.
// Returns results as []RipgrepResult.
func RipgrepSearch(ctx context.Context, pattern string, path string, includePattern string, maxLines int) ([]RipgrepResult, error) {
	// Check if rg is available
	if !HasRipgrep() {
		return nil, fmt.Errorf("ripgrep (rg) not found")
	}

	args := BuildRipgrepArgs(pattern, path, includePattern)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Exit code 1 means no matches found (not an error)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return []RipgrepResult{}, nil
			}
		}
		return nil, fmt.Errorf("ripgrep failed: %w, stderr: %s", err, stderr.String())
	}

	return parseRipgrepOutput(stdout.Bytes(), maxLines)
}

// parseRipgrepOutput parses ripgrep output into RipgrepResult slices.
// Output format: filepath:lineNum:content
func parseRipgrepOutput(output []byte, maxLines int) ([]RipgrepResult, error) {
	var results []RipgrepResult
	count := 0

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if count >= maxLines {
			break
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse format: filepath:lineNum:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		file := parts[0]
		var lineNum int
		fmt.Sscanf(parts[1], "%d", &lineNum)
		content := parts[2]

		results = append(results, RipgrepResult{
			File:    file,
			LineNum: lineNum,
			Content: content,
		})
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading output: %w", err)
	}

	return results, nil
}

// GrepSearch executes a grep search (fallback when rg is not available).
func GrepSearch(ctx context.Context, pattern string, path string, maxLines int) ([]RipgrepResult, error) {
	if !HasGrep() {
		return nil, fmt.Errorf("grep not found")
	}

	args := BuildGrepArgs(pattern, path)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "grep", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return []RipgrepResult{}, nil
			}
		}
		return nil, fmt.Errorf("grep failed: %w, stderr: %s", err, stderr.String())
	}

	return parseGrepOutput(stdout.Bytes(), maxLines)
}

// parseGrepOutput parses grep output into RipgrepResult slices.
// Output format: filepath:lineNum:content
func parseGrepOutput(output []byte, maxLines int) ([]RipgrepResult, error) {
	var results []RipgrepResult
	count := 0

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if count >= maxLines {
			break
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse format: filepath:lineNum:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		file := parts[0]
		var lineNum int
		fmt.Sscanf(parts[1], "%d", &lineNum)
		content := parts[2]

		results = append(results, RipgrepResult{
			File:    file,
			LineNum: lineNum,
			Content: content,
		})
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading output: %w", err)
	}

	return results, nil
}

// CompileRegex compiles a regex pattern.
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}

// WalkAndGrep walks a directory and searches for regex matches.
func WalkAndGrep(ctx context.Context, path string, includePattern string, re *regexp.Regexp, resultFn func(RipgrepResult) bool) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			// Skip hidden directories and common non-essential directories
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != ".git" {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" || name == "bin" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check include pattern if specified
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, filepath.Base(filePath))
			if err != nil || !matched {
				return nil
			}
		}

		// Read file and search
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil // Skip unreadable files
		}

		// Skip binary files (simple heuristic: check for null bytes)
		if bytes.Contains(content, []byte{0}) {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if re.MatchString(line) {
				result := RipgrepResult{
					File:    filePath,
					LineNum: lineNum + 1, // 1-based line numbers
					Content: line,
				}
				if !resultFn(result) {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})
}
