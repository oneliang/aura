// Package jsonl provides JSONL-based storage for messages.
package jsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/user"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/oneliang/aura/storage/pkg/message"
)

const (
	// initialScannerBufferSize is the initial buffer size for bufio.Scanner (64 KB).
	initialScannerBufferSize = 64 * 1024
	// maxScannerCapacity is the maximum buffer size for bufio.Scanner (10 MB).
	// Prevents "token too long" errors when a single JSONL line exceeds the default 64 KB limit.
	maxScannerCapacity = 10 * 1024 * 1024
)

// MessageStore provides JSONL-based storage for messages, partitioned by sessionID.
type MessageStore struct {
	dataDir string
	mu      sync.Mutex
}

// NewMessageStore creates a new JSONL message store in the specified directory.
func NewMessageStore(dataDir string) (*MessageStore, error) {
	if err := ffp.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	return &MessageStore{dataDir: dataDir}, nil
}

// filePath returns the path to a session's JSONL file.
func (s *MessageStore) filePath(sessionID string) string {
	return filepath.Join(s.dataDir, sessionID+".jsonl")
}

// Append appends a message to the session's JSONL file.
// Validates that the message userID matches the session owner (in multi-user mode).
func (s *MessageStore) Append(ctx context.Context, msg *message.Message) error {
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.filePath(msg.SessionID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Get retrieves messages from a session file, returning the last limit messages.
// If limit <= 0, all messages are returned.
// In multi-user mode (userID != ""), verifies the session belongs to the user.
func (s *MessageStore) Get(ctx context.Context, sessionID string, limit int, userID string) ([]message.Message, error) {
	path := s.filePath(sessionID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []message.Message{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	var messages []message.Message
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, initialScannerBufferSize), maxScannerCapacity)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg message.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // 跳过格式错误的行
		}
		// In multi-user mode, verify message belongs to user
		if !user.HasOwnership(userID, msg.UserID) {
			continue
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading session file: %w", err)
	}

	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// DeleteSession removes all messages for a session.
func (s *MessageStore) DeleteSession(sessionID string) error {
	err := os.Remove(s.filePath(sessionID))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}
	return nil
}

// TruncateSession truncates a session's JSONL file to zero length (keeps the file but clears contents).
func (s *MessageStore) TruncateSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.filePath(sessionID), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file for truncation: %w", err)
	}
	defer file.Close()
	return nil
}
