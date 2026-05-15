package commands

import (
	"testing"
)

func TestGetCommandContext(t *testing.T) {
	ctx := GetCommandContext()
	if ctx == nil {
		t.Error("GetCommandContext() returned nil")
	}
}

func TestSetCommandContext(t *testing.T) {
	// Save original context
	originalCtx := GetCommandContext()
	defer SetCommandContext(originalCtx)

	testCtx := &CommandContext{
		Config: nil,
		Logger: nil,
	}

	SetCommandContext(testCtx)

	ctx := GetCommandContext()
	if ctx == nil {
		t.Fatal("GetCommandContext() returned nil after SetCommandContext")
	}

	if ctx != testCtx {
		t.Error("SetCommandContext did not set the context correctly")
	}
}

func TestDefaultCommandContext(t *testing.T) {
	ctx := DefaultCommandContext()
	if ctx == nil {
		t.Fatal("DefaultCommandContext() returned nil")
	}

	// Check that default factories are set
	if ctx.ConfigLoader == nil {
		t.Error("ConfigLoader is nil")
	}

	if ctx.HomeDirProvider == nil {
		t.Error("HomeDirProvider is nil")
	}

	if ctx.KnowledgeStoreFactory == nil {
		t.Error("KnowledgeStoreFactory is nil")
	}

	if ctx.PermissionManagerFactory == nil {
		t.Error("PermissionManagerFactory is nil")
	}
}

func TestCommandContext_Concurrent(t *testing.T) {
	// Test concurrent access to global context
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			ctx := GetCommandContext()
			if ctx == nil {
				t.Error("GetCommandContext() returned nil in goroutine")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCommandContext_InitializedByDefault(t *testing.T) {
	// The init() function should have initialized globalCmdCtx
	ctx := GetCommandContext()
	if ctx == nil {
		t.Error("Global command context was not initialized by init()")
	}
}
