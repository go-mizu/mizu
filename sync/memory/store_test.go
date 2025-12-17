package memory_test

import (
	"context"
	"sync"
	"testing"

	gosync "github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

func TestStore_GetSetDelete(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	// Set
	data := []byte(`{"name":"test"}`)
	if err := s.Set(ctx, "scope1", "entity1", "id1", data); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	got, err := s.Get(ctx, "scope1", "entity1", "id1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("Get returned %q, want %q", got, data)
	}

	// Delete
	if err := s.Delete(ctx, "scope1", "entity1", "id1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get after delete
	_, err = s.Get(ctx, "scope1", "entity1", "id1")
	if err != gosync.ErrNotFound {
		t.Errorf("Get after delete returned err=%v, want ErrNotFound", err)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	_, err := s.Get(ctx, "scope", "entity", "nonexistent")
	if err != gosync.ErrNotFound {
		t.Errorf("Get returned err=%v, want ErrNotFound", err)
	}

	// Non-existent scope
	_, err = s.Get(ctx, "nonexistent", "entity", "id")
	if err != gosync.ErrNotFound {
		t.Errorf("Get returned err=%v, want ErrNotFound", err)
	}

	// Non-existent entity
	s.Set(ctx, "scope", "entity", "id", []byte("data"))
	_, err = s.Get(ctx, "scope", "other", "id")
	if err != gosync.ErrNotFound {
		t.Errorf("Get returned err=%v, want ErrNotFound", err)
	}
}

func TestStore_Snapshot_Empty(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	snap, err := s.Snapshot(ctx, "scope")
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if len(snap) != 0 {
		t.Errorf("Snapshot returned %d entities, want 0", len(snap))
	}
}

func TestStore_Snapshot_WithData(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	// Add data
	s.Set(ctx, "scope", "users", "u1", []byte(`{"id":"u1"}`))
	s.Set(ctx, "scope", "users", "u2", []byte(`{"id":"u2"}`))
	s.Set(ctx, "scope", "posts", "p1", []byte(`{"id":"p1"}`))
	s.Set(ctx, "other", "users", "u3", []byte(`{"id":"u3"}`)) // Different scope

	snap, err := s.Snapshot(ctx, "scope")
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	if len(snap) != 2 {
		t.Errorf("Snapshot returned %d entities, want 2", len(snap))
	}
	if len(snap["users"]) != 2 {
		t.Errorf("users has %d items, want 2", len(snap["users"]))
	}
	if len(snap["posts"]) != 1 {
		t.Errorf("posts has %d items, want 1", len(snap["posts"]))
	}
}

func TestStore_Snapshot_IsCopy(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	original := []byte(`{"id":"1"}`)
	s.Set(ctx, "scope", "entity", "id", original)

	snap, _ := s.Snapshot(ctx, "scope")
	// Modify the snapshot
	snap["entity"]["id"][0] = 'X'

	// Original should be unchanged
	got, _ := s.Get(ctx, "scope", "entity", "id")
	if string(got) != string(original) {
		t.Errorf("Snapshot modification affected original data")
	}
}

func TestStore_Get_IsCopy(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	original := []byte(`{"id":"1"}`)
	s.Set(ctx, "scope", "entity", "id", original)

	got, _ := s.Get(ctx, "scope", "entity", "id")
	// Modify the result
	got[0] = 'X'

	// Getting again should return original
	got2, _ := s.Get(ctx, "scope", "entity", "id")
	if string(got2) != string(original) {
		t.Errorf("Get modification affected stored data")
	}
}

func TestStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i%26))
			s.Set(ctx, "scope", "entity", id, []byte("data"))
			s.Get(ctx, "scope", "entity", id)
			s.Snapshot(ctx, "scope")
			s.Delete(ctx, "scope", "entity", id)
		}(i)
	}
	wg.Wait()
}

func TestStore_Clear(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	s.Set(ctx, "scope", "entity", "id", []byte("data"))
	s.Clear()

	_, err := s.Get(ctx, "scope", "entity", "id")
	if err != gosync.ErrNotFound {
		t.Errorf("Get after Clear returned err=%v, want ErrNotFound", err)
	}
}

func TestStore_DeleteCleansEmptyMaps(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	// Add and remove
	s.Set(ctx, "scope", "entity", "id", []byte("data"))
	s.Delete(ctx, "scope", "entity", "id")

	// Snapshot should be empty
	snap, _ := s.Snapshot(ctx, "scope")
	if len(snap) != 0 {
		t.Errorf("Snapshot after delete has %d entities, want 0", len(snap))
	}
}
