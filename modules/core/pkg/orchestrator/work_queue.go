package orchestrator

import (
	"sort"
	"sync"
)

// WorkQueue is a priority-ordered, concurrency-safe queue of collaboration documents.
// Priority ordering: urgent > normal > low.
// Documents within the same priority level are processed in FIFO order.
type WorkQueue struct {
	agentID string
	urgent  []*CollaboDoc
	normal  []*CollaboDoc
	low     []*CollaboDoc
	mu      sync.RWMutex
}

// NewWorkQueue creates a new work queue for the given agent.
func NewWorkQueue(agentID string) *WorkQueue {
	return &WorkQueue{
		agentID: agentID,
		urgent:  make([]*CollaboDoc, 0),
		normal:  make([]*CollaboDoc, 0),
		low:     make([]*CollaboDoc, 0),
	}
}

// Enqueue adds a document to the appropriate priority queue.
// Idempotent: if the document is already in the queue, it is not added again.
func (q *WorkQueue) Enqueue(doc *CollaboDoc) {
	if doc == nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if already enqueued (by ID)
	if q.contains(doc.ID) {
		return
	}

	switch doc.Priority {
	case PriorityUrgent:
		q.urgent = append(q.urgent, doc)
	case PriorityLow:
		q.low = append(q.low, doc)
	default:
		q.normal = append(q.normal, doc)
	}
}

// Dequeue pops the highest-priority pending document.
// Returns nil, false if the queue is empty.
func (q *WorkQueue) Dequeue() (*CollaboDoc, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check urgent first
	if len(q.urgent) > 0 {
		doc := q.urgent[0]
		q.urgent = q.urgent[1:]
		return doc, true
	}

	// Then normal
	if len(q.normal) > 0 {
		doc := q.normal[0]
		q.normal = q.normal[1:]
		return doc, true
	}

	// Then low
	if len(q.low) > 0 {
		doc := q.low[0]
		q.low = q.low[1:]
		return doc, true
	}

	return nil, false
}

// Remove removes a document from any priority level.
// Used when a document is completed, rejected, or cancelled.
func (q *WorkQueue) Remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.urgent = q.filterByID(q.urgent, id)
	q.normal = q.filterByID(q.normal, id)
	q.low = q.filterByID(q.low, id)
}

// Len returns total pending items in the queue.
func (q *WorkQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.urgent) + len(q.normal) + len(q.low)
}

// Peek returns the next document without removing it.
// Returns nil if the queue is empty.
func (q *WorkQueue) Peek() *CollaboDoc {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.urgent) > 0 {
		return q.urgent[0]
	}
	if len(q.normal) > 0 {
		return q.normal[0]
	}
	if len(q.low) > 0 {
		return q.low[0]
	}
	return nil
}

// Snapshot returns a copy of all documents in priority order (for status display).
func (q *WorkQueue) Snapshot() []*CollaboDoc {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*CollaboDoc, 0, q.Len())
	result = append(result, q.urgent...)
	result = append(result, q.normal...)
	result = append(result, q.low...)
	return result
}

// Stats returns queue statistics.
func (q *WorkQueue) Stats() (urgent, normal, low int) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.urgent), len(q.normal), len(q.low)
}

// contains checks if a document ID is already in any queue (requires lock held).
func (q *WorkQueue) contains(id string) bool {
	for _, doc := range q.urgent {
		if doc.ID == id {
			return true
		}
	}
	for _, doc := range q.normal {
		if doc.ID == id {
			return true
		}
	}
	for _, doc := range q.low {
		if doc.ID == id {
			return true
		}
	}
	return false
}

// filterByID removes a document by ID from a slice.
func (q *WorkQueue) filterByID(docs []*CollaboDoc, id string) []*CollaboDoc {
	result := make([]*CollaboDoc, 0, len(docs))
	for _, doc := range docs {
		if doc.ID != id {
			result = append(result, doc)
		}
	}
	return result
}

// SortByPriority sorts a slice of documents by priority (urgent > normal > low).
func SortByPriority(docs []*CollaboDoc) []*CollaboDoc {
	priorityOrder := map[Priority]int{
		PriorityUrgent: 0,
		PriorityNormal: 1,
		PriorityLow:    2,
	}

	sort.Slice(docs, func(i, j int) bool {
		return priorityOrder[docs[i].Priority] < priorityOrder[docs[j].Priority]
	})

	return docs
}
