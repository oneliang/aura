// Package filepath provides tests for the filepath package.
package filepath

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetHomeDir tests GetHomeDir function.
func TestGetHomeDir(t *testing.T) {
	homeDir := GetHomeDir()
	if homeDir == "" {
		t.Error("GetHomeDir() returned empty string")
	}
	if homeDir == "." {
		t.Log("GetHomeDir() returned fallback '.' - home directory may not be available")
	}
}

// TestEnsureDir tests EnsureDir function.
func TestEnsureDir(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Test creating new directory
	newDir := filepath.Join(tmpDir, "test", "nested", "dir")
	err := EnsureDir(newDir)
	if err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("Directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path should be a directory")
	}

	// Test creating existing directory (should not error)
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir() on existing dir should not error: %v", err)
	}
}

// TestExpandTilde tests ExpandTilde function.
func TestExpandTilde(t *testing.T) {
	homeDir := GetHomeDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde slash",
			path: "~/test",
			want: filepath.Join(homeDir, "test"),
		},
		{
			name: "tilde only",
			path: "~",
			want: homeDir,
		},
		{
			name: "no tilde",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "relative path",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde without slash",
			path: "~user",
			want: "~user", // Should not expand ~user pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTilde(tt.path)
			if got != tt.want {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestExpandTilde_BackslashSeparator tests ExpandTilde with backslash separator (Windows).
func TestExpandTilde_BackslashSeparator(t *testing.T) {
	// This test is mainly for Windows compatibility
	homeDir := GetHomeDir()

	// On Unix systems, backslash is a valid filename character, not a separator
	// So ~\test would not be expanded (returns as-is)
	result := ExpandTilde("~\\test")
	if filepath.Separator == '\\' {
		// Windows - should expand
		expected := filepath.Join(homeDir, "test")
		if result != expected {
			t.Errorf("ExpandTilde(\"~\\\\test\") = %q, want %q", result, expected)
		}
	} else {
		// Unix - should not expand (backslash is part of filename)
		if result != "~\\test" {
			t.Errorf("ExpandTilde(\"~\\\\test\") on Unix should return \"~\\\\test\", got %q", result)
		}
	}
}

// TestJoinSafe tests JoinSafe function.
func TestJoinSafe(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{
			name:     "normal join",
			segments: []string{"a", "b", "c"},
			want:     filepath.Join("a", "b", "c"),
		},
		{
			name:     "with empty segments",
			segments: []string{"a", "", "c"},
			want:     filepath.Join("a", "c"),
		},
		{
			name:     "all empty",
			segments: []string{"", "", ""},
			want:     "",
		},
		{
			name:     "single segment",
			segments: []string{"alone"},
			want:     "alone",
		},
		{
			name:     "empty input",
			segments: []string{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinSafe(tt.segments...)
			if got != tt.want {
				t.Errorf("JoinSafe(%v) = %q, want %q", tt.segments, got, tt.want)
			}
		})
	}
}

// TestAbsRelativeToBase tests AbsRelativeToBase function.
func TestAbsRelativeToBase(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		base    string
		rel     string
		wantErr bool
	}{
		{
			name: "with base",
			base: tmpDir,
			rel:  "sub/dir",
		},
		{
			name: "empty base uses cwd",
			base: "",
			rel:  "relative",
		},
		{
			name: "absolute relative path",
			base: tmpDir,
			rel:  "/absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AbsRelativeToBase(tt.base, tt.rel)
			if (err != nil) != tt.wantErr {
				t.Errorf("AbsRelativeToBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == "" {
					t.Error("AbsRelativeToBase() should return non-empty path")
				}
				// Verify it's an absolute path
				if !filepath.IsAbs(got) {
					t.Errorf("Result should be absolute: %q", got)
				}
			}
		})
	}
}

// TestIsSubPath tests IsSubPath function.
func TestIsSubPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "direct subpath",
			parent: tmpDir,
			child:  filepath.Join(tmpDir, "sub"),
			want:   true,
		},
		{
			name:   "nested subpath",
			parent: tmpDir,
			child:  filepath.Join(tmpDir, "a", "b", "c"),
			want:   true,
		},
		{
			name:   "same path returns true",
			parent: tmpDir,
			child:  tmpDir,
			want:   true, // Same path is considered a subpath
		},
		{
			name:   "not subpath",
			parent: tmpDir,
			child:  "/other/path",
			want:   false,
		},
		{
			name:   "parent is prefix but not path separator",
			parent: tmpDir,
			child:  tmpDir + "_suffix",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSubPath(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("IsSubPath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}

// TestIsSubPath_InvalidPaths tests IsSubPath with invalid paths.
func TestIsSubPath_InvalidPaths(t *testing.T) {
	// Invalid paths should return false
	if IsSubPath("", "") {
		t.Error("IsSubPath(\"\", \"\") should return false")
	}

	if IsSubPath("/nonexistent/parent", "/nonexistent/child") {
		// This may return true or false depending on implementation
		// Just verify it doesn't panic
	}
}

// TestAbsRelativeToBase_InvalidBase tests AbsRelativeToBase with invalid base.
func TestAbsRelativeToBase_InvalidBase(t *testing.T) {
	// Test with a base that doesn't exist (should still work as it just joins paths)
	_, err := AbsRelativeToBase("/nonexistent/base", "relative")
	if err != nil {
		t.Errorf("AbsRelativeToBase() with nonexistent base should not error: %v", err)
	}
}
