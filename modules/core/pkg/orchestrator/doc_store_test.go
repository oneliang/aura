package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestDocStore creates a temporary directory and document store for testing.
func setupTestDocStore(t *testing.T) (*DocStore, string, func()) {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "doc-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewDocStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create doc store: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return store, tempDir, cleanup
}

// createTestDoc creates a test document with the given ID.
func createTestDoc(id, from, to, title, body string, status DocStatus) *CollaboDoc {
	return &CollaboDoc{
		ID:        id,
		Type:      DocTypeTaskAssign,
		From:      from,
		To:        to,
		Priority:  PriorityNormal,
		Status:    status,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestNewDocStore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "doc-store-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		store, err := NewDocStore(tempDir)
		if err != nil {
			t.Fatalf("NewDocStore() error = %v", err)
		}
		if store == nil {
			t.Fatal("NewDocStore() returned nil")
		}
		if store.docsDir != tempDir {
			t.Errorf("docsDir = %q, want %q", store.docsDir, tempDir)
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "doc-store-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		newDir := filepath.Join(tempDir, "new-dir")
		store, err := NewDocStore(newDir)
		if err != nil {
			t.Fatalf("NewDocStore() error = %v", err)
		}
		if store == nil {
			t.Fatal("NewDocStore() returned nil")
		}

		// Verify directory was created
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			t.Error("Directory was not created")
		}
	})

	t.Run("empty directory path", func(t *testing.T) {
		_, err := NewDocStore("")
		if err == nil {
			t.Error("NewDocStore() with empty path should return error")
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		// Use a path that should fail on most systems
		invalidPath := "/root/protected/invalid/path/that/should/fail"
		_, err := NewDocStore(invalidPath)
		if err == nil {
			os.RemoveAll(invalidPath)
			t.Error("NewDocStore() with invalid path should return error")
		}
	})
}

func TestDocStoreFilename(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	docID := "test-doc-123"
	expected := filepath.Join(store.docsDir, docID+".md")
	got := store.filename(docID)

	if got != expected {
		t.Errorf("filename() = %q, want %q", got, expected)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"safe filename", "test-doc-123", "test-doc-123"},
		{"slash", "test/doc", "test-doc"},
		{"backslash", "test\\doc", "test-doc"},
		{"parent dir", "test..doc", "test-doc"},
		{"colon", "test:doc", "test-doc"},
		{"asterisk", "test*doc", "test-doc"},
		{"question mark", "test?doc", "test-doc"},
		{"quote", "test\"doc", "test-doc"},
		{"less than", "test<doc", "test-doc"},
		{"greater than", "test>doc", "test-doc"},
		{"pipe", "test|doc", "test-doc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDocStoreSave(t *testing.T) {
	t.Run("save single doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := createTestDoc("test-doc-1", "agent-1", "agent-2", "Test Doc", "Body content", DocStatusPending)

		err := store.Save(doc)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file was created
		filePath := store.filename(doc.ID)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Document file was not created")
		}
	})

	t.Run("save nil doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		err := store.Save(nil)
		if err == nil {
			t.Error("Save(nil) should return error")
		}
	})

	t.Run("save doc without ID", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := &CollaboDoc{
			Title: "No ID Doc",
		}

		err := store.Save(doc)
		if err == nil {
			t.Error("Save() with empty ID should return error")
		}
	})

	t.Run("overwrite existing doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := createTestDoc("test-doc-overwrite", "agent-1", "agent-2", "Original", "Original body", DocStatusPending)
		if err := store.Save(doc); err != nil {
			t.Fatalf("First Save() error = %v", err)
		}

		// Modify and save again
		doc.Title = "Updated"
		doc.Body = "Updated body"
		if err := store.Save(doc); err != nil {
			t.Fatalf("Second Save() error = %v", err)
		}

		// Load and verify
		loaded, err := store.Load(doc.ID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Title != "Updated" {
			t.Errorf("Title = %q, want 'Updated'", loaded.Title)
		}
		if loaded.Body != "Updated body" {
			t.Errorf("Body = %q, want 'Updated body'", loaded.Body)
		}
	})
}

func TestDocStoreLoad(t *testing.T) {
	t.Run("load existing doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := createTestDoc("test-doc-load", "agent-1", "agent-2", "Load Test", "Load body", DocStatusCompleted)
		if err := store.Save(doc); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := store.Load(doc.ID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.ID != doc.ID {
			t.Errorf("ID = %q, want %q", loaded.ID, doc.ID)
		}
		if loaded.Status != DocStatusCompleted {
			t.Errorf("Status = %q, want %q", loaded.Status, DocStatusCompleted)
		}
	})

	t.Run("load non-existent doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		_, err := store.Load("non-existent-doc")
		if err == nil {
			t.Error("Load() non-existent doc should return error")
		}
	})

	t.Run("load doc without ID", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		_, err := store.Load("")
		if err == nil {
			t.Error("Load(\"\") should return error")
		}
	})
}

func TestDocStoreList(t *testing.T) {
	t.Run("list all docs", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		// Create 3 docs with different statuses
		docs := []*CollaboDoc{
			createTestDoc("doc-1", "agent-1", "agent-2", "Doc 1", "Body 1", DocStatusPending),
			createTestDoc("doc-2", "agent-1", "agent-2", "Doc 2", "Body 2", DocStatusInProgress),
			createTestDoc("doc-3", "agent-1", "agent-2", "Doc 3", "Body 3", DocStatusCompleted),
		}

		for _, doc := range docs {
			if err := store.Save(doc); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		allDocs, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(allDocs) != 3 {
			t.Errorf("Expected 3 docs, got %d", len(allDocs))
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		// Create 3 docs with different statuses
		docs := []*CollaboDoc{
			createTestDoc("doc-pending", "agent-1", "agent-2", "Pending", "Body", DocStatusPending),
			createTestDoc("doc-progress", "agent-1", "agent-2", "InProgress", "Body", DocStatusInProgress),
			createTestDoc("doc-completed", "agent-1", "agent-2", "Completed", "Body", DocStatusCompleted),
		}

		for _, doc := range docs {
			if err := store.Save(doc); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		// Filter by completed status
		completedDocs, err := store.List(DocStatusCompleted)
		if err != nil {
			t.Fatalf("List(completed) error = %v", err)
		}
		if len(completedDocs) != 1 {
			t.Errorf("Expected 1 completed doc, got %d", len(completedDocs))
		}
		if completedDocs[0].ID != "doc-completed" {
			t.Errorf("Expected doc-completed, got %s", completedDocs[0].ID)
		}
	})

	t.Run("list empty store", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		docs, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		// nil or empty slice is acceptable for empty store
		if len(docs) != 0 {
			t.Errorf("Expected 0 docs, got %d", len(docs))
		}
	})
}

func TestDocStoreDelete(t *testing.T) {
	t.Run("delete existing doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := createTestDoc("test-doc-delete", "agent-1", "agent-2", "Delete Test", "Delete body", DocStatusPending)
		if err := store.Save(doc); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file exists
		filePath := store.filename(doc.ID)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("Document file should exist before delete")
		}

		// Delete doc
		err := store.Delete(doc.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify file is gone
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("Document file should be deleted")
		}

		// Load should return error for deleted doc
		_, err = store.Load(doc.ID)
		if err == nil {
			t.Error("Load() after delete should return error")
		}
	})

	t.Run("delete non-existent doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		err := store.Delete("non-existent-doc")
		if err == nil {
			t.Error("Delete() non-existent doc should return error")
		}
	})

	t.Run("delete doc without ID", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		err := store.Delete("")
		if err == nil {
			t.Error("Delete(\"\") should return error")
		}
	})
}

func TestDocStoreUpdateStatus(t *testing.T) {
	t.Run("update status", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		doc := createTestDoc("test-doc-status", "agent-1", "agent-2", "Status Test", "Status body", DocStatusPending)
		if err := store.Save(doc); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Update status
		err := store.UpdateStatus(doc.ID, DocStatusInProgress, "Starting work")
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}

		// Load and verify
		loaded, err := store.Load(doc.ID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Status != DocStatusInProgress {
			t.Errorf("Status = %q, want %q", loaded.Status, DocStatusInProgress)
		}
		if len(loaded.History) != 1 {
			t.Errorf("Expected 1 history entry, got %d", len(loaded.History))
		}
	})

	t.Run("update status non-existent doc", func(t *testing.T) {
		store, _, cleanup := setupTestDocStore(t)
		defer cleanup()

		err := store.UpdateStatus("non-existent-doc", DocStatusCompleted, "")
		if err == nil {
			t.Error("UpdateStatus() non-existent doc should return error")
		}
	})
}

func TestDocStoreIntegration(t *testing.T) {
	store, _, cleanup := setupTestDocStore(t)
	defer cleanup()

	// Test complete workflow: save, load, list, update, delete
	docID := "integration-doc"

	// Phase 1: Save
	t.Run("save phase", func(t *testing.T) {
		doc := createTestDoc(docID, "orchestrator", "agent-1", "Integration Test", "Test body", DocStatusPending)
		if err := store.Save(doc); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	})

	// Phase 2: Load and verify
	t.Run("load phase", func(t *testing.T) {
		loaded, err := store.Load(docID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.ID != docID {
			t.Errorf("ID = %q, want %q", loaded.ID, docID)
		}
		if loaded.Status != DocStatusPending {
			t.Errorf("Status = %q, want %q", loaded.Status, DocStatusPending)
		}
	})

	// Phase 3: List all
	t.Run("list phase", func(t *testing.T) {
		allDocs, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(allDocs) != 1 {
			t.Errorf("Expected 1 doc, got %d", len(allDocs))
		}
	})

	// Phase 4: Update status
	t.Run("update phase", func(t *testing.T) {
		err := store.UpdateStatus(docID, DocStatusCompleted, "Done")
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}

		loaded, err := store.Load(docID)
		if err != nil {
			t.Fatalf("Load() after update error = %v", err)
		}
		if loaded.Status != DocStatusCompleted {
			t.Errorf("Status = %q, want %q", loaded.Status, DocStatusCompleted)
		}
	})

	// Phase 5: Delete
	t.Run("delete phase", func(t *testing.T) {
		err := store.Delete(docID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = store.Load(docID)
		if err == nil {
			t.Error("Load() after delete should return error")
		}
	})
}
