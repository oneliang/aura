package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/rs/zerolog"
)

// zerologTarget implements Log interface using zerolog.
type zerologTarget struct {
	zl      zerolog.Logger
	logFile *os.File
}

// newZerologTarget creates a zerolog-based Log implementation.
func newZerologTarget(cfg Config) *zerologTarget {
	var writers []io.Writer
	var logFile *os.File

	// Primary output
	var primary io.Writer = os.Stdout
	var primaryFile *os.File

	if cfg.Output == "file" {
		logPath := cfg.Path
		if logPath == "" {
			logPath = ffp.AuraHomePathOrDefault("aura.log")
			if logPath == "" {
				logPath = "aura.log"
			}
		}

		logDir := filepath.Dir(logPath)
		if err := ffp.EnsureDir(logDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot create log directory %s: %v, using stdout\n", logDir, err)
		} else {
			var err error
			primaryFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot open log file %s: %v, using stdout\n", logPath, err)
			} else {
				primary = primaryFile
				logFile = primaryFile
			}
		}
	} else if cfg.Output == "stderr" {
		primary = os.Stderr
	}

	// Apply text format (ConsoleWriter) only for non-file primary output
	// or when file output succeeded and format is text
	if cfg.Format == "text" {
		noColor := cfg.Output == "file"
		primary = zerolog.ConsoleWriter{
			Out:        primary,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    noColor,
		}
	}
	writers = append(writers, primary)

	// Optional JSONL secondary output (audit/debug log to separate file)
	if cfg.JSONLPath != "" {
		jsonlDir := filepath.Dir(cfg.JSONLPath)
		if err := ffp.EnsureDir(jsonlDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot create JSONL log directory %s: %v\n", jsonlDir, err)
		} else {
			jsonlFile, err := os.OpenFile(cfg.JSONLPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot open JSONL log file %s: %v\n", cfg.JSONLPath, err)
			} else {
				writers = append(writers, jsonlFile)
			}
		}
	}

	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	level := parseZerologLevel(cfg.Level)
	zl := zerolog.New(output).Level(level).With().Timestamp().Logger()

	// Add module field to context if specified
	if cfg.Module != "" {
		zl = zl.With().Str("module", cfg.Module).Logger()
	}

	return &zerologTarget{
		zl:      zl,
		logFile: logFile,
	}
}

// Debug logs a debug message with key-value pairs.
func (t *zerologTarget) Debug(msg string, kv ...any) {
	t.log(zerolog.DebugLevel, msg, kv)
}

// Info logs an info message with key-value pairs.
func (t *zerologTarget) Info(msg string, kv ...any) {
	t.log(zerolog.InfoLevel, msg, kv)
}

// Warn logs a warning message with key-value pairs.
func (t *zerologTarget) Warn(msg string, kv ...any) {
	t.log(zerolog.WarnLevel, msg, kv)
}

// Error logs an error message with key-value pairs.
func (t *zerologTarget) Error(msg string, kv ...any) {
	t.log(zerolog.ErrorLevel, msg, kv)
}

// log writes a message at the specified level with key-value pairs.
func (t *zerologTarget) log(level zerolog.Level, msg string, kv []any) {
	event := t.zl.WithLevel(level)
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			key := fmt.Sprintf("%v", kv[i])
			event.Interface(key, kv[i+1])
		}
	}
	event.Msg(msg)
}

// Close closes the log file if opened.
func (t *zerologTarget) Close() error {
	if t.logFile != nil {
		return t.logFile.Close()
	}
	return nil
}

// parseZerologLevel converts string level to zerolog.Level.
func parseZerologLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}