// Package loader provides generic loading utilities for YAML frontmatter files.
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sharedfilepath "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"gopkg.in/yaml.v3"
)

// FileSpec specifies which files to look for in each subdirectory.
type FileSpec struct {
	FileName string // e.g., "SKILL.md", "AGENT.md"
}

// ParseFunc parses content into the target type T.
// Returns parsed metadata, body content, and error.
type ParseFunc[T any] func(content, filePath string) (T, string, error)

// Result contains the loaded item with its file path.
type Result[T any] struct {
	Item     T
	FilePath string
}

// LoadFromDirectories loads items from multiple base directories.
// 这是纯函数，不维护状态，符合 Layer 1 的无状态原则。
func LoadFromDirectories[T any](
	baseDirs []string,
	fileSpec FileSpec,
	parseFn ParseFunc[T],
) ([]Result[T], error) {
	var results []Result[T]

	for _, baseDir := range baseDirs {
		expandedDir := sharedfilepath.ExpandTilde(baseDir)

		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(expandedDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", expandedDir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			filePath := filepath.Join(expandedDir, entry.Name(), fileSpec.FileName)

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				continue
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			item, _, err := parseFn(string(content), filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
			}

			results = append(results, Result[T]{
				Item:     item,
				FilePath: filePath,
			})
		}
	}

	return results, nil
}

// ParseYAMLFrontmatter is a helper for parsing YAML frontmatter.
// 返回 YAML 内容、body 内容。
func ParseYAMLFrontmatter(content string) (yamlContent, body string, err error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("missing YAML frontmatter")
	}

	endIdx := strings.Index(content[4:], "---")
	if endIdx == -1 {
		return "", "", fmt.Errorf("unclosed YAML frontmatter")
	}
	endIdx += 4

	yamlContent = content[4:endIdx]
	body = strings.TrimSpace(content[endIdx+3:])
	return yamlContent, body, nil
}

// UnmarshalYAMLFrontmatter parses YAML frontmatter into target struct.
func UnmarshalYAMLFrontmatter[T any](content string, target *T) (body string, err error) {
	yamlContent, body, err := ParseYAMLFrontmatter(content)
	if err != nil {
		return "", err
	}

	if err := yaml.Unmarshal([]byte(yamlContent), target); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	return body, nil
}
