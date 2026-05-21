// Package logger provides structured logging for aura.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/rs/zerolog"
)

// auditLogger wraps a zerolog-based logger writing to a JSONL file.
type auditLogger struct {
	log     *Logger
	file    *os.File
	mu      sync.Mutex
	enabled bool
}

func newAuditLogger(path string) (*auditLogger, error) {
	dir := filepath.Dir(path)
	if err := ffp.EnsureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}

	zl := zerolog.New(f).With().Timestamp().Logger()

	return &auditLogger{
		log:     &Logger{Logger: zl},
		file:    f,
		enabled: true,
	}, nil
}

func (a *auditLogger) write(entry map[string]any) {
	if a == nil || !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.log == nil || a.file == nil {
		return
	}

	a.log.Info().Interface("entry", entry).Msg("")
}

func (a *auditLogger) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.file != nil {
		err := a.file.Close()
		a.file = nil
		a.enabled = false
		return err
	}
	return nil
}

//
// LLM Audit Logger
//

// LLMAuditLogger writes structured JSONL entries for LLM interactions.
type LLMAuditLogger struct {
	audit *auditLogger
}

// LLMLogEntry represents a single LLM interaction log entry.
type LLMLogEntry struct {
	Timestamp  int64                  `json:"timestamp"`
	RequestID  string                 `json:"request_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	Method     string                 `json:"method"`
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model"`
	Messages   any                    `json:"messages,omitempty"`
	InputTexts []string               `json:"input_texts,omitempty"`
	Response   any                    `json:"response,omitempty"`
	DurationMs int64                  `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewLLMAuditLogger creates an LLM audit logger writing to the given JSONL path.
func NewLLMAuditLogger(path string) (*LLMAuditLogger, error) {
	a, err := newAuditLogger(path)
	if err != nil {
		return nil, err
	}
	return &LLMAuditLogger{audit: a}, nil
}

// GetLLMAuditLogger returns the global LLM audit logger, creating it on first call.
// Returns nil if the logger cannot be initialized (e.g., log dir unavailable).
func GetLLMAuditLogger() *LLMAuditLogger {
	llmAuditOnce.Do(func() {
		logPath, err := ffp.AuraHomePath("logs/llm_requests.log")
		if err != nil {
			return
		}
		if err := ffp.EnsureDir(filepath.Dir(logPath)); err != nil {
			return
		}
		llmAuditLogger, err = NewLLMAuditLogger(logPath)
		if err != nil {
			return
		}
	})
	return llmAuditLogger
}

// CloseLLMAuditLogger closes the global LLM audit logger.
func CloseLLMAuditLogger() error {
	if llmAuditLogger != nil {
		return llmAuditLogger.Close()
	}
	return nil
}

// Log writes an LLM log entry.
func (l *LLMAuditLogger) Log(entry LLMLogEntry) {
	if l == nil || l.audit == nil {
		return
	}
	l.audit.mu.Lock()
	defer l.audit.mu.Unlock()

	if !l.audit.enabled || l.audit.file == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal LLM log entry: %v\n", err)
		return
	}

	_, err = l.audit.file.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write LLM log entry: %v\n", err)
	}
}

// Close closes the underlying audit logger.
func (l *LLMAuditLogger) Close() error {
	if l == nil || l.audit == nil {
		return nil
	}
	return l.audit.Close()
}

var (
	llmAuditLogger *LLMAuditLogger
	llmAuditOnce   sync.Once
)

//
// Delegation Audit Logger
//

// DelegationAuditLogger writes structured JSONL entries for agent delegation.
type DelegationAuditLogger struct {
	audit *auditLogger
}

// DelegationLogEntry represents a single delegation log entry.
type DelegationLogEntry struct {
	Timestamp     int64                  `json:"timestamp"`
	RequestID     string                 `json:"request_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	Event         string                 `json:"event"`
	AgentName     string                 `json:"agent_name,omitempty"`
	Step          string                 `json:"step,omitempty"`
	DurationMs    int64                  `json:"duration_ms,omitempty"`
	Path          string                 `json:"path,omitempty"`
	TaskPreview   string                 `json:"task_preview,omitempty"`
	ResultPreview string                 `json:"result_preview,omitempty"`
	Success       *bool                  `json:"success,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewDelegationAuditLogger creates a delegation audit logger writing to the given JSONL path.
func NewDelegationAuditLogger(path string) (*DelegationAuditLogger, error) {
	a, err := newAuditLogger(path)
	if err != nil {
		return nil, err
	}
	return &DelegationAuditLogger{audit: a}, nil
}

// GetDelegationAuditLogger returns the global delegation audit logger, creating it on first call.
func GetDelegationAuditLogger() *DelegationAuditLogger {
	delegationAuditOnce.Do(func() {
		logPath, err := ffp.AuraHomePath("logs/agent_delegation.log")
		if err != nil {
			return
		}
		if err := ffp.EnsureDir(filepath.Dir(logPath)); err != nil {
			return
		}
		delegationAuditLogger, err = NewDelegationAuditLogger(logPath)
		if err != nil {
			return
		}
	})
	return delegationAuditLogger
}

// CloseDelegationAuditLogger closes the global delegation audit logger.
func CloseDelegationAuditLogger() error {
	if delegationAuditLogger != nil {
		return delegationAuditLogger.Close()
	}
	return nil
}

// Log writes a delegation log entry.
func (d *DelegationAuditLogger) Log(entry DelegationLogEntry) {
	if d == nil || d.audit == nil {
		return
	}
	d.audit.mu.Lock()
	defer d.audit.mu.Unlock()

	if !d.audit.enabled || d.audit.file == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal delegation log entry: %v\n", err)
		return
	}

	_, err = d.audit.file.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write delegation log entry: %v\n", err)
	}
}

// Close closes the underlying audit logger.
func (d *DelegationAuditLogger) Close() error {
	if d == nil || d.audit == nil {
		return nil
	}
	return d.audit.Close()
}

// Start logs the beginning of a delegation.
func (d *DelegationAuditLogger) Start(requestID, sessionID, agentName, task string) {
	if d == nil {
		return
	}
	d.Log(DelegationLogEntry{
		Timestamp:   timestampNow(),
		RequestID:   requestID,
		SessionID:   sessionID,
		Event:       "start",
		AgentName:   agentName,
		TaskPreview: TruncateStr(task, 200),
	})
}

// Step logs a delegation step.
func (d *DelegationAuditLogger) Step(requestID, step string, durationMs int64) {
	if d == nil {
		return
	}
	d.Log(DelegationLogEntry{
		Timestamp:  timestampNow(),
		RequestID:  requestID,
		Event:      "step",
		Step:       step,
		DurationMs: durationMs,
	})
}

// Complete logs successful delegation completion.
func (d *DelegationAuditLogger) Complete(requestID, agentName string, totalDurationMs int64, path string, result string) {
	if d == nil {
		return
	}
	success := true
	d.Log(DelegationLogEntry{
		Timestamp:     timestampNow(),
		RequestID:     requestID,
		Event:         "complete",
		AgentName:     agentName,
		DurationMs:    totalDurationMs,
		Path:          path,
		ResultPreview: TruncateStr(result, 200),
		Success:       &success,
	})
}

// Error logs a delegation error.
func (d *DelegationAuditLogger) Error(requestID, agentName, step string, durationMs int64, err error) {
	if d == nil {
		return
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	d.Log(DelegationLogEntry{
		Timestamp:  timestampNow(),
		RequestID:  requestID,
		Event:      "error",
		AgentName:  agentName,
		Step:       step,
		DurationMs: durationMs,
		Error:      errStr,
	})
}

var (
	delegationAuditLogger *DelegationAuditLogger
	delegationAuditOnce   sync.Once
)

//
// Delegation File Logger (per-delegation independent log file)
//

// DelegationFileLogger provides per-delegation independent log file support.
type DelegationFileLogger struct {
	log       *Logger
	file      *os.File
	mu        sync.Mutex
	path      string
	requestID string
	agentName string
}

// NewDelegationFileLogger creates a per-delegation log file at ~/.aura/logs/delegation/{request_id}.log.
func NewDelegationFileLogger(requestID, agentName string) (*DelegationFileLogger, error) {
	logDir, err := ffp.AuraHomePath("logs")
	if err != nil {
		return nil, fmt.Errorf("failed to get log directory: %w", err)
	}

	delegationDir := filepath.Join(logDir, "delegation")
	if err := ffp.EnsureDir(delegationDir); err != nil {
		return nil, fmt.Errorf("failed to create delegation log directory: %w", err)
	}

	logPath := filepath.Join(delegationDir, requestID+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open delegation log file: %w", err)
	}

	zl := zerolog.New(f).With().Timestamp().
		Str("request_id", requestID).
		Str("agent_name", agentName).
		Logger()

	return &DelegationFileLogger{
		log:       &Logger{Logger: zl},
		file:      f,
		path:      logPath,
		requestID: requestID,
		agentName: agentName,
	}, nil
}

// Path returns the log file path.
func (l *DelegationFileLogger) Path() string {
	return l.path
}

// Writer returns an io.Writer for the log file, usable by other loggers.
func (l *DelegationFileLogger) Writer() io.Writer {
	return l
}

// Write implements io.Writer. Thread-safe.
func (l *DelegationFileLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return 0, fmt.Errorf("delegation logger closed")
	}
	return l.file.Write(p)
}

// Logger returns the underlying zerolog-based logger.
func (l *DelegationFileLogger) Logger() *Logger {
	return l.log
}

// Close closes the log file. Safe to call multiple times.
func (l *DelegationFileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

//
// Helpers
//

func timestampNow() int64 {
	return time.Now().UnixNano()
}

// TruncateStr truncates a string to maxLen characters.
func TruncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
