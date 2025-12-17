package sync

import (
	"context"
	"fmt"
	"testing"
)

// todoMutator creates a test mutator for todo operations.
func todoMutator() Mutator {
	mm := NewMutatorMap()

	mm.Register("todo/create", func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		id := m.Args["id"].(string)
		title := m.Args["title"].(string)

		todo := map[string]any{
			"id":        id,
			"title":     title,
			"completed": false,
		}

		if err := store.Set(ctx, m.Scope, "todo", id, todo); err != nil {
			return nil, err
		}

		return []Change{{
			Scope:  m.Scope,
			Entity: "todo",
			ID:     id,
			Op:     OpCreate,
			Data:   todo,
		}}, nil
	})

	mm.Register("todo/toggle", func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		id := m.Args["id"].(string)

		data, err := store.Get(ctx, m.Scope, "todo", id)
		if err != nil {
			return nil, err
		}

		todo := data.(map[string]any)
		todo["completed"] = !todo["completed"].(bool)

		if err := store.Set(ctx, m.Scope, "todo", id, todo); err != nil {
			return nil, err
		}

		return []Change{{
			Scope:  m.Scope,
			Entity: "todo",
			ID:     id,
			Op:     OpUpdate,
			Data:   todo,
		}}, nil
	})

	mm.Register("todo/delete", func(ctx context.Context, store Store, m Mutation) ([]Change, error) {
		id := m.Args["id"].(string)

		if err := store.Delete(ctx, m.Scope, "todo", id); err != nil {
			return nil, err
		}

		return []Change{{
			Scope:  m.Scope,
			Entity: "todo",
			ID:     id,
			Op:     OpDelete,
		}}, nil
	})

	return mm
}

func createTestEngine() *Engine {
	return New(Options{
		Store:     NewMemoryStore(),
		ChangeLog: NewMemoryChangeLog(),
		Mutator:   todoMutator(),
	})
}

func createTestEngineWithBroker(broker PokeBroker) *Engine {
	return New(Options{
		Store:     NewMemoryStore(),
		ChangeLog: NewMemoryChangeLog(),
		Mutator:   todoMutator(),
		Broker:    broker,
	})
}

func TestEngine_New(t *testing.T) {
	e := createTestEngine()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.Store() == nil {
		t.Error("expected non-nil store")
	}
	if e.ChangeLog() == nil {
		t.Error("expected non-nil changelog")
	}
}

func TestEngine_New_NilBroker(t *testing.T) {
	e := New(Options{
		Store:     NewMemoryStore(),
		ChangeLog: NewMemoryChangeLog(),
		Mutator:   todoMutator(),
		Broker:    nil, // Should default to NopBroker
	})

	ctx := context.Background()

	// Should not panic
	results, err := e.Push(ctx, []Mutation{{
		Name:  "todo/create",
		Scope: "test",
		Args:  map[string]any{"id": "1", "title": "Test"},
	}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !results[0].OK {
		t.Error("expected mutation to succeed")
	}
}

func TestEngine_Push_SingleMutation(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	results, err := e.Push(ctx, []Mutation{{
		Name:  "todo/create",
		Scope: "user:123",
		Args:  map[string]any{"id": "1", "title": "Buy milk"},
	}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if !r.OK {
		t.Errorf("expected OK, got error: %s", r.Error)
	}
	if r.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", r.Cursor)
	}
	if len(r.Changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(r.Changes))
	}
}

func TestEngine_Push_MultipleMutations(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	results, err := e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "3", "title": "Third"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i, r := range results {
		if !r.OK {
			t.Errorf("mutation %d failed: %s", i, r.Error)
		}
		if r.Cursor != uint64(i+1) {
			t.Errorf("expected cursor %d, got %d", i+1, r.Cursor)
		}
	}
}

func TestEngine_Push_Error(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	results, err := e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "unknown/action", Scope: "user:123"}, // This will fail
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Third"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if !results[0].OK {
		t.Error("first mutation should succeed")
	}
	if results[1].OK {
		t.Error("second mutation should fail")
	}
	if results[1].Error == "" {
		t.Error("expected error message for failed mutation")
	}
	if !results[2].OK {
		t.Error("third mutation should succeed despite previous failure")
	}
}

func TestEngine_Push_Broker(t *testing.T) {
	mb := &MockBroker{}
	e := createTestEngineWithBroker(mb)
	ctx := context.Background()

	e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "scope1", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "scope2", Args: map[string]any{"id": "2", "title": "Second"}},
		{Name: "todo/create", Scope: "scope1", Args: map[string]any{"id": "3", "title": "Third"}},
	})

	pokes := mb.Pokes()
	if len(pokes) != 2 {
		t.Fatalf("expected 2 pokes (unique scopes), got %d", len(pokes))
	}

	// Check that both scopes were poked
	scopes := make(map[string]bool)
	for _, p := range pokes {
		scopes[p.Scope] = true
	}
	if !scopes["scope1"] || !scopes["scope2"] {
		t.Error("expected both scopes to be poked")
	}
}

func TestEngine_Pull(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	// Push some data
	e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
	})

	// Pull from cursor 0
	changes, cursor, hasMore, err := e.Pull(ctx, "user:123", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}
	if cursor != 2 {
		t.Errorf("expected cursor 2, got %d", cursor)
	}
	if hasMore {
		t.Error("expected hasMore to be false")
	}
}

func TestEngine_Pull_FromCursor(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "3", "title": "Third"}},
	})

	// Pull from cursor 1 (should get changes 2 and 3)
	changes, cursor, _, err := e.Pull(ctx, "user:123", 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}
	if cursor != 3 {
		t.Errorf("expected cursor 3, got %d", cursor)
	}
}

func TestEngine_Pull_HasMore(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	// Push 10 items
	for i := 1; i <= 10; i++ {
		e.Push(ctx, []Mutation{{
			Name:  "todo/create",
			Scope: "user:123",
			Args:  map[string]any{"id": fmt.Sprintf("%d", i), "title": fmt.Sprintf("Todo %d", i)},
		}})
	}

	// Pull with limit 3
	changes, cursor, hasMore, err := e.Pull(ctx, "user:123", 0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 3 {
		t.Errorf("expected 3 changes, got %d", len(changes))
	}
	if cursor != 3 {
		t.Errorf("expected cursor 3, got %d", cursor)
	}
	if !hasMore {
		t.Error("expected hasMore to be true")
	}
}

func TestEngine_Snapshot(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
	})

	data, cursor, err := e.Snapshot(ctx, "user:123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cursor != 2 {
		t.Errorf("expected cursor 2, got %d", cursor)
	}
	if len(data["todo"]) != 2 {
		t.Errorf("expected 2 todos, got %d", len(data["todo"]))
	}
}

func TestEngine_Cursor(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	cursor, _ := e.Cursor(ctx)
	if cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", cursor)
	}

	e.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "test", Args: map[string]any{"id": "1", "title": "Test"}},
	})

	cursor, _ = e.Cursor(ctx)
	if cursor != 1 {
		t.Errorf("expected cursor 1, got %d", cursor)
	}
}

func TestEngine_FullFlow(t *testing.T) {
	e := createTestEngine()
	ctx := context.Background()

	// Create
	results, _ := e.Push(ctx, []Mutation{{
		Name:  "todo/create",
		Scope: "user:123",
		Args:  map[string]any{"id": "1", "title": "Buy milk"},
	}})
	if !results[0].OK {
		t.Fatal("create failed")
	}

	// Toggle
	results, _ = e.Push(ctx, []Mutation{{
		Name:  "todo/toggle",
		Scope: "user:123",
		Args:  map[string]any{"id": "1"},
	}})
	if !results[0].OK {
		t.Fatal("toggle failed")
	}

	// Verify via pull
	changes, _, _, _ := e.Pull(ctx, "user:123", 0, 100)
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}

	lastChange := changes[1]
	if lastChange.Op != OpUpdate {
		t.Errorf("expected update op, got %s", lastChange.Op)
	}

	todo := lastChange.Data.(map[string]any)
	if !todo["completed"].(bool) {
		t.Error("expected completed to be true after toggle")
	}

	// Delete
	results, _ = e.Push(ctx, []Mutation{{
		Name:  "todo/delete",
		Scope: "user:123",
		Args:  map[string]any{"id": "1"},
	}})
	if !results[0].OK {
		t.Fatal("delete failed")
	}

	// Verify empty snapshot
	data, _, _ := e.Snapshot(ctx, "user:123")
	if len(data["todo"]) != 0 {
		t.Error("expected no todos after delete")
	}
}
