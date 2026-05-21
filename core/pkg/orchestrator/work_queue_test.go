package orchestrator

import (
	"testing"
)

func TestWorkQueue_EnqueueDequeue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	doc := &CollaboDoc{
		ID:       "doc-1",
		Title:    "Test Doc",
		Priority: PriorityNormal,
	}

	q.Enqueue(doc)

	if got := q.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1", got)
	}

	got, ok := q.Dequeue()
	if !ok {
		t.Fatal("Dequeue() returned false, want true")
	}
	if got.ID != doc.ID {
		t.Errorf("Dequeue() ID = %s, want %s", got.ID, doc.ID)
	}

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after dequeue = %d, want 0", got)
	}
}

func TestWorkQueue_PriorityOrdering(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Enqueue in mixed order
	q.Enqueue(&CollaboDoc{ID: "low-1", Priority: PriorityLow, Title: "Low 1"})
	q.Enqueue(&CollaboDoc{ID: "urgent-1", Priority: PriorityUrgent, Title: "Urgent 1"})
	q.Enqueue(&CollaboDoc{ID: "normal-1", Priority: PriorityNormal, Title: "Normal 1"})
	q.Enqueue(&CollaboDoc{ID: "urgent-2", Priority: PriorityUrgent, Title: "Urgent 2"})

	// Should dequeue in priority order: urgent first, then normal, then low
	expected := []string{"urgent-1", "urgent-2", "normal-1", "low-1"}

	for _, wantID := range expected {
		doc, ok := q.Dequeue()
		if !ok {
			t.Fatalf("Dequeue() returned false, want true for %s", wantID)
		}
		if doc.ID != wantID {
			t.Errorf("Dequeue() ID = %s, want %s", doc.ID, wantID)
		}
	}

	if got := q.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
}

func TestWorkQueue_IdempotentEnqueue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	doc := &CollaboDoc{
		ID:       "doc-1",
		Priority: PriorityNormal,
		Title:    "Test",
	}

	q.Enqueue(doc)
	q.Enqueue(doc) // Should not duplicate
	q.Enqueue(doc) // Should not duplicate

	if got := q.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1 (idempotent enqueue)", got)
	}
}

func TestWorkQueue_Remove(t *testing.T) {
	q := NewWorkQueue("test-agent")

	docs := []*CollaboDoc{
		{ID: "doc-1", Priority: PriorityNormal, Title: "Doc 1"},
		{ID: "doc-2", Priority: PriorityUrgent, Title: "Doc 2"},
		{ID: "doc-3", Priority: PriorityLow, Title: "Doc 3"},
	}

	for _, doc := range docs {
		q.Enqueue(doc)
	}

	if got := q.Len(); got != 3 {
		t.Fatalf("Len() = %d, want 3", got)
	}

	q.Remove("doc-2")

	if got := q.Len(); got != 2 {
		t.Errorf("Len() after remove = %d, want 2", got)
	}

	// Verify doc-2 is gone
	for q.Len() > 0 {
		doc, _ := q.Dequeue()
		if doc.ID == "doc-2" {
			t.Error("Remove() did not remove doc-2")
		}
	}
}

func TestWorkQueue_Peek(t *testing.T) {
	q := NewWorkQueue("test-agent")

	doc := &CollaboDoc{
		ID:       "doc-1",
		Priority: PriorityNormal,
		Title:    "Test",
	}

	q.Enqueue(doc)

	// Peek should return same doc without removing
	peeked1 := q.Peek()
	peeked2 := q.Peek()

	if peeked1.ID != doc.ID || peeked2.ID != doc.ID {
		t.Error("Peek() returned wrong document")
	}

	if got := q.Len(); got != 1 {
		t.Errorf("Len() after Peek() = %d, want 1", got)
	}
}

func TestWorkQueue_EmptyDequeue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	_, ok := q.Dequeue()
	if ok {
		t.Error("Dequeue() on empty queue returned true, want false")
	}
}

func TestWorkQueue_Stats(t *testing.T) {
	q := NewWorkQueue("test-agent")

	q.Enqueue(&CollaboDoc{ID: "u1", Priority: PriorityUrgent})
	q.Enqueue(&CollaboDoc{ID: "u2", Priority: PriorityUrgent})
	q.Enqueue(&CollaboDoc{ID: "n1", Priority: PriorityNormal})
	q.Enqueue(&CollaboDoc{ID: "n2", Priority: PriorityNormal})
	q.Enqueue(&CollaboDoc{ID: "n3", Priority: PriorityNormal})
	q.Enqueue(&CollaboDoc{ID: "l1", Priority: PriorityLow})

	urgent, normal, low := q.Stats()

	if urgent != 2 {
		t.Errorf("Stats urgent = %d, want 2", urgent)
	}
	if normal != 3 {
		t.Errorf("Stats normal = %d, want 3", normal)
	}
	if low != 1 {
		t.Errorf("Stats low = %d, want 1", low)
	}
}

func TestWorkQueue_Snapshot(t *testing.T) {
	q := NewWorkQueue("test-agent")

	q.Enqueue(&CollaboDoc{ID: "l1", Priority: PriorityLow})
	q.Enqueue(&CollaboDoc{ID: "u1", Priority: PriorityUrgent})
	q.Enqueue(&CollaboDoc{ID: "n1", Priority: PriorityNormal})

	snapshot := q.Snapshot()

	if len(snapshot) != 3 {
		t.Fatalf("Snapshot() len = %d, want 3", len(snapshot))
	}

	// Should be in priority order
	expected := []string{"u1", "n1", "l1"}
	for i, want := range expected {
		if snapshot[i].ID != want {
			t.Errorf("Snapshot[%d].ID = %s, want %s", i, snapshot[i].ID, want)
		}
	}

	// Original queue should be unchanged
	if got := q.Len(); got != 3 {
		t.Errorf("Len() after Snapshot() = %d, want 3", got)
	}
}

func TestSortByPriority(t *testing.T) {
	docs := []*CollaboDoc{
		{ID: "l1", Priority: PriorityLow},
		{ID: "u1", Priority: PriorityUrgent},
		{ID: "n1", Priority: PriorityNormal},
		{ID: "u2", Priority: PriorityUrgent},
		{ID: "l2", Priority: PriorityLow},
	}

	sorted := SortByPriority(docs)

	expected := []string{"u1", "u2", "n1", "l1", "l2"}
	for i, want := range expected {
		if sorted[i].ID != want {
			t.Errorf("Sorted[%d].ID = %s, want %s", i, sorted[i].ID, want)
		}
	}
}

func TestWorkQueue_ConcurrentEnqueue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Enqueue from multiple goroutines
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			q.Enqueue(&CollaboDoc{
				ID:       string(rune(id)),
				Priority: PriorityNormal,
				Title:    "Test",
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 100; i++ {
		<-done
	}

	if got := q.Len(); got != 100 {
		t.Errorf("Len() after concurrent enqueue = %d, want 100", got)
	}
}

func TestWorkQueue_PeekEmptyQueue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	doc := q.Peek()
	if doc != nil {
		t.Errorf("Peek() on empty queue = %v, want nil", doc)
	}
}

func TestWorkQueue_PeekUrgentPriority(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Add docs in mixed order
	q.Enqueue(&CollaboDoc{ID: "normal-1", Priority: PriorityNormal, Title: "Normal"})
	q.Enqueue(&CollaboDoc{ID: "low-1", Priority: PriorityLow, Title: "Low"})
	q.Enqueue(&CollaboDoc{ID: "urgent-1", Priority: PriorityUrgent, Title: "Urgent"})

	// Peek should return urgent first
	peeked := q.Peek()
	if peeked.ID != "urgent-1" {
		t.Errorf("Peek() ID = %s, want urgent-1", peeked.ID)
	}

	// Queue should be unchanged
	if got := q.Len(); got != 3 {
		t.Errorf("Len() after Peek() = %d, want 3", got)
	}
}

func TestWorkQueue_PeekLowPriority(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Add only low priority doc
	q.Enqueue(&CollaboDoc{ID: "low-1", Priority: PriorityLow, Title: "Low"})

	peeked := q.Peek()
	if peeked.ID != "low-1" {
		t.Errorf("Peek() ID = %s, want low-1", peeked.ID)
	}
	if peeked.Priority != PriorityLow {
		t.Errorf("Peek() Priority = %v, want PriorityLow", peeked.Priority)
	}
}

func TestWorkQueue_PeekPriorityOrder(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Add docs in reverse priority order
	q.Enqueue(&CollaboDoc{ID: "low-1", Priority: PriorityLow})
	q.Enqueue(&CollaboDoc{ID: "normal-1", Priority: PriorityNormal})
	q.Enqueue(&CollaboDoc{ID: "urgent-1", Priority: PriorityUrgent})

	// Peek and dequeue should return in priority order
	expected := []string{"urgent-1", "normal-1", "low-1"}
	for _, wantID := range expected {
		peeked := q.Peek()
		if peeked.ID != wantID {
			t.Errorf("Peek() ID = %s, want %s", peeked.ID, wantID)
		}
		dequeued, _ := q.Dequeue()
		if dequeued.ID != wantID {
			t.Errorf("Dequeue() ID = %s, want %s", dequeued.ID, wantID)
		}
	}

	// Queue should be empty
	if got := q.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
}

func TestWorkQueue_NilEnqueue(t *testing.T) {
	q := NewWorkQueue("test-agent")

	// Should not panic
	q.Enqueue(nil)

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after nil enqueue = %d, want 0", got)
	}
}
