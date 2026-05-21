package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoplsClient_Language(t *testing.T) {
	client := NewGoplsClient(".")
	if client.Language() != "go" {
		t.Errorf("Language() = %q, want %q", client.Language(), "go")
	}
}

func TestGoplsClient_IsAvailable(t *testing.T) {
	client := NewGoplsClient(".")
	available := client.IsAvailable()
	t.Logf("gopls available: %v", available)

	// Check consistency with exec.LookPath
	_, err := os.Stat(client.serverPath)
	if err == nil && !available {
		t.Error("serverPath exists but IsAvailable returns false")
	}
}

func TestGoplsClient_Execute_Symbols(t *testing.T) {
	// Skip if gopls not available
	client := NewGoplsClient(".")
	if !client.IsAvailable() {
		t.Skip("gopls not available")
	}

	// Use current test file as target
	testFile := filepath.Join(".", "gopls_test.go")

	ctx := context.Background()
	result, err := client.Execute(ctx, OpSymbols, Params{File: testFile})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Symbols output should contain function names
	if result.Error != "" {
		t.Logf("Result error: %s", result.Error)
	}
	t.Logf("Symbols output: %s", result.Content)
}

func TestGoplsClient_Execute_Diagnostics(t *testing.T) {
	client := NewGoplsClient(".")
	if !client.IsAvailable() {
		t.Skip("gopls not available")
	}

	// Use current test file
	testFile := filepath.Join(".", "gopls_test.go")

	ctx := context.Background()
	result, err := client.Execute(ctx, OpDiagnostics, Params{File: testFile})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	t.Logf("Diagnostics output: %s", result.Content)
}

func TestGoplsClient_Execute_Definition(t *testing.T) {
	client := NewGoplsClient(".")
	if !client.IsAvailable() {
		t.Skip("gopls not available")
	}

	// Use current test file with line/col
	testFile := filepath.Join(".", "gopls_test.go")

	ctx := context.Background()
	result, err := client.Execute(ctx, OpDefinition, Params{
		File:   testFile,
		Line:   6,  // TestGoplsClient_Language function
		Column: 6,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	t.Logf("Definition output: %s", result.Content)
}

func TestGoplsClient_Execute_InvalidOperation(t *testing.T) {
	client := NewGoplsClient(".")
	if !client.IsAvailable() {
		t.Skip("gopls not available")
	}

	ctx := context.Background()
	_, err := client.Execute(ctx, Operation("invalid"), Params{File: "test.go"})
	if err == nil {
		t.Error("Execute should return error for invalid operation")
	}
}

func TestGoplsClient_Execute_MissingParams(t *testing.T) {
	client := NewGoplsClient(".")
	if !client.IsAvailable() {
		t.Skip("gopls not available")
	}

	ctx := context.Background()

	// Test missing file
	_, err := client.Execute(ctx, OpSymbols, Params{})
	if err == nil {
		t.Error("Execute should return error for missing file")
	}

	// Test missing line/col for definition
	_, err = client.Execute(ctx, OpDefinition, Params{File: "test.go"})
	if err == nil {
		t.Error("Execute should return error for missing line/col for definition")
	}

	// Test missing newName for rename
	_, err = client.Execute(ctx, OpRename, Params{File: "test.go", Line: 1, Column: 1})
	if err == nil {
		t.Error("Execute should return error for missing newName for rename")
	}
}