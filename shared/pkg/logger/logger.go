// Package logger provides structured logging for aura.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger.
type Logger struct {
	zerolog.Logger
	logFile   *os.File // Keep reference for cleanup (primary output)
	jsonlFile *os.File // Keep reference for cleanup (JSONL secondary output)
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

// New creates a new logger instance.
func New(cfg Config) *Logger {
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
	var jsonlFile *os.File
	if cfg.JSONLPath != "" {
		jsonlDir := filepath.Dir(cfg.JSONLPath)
		if err := ffp.EnsureDir(jsonlDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot create JSONL log directory %s: %v\n", jsonlDir, err)
		} else {
			var err error
			jsonlFile, err = os.OpenFile(cfg.JSONLPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	level := parseLevel(cfg.Level)
	zl := zerolog.New(output).Level(level)

	// Add module field to context if specified
	ctx := zl.With().Timestamp()
	if cfg.Module != "" {
		ctx = ctx.Str("module", cfg.Module)
	}

	return &Logger{
		Logger:    ctx.Logger(),
		logFile:   logFile,
		jsonlFile: jsonlFile,
	}
}

// Close closes the log files if opened.
func (l *Logger) Close() error {
	var err error
	if l.jsonlFile != nil {
		err = l.jsonlFile.Close()
	}
	if l.logFile != nil {
		if err2 := l.logFile.Close(); err2 != nil {
			err = err2
		}
	}
	return err
}

func parseLevel(level string) zerolog.Level {
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

// Default creates a default logger with text format.
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
}

// WithField adds a field to the logger.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Logger: l.Logger.With().Interface(key, value).Logger()}
}

// WithFields adds multiple fields to the logger.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{Logger: ctx.Logger()}
}

// WithModule returns a new Logger with the "module" field set.
func (l *Logger) WithModule(module string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("module", module).Logger()}
}

// NewNamed creates a logger with a module name baked into the zerolog context.
// Equivalent to New(cfg).WithModule(cfg.Module) but more efficient.
func NewNamed(cfg Config) *Logger {
	return New(cfg)
}
