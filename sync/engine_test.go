package sync_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

// testMutator creates a simple mutator for testing
func testMutator() *sync.MutatorMap {
	m := sync.NewMutatorMap()

	m.Register("create", func(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		entity, _ := mut.Args["entity"].(string)
		id, _ := mut.Args["id"].(string)
		data, _ := mut.Args["data"].(string)

		if err := store.Set(ctx, mut.Scope, entity, id, []byte(data)); err != nil {
			return nil, err
		}

		return []sync.Change{
			{Entity: entity, ID: id, Op: sync.Create, Data: []byte(data)},
		}, nil
	})

	m.Register("update", func(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		entity, _ := mut.Args["entity"].(string)
		id, _ := mut.Args["id"].(string)
		data, _ := mut.Args["data"].(string)

		if _, err := store.Get(ctx, mut.Scope, entity, id); err != nil {
			return nil, err
		}

		if err := store.Set(ctx, mut.Scope, entity, id, []byte(data)); err != nil {
			return nil, err
		}

		return []sync.Change{
			{Entity: entity, ID: id, Op: sync.Update, Data: []byte(data)},
		}, nil
	})

	m.Register("delete", func(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		entity, _ := mut.Args["entity"].(string)
		id, _ := mut.Args["id"].(string)

		if err := store.Delete(ctx, mut.Scope, entity, id); err != nil {
			return nil, err
		}

		return []sync.Change{
			{Entity: entity, ID: id, Op: sync.Delete},
		}, nil
	})

	m.Register("noop", func(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		return nil, nil
	})

	return m
}

func newTestEngine() *sync.Engine {
	return sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
	})
}

func TestEngine_Push_Success(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutations := []sync.Mutation{
		{
			ID:    "mut-1",
			Name:  "create",
			Scope: "test",
			Args:  map[string]any{"entity": "users", "id": "u1", "data": `{"name":"Alice"}`},
		},
	}

	results, err := e.Push(ctx, mutations)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Push returned %d results, want 1", len(results))
	}

	r := results[0]
	if !r.OK {
		t.Errorf("result.OK = false, want true")
	}
	if r.Cursor != 1 {
		t.Errorf("result.Cursor = %d, want 1", r.Cursor)
	}
	if len(r.Changes) != 1 {
		t.Errorf("result.Changes has %d items, want 1", len(r.Changes))
	}

	// Verify data was stored
	data, err := e.Store().Get(ctx, "test", "users", "u1")
	if err != nil {
		t.Errorf("Store.Get failed: %v", err)
	}
	if string(data) != `{"name":"Alice"}` {
		t.Errorf("Stored data = %q, want %q", data, `{"name":"Alice"}`)
	}
}

func TestEngine_Push_Idempotency(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutation := sync.Mutation{
		ID:    "mut-1",
		Name:  "create",
		Scope: "test",
		Args:  map[string]any{"entity": "users", "id": "u1", "data": `{"name":"Alice"}`},
	}

	// First push
	results1, _ := e.Push(ctx, []sync.Mutation{mutation})

	// Second push with same ID
	results2, _ := e.Push(ctx, []sync.Mutation{mutation})

	// Should return same result without re-executing
	if results1[0].Cursor != results2[0].Cursor {
		t.Errorf("Idempotent push returned different cursors: %d vs %d",
			results1[0].Cursor, results2[0].Cursor)
	}

	// Verify only one change in log
	changes, _, _ := e.Pull(ctx, "test", 0, 100)
	if len(changes) != 1 {
		t.Errorf("After idempotent push, log has %d changes, want 1", len(changes))
	}
}

func TestEngine_Push_NoID_NoIdempotency(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutation := sync.Mutation{
		Name:  "noop",
		Scope: "test",
	}

	// Push twice without ID
	e.Push(ctx, []sync.Mutation{mutation})
	e.Push(ctx, []sync.Mutation{mutation})

	// Both should execute (no idempotency without ID)
	// Since noop returns no changes, cursor stays at 0
}

func TestEngine_Push_UnknownMutation(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutations := []sync.Mutation{
		{Name: "unknown", Scope: "test"},
	}

	results, err := e.Push(ctx, mutations)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	r := results[0]
	if r.OK {
		t.Error("result.OK = true, want false for unknown mutation")
	}
	if r.Code != sync.CodeUnknown {
		t.Errorf("result.Code = %q, want %q", r.Code, sync.CodeUnknown)
	}
}

func TestEngine_Push_MutationError(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Try to update non-existent entity
	mutations := []sync.Mutation{
		{
			Name:  "update",
			Scope: "test",
			Args:  map[string]any{"entity": "users", "id": "nonexistent", "data": "{}"},
		},
	}

	results, _ := e.Push(ctx, mutations)
	r := results[0]

	if r.OK {
		t.Error("result.OK = true, want false for error")
	}
	if r.Code != sync.CodeNotFound {
		t.Errorf("result.Code = %q, want %q", r.Code, sync.CodeNotFound)
	}
}

func TestEngine_Push_MultipleScopes(t *testing.T) {
	ctx := context.Background()

	var notifications []struct {
		scope  string
		cursor uint64
	}

	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
		Notify: sync.NotifierFunc(func(scope string, cursor uint64) {
			notifications = append(notifications, struct {
				scope  string
				cursor uint64
			}{scope, cursor})
		}),
	})

	mutations := []sync.Mutation{
		{Name: "create", Scope: "scope1", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
		{Name: "create", Scope: "scope2", Args: map[string]any{"entity": "e", "id": "2", "data": "{}"}},
	}

	e.Push(ctx, mutations)

	if len(notifications) != 2 {
		t.Errorf("Got %d notifications, want 2", len(notifications))
	}
}

func TestEngine_Push_NotifyCalled(t *testing.T) {
	ctx := context.Background()

	var notified bool
	var notifiedScope string
	var notifiedCursor uint64

	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
		Notify: sync.NotifierFunc(func(scope string, cursor uint64) {
			notified = true
			notifiedScope = scope
			notifiedCursor = cursor
		}),
	})

	mutations := []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	}

	e.Push(ctx, mutations)

	if !notified {
		t.Error("Notifier was not called")
	}
	if notifiedScope != "test" {
		t.Errorf("notified scope = %q, want %q", notifiedScope, "test")
	}
	if notifiedCursor != 1 {
		t.Errorf("notified cursor = %d, want 1", notifiedCursor)
	}
}

func TestEngine_Push_DefaultScope(t *testing.T) {
	ctx := context.Background()

	var notifiedScope string
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
		Notify: sync.NotifierFunc(func(scope string, cursor uint64) {
			notifiedScope = scope
		}),
	})

	mutations := []sync.Mutation{
		{Name: "create", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	}

	e.Push(ctx, mutations)

	if notifiedScope != "_default" {
		t.Errorf("notified scope = %q, want %q", notifiedScope, "_default")
	}
}

func TestEngine_Pull_Empty(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	changes, hasMore, err := e.Pull(ctx, "test", 0, 100)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("Pull returned %d changes, want 0", len(changes))
	}
	if hasMore {
		t.Error("Pull returned hasMore=true, want false")
	}
}

func TestEngine_Pull_WithChanges(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Create some data
	mutations := []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "users", "id": "1", "data": "{}"}},
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "users", "id": "2", "data": "{}"}},
	}
	e.Push(ctx, mutations)

	changes, hasMore, err := e.Pull(ctx, "test", 0, 100)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("Pull returned %d changes, want 2", len(changes))
	}
	if hasMore {
		t.Error("Pull returned hasMore=true, want false")
	}

	// Verify change fields
	if changes[0].Cursor != 1 || changes[1].Cursor != 2 {
		t.Errorf("Change cursors incorrect")
	}
	if changes[0].Op != sync.Create {
		t.Errorf("changes[0].Op = %q, want %q", changes[0].Op, sync.Create)
	}
}

func TestEngine_Pull_Pagination(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Create 10 items
	for i := 0; i < 10; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{
				"entity": "items",
				"id":     string(rune('a' + i)),
				"data":   "{}",
			}},
		})
	}

	// Pull first page
	page1, hasMore1, _ := e.Pull(ctx, "test", 0, 3)
	if len(page1) != 3 {
		t.Errorf("Page 1 has %d items, want 3", len(page1))
	}
	if !hasMore1 {
		t.Error("Page 1 hasMore=false, want true")
	}

	// Pull second page
	lastCursor := page1[len(page1)-1].Cursor
	page2, hasMore2, _ := e.Pull(ctx, "test", lastCursor, 3)
	if len(page2) != 3 {
		t.Errorf("Page 2 has %d items, want 3", len(page2))
	}
	if !hasMore2 {
		t.Error("Page 2 hasMore=false, want true")
	}

	// Pull until end
	lastCursor = page2[len(page2)-1].Cursor
	page3, hasMore3, _ := e.Pull(ctx, "test", lastCursor, 100)
	if len(page3) != 4 {
		t.Errorf("Page 3 has %d items, want 4", len(page3))
	}
	if hasMore3 {
		t.Error("Page 3 hasMore=true, want false")
	}
}

func TestEngine_Pull_DefaultScope(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Create data with no scope
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	})

	// Pull with empty scope should use _default
	changes, _, _ := e.Pull(ctx, "", 0, 100)
	if len(changes) != 1 {
		t.Errorf("Pull with empty scope returned %d changes, want 1", len(changes))
	}
}

func TestEngine_Snapshot_Empty(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	data, cursor, err := e.Snapshot(ctx, "test")
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Snapshot returned %d entities, want 0", len(data))
	}
	if cursor != 0 {
		t.Errorf("cursor = %d, want 0", cursor)
	}
}

func TestEngine_Snapshot_WithData(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Create data
	mutations := []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "users", "id": "1", "data": `{"name":"A"}`}},
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "users", "id": "2", "data": `{"name":"B"}`}},
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "posts", "id": "1", "data": `{"title":"P"}`}},
	}
	e.Push(ctx, mutations)

	data, cursor, err := e.Snapshot(ctx, "test")
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	if len(data) != 2 {
		t.Errorf("Snapshot has %d entity types, want 2", len(data))
	}
	if len(data["users"]) != 2 {
		t.Errorf("users has %d items, want 2", len(data["users"]))
	}
	if len(data["posts"]) != 1 {
		t.Errorf("posts has %d items, want 1", len(data["posts"]))
	}
	if cursor != 3 {
		t.Errorf("cursor = %d, want 3", cursor)
	}
}

func TestEngine_Accessors(t *testing.T) {
	store := memory.NewStore()
	log := memory.NewLog()

	e := sync.New(sync.Options{
		Store:   store,
		Log:     log,
		Mutator: testMutator(),
	})

	if e.Store() != store {
		t.Error("Store() returned wrong store")
	}
	if e.Log() != log {
		t.Error("Log() returned wrong log")
	}
}

func TestMutatorMap_Register(t *testing.T) {
	ctx := context.Background()
	m := sync.NewMutatorMap()

	called := false
	m.Register("test", func(ctx context.Context, s sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		called = true
		return nil, nil
	})

	m.Apply(ctx, nil, sync.Mutation{Name: "test"})
	if !called {
		t.Error("Registered handler was not called")
	}
}

func TestMutatorFunc(t *testing.T) {
	ctx := context.Background()
	called := false

	var f sync.MutatorFunc = func(ctx context.Context, s sync.Store, m sync.Mutation) ([]sync.Change, error) {
		called = true
		return []sync.Change{{Entity: "e", ID: "1", Op: sync.Create}}, nil
	}

	changes, err := f.Apply(ctx, nil, sync.Mutation{})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !called {
		t.Error("MutatorFunc was not called")
	}
	if len(changes) != 1 {
		t.Errorf("Got %d changes, want 1", len(changes))
	}
}

func TestMultiNotifier(t *testing.T) {
	var calls []string

	n := sync.MultiNotifier{
		sync.NotifierFunc(func(scope string, cursor uint64) {
			calls = append(calls, "first")
		}),
		sync.NotifierFunc(func(scope string, cursor uint64) {
			calls = append(calls, "second")
		}),
	}

	n.Notify("scope", 1)

	if len(calls) != 2 {
		t.Errorf("Got %d calls, want 2", len(calls))
	}
	if calls[0] != "first" || calls[1] != "second" {
		t.Errorf("Calls in wrong order: %v", calls)
	}
}
