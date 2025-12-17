package sync

import (
	"testing"
)

func TestQueue_Push(t *testing.T) {
	q := NewQueue()

	id := q.Push(Mutation{Name: "test"})

	if id == "" {
		t.Error("Push should return an ID")
	}

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}

func TestQueue_PushWithID(t *testing.T) {
	q := NewQueue()

	id := q.Push(Mutation{ID: "custom-id", Name: "test"})

	if id != "custom-id" {
		t.Errorf("Push() = %s, want custom-id", id)
	}
}

func TestQueue_PushDuplicate(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{ID: "dup", Name: "test1"})
	q.Push(Mutation{ID: "dup", Name: "test2"}) // Duplicate

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1 (no duplicate)", q.Len())
	}
}

func TestQueue_Pending(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{Name: "a"})
	q.Push(Mutation{Name: "b"})
	q.Push(Mutation{Name: "c"})

	pending := q.Pending()
	if len(pending) != 3 {
		t.Errorf("Pending() len = %d, want 3", len(pending))
	}

	// Verify order
	if pending[0].Name != "a" || pending[1].Name != "b" || pending[2].Name != "c" {
		t.Error("Pending should preserve order")
	}

	// Verify sequence numbers
	if pending[0].Seq != 1 || pending[1].Seq != 2 || pending[2].Seq != 3 {
		t.Error("Pending should have correct sequence numbers")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{Name: "a"})
	q.Push(Mutation{Name: "b"})

	q.Clear()

	if q.Len() != 0 {
		t.Errorf("Len() = %d, want 0", q.Len())
	}
}

func TestQueue_Remove(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{ID: "a", Name: "a"})
	q.Push(Mutation{ID: "b", Name: "b"})
	q.Push(Mutation{ID: "c", Name: "c"})

	q.Remove("b")

	if q.Len() != 2 {
		t.Errorf("Len() = %d, want 2", q.Len())
	}

	pending := q.Pending()
	if pending[0].ID != "a" || pending[1].ID != "c" {
		t.Error("Remove should preserve remaining order")
	}
}

func TestQueue_RemoveUpTo(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{Name: "a"}) // seq 1
	q.Push(Mutation{Name: "b"}) // seq 2
	q.Push(Mutation{Name: "c"}) // seq 3
	q.Push(Mutation{Name: "d"}) // seq 4

	q.RemoveUpTo(2)

	if q.Len() != 2 {
		t.Errorf("Len() = %d, want 2", q.Len())
	}

	pending := q.Pending()
	if pending[0].Seq != 3 || pending[1].Seq != 4 {
		t.Error("RemoveUpTo should remove mutations up to and including seq")
	}
}

func TestQueue_Get(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{ID: "a", Name: "test-a"})

	m, ok := q.Get("a")
	if !ok {
		t.Fatal("Get should return true for existing")
	}
	if m.Name != "test-a" {
		t.Errorf("Get().Name = %s, want test-a", m.Name)
	}

	_, ok = q.Get("nonexistent")
	if ok {
		t.Error("Get should return false for non-existent")
	}
}

func TestQueue_Has(t *testing.T) {
	q := NewQueue()

	if q.Has("a") {
		t.Error("Has should return false before push")
	}

	q.Push(Mutation{ID: "a", Name: "test"})

	if !q.Has("a") {
		t.Error("Has should return true after push")
	}
}

func TestQueue_ClientID(t *testing.T) {
	q := NewQueue()

	id1 := q.ClientID()
	if id1 == "" {
		t.Error("ClientID should not be empty")
	}

	q.SetClientID("custom-client")

	if q.ClientID() != "custom-client" {
		t.Errorf("ClientID() = %s, want custom-client", q.ClientID())
	}
}

func TestQueue_MutationHasClientID(t *testing.T) {
	q := NewQueue()
	q.SetClientID("my-client")

	q.Push(Mutation{Name: "test"})

	pending := q.Pending()
	if pending[0].Client != "my-client" {
		t.Errorf("Mutation.Client = %s, want my-client", pending[0].Client)
	}
}

func TestQueue_CurrentSeq(t *testing.T) {
	q := NewQueue()

	if q.CurrentSeq() != 0 {
		t.Errorf("CurrentSeq() = %d, want 0", q.CurrentSeq())
	}

	q.Push(Mutation{Name: "a"})
	q.Push(Mutation{Name: "b"})

	if q.CurrentSeq() != 2 {
		t.Errorf("CurrentSeq() = %d, want 2", q.CurrentSeq())
	}
}

func TestQueue_Load(t *testing.T) {
	q := NewQueue()

	mutations := []Mutation{
		{ID: "a", Name: "a", Seq: 10},
		{ID: "b", Name: "b", Seq: 11},
	}

	q.Load(mutations)

	if q.Len() != 2 {
		t.Errorf("Len() = %d, want 2", q.Len())
	}

	if q.CurrentSeq() != 11 {
		t.Errorf("CurrentSeq() = %d, want 11", q.CurrentSeq())
	}

	// New pushes should continue from highest seq
	q.Push(Mutation{Name: "c"})
	pending := q.Pending()
	if pending[2].Seq != 12 {
		t.Errorf("New mutation seq = %d, want 12", pending[2].Seq)
	}
}

func TestQueue_CreatedAt(t *testing.T) {
	q := NewQueue()

	q.Push(Mutation{Name: "test"})

	pending := q.Pending()
	if pending[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set automatically")
	}
}
