// Package constants provides tests for the constants package.
package constants

import (
	"strings"
	"testing"
	"time"
)

// TestVersionConstants tests version constants.
func TestVersionConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if BuildCommit == "" {
		t.Error("BuildCommit should not be empty")
	}
}

// TestDefaultHomeDir tests DefaultHomeDir constant.
func TestDefaultHomeDir(t *testing.T) {
	if DefaultHomeDir == "" {
		t.Error("DefaultHomeDir should not be empty")
	}
	if !strings.HasPrefix(DefaultHomeDir, ".") {
		t.Errorf("DefaultHomeDir = %q, should start with '.'", DefaultHomeDir)
	}
}

// TestDefaultConfigFile tests DefaultConfigFile constant.
func TestDefaultConfigFile(t *testing.T) {
	if DefaultConfigFile == "" {
		t.Error("DefaultConfigFile should not be empty")
	}
	if !strings.HasSuffix(DefaultConfigFile, ".yaml") {
		t.Errorf("DefaultConfigFile = %q, should end with '.yaml'", DefaultConfigFile)
	}
}

// TestDefaultProfileFile tests DefaultProfileFile constant.
func TestDefaultProfileFile(t *testing.T) {
	if DefaultProfileFile == "" {
		t.Error("DefaultProfileFile should not be empty")
	}
	if !strings.HasSuffix(DefaultProfileFile, ".yaml") {
		t.Errorf("DefaultProfileFile = %q, should end with '.yaml'", DefaultProfileFile)
	}
}

// TestDirectoryConstants tests directory constants.
func TestDirectoryConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"DirKnowledge", DirKnowledge},
		{"DirSessions", DirSessions},
		{"DirRoles", DirRoles},
		{"DirSkills", DirSkills},
		{"DirMemory", DirMemory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

// TestToolConstants tests tool constants.
func TestToolConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"ToolShellExec", ToolShellExec},
		{"ToolSSHExec", ToolSSHExec},
		{"ToolSystemInfo", ToolSystemInfo},
		{"ToolFileRead", ToolFileRead},
		{"ToolFileWrite", ToolFileWrite},
		{"ToolFileSearch", ToolFileSearch},
		{"ToolFileList", ToolFileList},
		{"ToolCodeNavigate", ToolCodeNavigate},
		{"ToolWebFetch", ToolWebFetch},
		{"ToolWebSearch", ToolWebSearch},
		{"ToolDateTime", ToolDateTime},
		{"ToolText", ToolText},
		{"ToolCalculator", ToolCalculator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

// TestTimeoutConstants tests timeout constants.
func TestTimeoutConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Duration
		minValue time.Duration
	}{
		{"DefaultToolTimeout", DefaultToolTimeout, 10 * time.Second},
		{"DefaultShellTimeout", DefaultShellTimeout, 10 * time.Second},
		{"DefaultSSHTimeout", DefaultSSHTimeout, 10 * time.Second},
		{"DefaultWebTimeout", DefaultWebTimeout, 10 * time.Second},
		{"DefaultLLMTimeout", DefaultLLMTimeout, 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value <= 0 {
				t.Errorf("%s should be positive", tt.name)
			}
			if tt.value < tt.minValue {
				t.Errorf("%s = %v, should be at least %v", tt.name, tt.value, tt.minValue)
			}
		})
	}
}

// TestAPIPortConstant tests DefaultAPIPort constant.
func TestAPIPortConstant(t *testing.T) {
	if DefaultAPIPort <= 0 {
		t.Errorf("DefaultAPIPort = %d, should be positive", DefaultAPIPort)
	}
	if DefaultAPIPort < 1024 || DefaultAPIPort > 65535 {
		t.Errorf("DefaultAPIPort = %d, should be a valid port number", DefaultAPIPort)
	}
}

// TestAllToolConstantsAreUnique tests that all tool constants have unique values.
func TestAllToolConstantsAreUnique(t *testing.T) {
	toolValues := []string{
		ToolShellExec,
		ToolSSHExec,
		ToolSystemInfo,
		ToolFileRead,
		ToolFileWrite,
		ToolFileSearch,
		ToolFileList,
		ToolCodeNavigate,
		ToolWebFetch,
		ToolWebSearch,
		ToolDateTime,
		ToolText,
		ToolCalculator,
	}

	seen := make(map[string]bool)
	for _, tool := range toolValues {
		if seen[tool] {
			t.Errorf("Duplicate tool value: %q", tool)
		}
		seen[tool] = true
	}
}

// TestDirectoryConstantsAreUnique tests that all directory constants have unique values.
func TestDirectoryConstantsAreUnique(t *testing.T) {
	dirValues := []string{
		DirKnowledge,
		DirSessions,
		DirRoles,
		DirSkills,
		DirMemory,
	}

	seen := make(map[string]bool)
	for _, dir := range dirValues {
		if seen[dir] {
			t.Errorf("Duplicate directory value: %q", dir)
		}
		seen[dir] = true
	}
}
