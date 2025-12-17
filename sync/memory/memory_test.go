package memory_test

import (
	"context"
	gosync "sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

// -----------------------------------------------------------------------------
// Store Tests
// -----------------------------------------------------------------------------

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
	if err != sync.ErrNotFound {
		t.Errorf("Get after delete returned err=%v, want ErrNotFound", err)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore()

	_, err := s.Get(ctx, "scope", "entity", "nonexistent")
	if err != sync.ErrNotFound {
		t.Errorf("Get returned err=%v, want ErrNotFound", err)
	}

	// Non-existent scope
	_, err = s.Get(ctx, "nonexistent", "entity", "id")
	if err != sync.ErrNotFound {
		t.Errorf("Get returned err=%v, want ErrNotFound", err)
	}

	// Non-existent entity
	s.Set(ctx, "scope", "entity", "id", []byte("data"))
	_, err = s.Get(ctx, "scope", "other", "id")
	if err != sync.ErrNotFound {
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

	var wg gosync.WaitGroup
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

// -----------------------------------------------------------------------------
// Log Tests
// -----------------------------------------------------------------------------

func TestLog_Append_SingleChange(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	changes := []sync.Change{
		{Entity: "users", ID: "1", Op: sync.Create, Data: []byte(`{}`)},
	}

	cursor, err := l.Append(ctx, "scope", changes)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if cursor != 1 {
		t.Errorf("cursor = %d, want 1", cursor)
	}

	// Verify cursor was assigned
	since, err := l.Since(ctx, "scope", 0, 10)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(since) != 1 {
		t.Fatalf("Since returned %d changes, want 1", len(since))
	}
	if since[0].Cursor != 1 {
		t.Errorf("change.Cursor = %d, want 1", since[0].Cursor)
	}
}

func TestLog_Append_MultipleChanges(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	changes := []sync.Change{
		{Entity: "users", ID: "1", Op: sync.Create},
		{Entity: "users", ID: "2", Op: sync.Create},
		{Entity: "posts", ID: "1", Op: sync.Create},
	}

	cursor, err := l.Append(ctx, "scope", changes)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if cursor != 3 {
		t.Errorf("cursor = %d, want 3", cursor)
	}

	since, _ := l.Since(ctx, "scope", 0, 10)
	if len(since) != 3 {
		t.Errorf("Since returned %d changes, want 3", len(since))
	}

	// Verify sequential cursors
	for i, c := range since {
		want := uint64(i + 1)
		if c.Cursor != want {
			t.Errorf("changes[%d].Cursor = %d, want %d", i, c.Cursor, want)
		}
	}
}

func TestLog_Append_Empty(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Append some initial changes
	l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: "1", Op: sync.Create}})

	// Append empty should return current cursor
	cursor, err := l.Append(ctx, "scope", []sync.Change{})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if cursor != 1 {
		t.Errorf("cursor = %d, want 1", cursor)
	}
}

func TestLog_Since_Empty(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	changes, err := l.Since(ctx, "scope", 0, 10)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("Since returned %d changes, want 0", len(changes))
	}
}

func TestLog_Since_WithCursor(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Add 5 changes
	for i := 0; i < 5; i++ {
		l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: sync.Create}})
	}

	// Get changes since cursor 2
	changes, err := l.Since(ctx, "scope", 2, 10)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(changes) != 3 {
		t.Errorf("Since(cursor=2) returned %d changes, want 3", len(changes))
	}
	if changes[0].Cursor != 3 {
		t.Errorf("first change cursor = %d, want 3", changes[0].Cursor)
	}
}

func TestLog_Since_Limit(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Add 10 changes
	for i := 0; i < 10; i++ {
		l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: sync.Create}})
	}

	changes, err := l.Since(ctx, "scope", 0, 3)
	if err != nil {
		t.Fatalf("Since failed: %v", err)
	}
	if len(changes) != 3 {
		t.Errorf("Since(limit=3) returned %d changes, want 3", len(changes))
	}
}

func TestLog_Cursor(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Initial cursor should be 0
	cursor, err := l.Cursor(ctx, "scope")
	if err != nil {
		t.Fatalf("Cursor failed: %v", err)
	}
	if cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", cursor)
	}

	// Add changes
	l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: "1", Op: sync.Create}})
	l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: "2", Op: sync.Create}})

	cursor, _ = l.Cursor(ctx, "scope")
	if cursor != 2 {
		t.Errorf("cursor after appends = %d, want 2", cursor)
	}
}

func TestLog_Trim(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Add 5 changes
	for i := 0; i < 5; i++ {
		l.Append(ctx, "scope", []sync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: sync.Create}})
	}

	// Trim before cursor 3
	if err := l.Trim(ctx, "scope", 3); err != nil {
		t.Fatalf("Trim failed: %v", err)
	}

	// Should only have changes with cursor >= 3
	changes, _ := l.Since(ctx, "scope", 0, 10)
	if len(changes) != 3 {
		t.Errorf("After trim, got %d changes, want 3", len(changes))
	}
	if changes[0].Cursor != 3 {
		t.Errorf("first change cursor = %d, want 3", changes[0].Cursor)
	}
}

func TestLog_Scoped(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	// Add changes to different scopes
	l.Append(ctx, "scope1", []sync.Change{{Entity: "e", ID: "1", Op: sync.Create}})
	l.Append(ctx, "scope2", []sync.Change{{Entity: "e", ID: "2", Op: sync.Create}})
	l.Append(ctx, "scope1", []sync.Change{{Entity: "e", ID: "3", Op: sync.Create}})

	// Check scope1
	changes1, _ := l.Since(ctx, "scope1", 0, 10)
	if len(changes1) != 2 {
		t.Errorf("scope1 has %d changes, want 2", len(changes1))
	}

	// Check scope2
	changes2, _ := l.Since(ctx, "scope2", 0, 10)
	if len(changes2) != 1 {
		t.Errorf("scope2 has %d changes, want 1", len(changes2))
	}

	// Cursors should be scoped
	c1, _ := l.Cursor(ctx, "scope1")
	c2, _ := l.Cursor(ctx, "scope2")
	if c1 != 3 || c2 != 2 {
		t.Errorf("cursors = (%d, %d), want (3, 2)", c1, c2)
	}
}

func TestLog_SetsScope(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	changes := []sync.Change{{Entity: "e", ID: "1", Op: sync.Create}}
	l.Append(ctx, "myScope", changes)

	got, _ := l.Since(ctx, "myScope", 0, 10)
	if got[0].Scope != "myScope" {
		t.Errorf("change.Scope = %q, want %q", got[0].Scope, "myScope")
	}
}

func TestLog_Concurrency(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	var wg gosync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			scope := "scope"
			if i%2 == 0 {
				scope = "scope2"
			}
			l.Append(ctx, scope, []sync.Change{{
				Entity: "e",
				ID:     string(rune('a' + i%26)),
				Op:     sync.Create,
				Time:   time.Now(),
			}})
			l.Since(ctx, scope, 0, 10)
			l.Cursor(ctx, scope)
		}(i)
	}
	wg.Wait()
}

// -----------------------------------------------------------------------------
// Applied Tests
// -----------------------------------------------------------------------------

func TestApplied_GetPut(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	result := sync.Result{
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

	result1 := sync.Result{OK: true, Cursor: 1}
	result2 := sync.Result{OK: true, Cursor: 2}

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

	a.Put(ctx, "scope", "key", sync.Result{Cursor: 1})
	a.Put(ctx, "scope", "key", sync.Result{Cursor: 2})

	got, _, _ := a.Get(ctx, "scope", "key")
	if got.Cursor != 2 {
		t.Errorf("After overwrite, cursor = %d, want 2", got.Cursor)
	}
}

func TestApplied_Concurrency(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	var wg gosync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			result := sync.Result{OK: true, Cursor: uint64(i)}
			a.Put(ctx, "scope", key, result)
			a.Get(ctx, "scope", key)
		}(i)
	}
	wg.Wait()
}

func TestApplied_PreservesChanges(t *testing.T) {
	ctx := context.Background()
	a := memory.NewApplied()

	changes := []sync.Change{
		{Cursor: 1, Entity: "users", ID: "1", Op: sync.Create},
		{Cursor: 2, Entity: "users", ID: "2", Op: sync.Update},
	}
	result := sync.Result{
		OK:      true,
		Cursor:  2,
		Changes: changes,
	}

	a.Put(ctx, "scope", "key", result)

	got, _, _ := a.Get(ctx, "scope", "key")
	if len(got.Changes) != 2 {
		t.Errorf("Got %d changes, want 2", len(got.Changes))
	}
	if got.Changes[0].Entity != "users" || got.Changes[1].Op != sync.Update {
		t.Error("Changes not preserved correctly")
	}
}
