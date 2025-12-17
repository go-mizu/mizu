package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	gosync "github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

func TestLog_Append_SingleChange(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	changes := []gosync.Change{
		{Entity: "users", ID: "1", Op: gosync.Create, Data: []byte(`{}`)},
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

	changes := []gosync.Change{
		{Entity: "users", ID: "1", Op: gosync.Create},
		{Entity: "users", ID: "2", Op: gosync.Create},
		{Entity: "posts", ID: "1", Op: gosync.Create},
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
	l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: "1", Op: gosync.Create}})

	// Append empty should return current cursor
	cursor, err := l.Append(ctx, "scope", []gosync.Change{})
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
		l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: gosync.Create}})
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
		l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: gosync.Create}})
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
	l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: "1", Op: gosync.Create}})
	l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: "2", Op: gosync.Create}})

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
		l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: string(rune('a' + i)), Op: gosync.Create}})
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
	l.Append(ctx, "scope1", []gosync.Change{{Entity: "e", ID: "1", Op: gosync.Create}})
	l.Append(ctx, "scope2", []gosync.Change{{Entity: "e", ID: "2", Op: gosync.Create}})
	l.Append(ctx, "scope1", []gosync.Change{{Entity: "e", ID: "3", Op: gosync.Create}})

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

	changes := []gosync.Change{{Entity: "e", ID: "1", Op: gosync.Create}}
	l.Append(ctx, "myScope", changes)

	got, _ := l.Since(ctx, "myScope", 0, 10)
	if got[0].Scope != "myScope" {
		t.Errorf("change.Scope = %q, want %q", got[0].Scope, "myScope")
	}
}

func TestLog_Concurrency(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			scope := "scope"
			if i%2 == 0 {
				scope = "scope2"
			}
			l.Append(ctx, scope, []gosync.Change{{
				Entity: "e",
				ID:     string(rune('a' + i%26)),
				Op:     gosync.Create,
				Time:   time.Now(),
			}})
			l.Since(ctx, scope, 0, 10)
			l.Cursor(ctx, scope)
		}(i)
	}
	wg.Wait()
}

func TestLog_Clear(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	l.Append(ctx, "scope", []gosync.Change{{Entity: "e", ID: "1", Op: gosync.Create}})
	l.Clear()

	cursor, _ := l.Cursor(ctx, "scope")
	if cursor != 0 {
		t.Errorf("cursor after Clear = %d, want 0", cursor)
	}

	changes, _ := l.Since(ctx, "scope", 0, 10)
	if len(changes) != 0 {
		t.Errorf("changes after Clear = %d, want 0", len(changes))
	}
}

func TestLog_Len(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLog()

	if l.Len("scope") != 0 {
		t.Errorf("initial Len = %d, want 0", l.Len("scope"))
	}

	l.Append(ctx, "scope", []gosync.Change{
		{Entity: "e", ID: "1", Op: gosync.Create},
		{Entity: "e", ID: "2", Op: gosync.Create},
	})

	if l.Len("scope") != 2 {
		t.Errorf("Len after append = %d, want 2", l.Len("scope"))
	}
}
