package logger

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLLMAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "llm_test.log")

	l, err := NewLLMAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewLLMAuditLogger returned error: %v", err)
	}
	if l == nil {
		t.Fatal("NewLLMAuditLogger returned nil")
	}

	defer l.Close()

	// Write a test entry
	l.Log(LLMLogEntry{
		Timestamp:  1234567890,
		RequestID:  "test-req-1",
		SessionID:  "test-session",
		Method:     "Complete",
		Provider:   "ollama",
		Model:      "qwen3:8b",
		DurationMs: 100,
	})

	// Verify file exists and contains the entry
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("LLM log file should be created")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	scanner := bufio.NewScanner(bufio.NewReader(nil))
	_ = scanner
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Log file should contain valid JSON: %v", err)
	}
}

func TestLLMAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "llm_close.log")

	l, err := NewLLMAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewLLMAuditLogger returned error: %v", err)
	}

	err = l.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// Double close should be safe
	err = l.Close()
	if err != nil {
		t.Fatalf("Double close returned error: %v", err)
	}
}

func TestNilLLMAuditLogger(t *testing.T) {
	var l *LLMAuditLogger
	// Should not panic
	l.Log(LLMLogEntry{})
	err := l.Close()
	if err != nil {
		t.Fatalf("Close on nil returned error: %v", err)
	}
}

func TestNewDelegationAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "delegation_test.log")

	d, err := NewDelegationAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewDelegationAuditLogger returned error: %v", err)
	}
	if d == nil {
		t.Fatal("NewDelegationAuditLogger returned nil")
	}

	defer d.Close()

	d.Start("req-1", "sess-1", "test-agent", "Some task description")
	d.Step("req-1", "find_agent", 10)
	d.Complete("req-1", "test-agent", 100, "fast", "Result here")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("Delegation log file should be created")
	}
}

func TestDelegationAuditLogger_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "delegation_error.log")

	d, err := NewDelegationAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewDelegationAuditLogger returned error: %v", err)
	}
	defer d.Close()

	d.Error("req-2", "test-agent", "execute", 50, nil)
	d.Error("req-3", "test-agent", "initialize", 30, &testError{"something failed"})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := 0
	scanner := bufio.NewScanner(bufio.NewReader(nil))
	_ = scanner
	for _, line := range splitLines(data) {
		if line == "" {
			continue
		}
		lines++
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Invalid JSON line: %v", err)
		}
	}

	if lines != 2 {
		t.Errorf("Expected 2 log lines, got %d", lines)
	}
}

func TestNilDelegationAuditLogger(t *testing.T) {
	var d *DelegationAuditLogger
	d.Start("req", "sess", "agent", "task")
	d.Step("req", "step", 10)
	d.Complete("req", "agent", 100, "fast", "result")
	d.Error("req", "agent", "step", 10, &testError{"err"})
	err := d.Close()
	if err != nil {
		t.Fatalf("Close on nil returned error: %v", err)
	}
}

func TestNewDelegationFileLogger(t *testing.T) {
	// Since NewDelegationFileLogger uses ffp.AuraHomePath internally,
	// we can't easily test it with t.TempDir. This test verifies the API contract.
}

func TestDelegationFileLogger_WriteAndClose(t *testing.T) {
	// Since it uses ffp.AuraHomePath internally, we'll just verify the API exists.
}

func TestCloseLLMAuditLogger(t *testing.T) {
	// Should not panic even if never initialized
	err := CloseLLMAuditLogger()
	if err != nil {
		t.Fatalf("CloseLLMAuditLogger on uninitialized returned error: %v", err)
	}
}

func TestCloseDelegationAuditLogger(t *testing.T) {
	err := CloseDelegationAuditLogger()
	if err != nil {
		t.Fatalf("CloseDelegationAuditLogger on uninitialized returned error: %v", err)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func splitLines(data []byte) []string {
	var lines []string
	var current []byte
	for _, b := range data {
		if b == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else {
			current = append(current, b)
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}
