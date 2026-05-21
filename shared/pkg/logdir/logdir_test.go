// Package logdir provides tests for the logdir package.
package logdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetLogDir tests GetLogDir function.
func TestGetLogDir(t *testing.T) {
	logDir, err := GetLogDir()
	if err != nil {
		t.Fatalf("GetLogDir() error = %v", err)
	}
	if logDir == "" {
		t.Fatal("GetLogDir() returned empty string")
	}

	// Verify it ends with .aura/logs
	if !strings.HasSuffix(logDir, ".aura/logs") {
		t.Errorf("GetLogDir() = %q, should end with .aura/logs", logDir)
	}

	// Verify directory exists
	info, err := os.Stat(logDir)
	if err != nil {
		t.Errorf("Log directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Log path should be a directory")
	}
}

// TestGetLogFile tests GetLogFile function.
func TestGetLogFile(t *testing.T) {
	filename := "test.log"
	logFile, err := GetLogFile(filename)
	if err != nil {
		t.Fatalf("GetLogFile() error = %v", err)
	}
	if logFile == "" {
		t.Fatal("GetLogFile() returned empty string")
	}

	// Verify it contains the filename
	if !strings.HasSuffix(logFile, filename) {
		t.Errorf("GetLogFile() = %q, should end with %q", logFile, filename)
	}

	// Verify parent directory exists
	dir := filepath.Dir(logFile)
	info, err := os.Stat(dir)
	if err != nil {
		t.Errorf("Log directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Log path should be a directory")
	}
}

// TestGetLogFile_WithSubDir tests GetLogFile with subdirectory.
func TestGetLogFile_WithSubDir(t *testing.T) {
	filename := "subdir/test.log"
	logFile, err := GetLogFile(filename)
	if err != nil {
		t.Fatalf("GetLogFile() error = %v", err)
	}

	// Verify it contains the filename
	if !strings.HasSuffix(logFile, filename) {
		t.Errorf("GetLogFile() = %q, should end with %q", logFile, filename)
	}
}

// TestGetLogFile_EmptyFilename tests GetLogFile with empty filename.
func TestGetLogFile_EmptyFilename(t *testing.T) {
	logFile, err := GetLogFile("")
	if err != nil {
		t.Fatalf("GetLogFile() error = %v", err)
	}

	// Should still return a valid path
	if logFile == "" {
		t.Error("GetLogFile() should return a path even with empty filename")
	}
}
