package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	cfg := Config{Level: "debug", Format: "json", Output: "stdout"}
	l1 := r.Register("test", cfg)
	l2 := r.Register("test", cfg)

	if l1 != l2 {
		t.Error("Register should return the same instance for the same name")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	r.Register("mylogger", Config{Level: "info", Format: "json", Output: "stdout"})

	l, ok := r.Get("mylogger")
	if !ok {
		t.Fatal("Get should find registered logger")
	}
	if l == nil {
		t.Fatal("Get returned nil")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("Get should not find nonexistent logger")
	}
}

func TestRegistry_MustGet(t *testing.T) {
	r := NewRegistry()

	r.Register("exists", Config{Level: "info", Format: "json", Output: "stdout"})

	l := r.MustGet("exists")
	if l == nil {
		t.Fatal("MustGet returned nil for existing logger")
	}

	// Should fall back to Default for nonexistent
	l2 := r.MustGet("nonexistent")
	if l2 == nil {
		t.Fatal("MustGet should fall back to Default")
	}
}

func TestRegistry_Default(t *testing.T) {
	r := NewRegistry()

	l1 := r.Default()
	l2 := r.Default()

	if l1 != l2 {
		t.Error("Default should return the same instance")
	}
}

func TestRegistry_CloseAll(t *testing.T) {
	r := NewRegistry()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	r.Register("filelog", Config{Level: "info", Format: "json", Output: "file", Path: logPath})
	r.Register("stdout", Config{Level: "info", Format: "json", Output: "stdout"})

	err := r.CloseAll()
	if err != nil {
		t.Fatalf("CloseAll returned error: %v", err)
	}

	// After CloseAll, registry should be empty
	if len(r.loggers) != 0 {
		t.Error("Registry should be empty after CloseAll")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Test global registry functions
	l := Register("global_test", Config{Level: "info", Format: "json", Output: "stdout"})
	if l == nil {
		t.Fatal("Register returned nil")
	}

	l2, ok := Get("global_test")
	if !ok {
		t.Fatal("Get should find global registered logger")
	}
	if l != l2 {
		t.Error("Get should return same instance as Register")
	}

	l3 := MustGet("global_test")
	if l3 != l {
		t.Error("MustGet should return same instance")
	}
}

func TestNewNamed(t *testing.T) {
	log := NewNamed(Config{Level: "info", Format: "json", Output: "stdout", Module: "test-module"})
	if log == nil {
		t.Fatal("NewNamed returned nil")
	}
}

func TestWithModule(t *testing.T) {
	log := Default()
	modLog := log.WithModule("test-module")
	if modLog == nil {
		t.Fatal("WithModule returned nil")
	}
	if modLog == log {
		t.Error("WithModule should return a new logger instance")
	}
}

func TestNewFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	log := New(Config{
		Level:  "info",
		Format: "json",
		Output: "file",
		Path:   logPath,
	})

	if log == nil {
		t.Fatal("New file logger returned nil")
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file should be created")
	}

	log.Close()
}

func TestNewJSONLPath(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "audit.jsonl")

	log := New(Config{
		Level:     "debug",
		Format:    "text",
		Output:    "stdout",
		JSONLPath: jsonlPath,
	})

	if log == nil {
		t.Fatal("New with JSONLPath returned nil")
	}

	log.Close()
}
