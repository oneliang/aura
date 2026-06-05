// Package logger provides structured logging for aura.
package logger

import (
	"os"
)

// Level represents log level.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Logger provides structured logging with key-value pairs.
type Logger struct {
	target  Log       // 唯一输出源
	logFile *os.File  // 用于清理（从 zerologTarget 获取）
	module  string    // 模块名（自动添加到所有日志）
}

// Config represents logger configuration.
type Config struct {
	Level     string // debug, info, warn, error
	Format    string // json, text
	Output    string // stdout, stderr, file
	Path      string // log file path (used when Output="file")
	Module    string // component/module name, auto-added as "module" field
	JSONLPath string // if non-empty, also write JSONL entries to this file (no ConsoleWriter)
}

// New creates a new logger instance with default zerologTarget.
func New(cfg Config) *Logger {
	target := newZerologTarget(cfg)
	return &Logger{
		target:  target,
		logFile: target.logFile,
		module:  cfg.Module,
	}
}

// SetTarget replaces the output target.
// Use this to inject external logger implementations.
func (l *Logger) SetTarget(target Log) {
	l.target = target
}

// Debug logs a debug message with key-value pairs.
func (l *Logger) Debug(msg string, kv ...any) {
	l.log(DebugLevel, msg, kv...)
}

// Info logs an info message with key-value pairs.
func (l *Logger) Info(msg string, kv ...any) {
	l.log(InfoLevel, msg, kv...)
}

// Warn logs a warning message with key-value pairs.
func (l *Logger) Warn(msg string, kv ...any) {
	l.log(WarnLevel, msg, kv...)
}

// Error logs an error message with key-value pairs.
func (l *Logger) Error(msg string, kv ...any) {
	l.log(ErrorLevel, msg, kv...)
}

// log writes a message at the specified level.
func (l *Logger) log(level Level, msg string, kv ...any) {
	// Prepend module if set
	if l.module != "" {
		kv = append([]any{"module", l.module}, kv...)
	}

	switch level {
	case DebugLevel:
		l.target.Debug(msg, kv...)
	case InfoLevel:
		l.target.Info(msg, kv...)
	case WarnLevel:
		l.target.Warn(msg, kv...)
	case ErrorLevel:
		l.target.Error(msg, kv...)
	}
}

// WithModule returns a new Logger with the "module" field set.
func (l *Logger) WithModule(module string) *Logger {
	return &Logger{
		target:  l.target,
		logFile: l.logFile,
		module:  module,
	}
}

// WithField returns a new Logger with a preset field.
// Note: This creates a wrapper that prepends the field to each log call.
func (l *Logger) WithField(key string, value any) *Logger {
	// For compatibility, return a Logger that will prepend this field
	// We implement this by creating a wrapper target
	return &Logger{
		target:  &fieldWrapper{base: l.target, fields: []any{key, value}},
		logFile: l.logFile,
		module:  l.module,
	}
}

// WithFields returns a new Logger with multiple preset fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	kv := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		kv = append(kv, k, v)
	}
	return &Logger{
		target:  &fieldWrapper{base: l.target, fields: kv},
		logFile: l.logFile,
		module:  l.module,
	}
}

// Close closes the log files if opened.
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Default creates a default logger with text format.
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
}

// NewNamed creates a logger with a module name.
// Equivalent to New(cfg) but cfg.Module is used.
func NewNamed(cfg Config) *Logger {
	return New(cfg)
}

// parseLevel converts string level to Level type.
func parseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// fieldWrapper wraps a Log target and prepends preset fields.
type fieldWrapper struct {
	base   Log
	fields []any
}

func (w *fieldWrapper) Debug(msg string, kv ...any) {
	w.base.Debug(msg, append(w.fields, kv...)...)
}

func (w *fieldWrapper) Info(msg string, kv ...any) {
	w.base.Info(msg, append(w.fields, kv...)...)
}

func (w *fieldWrapper) Warn(msg string, kv ...any) {
	w.base.Warn(msg, append(w.fields, kv...)...)
}

func (w *fieldWrapper) Error(msg string, kv ...any) {
	w.base.Error(msg, append(w.fields, kv...)...)
}