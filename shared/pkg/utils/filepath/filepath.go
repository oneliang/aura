// Package filepath provides common path utility functions.
package filepath

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// GetHomeDir returns the user's home directory.
// Returns "." if the home directory cannot be determined.
func GetHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return homeDir
}

// EnsureDir creates a directory and all its parent directories if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ExpandTilde expands a leading tilde in the path to the home directory.
// Returns the path unchanged if it doesn't start with ~.
func ExpandTilde(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	homeDir := GetHomeDir()
	if len(path) == 1 {
		return homeDir
	}

	// Handle ~/ and ~user/ patterns
	if path[1] == '/' || path[1] == filepath.Separator {
		return filepath.Join(homeDir, path[2:])
	}

	return path
}

// JoinSafe joins multiple path segments, ignoring empty segments.
func JoinSafe(segments ...string) string {
	var nonEmpty []string
	for _, s := range segments {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return filepath.Join(nonEmpty...)
}

// AbsRelativeToBase returns the absolute path relative to a base directory.
// If base is empty, uses the current working directory.
func AbsRelativeToBase(base, rel string) (string, error) {
	if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return filepath.Abs(filepath.Join(base, rel))
}

// IsSubPath checks if child is a subpath of parent.
// Returns true if child is equal to or under parent.
// Returns false for empty paths.
func IsSubPath(parent, child string) bool {
	// Empty paths are not valid
	if parent == "" || child == "" {
		return false
	}

	parent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	child, err = filepath.Abs(child)
	if err != nil {
		return false
	}

	// Use filepath.Rel for accurate subpath checking
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	// If rel doesn't start with "..", child is within parent
	// "." means same path, which is considered a subpath
	return !strings.HasPrefix(rel, "..")
}

// IsPathInDirs checks if a path is within any of the given directories.
// Returns true if the path equals to or is a subpath of any directory in dirs.
func IsPathInDirs(path string, dirs []string) bool {
	// If no dirs configured, return false (deny by default)
	if len(dirs) == 0 {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, dir := range dirs {
		if IsSubPath(dir, absPath) {
			return true
		}
	}

	return false
}

// IsPathInDirsOrDefault checks if a path is within any of the given directories.
// If dirs is empty, returns true (allow all, for backward compatibility).
func IsPathInDirsOrDefault(path string, dirs []string) bool {
	// If no dirs configured, allow all (backward compatibility)
	if len(dirs) == 0 {
		return true
	}
	return IsPathInDirs(path, dirs)
}

// AuraHomePath returns a path relative to ~/.aura.
// Subpath elements are joined with the home directory.
func AuraHomePath(subpath ...string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	parts := append([]string{homeDir, constants.DefaultHomeDir}, subpath...)
	return filepath.Join(parts...), nil
}

// MustAuraHomePath is like AuraHomePath but panics on error.
// Use only in initialization code where home directory is expected to exist.
func MustAuraHomePath(subpath ...string) string {
	path, err := AuraHomePath(subpath...)
	if err != nil {
		panic(err)
	}
	return path
}

// AuraHomePathOrDefault returns a path relative to ~/.aura.
// If the home directory cannot be determined, returns an empty string.
// Use this when you want to gracefully handle missing home directories.
func AuraHomePathOrDefault(subpath ...string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	parts := append([]string{homeDir, constants.DefaultHomeDir}, subpath...)
	return filepath.Join(parts...)
}

// EnsureDirWithMode creates a directory and all its parent directories with custom permissions.
func EnsureDirWithMode(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
