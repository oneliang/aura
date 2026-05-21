package taskstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/shared/pkg/tasks"
)

func setupTestStore(t *testing.T) (*TaskStore, string) {
	t.Helper()
	dir := t.TempDir()
	store := New(dir, "test-session-1")
	return store, dir
}

func TestTaskStore_SaveAndLoad(t *testing.T) {
	store, _ := setupTestStore(t)

	ts := []tasks.Task{
		{ID: 1, Content: "Task A", Status: "pending"},
		{ID: 2, Content: "Task B", Status: "in_progress"},
		{ID: 3, Content: "Task C", Status: "completed", Notes: "Done"},
	}

	if err := store.Save(ts); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(loaded))
	}
	if loaded[0].Content != "Task A" || loaded[0].Status != "pending" {
		t.Fatalf("task 1 mismatch: %+v", loaded[0])
	}
	if loaded[2].Notes != "Done" {
		t.Fatalf("task 3 notes mismatch: %q", loaded[2].Notes)
	}
}

func TestTaskStore_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := New(dir, "no-session")

	ts, err := store.Load()
	if err != nil {
		t.Fatalf("expected nil error for non-existent file, got %v", err)
	}
	if ts != nil {
		t.Fatalf("expected nil tasks, got %+v", ts)
	}
}

func TestTaskStore_Overwrite(t *testing.T) {
	store, _ := setupTestStore(t)

	// First save
	store.Save([]tasks.Task{{ID: 1, Content: "Original", Status: "pending"}})

	// Overwrite
	store.Save([]tasks.Task{
		{ID: 1, Content: "Original", Status: "completed"},
		{ID: 2, Content: "New task", Status: "pending"},
	})

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 tasks after overwrite, got %d", len(loaded))
	}
	if loaded[0].Status != "completed" {
		t.Fatalf("expected task 1 to be completed, got %q", loaded[0].Status)
	}
}

func TestTaskStore_FileExists(t *testing.T) {
	store, dir := setupTestStore(t)

	store.Save([]tasks.Task{{ID: 1, Content: "Test", Status: "pending"}})

	expected := filepath.Join(dir, "test-session-1.tasks.json")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", expected)
	}
}
