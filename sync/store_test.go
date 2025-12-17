package sync

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryStore_SetGet(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	todo := map[string]any{
		"id":        "1",
		"title":     "Test",
		"completed": false,
	}

	err := s.Set(ctx, "user:123", "todo", "1", todo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := s.Get(ctx, "user:123", "todo", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := data.(map[string]any)
	if result["title"] != "Test" {
		t.Errorf("expected title 'Test', got %v", result["title"])
	}
}

func TestMemoryStore_ScopedIsolation(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope1", "todo", "1", map[string]any{"title": "Scope1 Todo"})
	s.Set(ctx, "scope2", "todo", "1", map[string]any{"title": "Scope2 Todo"})

	data1, _ := s.Get(ctx, "scope1", "todo", "1")
	data2, _ := s.Get(ctx, "scope2", "todo", "1")

	if data1.(map[string]any)["title"] != "Scope1 Todo" {
		t.Error("scope isolation failed for scope1")
	}
	if data2.(map[string]any)["title"] != "Scope2 Todo" {
		t.Error("scope isolation failed for scope2")
	}
}

func TestMemoryStore_EntityIsolation(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"type": "todo"})
	s.Set(ctx, "scope", "user", "1", map[string]any{"type": "user"})

	todoData, _ := s.Get(ctx, "scope", "todo", "1")
	userData, _ := s.Get(ctx, "scope", "user", "1")

	if todoData.(map[string]any)["type"] != "todo" {
		t.Error("entity isolation failed for todo")
	}
	if userData.(map[string]any)["type"] != "user" {
		t.Error("entity isolation failed for user")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"title": "Test"})

	err := s.Delete(ctx, "scope", "todo", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Get(ctx, "scope", "todo", "1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_DeleteNonExistent(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	// Should not error
	err := s.Delete(ctx, "scope", "todo", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error deleting non-existent: %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"id": "1"})
	s.Set(ctx, "scope", "todo", "2", map[string]any{"id": "2"})
	s.Set(ctx, "scope", "todo", "3", map[string]any{"id": "3"})

	items, err := s.List(ctx, "scope", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestMemoryStore_ListEmpty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	items, err := s.List(ctx, "unknown", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items for unknown scope, got %d", len(items))
	}
}

func TestMemoryStore_Snapshot(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"id": "1"})
	s.Set(ctx, "scope", "todo", "2", map[string]any{"id": "2"})
	s.Set(ctx, "scope", "user", "u1", map[string]any{"id": "u1"})

	snap, err := s.Snapshot(ctx, "scope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snap) != 2 {
		t.Errorf("expected 2 entity types, got %d", len(snap))
	}
	if len(snap["todo"]) != 2 {
		t.Errorf("expected 2 todos, got %d", len(snap["todo"]))
	}
	if len(snap["user"]) != 1 {
		t.Errorf("expected 1 user, got %d", len(snap["user"]))
	}
}

func TestMemoryStore_SnapshotIsCopy(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"id": "1"})

	snap, _ := s.Snapshot(ctx, "scope")

	// Modify snapshot
	snap["todo"]["999"] = map[string]any{"id": "999"}

	// Original should be unchanged
	_, err := s.Get(ctx, "scope", "todo", "999")
	if err != ErrNotFound {
		t.Error("snapshot modification affected original store")
	}
}

func TestMemoryStore_SnapshotEmpty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	snap, err := s.Snapshot(ctx, "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snap == nil {
		t.Error("expected non-nil empty map")
	}
	if len(snap) != 0 {
		t.Errorf("expected empty snapshot, got %d entities", len(snap))
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.Get(ctx, "unknown", "todo", "1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown scope, got %v", err)
	}

	s.Set(ctx, "scope", "todo", "1", map[string]any{})

	_, err = s.Get(ctx, "scope", "unknown", "1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown entity, got %v", err)
	}

	_, err = s.Get(ctx, "scope", "todo", "unknown")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown id, got %v", err)
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Set(ctx, "scope", "todo", "1", map[string]any{"id": "1"})
	s.Clear()

	_, err := s.Get(ctx, "scope", "todo", "1")
	if err != ErrNotFound {
		t.Error("expected ErrNotFound after clear")
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	const n = 100

	// Concurrent sets
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Set(ctx, "scope", "todo", string(rune('0'+i)), map[string]any{"i": i})
		}(i)
	}

	// Concurrent gets
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Get(ctx, "scope", "todo", string(rune('0'+i)))
		}(i)
	}

	wg.Wait()

	// Just verify no panics occurred
	items, _ := s.List(ctx, "scope", "todo")
	if len(items) != n {
		t.Errorf("expected %d items, got %d", n, len(items))
	}
}
