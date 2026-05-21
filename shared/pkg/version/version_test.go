// Package version provides tests for version information.
package version

import (
	"strings"
	"testing"
)

// TestGetVersion tests version retrieval.
func TestGetVersion(t *testing.T) {
	v := GetVersion()

	if v == "" {
		t.Error("GetVersion() returned empty string")
	}
}

// TestGetVersionDefault tests default version when empty.
func TestGetVersionDefault(t *testing.T) {
	// Save original
	origVersion := Version
	defer func() { Version = origVersion }()

	// Test with empty Version
	Version = ""
	v := GetVersion()

	if v != "v0.0.1-alpha" {
		t.Errorf("GetVersion() with empty Version = %q, want %q", v, "v0.0.1-alpha")
	}
}

// TestGetVersionWithBuildInfo tests version with build info.
func TestGetVersionWithBuildInfo(t *testing.T) {
	// Save original
	origVersion := Version
	defer func() { Version = origVersion }()

	// Test with set Version
	Version = "v1.0.0"
	v := GetVersion()

	if v != "v1.0.0" {
		t.Errorf("GetVersion() = %q, want %q", v, "v1.0.0")
	}
}

// TestGetBuildTime tests build time retrieval.
func TestGetBuildTime(t *testing.T) {
	bt := GetBuildTime()

	// Should not be empty (either actual value or "unknown")
	if bt == "" {
		t.Error("GetBuildTime() returned empty string")
	}
}

// TestGetBuildTimeDefault tests default build time when empty.
func TestGetBuildTimeDefault(t *testing.T) {
	// Save original
	origBuildTime := BuildTime
	defer func() { BuildTime = origBuildTime }()

	// Test with empty BuildTime
	BuildTime = ""
	bt := GetBuildTime()

	if bt != "unknown" {
		t.Errorf("GetBuildTime() with empty BuildTime = %q, want %q", bt, "unknown")
	}
}

// TestFullVersion tests full version string.
func TestFullVersion(t *testing.T) {
	fv := FullVersion()

	if fv == "" {
		t.Error("FullVersion() returned empty string")
	}
	if !strings.Contains(fv, " (built: ") {
		t.Errorf("FullVersion() = %q, should contain ' (built: '", fv)
	}
	if !strings.HasSuffix(fv, ")") {
		t.Errorf("FullVersion() = %q, should end with ')'", fv)
	}
}

// TestFullVersionWithCustomInfo tests full version with custom info.
func TestFullVersionWithCustomInfo(t *testing.T) {
	// Save original
	origVersion := Version
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
	}()

	// Set custom values
	Version = "v2.0.0"
	BuildTime = "2024-01-15_10:30:45"

	fv := FullVersion()

	expected := "v2.0.0 (built: 2024-01-15_10:30:45)"
	if fv != expected {
		t.Errorf("FullVersion() = %q, want %q", fv, expected)
	}
}

// TestFullVersionWithDefaultInfo tests full version with default info.
func TestFullVersionWithDefaultInfo(t *testing.T) {
	// Save original
	origVersion := Version
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
	}()

	// Set to empty to test defaults
	Version = ""
	BuildTime = ""

	fv := FullVersion()

	if !strings.Contains(fv, "v0.0.1-alpha") {
		t.Errorf("FullVersion() = %q, should contain default version", fv)
	}
	if !strings.Contains(fv, "unknown") {
		t.Errorf("FullVersion() = %q, should contain 'unknown'", fv)
	}
}

// TestVersionConstants tests version constants.
func TestVersionConstants(t *testing.T) {
	// Version should have a default value
	if Version == "" {
		t.Log("Version constant is empty (will use default in GetVersion)")
	}
	// BuildTime can be empty (will use default in GetBuildTime)
}

// TestVersionFormat tests version format validation.
func TestVersionFormat(t *testing.T) {
	v := GetVersion()

	// Version should start with 'v' (semantic versioning convention)
	if !strings.HasPrefix(v, "v") && !strings.HasPrefix(v, "dev") {
		t.Logf("Version %q doesn't follow semantic versioning (should start with 'v')", v)
	}
}

// TestVersionComponents tests parsing version components.
func TestVersionComponents(t *testing.T) {
	// Save original
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "v1.2.3-alpha"
	v := GetVersion()

	if !strings.Contains(v, "1") || !strings.Contains(v, "2") || !strings.Contains(v, "3") {
		t.Logf("Version %q may not contain expected components", v)
	}
}
