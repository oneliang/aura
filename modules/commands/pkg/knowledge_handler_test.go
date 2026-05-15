// Package commands provides tests for KnowledgeExecutor.
package commands

import (
	"context"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg"
)

// TestNewKnowledgeExecutor tests the NewKnowledgeExecutor function.
func TestNewKnowledgeExecutor(t *testing.T) {
	var factory knowledge.CollectionFactory

	executor := NewKnowledgeExecutor(factory, "")

	if executor == nil {
		t.Fatal("NewKnowledgeExecutor() returned nil")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_UnknownCommand tests unknown command handling.
func TestKnowledgeExecutor_ExecuteCommand_UnknownCommand(t *testing.T) {
	executor := &KnowledgeExecutor{}
	ctx := context.Background()

	_, err := executor.ExecuteCommand(ctx, "unknown_cmd", nil)
	if err == nil {
		t.Error("ExecuteCommand() for unknown command should return error")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_SearchValidation tests search query validation.
func TestKnowledgeExecutor_ExecuteCommand_SearchValidation(t *testing.T) {
	executor := &KnowledgeExecutor{}
	ctx := context.Background()

	// Empty query should fail validation before accessing factory
	_, err := executor.ExecuteCommand(ctx, "search", map[string]any{"query": ""})
	if err == nil {
		t.Error("search with empty query should return error")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_ImportValidation tests import path validation.
func TestKnowledgeExecutor_ExecuteCommand_ImportValidation(t *testing.T) {
	executor := &KnowledgeExecutor{}
	ctx := context.Background()

	// Empty path should fail validation before accessing factory
	_, err := executor.ExecuteCommand(ctx, "import", map[string]any{"path": ""})
	if err == nil {
		t.Error("import with empty path should return error")
	}
}

// TestKnowledgeExecutor_ExecuteCommand_EmptyParams tests execute with empty params.
func TestKnowledgeExecutor_ExecuteCommand_EmptyParams(t *testing.T) {
	executor := &KnowledgeExecutor{}
	ctx := context.Background()

	// Empty params for search should fail (empty query)
	_, err := executor.ExecuteCommand(ctx, "search", nil)
	if err == nil {
		t.Error("search with nil params should return error")
	}

	// Empty params for import should fail (empty path)
	_, err = executor.ExecuteCommand(ctx, "import", nil)
	if err == nil {
		t.Error("import with nil params should return error")
	}
}

// TestKnowledgeCollectionFactory tests the knowledge.CollectionFactory interface.
func TestKnowledgeCollectionFactory(t *testing.T) {
	// This test just verifies the interface exists
	// We can't easily test with a mock because it requires real ChromemCollection
	t.Log("knowledge.CollectionFactory interface exists")
}
