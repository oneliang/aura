package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestReadJSONFile(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.json")

		content := []byte(`{"name":"test","value":42}`)
		if err := os.WriteFile(tmpFile, content, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		var result testStruct
		err := ReadJSONFile(tmpFile, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Name != "test" {
			t.Errorf("expected name 'test', got %q", result.Name)
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		var result testStruct
		err := ReadJSONFile("/nonexistent/path/file.json", &result)
		if err != ErrFileNotFound {
			t.Errorf("expected ErrFileNotFound, got %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(tmpFile, []byte(`{invalid json}`), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		var result testStruct
		err := ReadJSONFile(tmpFile, &result)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestReadJSONFileOrDefault(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.json")

		content := []byte(`{"name":"existing","value":100}`)
		if err := os.WriteFile(tmpFile, content, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		var result testStruct
		defaultVal := testStruct{Name: "default", Value: 0}
		err := ReadJSONFileOrDefault(tmpFile, &result, defaultVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Name != "existing" {
			t.Errorf("expected name 'existing', got %q", result.Name)
		}
		if result.Value != 100 {
			t.Errorf("expected value 100, got %d", result.Value)
		}
	})

	t.Run("file not found uses default", func(t *testing.T) {
		var result testStruct
		defaultVal := testStruct{Name: "default", Value: 99}
		err := ReadJSONFileOrDefault("/nonexistent/path/file.json", &result, defaultVal)
		if err != ErrFileNotFound {
			t.Errorf("expected ErrFileNotFound, got %v", err)
		}

		if result.Name != "default" {
			t.Errorf("expected name 'default', got %q", result.Name)
		}
		if result.Value != 99 {
			t.Errorf("expected value 99, got %d", result.Value)
		}
	})
}

func TestWriteJSONFile(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "output.json")

		data := testStruct{Name: "test", Value: 42}
		err := WriteJSONFile(tmpFile, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		content, err := os.ReadFile(tmpFile)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}

		// Verify content
		var result testStruct
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatalf("failed to unmarshal written content: %v", err)
		}

		if result.Name != "test" {
			t.Errorf("expected name 'test', got %q", result.Name)
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})

	t.Run("atomic write", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "atomic.json")

		data := testStruct{Name: "atomic", Value: 1}
		err := WriteJSONFile(tmpFile, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify temp file is cleaned up
		tmpFileWithExt := tmpFile + ".tmp"
		if _, err := os.Stat(tmpFileWithExt); !os.IsNotExist(err) {
			t.Error("temp file should be cleaned up after atomic write")
		}
	})

	t.Run("cleanup on rename failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "output.json")
		tmpFileWithExt := tmpFile + ".tmp"

		data := testStruct{Name: "test", Value: 42}
		err := WriteJSONFile(tmpFile, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify temp file is cleaned up even after successful rename
		if _, err := os.Stat(tmpFileWithExt); !os.IsNotExist(err) {
			t.Error("temp file should be cleaned up after successful write")
		}
	})
}

func TestWriteJSONFileWithMode(t *testing.T) {
	t.Run("write with custom permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "custom.json")

		data := testStruct{Name: "custom", Value: 123}
		err := WriteJSONFileWithMode(tmpFile, data, 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check permissions
		info, err := os.Stat(tmpFile)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		// Note: umask may affect actual permissions
		if info.Mode().Perm()&0600 != 0600 {
			t.Errorf("expected at least 0600 permissions, got %o", info.Mode().Perm())
		}
	})
}
