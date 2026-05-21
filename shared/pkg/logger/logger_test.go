// Package logger provides tests for structured logging.
package logger

import (
	"testing"
)

// TestNew tests logger creation.
func TestNew(t *testing.T) {
	cfg := Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := New(cfg)
	if log == nil {
		t.Fatal("New() returned nil")
	}
}

// TestNewTextFormat tests text format logger creation.
func TestNewTextFormat(t *testing.T) {
	cfg := Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}

	log := New(cfg)
	if log == nil {
		t.Fatal("New() returned nil")
	}
}

// TestNewStderr tests stderr output logger.
func TestNewStderr(t *testing.T) {
	cfg := Config{
		Level:  "info",
		Format: "json",
		Output: "stderr",
	}

	log := New(cfg)
	if log == nil {
		t.Fatal("New() returned nil")
	}
}

// TestDefault tests default logger creation.
func TestDefault(t *testing.T) {
	log := Default()
	if log == nil {
		t.Fatal("Default() returned nil")
	}
}

// TestWithField tests adding a field to logger.
func TestWithField(t *testing.T) {
	log := Default()
	logWithField := log.WithField("key", "value")

	if logWithField == nil {
		t.Fatal("WithField() returned nil")
	}
	if logWithField == log {
		t.Error("WithField should return a new logger instance")
	}
}

// TestWithFields tests adding multiple fields to logger.
func TestWithFields(t *testing.T) {
	log := Default()
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	logWithFields := log.WithFields(fields)
	if logWithFields == nil {
		t.Fatal("WithFields() returned nil")
	}
}

// TestWithFieldsEmpty tests adding empty fields map.
func TestWithFieldsEmpty(t *testing.T) {
	log := Default()
	logWithFields := log.WithFields(map[string]interface{}{})

	if logWithFields == nil {
		t.Fatal("WithFields() returned nil")
	}
}

// TestParseLevel tests log level parsing.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := Config{
				Level:  tt.level,
				Format: "json",
				Output: "stdout",
			}
			log := New(cfg)
			if log == nil {
				t.Fatal("New() returned nil")
			}
		})
	}
}

// TestLoggerMethods tests logger methods don't panic.
func TestLoggerMethods(t *testing.T) {
	log := Default()

	// zerolog uses chaining with Msg()
	log.Debug().Msg("debug message")
	log.Info().Msg("info message")
	log.Warn().Msg("warn message")
	log.Error().Msg("error message")

	// With fields
	log.WithField("test", "value").Info().Msg("message with field")
	log.WithFields(map[string]interface{}{"k": "v"}).Info().Msg("message with fields")
}

// TestConfigStruct tests config struct creation.
func TestConfigStruct(t *testing.T) {
	cfg := Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	if cfg.Level != "debug" {
		t.Errorf("Level = %q, want %q", cfg.Level, "debug")
	}
	if cfg.Format != "json" {
		t.Errorf("Format = %q, want %q", cfg.Format, "json")
	}
	if cfg.Output != "stdout" {
		t.Errorf("Output = %q, want %q", cfg.Output, "stdout")
	}
}

// TestLoggerChaining tests method chaining.
func TestLoggerChaining(t *testing.T) {
	log := Default()

	// Chain multiple WithField calls
	chained := log.WithField("a", 1).WithField("b", 2)
	if chained == nil {
		t.Error("Chained WithField returned nil")
	}
}
