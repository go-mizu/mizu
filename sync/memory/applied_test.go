package memory_test

import (
	"context"
	"sync"
	"testing"

	gosync "github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

func TestApplied_GetPut(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	result := gosync.Result{
		OK:     true,
		Cursor: 42,
	}

	// Put
	if err := a.Put(ctx, "scope", "mut-1", result); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	got, found, err := a.Get(ctx, "scope", "mut-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Get returned found=false, want true")
	}
	if got.OK != result.OK || got.Cursor != result.Cursor {
		t.Errorf("Get returned %+v, want %+v", got, result)
	}
}

func TestApplied_GetNotFound(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	_, found, err := a.Get(ctx, "scope", "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Get returned found=true, want false")
	}

	// Non-existent scope
	_, found, _ = a.Get(ctx, "nonexistent", "key")
	if found {
		t.Error("Get for nonexistent scope returned found=true")
	}
}

func TestApplied_Scoped(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	result1 := gosync.Result{OK: true, Cursor: 1}
	result2 := gosync.Result{OK: true, Cursor: 2}

	a.Put(ctx, "scope1", "key", result1)
	a.Put(ctx, "scope2", "key", result2)

	got1, found1, _ := a.Get(ctx, "scope1", "key")
	got2, found2, _ := a.Get(ctx, "scope2", "key")

	if !found1 || !found2 {
		t.Fatal("Expected both results to be found")
	}
	if got1.Cursor != 1 || got2.Cursor != 2 {
		t.Errorf("Scoped results incorrect: got cursors %d, %d, want 1, 2", got1.Cursor, got2.Cursor)
	}
}

func TestApplied_Overwrite(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	a.Put(ctx, "scope", "key", gosync.Result{Cursor: 1})
	a.Put(ctx, "scope", "key", gosync.Result{Cursor: 2})

	got, _, _ := a.Get(ctx, "scope", "key")
	if got.Cursor != 2 {
		t.Errorf("After overwrite, cursor = %d, want 2", got.Cursor)
	}
}

func TestApplied_Concurrency(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			result := gosync.Result{OK: true, Cursor: uint64(i)}
			a.Put(ctx, "scope", key, result)
			a.Get(ctx, "scope", key)
		}(i)
	}
	wg.Wait()
}

func TestApplied_Clear(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	a.Put(ctx, "scope", "key", gosync.Result{OK: true})
	a.Clear()

	_, found, _ := a.Get(ctx, "scope", "key")
	if found {
		t.Error("After Clear, Get returned found=true")
	}
}

func TestApplied_Len(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	if a.Len("scope") != 0 {
		t.Errorf("initial Len = %d, want 0", a.Len("scope"))
	}

	a.Put(ctx, "scope", "key1", gosync.Result{})
	a.Put(ctx, "scope", "key2", gosync.Result{})
	a.Put(ctx, "other", "key3", gosync.Result{})

	if a.Len("scope") != 2 {
		t.Errorf("Len(scope) = %d, want 2", a.Len("scope"))
	}
	if a.Len("other") != 1 {
		t.Errorf("Len(other) = %d, want 1", a.Len("other"))
	}
}

func TestApplied_PreservesChanges(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	changes := []gosync.Change{
		{Cursor: 1, Entity: "users", ID: "1", Op: gosync.Create},
		{Cursor: 2, Entity: "users", ID: "2", Op: gosync.Update},
	}
	result := gosync.Result{
		OK:      true,
		Cursor:  2,
		Changes: changes,
	}

	a.Put(ctx, "scope", "key", result)

	got, _, _ := a.Get(ctx, "scope", "key")
	if len(got.Changes) != 2 {
		t.Errorf("Got %d changes, want 2", len(got.Changes))
	}
	if got.Changes[0].Entity != "users" || got.Changes[1].Op != gosync.Update {
		t.Error("Changes not preserved correctly")
	}
}
