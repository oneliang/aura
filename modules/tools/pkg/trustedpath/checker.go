// Package trustedpath provides trusted path checking interface for tools.
package trustedpath

import (
	"github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// Checker defines the interface for checking trusted paths.
// This interface allows tools to check if a path is within trusted directories
// without depending on core/permissions package.
type Checker interface {
	// IsTrustedPath checks if a path is within any trusted directory.
	// Returns true if the path is trusted, false otherwise.
	IsTrustedPath(path string) bool
}

// nopChecker is a no-op checker that always returns true.
// Used for backward compatibility when no checker is provided.
type nopChecker struct{}

// IsTrustedPath always returns true for nopChecker.
func (nopChecker) IsTrustedPath(path string) bool {
	return true
}

// NopChecker returns a checker that always returns true.
// This is used for backward compatibility and testing.
func NopChecker() Checker {
	return nopChecker{}
}

// listChecker is a checker that checks against a list of trusted directories.
type listChecker struct {
	trustedDirs []string
}

// IsTrustedPath checks if a path is within any trusted directory.
func (c *listChecker) IsTrustedPath(path string) bool {
	// Use shared utility function for consistent behavior
	return filepath.IsPathInDirsOrDefault(path, c.trustedDirs)
}

// NewChecker creates a new checker with the given trusted directories.
func NewChecker(trustedDirs []string) Checker {
	return &listChecker{trustedDirs: trustedDirs}
}
