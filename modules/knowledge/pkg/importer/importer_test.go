// Package importer provides document importers for the knowledge base.
package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// MockCollection is a mock implementation of storage.Collection for testing.
type MockCollection struct {
	documents []storage.Document
	addError  error
}

func (m *MockCollection) Add(ctx context.Context, documents []storage.Document) error {
	if m.addError != nil {
		return m.addError
	}
	m.documents = append(m.documents, documents...)
	return nil
}

func (m *MockCollection) Query(ctx context.Context, query string, nResults int) ([]storage.Result, error) {
	return nil, nil
}

func (m *MockCollection) Delete(ctx context.Context, ids []string) error {
	return nil
}

func (m *MockCollection) Count(ctx context.Context) (int, error) {
	return len(m.documents), nil
}

func TestNewImporter(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection)

	if importer == nil {
		t.Fatal("New() returned nil")
	}
	if importer.chunkSize != 1000 {
		t.Errorf("default chunkSize = %d, want 1000", importer.chunkSize)
	}
	if importer.overlap != 200 {
		t.Errorf("default overlap = %d, want 200", importer.overlap)
	}
}

func TestNewImporterWithOptions(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection, WithChunkSize(500), WithOverlap(100))

	if importer.chunkSize != 500 {
		t.Errorf("chunkSize = %d, want 500", importer.chunkSize)
	}
	if importer.overlap != 100 {
		t.Errorf("overlap = %d, want 100", importer.overlap)
	}
}

func TestWithChunkSize(t *testing.T) {
	importer := &Importer{chunkSize: 1000}
	option := WithChunkSize(500)
	option(importer)

	if importer.chunkSize != 500 {
		t.Errorf("After WithChunkSize(500), chunkSize = %d, want 500", importer.chunkSize)
	}
}

func TestWithOverlap(t *testing.T) {
	importer := &Importer{overlap: 200}
	option := WithOverlap(50)
	option(importer)

	if importer.overlap != 50 {
		t.Errorf("After WithOverlap(50), overlap = %d, want 50", importer.overlap)
	}
}

func TestImporterImportFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	testContent := "This is test content for the importer."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	count, err := importer.ImportFile(ctx, testFile)
	if err != nil {
		t.Errorf("ImportFile() error = %v", err)
	}
	if count < 1 {
		t.Errorf("ImportFile() imported %d chunks, want >= 1", count)
	}
	if len(collection.documents) != count {
		t.Errorf("Collection has %d documents, want %d", len(collection.documents), count)
	}
}

func TestImporterImportFileNonExistent(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	_, err := importer.ImportFile(ctx, "/non/existent/file.txt")
	if err == nil {
		t.Error("ImportFile() should error for non-existent file")
	}
	if !strings.Contains(err.Error(), "read file") {
		t.Errorf("Error should mention 'read file', got: %v", err)
	}
}

func TestImporterImportFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	err := os.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	importer.ImportFile(ctx, testFile)

	if len(collection.documents) == 0 {
		t.Fatal("ImportFile() should add documents")
	}

	doc := collection.documents[0]
	if doc.Metadata["source"] != "test.md" {
		t.Errorf("Metadata[source] = %v, want 'test.md'", doc.Metadata["source"])
	}
	if doc.Metadata["path"] != testFile {
		t.Errorf("Metadata[path] = %v, want %v", doc.Metadata["path"], testFile)
	}
	if doc.Metadata["type"] != ".md" {
		t.Errorf("Metadata[type] = %v, want '.md'", doc.Metadata["type"])
	}
}

func TestImporterImportDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"test1.md":  "Markdown content 1",
		"test2.txt": "Text content 2",
		"test3.go":  "package main\n\nfunc main() {}",
		"skip.json": `{"skip": true}`, // Should be imported (json is in textExtensions)
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	count, err := importer.ImportDir(ctx, tmpDir)
	if err != nil {
		t.Errorf("ImportDir() error = %v", err)
	}
	if count < 1 {
		t.Errorf("ImportDir() imported %d chunks, want >= 1", count)
	}
}

func TestImporterImportDirNonExistent(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	count, err := importer.ImportDir(ctx, "/non/existent/dir")
	if err != nil {
		t.Errorf("ImportDir() should handle non-existent dir gracefully, got error = %v", err)
	}
	if count != 0 {
		t.Errorf("ImportDir() should return 0 for non-existent dir, got %d", count)
	}
}

func TestImporterImportText(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection)

	ctx := context.Background()
	count, err := importer.ImportText(ctx, "Test text content", "test-source")
	if err != nil {
		t.Errorf("ImportText() error = %v", err)
	}
	if count < 1 {
		t.Errorf("ImportText() imported %d chunks, want >= 1", count)
	}

	if len(collection.documents) == 0 {
		t.Fatal("ImportText() should add documents")
	}

	doc := collection.documents[0]
	if doc.Metadata["source"] != "test-source" {
		t.Errorf("Metadata[source] = %v, want 'test-source'", doc.Metadata["source"])
	}
}

func TestImporterChunkSmallText(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection, WithChunkSize(1000), WithOverlap(200))

	text := "Small text"
	chunks := importer.chunk(text)

	if len(chunks) != 1 {
		t.Errorf("chunk() small text returned %d chunks, want 1", len(chunks))
	}
	if chunks[0] != "Small text" {
		t.Errorf("chunk() small text = %q, want 'Small text'", chunks[0])
	}
}

func TestImporterChunkLargeText(t *testing.T) {
	// Create text larger than chunk size
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("This is line ")
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString("\n")
	}
	largeText := sb.String()

	collection := &MockCollection{}
	importer := New(collection, WithChunkSize(100), WithOverlap(20))

	chunks := importer.chunk(largeText)

	if len(chunks) < 2 {
		t.Errorf("chunk() large text returned %d chunks, want >= 2", len(chunks))
	}

	// Verify each chunk is within chunk size
	for i, chunk := range chunks {
		if len(chunk) > 150 { // Allow some overhead
			t.Errorf("Chunk %d length = %d, want <= 150", i, len(chunk))
		}
	}
}

func TestImporterChunkOverlap(t *testing.T) {
	// Create text that will require multiple chunks with overlap
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString("Line ")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString(" of test content\n")
	}
	text := sb.String()

	collection := &MockCollection{}
	importer := New(collection, WithChunkSize(50), WithOverlap(10))
	chunks := importer.chunk(text)

	if len(chunks) < 2 {
		t.Errorf("chunk() with overlap returned %d chunks, want >= 2", len(chunks))
	}
}

func TestImporterChunkEmptyText(t *testing.T) {
	collection := &MockCollection{}
	importer := New(collection)

	chunks := importer.chunk("")

	// chunk() returns []string{""} for empty text (one empty chunk)
	// because strings.TrimSpace("") returns ""
	if len(chunks) != 1 {
		t.Errorf("chunk() empty text returned %d chunks, want 1", len(chunks))
	}
	if chunks[0] != "" {
		t.Errorf("chunk() empty text chunk = %q, want ''", chunks[0])
	}
}

func TestImporterCollectionAddError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	err := os.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	collection := &MockCollection{
		addError: context.Canceled,
	}
	importer := New(collection)

	ctx := context.Background()
	_, err = importer.ImportFile(ctx, testFile)
	if err == nil {
		t.Error("ImportFile() should error when collection.Add fails")
	}
	if !strings.Contains(err.Error(), "add documents") {
		t.Errorf("Error should mention 'add documents', got: %v", err)
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".md", true},
		{".txt", true},
		{".go", true},
		{".py", true},
		{".js", true},
		{".ts", true},
		{".java", true},
		{".rs", true},
		{".yaml", true},
		{".yml", true},
		{".json", true},
		{".html", true},
		{".css", true},
		{".sql", true},
		{".rb", true},
		{".sh", true},
		{".toml", true},
		{".cpp", true},
		{".h", true},
		{".c", true},
		{".png", false},
		{".jpg", false},
		{".pdf", false},
		{".exe", false},
		{".bin", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := isTextFile("test" + tt.ext)
			if result != tt.expected {
				t.Errorf("isTextFile(test%s) = %v, want %v", tt.ext, result, tt.expected)
			}
		})
	}
}
