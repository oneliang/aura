package orchestrator

import (
	"sync"
	"time"
)

// TaskRegistry is an in-memory index mapping doc IDs to their CollaboDoc for fast lookup.
// It also tracks dependency satisfaction.
type TaskRegistry struct {
	mu   sync.RWMutex
	docs map[string]*CollaboDoc
}

// NewTaskRegistry creates a new task registry.
func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		docs: make(map[string]*CollaboDoc),
	}
}

// Register adds or updates a document in the registry.
func (r *TaskRegistry) Register(doc *CollaboDoc) {
	if doc == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ID] = doc
}

// Unregister removes a document from the registry.
func (r *TaskRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.docs, id)
}

// Get retrieves a document by ID.
func (r *TaskRegistry) Get(id string) (*CollaboDoc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, exists := r.docs[id]
	return doc, exists
}

// List returns all documents in the registry.
func (r *TaskRegistry) List() []*CollaboDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docs := make([]*CollaboDoc, 0, len(r.docs))
	for _, doc := range r.docs {
		docs = append(docs, doc)
	}
	return docs
}

// ListByStatus returns documents filtered by status.
func (r *TaskRegistry) ListByStatus(status DocStatus) []*CollaboDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var docs []*CollaboDoc
	for _, doc := range r.docs {
		if doc.Status == status {
			docs = append(docs, doc)
		}
	}
	return docs
}

// ListByAgent returns documents assigned to a specific agent (by To field).
func (r *TaskRegistry) ListByAgent(agentID string) []*CollaboDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var docs []*CollaboDoc
	for _, doc := range r.docs {
		if doc.To == agentID || doc.To == "" || doc.To == "any" {
			docs = append(docs, doc)
		}
	}
	return docs
}

// DepsReady returns true if all dependencies of doc are in DocStatusCompleted.
// The completedIDs map should contain IDs of all completed documents.
func (r *TaskRegistry) DepsReady(doc *CollaboDoc, completedIDs map[string]bool) bool {
	if doc == nil {
		return true
	}

	if len(doc.Dependencies) == 0 {
		return true
	}

	for _, depID := range doc.Dependencies {
		if !completedIDs[depID] {
			return false
		}
	}
	return true
}

// GetCompletedIDs returns a map of all completed document IDs.
func (r *TaskRegistry) GetCompletedIDs() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	completed := make(map[string]bool)
	for _, doc := range r.docs {
		if doc.Status == DocStatusCompleted {
			completed[doc.ID] = true
		}
	}
	return completed
}

// GetPendingByAgent returns pending documents for a specific agent.
func (r *TaskRegistry) GetPendingByAgent(agentID string) []*CollaboDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var docs []*CollaboDoc
	for _, doc := range r.docs {
		if doc.Status == DocStatusPending && (doc.To == agentID || doc.To == "" || doc.To == "any") {
			docs = append(docs, doc)
		}
	}
	return docs
}

// Count returns the total number of documents in the registry.
func (r *TaskRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.docs)
}

// Clear removes all documents from the registry.
func (r *TaskRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs = make(map[string]*CollaboDoc)
}

// SyncFromStore loads all documents from a DocStore into the registry.
func (r *TaskRegistry) SyncFromStore(store *DocStore) error {
	docs, err := store.List()
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, doc := range docs {
		r.docs[doc.ID] = doc
	}
	return nil
}

// GetStaleDocs returns documents that haven't been updated since the threshold.
func (r *TaskRegistry) GetStaleDocs(threshold time.Duration) []*CollaboDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cutoff := time.Now().Add(-threshold)
	var stale []*CollaboDoc

	for _, doc := range r.docs {
		// Only check active documents (pending or in_progress)
		if doc.Status == DocStatusPending || doc.Status == DocStatusInProgress {
			if doc.UpdatedAt.Before(cutoff) {
				stale = append(stale, doc)
			}
		}
	}
	return stale
}
