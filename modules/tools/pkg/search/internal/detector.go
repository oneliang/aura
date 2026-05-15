// Package internal provides internal utilities for search tools.
package internal

import (
	"os/exec"
	"sync"
)

// ToolDetector detects available search tools on the system.
type ToolDetector struct {
	hasRipgrep bool
	hasGrep    bool
	hasFind    bool
	once       sync.Once
}

// NewToolDetector creates a new tool detector.
func NewToolDetector() *ToolDetector {
	return &ToolDetector{}
}

// Detect detects available tools.
func (d *ToolDetector) Detect() {
	d.once.Do(func() {
		_, err := exec.LookPath("rg")
		d.hasRipgrep = (err == nil)

		_, err = exec.LookPath("grep")
		d.hasGrep = (err == nil)

		_, err = exec.LookPath("find")
		d.hasFind = (err == nil)
	})
}

// HasRipgrep checks if ripgrep is available.
func (d *ToolDetector) HasRipgrep() bool {
	d.Detect()
	return d.hasRipgrep
}

// HasGrep checks if grep is available.
func (d *ToolDetector) HasGrep() bool {
	d.Detect()
	return d.hasGrep
}

// HasFind checks if find is available.
func (d *ToolDetector) HasFind() bool {
	d.Detect()
	return d.hasFind
}

// GetBestGrep returns the best available grep tool.
func (d *ToolDetector) GetBestGrep() string {
	d.Detect()
	if d.hasRipgrep {
		return "rg"
	}
	if d.hasGrep {
		return "grep"
	}
	return "go_native"
}

// GetBestGlob returns the best available glob tool.
func (d *ToolDetector) GetBestGlob() string {
	d.Detect()
	if d.hasFind {
		return "find"
	}
	return "go_native"
}

// Global detector instance.
var defaultDetector = NewToolDetector()

// DetectTools detects available tools using the default detector.
func DetectTools() {
	defaultDetector.Detect()
}

// HasRipgrep checks if ripgrep is available using the default detector.
func HasRipgrep() bool {
	return defaultDetector.HasRipgrep()
}

// HasGrep checks if grep is available using the default detector.
func HasGrep() bool {
	return defaultDetector.HasGrep()
}

// HasFind checks if find is available using the default detector.
func HasFind() bool {
	return defaultDetector.HasFind()
}

// GetBestGrep returns the best available grep tool using the default detector.
func GetBestGrep() string {
	return defaultDetector.GetBestGrep()
}

// GetBestGlob returns the best available glob tool using the default detector.
func GetBestGlob() string {
	return defaultDetector.GetBestGlob()
}
