package sync_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// testData is a simple struct for test change data
type testData struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Op     string `json:"op"`
	Data   string `json:"data,omitempty"`
}

// testApplyFunc creates a simple apply function for testing
func testApplyFunc() sync.ApplyFunc {
	return func(ctx context.Context, mut sync.Mutation) ([]sync.Change, error) {
		// Parse args
		var args struct {
			Entity string `json:"entity"`
			ID     string `json:"id"`
			Data   string `json:"data"`
		}
		if mut.Args != nil {
			json.Unmarshal(mut.Args, &args)
		}

		switch mut.Name {
		case "create":
			data, _ := json.Marshal(testData{Entity: args.Entity, ID: args.ID, Op: "create", Data: args.Data})
			return []sync.Change{{Data: data}}, nil

		case "update":
			data, _ := json.Marshal(testData{Entity: args.Entity, ID: args.ID, Op: "update", Data: args.Data})
			return []sync.Change{{Data: data}}, nil

		case "delete":
			data, _ := json.Marshal(testData{Entity: args.Entity, ID: args.ID, Op: "delete"})
			return []sync.Change{{Data: data}}, nil

		case "noop":
			return nil, nil

		case "error":
			return nil, sync.ErrNotFound

		case "invalid":
			return nil, sync.ErrInvalidMutation

		default:
			return nil, errors.New("unknown mutation")
		}
	}
}

func newTestEngine() *sync.Engine {
	return sync.New(sync.Options{
		Log:    memory.NewLog(),
		Dedupe: memory.NewDedupe(),
		Apply:  testApplyFunc(),
	})
}

func makeArgs(entity, id, data string) json.RawMessage {
	args, _ := json.Marshal(map[string]string{
		"entity": entity,
		"id":     id,
		"data":   data,
	})
	return args
}

// -----------------------------------------------------------------------------
// Engine Tests
// -----------------------------------------------------------------------------

func TestEngine_Push_Success(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutations := []sync.Mutation{
		{
			ID:    "mut-1",
			Name:  "create",
			Scope: "test",
			Args:  makeArgs("users", "u1", `{"name":"Alice"}`),
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
}

func TestEngine_Push_Idempotency(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutation := sync.Mutation{
		ID:    "mut-1",
		Name:  "create",
		Scope: "test",
		Args:  makeArgs("users", "u1", `{"name":"Alice"}`),
	}

	// First push
	results1, _ := e.Push(ctx, []sync.Mutation{mutation})

	// Second push with same ID
	results2, _ := e.Push(ctx, []sync.Mutation{mutation})

	// Both should succeed
	if !results1[0].OK || !results2[0].OK {
		t.Error("Both pushes should succeed")
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
		Name:  "create",
		Scope: "test",
		Args:  makeArgs("users", "u1", `{}`),
	}

	// Push twice without ID
	e.Push(ctx, []sync.Mutation{mutation})
	e.Push(ctx, []sync.Mutation{mutation})

	// Both should execute (no idempotency without ID)
	changes, _, _ := e.Pull(ctx, "test", 0, 100)
	if len(changes) != 2 {
		t.Errorf("Without ID, both mutations should execute, got %d changes", len(changes))
	}
}

func TestEngine_Push_MutationError(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutations := []sync.Mutation{
		{Name: "error", Scope: "test"},
	}

	results, _ := e.Push(ctx, mutations)
	r := results[0]

	if r.OK {
		t.Error("result.OK = true, want false for error")
	}
	if r.Error == "" {
		t.Error("result.Error should not be empty")
	}
}

func TestEngine_Push_DefaultScope(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	mutations := []sync.Mutation{
		{Name: "create", Args: makeArgs("e", "1", "{}")},
	}

	results, _ := e.Push(ctx, mutations)
	if !results[0].OK {
		t.Error("Push should succeed")
	}

	// Pull with default scope
	changes, _, _ := e.Pull(ctx, "", 0, 100)
	if len(changes) != 1 {
		t.Errorf("Got %d changes in default scope, want 1", len(changes))
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
		{Name: "create", Scope: "test", Args: makeArgs("users", "1", "{}")},
		{Name: "create", Scope: "test", Args: makeArgs("users", "2", "{}")},
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
}

func TestEngine_Pull_Pagination(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine()

	// Create 10 items
	for i := 0; i < 10; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("items", string(rune('a'+i)), "{}")},
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
		{Name: "create", Args: makeArgs("e", "1", "{}")},
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
	if string(data) != "{}" {
		t.Errorf("Snapshot returned %q, want empty object", string(data))
	}
	if cursor != 0 {
		t.Errorf("cursor = %d, want 0", cursor)
	}
}

func TestEngine_Snapshot_WithSnapshotFunc(t *testing.T) {
	ctx := context.Background()

	e := sync.New(sync.Options{
		Log:   memory.NewLog(),
		Apply: testApplyFunc(),
		Snapshot: func(ctx context.Context, scope string) (json.RawMessage, uint64, error) {
			return json.RawMessage(`{"users":{"1":{"name":"Alice"}}}`), 5, nil
		},
	})

	data, cursor, err := e.Snapshot(ctx, "test")
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if string(data) != `{"users":{"1":{"name":"Alice"}}}` {
		t.Errorf("Snapshot returned %q, want users data", string(data))
	}
	if cursor != 5 {
		t.Errorf("cursor = %d, want 5", cursor)
	}
}

func TestEngine_Pull_CursorTooOld(t *testing.T) {
	ctx := context.Background()
	log := memory.NewLog()

	e := sync.New(sync.Options{
		Log:   log,
		Apply: testApplyFunc(),
	})

	// Create some data
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", string(rune('a'+i)), "{}")},
		})
	}

	// Trim log entries before cursor 3
	log.Trim(ctx, "test", 3)

	// Try to pull with a cursor that's been trimmed
	_, _, err := e.Pull(ctx, "test", 1, 100)
	if !errors.Is(err, sync.ErrCursorTooOld) {
		t.Errorf("Pull with old cursor should return ErrCursorTooOld, got %v", err)
	}

	// Pull with cursor at trim point should work
	changes, _, err := e.Pull(ctx, "test", 3, 100)
	if err != nil {
		t.Errorf("Pull with valid cursor should succeed, got %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("Got %d changes, want 2 (cursors 4 and 5)", len(changes))
	}
}

func TestEngine_InjectableTime(t *testing.T) {
	ctx := context.Background()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	e := sync.New(sync.Options{
		Log:   memory.NewLog(),
		Apply: testApplyFunc(),
		Now: func() time.Time {
			return fixedTime
		},
	})

	results, _ := e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: makeArgs("e", "1", "{}")},
	})

	if len(results[0].Changes) == 0 {
		t.Fatal("No changes returned")
	}

	changeTime := results[0].Changes[0].Time
	if !changeTime.Equal(fixedTime) {
		t.Errorf("Change time = %v, want %v", changeTime, fixedTime)
	}
}

func TestEngine_NoDedupe(t *testing.T) {
	ctx := context.Background()

	e := sync.New(sync.Options{
		Log:   memory.NewLog(),
		Apply: testApplyFunc(),
		// No Dedupe - idempotency disabled
	})

	mutation := sync.Mutation{
		ID:    "mut-1",
		Name:  "create",
		Scope: "test",
		Args:  makeArgs("users", "u1", `{}`),
	}

	// Push twice with same ID but no dedupe
	e.Push(ctx, []sync.Mutation{mutation})
	e.Push(ctx, []sync.Mutation{mutation})

	// Both should execute
	changes, _, _ := e.Pull(ctx, "test", 0, 100)
	if len(changes) != 2 {
		t.Errorf("Without dedupe, both mutations should execute, got %d changes", len(changes))
	}
}

func TestEngine_Log(t *testing.T) {
	log := memory.NewLog()
	e := sync.New(sync.Options{
		Log:   log,
		Apply: testApplyFunc(),
	})

	if e.Log() != log {
		t.Error("Log() should return the underlying log")
	}
}

func TestDefaultScope_Constant(t *testing.T) {
	if sync.DefaultScope != "_default" {
		t.Errorf("DefaultScope = %q, want %q", sync.DefaultScope, "_default")
	}
}

func TestErrCursorTooOld_Exported(t *testing.T) {
	if sync.ErrCursorTooOld == nil {
		t.Error("ErrCursorTooOld should not be nil")
	}
	if sync.ErrCursorTooOld.Error() != "sync: cursor too old" {
		t.Errorf("ErrCursorTooOld message = %q, want %q", sync.ErrCursorTooOld.Error(), "sync: cursor too old")
	}
}

func TestErrors_Exported(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{sync.ErrNotFound, "sync: not found"},
		{sync.ErrInvalidMutation, "sync: invalid mutation"},
		{sync.ErrConflict, "sync: conflict"},
		{sync.ErrCursorTooOld, "sync: cursor too old"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("%v.Error() = %q, want %q", tt.err, tt.err.Error(), tt.want)
		}
	}
}
