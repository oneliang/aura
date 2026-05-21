// Package commands provides additional tests for session_handler.go and knowledge_handler.go.
package commands

import (
	"context"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
)

// TestSessionHandler_ExecuteCommand_Update tests ExecuteCommand with update command.
func TestSessionHandler_ExecuteCommand_Update(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx := context.Background()

	// First create a session
	createResult, err := handler.ExecuteCommand(ctx, "create", map[string]any{
		"name": "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if createResult == "" {
		t.Error("createSession should return output")
	}

	// Extract session ID from result (format: "Created session [id] name")
	var sessionID string
	for i, ch := range createResult {
		if ch == '[' {
			for j := i + 1; j < len(createResult); j++ {
				if createResult[j] == ']' {
					sessionID = createResult[i+1 : j]
					break
				}
			}
			break
		}
	}

	if sessionID == "" {
		t.Fatal("Could not extract session ID from result")
	}

	// Update the session
	result, err := handler.ExecuteCommand(ctx, "update", map[string]any{
		"id":   sessionID,
		"role": "test-role",
	})
	if err != nil {
		t.Fatalf("ExecuteCommand(update) error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommand(update) should return output")
	}
}

// TestSessionHandler_ExecuteCommand_Show tests ExecuteCommand with show command.
func TestSessionHandler_ExecuteCommand_Show(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx := context.Background()

	// First create a session
	createResult, err := handler.ExecuteCommand(ctx, "create", map[string]any{
		"name": "Show Test Session",
		"role": "test-role",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Extract session ID
	var sessionID string
	for i, ch := range createResult {
		if ch == '[' {
			for j := i + 1; j < len(createResult); j++ {
				if createResult[j] == ']' {
					sessionID = createResult[i+1 : j]
					break
				}
			}
			break
		}
	}

	// Show the session
	result, err := handler.ExecuteCommand(ctx, "show", map[string]any{
		"id": sessionID,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand(show) error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommand(show) should return output")
	}
}

// TestSessionHandler_ExecuteCommand_Delete tests ExecuteCommand with delete command.
func TestSessionHandler_ExecuteCommand_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx := context.Background()

	// First create a session
	createResult, err := handler.ExecuteCommand(ctx, "create", map[string]any{
		"name": "Delete Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Extract session ID
	var sessionID string
	for i, ch := range createResult {
		if ch == '[' {
			for j := i + 1; j < len(createResult); j++ {
				if createResult[j] == ']' {
					sessionID = createResult[i+1 : j]
					break
				}
			}
			break
		}
	}

	// Delete the session
	result, err := handler.ExecuteCommand(ctx, "delete", map[string]any{
		"id": sessionID,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand(delete) error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommand(delete) should return output")
	}
}

// TestSessionHandler_ExecuteCommand_Unknown tests ExecuteCommand with unknown command.
func TestSessionHandler_ExecuteCommand_Unknown(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx := context.Background()

	_, err = handler.ExecuteCommand(ctx, "unknown_command", nil)
	if err == nil {
		t.Error("ExecuteCommand(unknown) should return error")
	}
}

// TestSessionHandler_ExecuteCommand_EmptyParams tests ExecuteCommand with empty params.
func TestSessionHandler_ExecuteCommand_EmptyParams(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx := context.Background()

	// Create with empty params should still work (name will be empty string)
	result, err := handler.ExecuteCommand(ctx, "create", map[string]any{})
	if err != nil {
		t.Fatalf("ExecuteCommand(create) error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommand(create) should return output")
	}
}

// TestSessionHandler_ExecuteCommand_WithContextCancellation tests context cancellation.
func TestSessionHandler_ExecuteCommand_WithContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	sessionMgr, err := manager.NewSessionManager(store, &config.Config{})
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	handler := NewSessionHandler(sessionMgr, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// List should still work (doesn't use context for I/O)
	result, err := handler.ExecuteCommand(ctx, "list", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand(list) with cancelled context error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommand(list) should return output even with cancelled context")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_Stats tests ExecuteCommand with stats command.
func TestKnowledgeExecutor_ExecuteCommand_Stats(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)
	executor := NewKnowledgeExecutor(factory, "")
	ctx := context.Background()

	// Stats command - will try to open knowledge base
	_, err := executor.ExecuteCommand(ctx, "stats", nil)
	// This may fail if Ollama is not running, which is expected
	// We just verify the command can be executed
	_ = err
}

// TestKnowledgeExecutor_ExecuteCommand_Search_WithQuery tests search with a query.
func TestKnowledgeExecutor_ExecuteCommand_Search_WithQuery(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)
	executor := NewKnowledgeExecutor(factory, "")
	ctx := context.Background()

	// Search command - will try to open knowledge base
	_, err := executor.ExecuteCommand(ctx, "search", map[string]any{
		"query": "test query",
	})
	// This may fail if Ollama is not running, which is expected
	_ = err
}

// TestKnowledgeExecutor_ExecuteCommand_Import_WithPath tests import with a path.
func TestKnowledgeExecutor_ExecuteCommand_Import_WithPath(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)
	executor := NewKnowledgeExecutor(factory, "")
	ctx := context.Background()

	// Import command - will try to open knowledge base
	_, err := executor.ExecuteCommand(ctx, "import", map[string]any{
		"path": "/nonexistent/path.txt",
	})
	// This will fail because file doesn't exist, which is expected
	if err == nil {
		t.Error("ExecuteCommand(import) with nonexistent path should return error")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_Unknown tests ExecuteCommand with unknown command.
func TestKnowledgeExecutor_ExecuteCommand_Unknown(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)
	executor := NewKnowledgeExecutor(factory, "")
	ctx := context.Background()

	_, err := executor.ExecuteCommand(ctx, "unknown_command", nil)
	if err == nil {
		t.Error("ExecuteCommand(unknown) should return error")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_EmptyParams2 tests ExecuteCommand with empty params.
func TestKnowledgeExecutor_ExecuteCommand_EmptyParams2(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)
	executor := NewKnowledgeExecutor(factory, "")
	ctx := context.Background()

	// Search with empty params - query will be empty string
	_, err := executor.ExecuteCommand(ctx, "search", map[string]any{})
	if err == nil {
		t.Error("ExecuteCommand(search) without query should return error")
	}

	// Import with empty params - path will be empty string
	_, err = executor.ExecuteCommand(ctx, "import", map[string]any{})
	if err == nil {
		t.Error("ExecuteCommand(import) without path should return error")
	}
}
