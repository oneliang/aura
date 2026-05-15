// Package internal provides internal utilities for filesystem tools.
package internal

import (
	"fmt"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
	"os"
)

// ValidatePath validates a path parameter and checks if it's trusted.
// Returns the validated path string or an error.
func ValidatePath(params map[string]any, checker trustedpath.Checker) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}

	if !checker.IsTrustedPath(path) {
		return "", fmt.Errorf("access denied: path %q is not in trusted directories", path)
	}

	return path, nil
}

// ValidateStringParam validates a required string parameter.
func ValidateStringParam(params map[string]any, name string) (string, error) {
	value, ok := params[name].(string)
	if !ok {
		return "", fmt.Errorf("%s parameter is required", name)
	}
	return value, nil
}

// ValidateStringParamOrDefault validates a string parameter or returns a default value.
func ValidateStringParamOrDefault(params map[string]any, name, defaultValue string) string {
	value, ok := params[name].(string)
	if !ok || value == "" {
		return defaultValue
	}
	return value
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
