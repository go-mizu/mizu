package sync_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// testMutator creates a simple mutator for testing
func testMutator() sync.Mutator {
	return sync.MutatorFunc(func(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
		switch mut.Name {
		case "create":
			entity, _ := mut.Args["entity"].(string)
			id, _ := mut.Args["id"].(string)
			data, _ := mut.Args["data"].(string)

			if err := store.Set(ctx, mut.Scope, entity, id, json.RawMessage(data)); err != nil {
				return nil, err
			}

			return []sync.Change{
				{Entity: entity, ID: id, Op: sync.Create, Data: json.RawMessage(data)},
			}, nil

		case "update":
			entity, _ := mut.Args["entity"].(string)
			id, _ := mut.Args["id"].(string)
			data, _ := mut.Args["data"].(string)

			if _, err := store.Get(ctx, mut.Scope, entity, id); err != nil {
				return nil, err
			}

			if err := store.Set(ctx, mut.Scope, entity, id, json.RawMessage(data)); err != nil {
				return nil, err
			}

			return []sync.Change{
				{Entity: entity, ID: id, Op: sync.Update, Data: json.RawMessage(data)},
			}, nil

		case "delete":
			entity, _ := mut.Args["entity"].(string)
			id, _ := mut.Args["id"].(string)

			if err := store.Delete(ctx, mut.Scope, entity, id); err != nil {
				return nil, err
			}

			return []sync.Change{
				{Entity: entity, ID: id, Op: sync.Delete},
			}, nil

		case "noop":
			return nil, nil

		default:
			return nil, sync.ErrUnknownMutation
		}
	})
}

func newTestEngine() *sync.Engine {
	return sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
	})
}

func newTestApp() (*mizu.App, *sync.Engine) {
	e := newTestEngine()
	app := mizu.New()
	e.Mount(app)
	return app, e
}

func doRequest(app *mizu.App, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)
	return rec
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
	if r.Code != "unknown_mutation" {
		t.Errorf("result.Code = %q, want %q", r.Code, "unknown_mutation")
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
	if r.Code != "not_found" {
		t.Errorf("result.Code = %q, want %q", r.Code, "not_found")
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

	if notifiedScope != sync.DefaultScope {
		t.Errorf("notified scope = %q, want %q", notifiedScope, sync.DefaultScope)
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

func TestNotifierFunc(t *testing.T) {
	var called bool
	var gotScope string
	var gotCursor uint64

	f := sync.NotifierFunc(func(scope string, cursor uint64) {
		called = true
		gotScope = scope
		gotCursor = cursor
	})

	f.Notify("test", 42)

	if !called {
		t.Error("NotifierFunc was not called")
	}
	if gotScope != "test" {
		t.Errorf("scope = %q, want %q", gotScope, "test")
	}
	if gotCursor != 42 {
		t.Errorf("cursor = %d, want 42", gotCursor)
	}
}

// -----------------------------------------------------------------------------
// HTTP Tests
// -----------------------------------------------------------------------------

func TestHTTP_Push_Success(t *testing.T) {
	app, _ := newTestApp()

	body := map[string]any{
		"mutations": []map[string]any{
			{
				"id":    "mut-1",
				"name":  "create",
				"scope": "test",
				"args":  map[string]any{"entity": "users", "id": "1", "data": `{}`},
			},
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Results []sync.Result `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Got %d results, want 1", len(resp.Results))
	}
	if !resp.Results[0].OK {
		t.Errorf("result.OK = false, want true")
	}
}

func TestHTTP_Push_BadRequest(t *testing.T) {
	app, _ := newTestApp()

	req := httptest.NewRequest("POST", "/_sync/push", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_Push_NoMutations(t *testing.T) {
	app, _ := newTestApp()

	body := map[string]any{"mutations": []map[string]any{}}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_Push_MultipleMutations(t *testing.T) {
	app, _ := newTestApp()

	body := map[string]any{
		"mutations": []map[string]any{
			{"name": "create", "scope": "test", "args": map[string]any{"entity": "e", "id": "1", "data": "{}"}},
			{"name": "create", "scope": "test", "args": map[string]any{"entity": "e", "id": "2", "data": "{}"}},
			{"name": "unknown", "scope": "test"}, // This one should fail
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Results []sync.Result `json:"results"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Results) != 3 {
		t.Fatalf("Got %d results, want 3", len(resp.Results))
	}
	if !resp.Results[0].OK || !resp.Results[1].OK {
		t.Error("First two results should be OK")
	}
	if resp.Results[2].OK {
		t.Error("Third result should not be OK")
	}
}

func TestHTTP_Pull_Success(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create some data
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "2", "data": "{}"}},
	})

	body := map[string]any{"scope": "test", "cursor": 0, "limit": 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Changes []sync.Change `json:"changes"`
		HasMore bool          `json:"has_more"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 2 {
		t.Errorf("Got %d changes, want 2", len(resp.Changes))
	}
	if resp.HasMore {
		t.Error("HasMore should be false")
	}
}

func TestHTTP_Pull_WithCursor(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create some data
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"}},
		})
	}

	body := map[string]any{"scope": "test", "cursor": 2, "limit": 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp struct {
		Changes []sync.Change `json:"changes"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 3 {
		t.Errorf("Got %d changes, want 3 (cursor 3,4,5)", len(resp.Changes))
	}
}

func TestHTTP_Pull_Pagination(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create 10 items
	for i := 0; i < 10; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"}},
		})
	}

	body := map[string]any{"scope": "test", "cursor": 0, "limit": 3}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp struct {
		Changes []sync.Change `json:"changes"`
		HasMore bool          `json:"has_more"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 3 {
		t.Errorf("Got %d changes, want 3", len(resp.Changes))
	}
	if !resp.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestHTTP_Pull_BadRequest(t *testing.T) {
	app, _ := newTestApp()

	req := httptest.NewRequest("POST", "/_sync/pull", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_Snapshot_Success(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create some data
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "users", "id": "1", "data": `{"n":"A"}`}},
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "posts", "id": "1", "data": `{"t":"P"}`}},
	})

	body := map[string]any{"scope": "test"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Data   map[string]map[string]json.RawMessage `json:"data"`
		Cursor uint64                                `json:"cursor"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Data) != 2 {
		t.Errorf("Got %d entity types, want 2", len(resp.Data))
	}
	if resp.Cursor != 2 {
		t.Errorf("Cursor = %d, want 2", resp.Cursor)
	}
}

func TestHTTP_Snapshot_Empty(t *testing.T) {
	app, _ := newTestApp()

	body := map[string]any{"scope": "empty"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Data map[string]map[string]json.RawMessage `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Data) != 0 {
		t.Errorf("Got %d entity types, want 0", len(resp.Data))
	}
}

func TestHTTP_Snapshot_BadRequest(t *testing.T) {
	app, _ := newTestApp()

	req := httptest.NewRequest("POST", "/_sync/snapshot", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_MountAt(t *testing.T) {
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Mutator: testMutator(),
	})

	app := mizu.New()
	e.MountAt(app, "/api/v1/sync")

	body := map[string]any{"scope": "test"}
	rec := doRequest(app, "POST", "/api/v1/sync/pull", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_DefaultScope(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create data with empty scope
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	})

	// Pull with empty scope should work
	body := map[string]any{"scope": "", "cursor": 0}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp struct {
		Changes []sync.Change `json:"changes"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 1 {
		t.Errorf("Got %d changes, want 1", len(resp.Changes))
	}
}

// -----------------------------------------------------------------------------
// New Feature Tests
// -----------------------------------------------------------------------------

func TestEngine_Pull_CursorTooOld(t *testing.T) {
	ctx := context.Background()
	log := memory.NewLog()

	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     log,
		Mutator: testMutator(),
	})

	// Create some data
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"}},
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

func TestHTTP_Pull_CursorTooOld(t *testing.T) {
	log := memory.NewLog()
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     log,
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
	})

	app := mizu.New()
	e.Mount(app)

	ctx := context.Background()

	// Create some data and trim
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"}},
		})
	}
	log.Trim(ctx, "test", 3)

	// Pull with trimmed cursor
	body := map[string]any{"scope": "test", "cursor": 1}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	if rec.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGone)
	}

	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Code != "cursor_too_old" {
		t.Errorf("code = %q, want %q", resp.Code, "cursor_too_old")
	}
}

func TestEngine_InjectableTime(t *testing.T) {
	ctx := context.Background()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Mutator: testMutator(),
		Now: func() time.Time {
			return fixedTime
		},
	})

	results, _ := e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	})

	if len(results[0].Changes) == 0 {
		t.Fatal("No changes returned")
	}

	changeTime := results[0].Changes[0].Time
	if !changeTime.Equal(fixedTime) {
		t.Errorf("Change time = %v, want %v", changeTime, fixedTime)
	}
}

func TestEngine_ScopeFunc(t *testing.T) {
	ctx := context.Background()

	var capturedScope string
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Mutator: testMutator(),
		ScopeFunc: func(ctx context.Context, claimed string) (string, error) {
			// Override scope to "authorized-scope"
			capturedScope = claimed
			return "authorized-scope", nil
		},
		Notify: sync.NotifierFunc(func(scope string, cursor uint64) {
			// Verify notification uses the overridden scope
			if scope != "authorized-scope" {
				panic("expected authorized-scope")
			}
		}),
	})

	// Client claims scope "user-scope"
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "user-scope", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	})

	if capturedScope != "user-scope" {
		t.Errorf("ScopeFunc received scope %q, want %q", capturedScope, "user-scope")
	}

	// Verify data is stored in the authorized scope
	changes, _, _ := e.Pull(ctx, "authorized-scope", 0, 100)
	if len(changes) != 1 {
		t.Errorf("Expected 1 change in authorized-scope, got %d", len(changes))
	}

	// Original scope should be empty
	changes, _, _ = e.Pull(ctx, "user-scope", 0, 100)
	if len(changes) != 0 {
		t.Errorf("Expected 0 changes in user-scope, got %d", len(changes))
	}
}

func TestEngine_ScopeFunc_Error(t *testing.T) {
	ctx := context.Background()

	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Mutator: testMutator(),
		ScopeFunc: func(ctx context.Context, claimed string) (string, error) {
			return "", errors.New("unauthorized")
		},
	})

	results, _ := e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
	})

	if results[0].OK {
		t.Error("Expected mutation to fail due to ScopeFunc error")
	}
	if results[0].Code != "internal_error" {
		t.Errorf("code = %q, want %q", results[0].Code, "internal_error")
	}
}

func TestEngine_MaxPullLimit(t *testing.T) {
	ctx := context.Background()

	e := sync.New(sync.Options{
		Store:        memory.NewStore(),
		Log:          memory.NewLog(),
		Mutator:      testMutator(),
		MaxPullLimit: 5, // Limit to 5
	})

	// Create 10 items
	for i := 0; i < 10; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"}},
		})
	}

	// Try to pull with a large limit
	changes, hasMore, _ := e.Pull(ctx, "test", 0, 1000)

	if len(changes) != 5 {
		t.Errorf("Got %d changes, want 5 (capped by MaxPullLimit)", len(changes))
	}
	if !hasMore {
		t.Error("Expected hasMore=true")
	}
}

func TestHTTP_MaxPushBatch(t *testing.T) {
	e := sync.New(sync.Options{
		Store:        memory.NewStore(),
		Log:          memory.NewLog(),
		Mutator:      testMutator(),
		MaxPushBatch: 3, // Limit to 3
	})

	app := mizu.New()
	e.Mount(app)

	// Try to push more than limit
	mutations := make([]map[string]any, 5)
	for i := 0; i < 5; i++ {
		mutations[i] = map[string]any{
			"name":  "create",
			"scope": "test",
			"args":  map[string]any{"entity": "e", "id": string(rune('a' + i)), "data": "{}"},
		}
	}

	body := map[string]any{"mutations": mutations}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "too many mutations in batch" {
		t.Errorf("error = %q, want %q", resp.Error, "too many mutations in batch")
	}
}

func TestDefaultScope_Constant(t *testing.T) {
	// Verify the constant is exported and correct
	if sync.DefaultScope != "_default" {
		t.Errorf("DefaultScope = %q, want %q", sync.DefaultScope, "_default")
	}
}

func TestErrCursorTooOld_Exported(t *testing.T) {
	// Verify the error is exported
	if sync.ErrCursorTooOld == nil {
		t.Error("ErrCursorTooOld should not be nil")
	}
	if sync.ErrCursorTooOld.Error() != "sync: cursor too old" {
		t.Errorf("ErrCursorTooOld message = %q, want %q", sync.ErrCursorTooOld.Error(), "sync: cursor too old")
	}
}
