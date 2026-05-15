// Package logdir provides unified log directory management for aura.
package logdir

import (
	"path/filepath"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// GetLogDir returns the path to the logs directory (~/.aura/logs/).
// It ensures the directory exists by creating it if necessary.
func GetLogDir() (string, error) {
	logDir := ffp.MustAuraHomePath("logs")

	// Ensure directory exists
	if err := ffp.EnsureDir(logDir); err != nil {
		return "", err
	}

	return logDir, nil
}

// GetLogFile returns the full path for a log file within the logs directory.
// The filename should be a simple name like "llm_requests.log" or "tui_debug.log".
// It ensures the logs directory exists.
func GetLogFile(filename string) (string, error) {
	logDir, err := GetLogDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(logDir, filename), nil
}
