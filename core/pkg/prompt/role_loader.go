// Package prompt provides system prompt building utilities.
package prompt

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// RoleLoader loads role-based system prompts from files.
type RoleLoader struct {
	baseDir string
}

// NewRoleLoader creates a new role loader.
// If baseDir is empty, defaults to ~/.aura/roles/
func NewRoleLoader(baseDir string) *RoleLoader {
	if baseDir == "" {
		baseDir = ffp.MustAuraHomePath(constants.DirRoles)
	}
	return &RoleLoader{
		baseDir: baseDir,
	}
}

// Load loads a role prompt from a file.
// Returns empty string if role is empty, file doesn't exist, or file is empty.
func (l *RoleLoader) Load(role string) string {
	if role == "" {
		return ""
	}

	rolePath := filepath.Join(l.baseDir, role+".md")
	content, err := os.ReadFile(rolePath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(content))
}

// Available returns true if a role file exists.
func (l *RoleLoader) Available(role string) bool {
	if role == "" {
		return false
	}
	rolePath := filepath.Join(l.baseDir, role+".md")
	_, err := os.Stat(rolePath)
	return err == nil
}

// List returns all available role names.
func (l *RoleLoader) List() ([]string, error) {
	entries, err := os.ReadDir(l.baseDir)
	if err != nil {
		return nil, err
	}

	var roles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			role := strings.TrimSuffix(entry.Name(), ".md")
			roles = append(roles, role)
		}
	}
	return roles, nil
}
