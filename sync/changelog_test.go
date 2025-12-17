package sync

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryChangeLog_Append(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	cursor, err := cl.Append(ctx, Change{
		Scope:  "test",
		Entity: "todo",
		ID:     "1",
		Op:     OpCreate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != 1 {
		t.Errorf("expected cursor 1, got %d", cursor)
	}

	cursor2, err := cl.Append(ctx, Change{
		Scope:  "test",
		Entity: "todo",
		ID:     "2",
		Op:     OpCreate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor2 != 2 {
		t.Errorf("expected cursor 2, got %d", cursor2)
	}

	if cl.Len() != 2 {
		t.Errorf("expected length 2, got %d", cl.Len())
	}
}

func TestMemoryChangeLog_Since_All(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		cl.Append(ctx, Change{
			Scope:  "test",
			Entity: "todo",
			ID:     string(rune('0' + i)),
			Op:     OpCreate,
		})
	}

	// Get all from cursor 0
	changes, err := cl.Since(ctx, "", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 5 {
		t.Errorf("expected 5 changes, got %d", len(changes))
	}
}

func TestMemoryChangeLog_Since_Cursor(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		cl.Append(ctx, Change{
			Scope:  "test",
			Entity: "todo",
			ID:     string(rune('0' + i)),
			Op:     OpCreate,
		})
	}

	// Get from cursor 3 (should get entries 4 and 5)
	changes, err := cl.Since(ctx, "", 3, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Cursor != 4 {
		t.Errorf("expected first change cursor 4, got %d", changes[0].Cursor)
	}
}

func TestMemoryChangeLog_Since_Scope(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	cl.Append(ctx, Change{Scope: "scope1", Entity: "todo", ID: "1", Op: OpCreate})
	cl.Append(ctx, Change{Scope: "scope2", Entity: "todo", ID: "2", Op: OpCreate})
	cl.Append(ctx, Change{Scope: "scope1", Entity: "todo", ID: "3", Op: OpCreate})

	changes, err := cl.Since(ctx, "scope1", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("expected 2 changes for scope1, got %d", len(changes))
	}
}

func TestMemoryChangeLog_Since_Limit(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		cl.Append(ctx, Change{
			Scope:  "test",
			Entity: "todo",
			ID:     string(rune('0' + i)),
			Op:     OpCreate,
		})
	}

	changes, err := cl.Since(ctx, "", 0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 3 {
		t.Errorf("expected 3 changes (limited), got %d", len(changes))
	}
}

func TestMemoryChangeLog_Since_Empty(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	cl.Append(ctx, Change{Scope: "test", Entity: "todo", ID: "1", Op: OpCreate})

	// Get from cursor at head
	changes, err := cl.Since(ctx, "", 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestMemoryChangeLog_Cursor(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	cursor, _ := cl.Cursor(ctx)
	if cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", cursor)
	}

	cl.Append(ctx, Change{Scope: "test", Entity: "todo", ID: "1", Op: OpCreate})
	cl.Append(ctx, Change{Scope: "test", Entity: "todo", ID: "2", Op: OpCreate})

	cursor, _ = cl.Cursor(ctx)
	if cursor != 2 {
		t.Errorf("expected cursor 2, got %d", cursor)
	}
}

func TestMemoryChangeLog_Trim(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		cl.Append(ctx, Change{
			Scope:  "test",
			Entity: "todo",
			ID:     string(rune('0' + i)),
			Op:     OpCreate,
		})
	}

	err := cl.Trim(ctx, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have entries 3, 4, 5 remaining
	if cl.Len() != 3 {
		t.Errorf("expected 3 entries after trim, got %d", cl.Len())
	}

	changes, _ := cl.Since(ctx, "", 0, 100)
	if changes[0].Cursor != 3 {
		t.Errorf("expected first entry cursor 3, got %d", changes[0].Cursor)
	}
}

func TestMemoryChangeLog_Concurrent(t *testing.T) {
	cl := NewMemoryChangeLog()
	ctx := context.Background()

	var wg sync.WaitGroup
	const n = 100

	// Concurrent appends
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cl.Append(ctx, Change{
				Scope:  "test",
				Entity: "todo",
				ID:     string(rune(i)),
				Op:     OpCreate,
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cl.Since(ctx, "", 0, 100)
		}()
	}

	wg.Wait()

	if cl.Len() != n {
		t.Errorf("expected %d entries, got %d", n, cl.Len())
	}
}
