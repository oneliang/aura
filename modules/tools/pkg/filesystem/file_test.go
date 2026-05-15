// Package filesystem provides file system tools.
package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tools "github.com/oneliang/aura/tools/pkg"
	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

// Helper function for tests
func newTestTools() (*ReadTool, *WriteTool, *SearchTool, *ListTool) {
	checker := trustedpath.NopChecker()
	return NewReadTool(checker), NewWriteTool(checker), NewSearchTool(checker), NewListTool(checker)
}

// Test ReadTool
func TestReadToolName(t *testing.T) {
	tool, _, _, _ := newTestTools()
	name := tool.Name()
	if name != "file_read" {
		t.Errorf("Name() = %v, want 'file_read'", name)
	}
}

func TestReadToolDescription(t *testing.T) {
	tool, _, _, _ := newTestTools()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "path") {
		t.Error("Description() should mention 'path' parameter")
	}
}

func TestReadToolExecute(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Test successful read
	params := map[string]any{"path": testFile}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	want := fmt.Sprintf("Read %d bytes from %s", len(testContent), testFile)
	if result.Content != want {
		t.Errorf("Execute() result = %q, want %q", result.Content, want)
	}
}

func TestReadToolExecuteMissingPath(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Test missing path parameter
	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Execute() should error when path is missing")
	}
}

func TestReadToolExecuteNonExistentFile(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Test non-existent file
	params := map[string]any{"path": "/non/existent/file.txt"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should return error status for non-existent file, got: %v", result.Status)
	}
}

func TestReadToolExecuteInvalidPathType(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Test invalid path type
	params := map[string]any{"path": 123}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is not a string")
	}
}

func TestReadToolExecuteDirectory(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Test reading a directory (should fail)
	tmpDir := t.TempDir()
	params := map[string]any{"path": tmpDir}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should return error status when path is a directory, got: %v", result.Status)
	}
}

func TestReadToolExecuteImageFile(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Create a temporary test image file (fake PNG content)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")
	testContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D} // PNG magic bytes

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	params := map[string]any{"path": testFile}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify result format
	if !strings.Contains(result.Content, "Image file: ") {
		t.Errorf("Execute() result should contain 'Image file: ', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "DataURI: ") {
		t.Errorf("Execute() result should contain 'DataURI: ', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "data:image/png;base64,") {
		t.Errorf("Execute() result should contain PNG dataURI, got: %s", result.Content)
	}
}

func TestReadToolExecuteJPGImage(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Create a temporary test JPG file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	testContent := []byte("fake jpg content")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	params := map[string]any{"path": testFile}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify result format for JPG
	if !strings.Contains(result.Content, "Image file: ") {
		t.Errorf("Execute() result should contain 'Image file: ', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "data:image/jpeg;base64,") {
		t.Errorf("Execute() result should contain JPEG dataURI, got: %s", result.Content)
	}
}

func TestReadToolExecuteGIFImage(t *testing.T) {
	tool, _, _, _ := newTestTools()
	ctx := context.Background()

	// Create a temporary test GIF file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "animation.gif")
	testContent := []byte("GIF89a")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	params := map[string]any{"path": testFile}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify result format for GIF
	if !strings.Contains(result.Content, "data:image/gif;base64,") {
		t.Errorf("Execute() result should contain GIF dataURI, got: %s", result.Content)
	}
}

// Test WriteTool
func TestWriteToolName(t *testing.T) {
	_, tool, _, _ := newTestTools()
	name := tool.Name()
	if name != "file_write" {
		t.Errorf("Name() = %v, want 'file_write'", name)
	}
}

func TestWriteToolDescription(t *testing.T) {
	_, tool, _, _ := newTestTools()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "path") || !contains(desc, "content") {
		t.Error("Description() should mention 'path' and 'content' parameters")
	}
}

func TestWriteToolRequiresConfirmation(t *testing.T) {
	_, tool, _, _ := newTestTools()
	if !tool.RequiresConfirmation() {
		t.Error("RequiresConfirmation() should return true for write operations")
	}
}

func TestWriteToolExecute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")
	testContent := "Test content"

	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{
		"path":    testFile,
		"content": testContent,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Content != "Successfully wrote to "+testFile {
		t.Errorf("Execute() result = %v, want 'Successfully wrote to %s'", result.Content, testFile)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("File content = %v, want %v", string(content), testContent)
	}
}

func TestWriteToolExecuteCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
	testContent := "Nested content"

	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{
		"path":    testFile,
		"content": testContent,
	}

	_, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should have been created")
	}
}

func TestWriteToolExecuteMissingPath(t *testing.T) {
	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"content": "test"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is missing")
	}
}

func TestWriteToolExecuteMissingContent(t *testing.T) {
	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": "/tmp/test.txt"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when content is missing")
	}
}

func TestWriteToolExecuteInvalidPathType(t *testing.T) {
	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": 123, "content": "test"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is not a string")
	}
}

func TestWriteToolExecuteInvalidContentType(t *testing.T) {
	_, tool, _, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": "/tmp/test.txt", "content": 123}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when content is not a string")
	}
}

// Test SearchTool
func TestSearchToolName(t *testing.T) {
	_, _, tool, _ := newTestTools()
	name := tool.Name()
	if name != "file_search" {
		t.Errorf("Name() = %v, want 'file_search'", name)
	}
}

func TestSearchToolDescription(t *testing.T) {
	_, _, tool, _ := newTestTools()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "path") || !contains(desc, "pattern") {
		t.Error("Description() should mention 'path' and 'pattern' parameters")
	}
}

func TestSearchToolExecute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("Hello World"), 0644)
	os.WriteFile(file2, []byte("Hello Go"), 0644)
	os.WriteFile(file3, []byte("Goodbye World"), 0644)

	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	// Search for "Hello"
	params := map[string]any{
		"path":    tmpDir,
		"pattern": "Hello",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Should find file1 and file2
	if !contains(result.Content, "2 files") {
		t.Errorf("Result should mention 2 files, got: %s", result.Content)
	}
	if !contains(result.Content, "file1.txt") || !contains(result.Content, "file2.txt") {
		t.Errorf("Result should contain file1.txt and file2.txt, got: %s", result.Content)
	}
}

func TestSearchToolExecuteNoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Hello"), 0644)

	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{
		"path":    tmpDir,
		"pattern": "NotFound",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !contains(result.Content, "0 files") {
		t.Errorf("Result should mention 0 files, got: %s", result.Content)
	}
}

func TestSearchToolExecuteMissingPath(t *testing.T) {
	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"pattern": "test"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is missing")
	}
}

func TestSearchToolExecuteMissingPattern(t *testing.T) {
	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": "/tmp"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when pattern is missing")
	}
}

func TestSearchToolExecuteInvalidPathType(t *testing.T) {
	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": 123, "pattern": "test"}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is not a string")
	}
}

func TestSearchToolExecuteNonExistentPath(t *testing.T) {
	_, _, tool, _ := newTestTools()
	ctx := context.Background()

	params := map[string]any{
		"path":    "/non/existent/path",
		"pattern": "test",
	}
	// SearchTool handles non-existent path gracefully by returning empty results
	// (filepath.Walk calls the error callback which returns nil to continue)
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() should handle non-existent path gracefully, got error: %v", err)
	}
	if !contains(result.Content, "0 files") {
		t.Errorf("Execute() should return '0 files' for non-existent path, got: %s", result.Content)
	}
}

// Test ListTool
func TestListToolName(t *testing.T) {
	_, _, _, tool := newTestTools()
	name := tool.Name()
	if name != "file_list" {
		t.Errorf("Name() = %v, want 'file_list'", name)
	}
}

func TestListToolDescription(t *testing.T) {
	_, _, _, tool := newTestTools()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "path") {
		t.Error("Description() should mention 'path' parameter")
	}
}

func TestListToolExecute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files and directories
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	_, _, _, tool := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": tmpDir}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !contains(result.Content, "Found 1 files and 1 directories in") {
		t.Errorf("Result should contain file and directory count, got: %s", result.Content)
	}
}

func TestListToolExecuteEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, _, tool := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": tmpDir}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !contains(result.Content, "Found 0 files and 0 directories in") {
		t.Error("Result should contain 'Found 0 files and 0 directories in' for empty directory")
	}
}

func TestListToolExecuteMissingPath(t *testing.T) {
	_, _, _, tool := newTestTools()
	ctx := context.Background()

	params := map[string]any{}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is missing")
	}
}

func TestListToolExecuteInvalidPathType(t *testing.T) {
	_, _, _, tool := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": 123}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when path is not a string")
	}
}

func TestListToolExecuteNonExistentPath(t *testing.T) {
	_, _, _, tool := newTestTools()
	ctx := context.Background()

	params := map[string]any{"path": "/non/existent/path"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should return error status for non-existent path, got: %v", result.Status)
	}
}

// Test AllTools
func TestAllTools(t *testing.T) {
	tools := AllTools()

	// Should return 4 tools
	if len(tools) != 4 {
		t.Errorf("AllTools() returned %d tools, want 4", len(tools))
	}

	// Verify all tool names are present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	expectedNames := []string{"file_read", "file_write", "file_search", "file_list"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("AllTools() missing tool: %s", name)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
