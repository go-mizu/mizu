package sync_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
)

func newTestApp() (*mizu.App, *sync.Engine) {
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Applied: memory.NewApplied(),
		Mutator: testMutator(),
	})

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

func TestHTTP_Push_Success(t *testing.T) {
	app, _ := newTestApp()

	body := sync.PushRequest{
		Mutations: []sync.Mutation{
			{
				ID:    "mut-1",
				Name:  "create",
				Scope: "test",
				Args:  map[string]any{"entity": "users", "id": "1", "data": `{}`},
			},
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp sync.PushResponse
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

	body := sync.PushRequest{Mutations: []sync.Mutation{}}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_Push_MultipleMutations(t *testing.T) {
	app, _ := newTestApp()

	body := sync.PushRequest{
		Mutations: []sync.Mutation{
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "1", "data": "{}"}},
			{Name: "create", Scope: "test", Args: map[string]any{"entity": "e", "id": "2", "data": "{}"}},
			{Name: "unknown", Scope: "test"}, // This one should fail
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp sync.PushResponse
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

	body := sync.PullRequest{Scope: "test", Cursor: 0, Limit: 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp sync.PullResponse
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

	body := sync.PullRequest{Scope: "test", Cursor: 2, Limit: 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp sync.PullResponse
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

	body := sync.PullRequest{Scope: "test", Cursor: 0, Limit: 3}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp sync.PullResponse
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

	body := sync.SnapshotRequest{Scope: "test"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp sync.SnapshotResponse
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

	body := sync.SnapshotRequest{Scope: "empty"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp sync.SnapshotResponse
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

	body := sync.PullRequest{Scope: "test"}
	rec := doRequest(app, "POST", "/api/v1/sync/pull", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTP_Handlers(t *testing.T) {
	e := sync.New(sync.Options{
		Store:   memory.NewStore(),
		Log:     memory.NewLog(),
		Mutator: testMutator(),
	})

	handlers := e.Handlers()

	if handlers.Push == nil {
		t.Error("Handlers().Push is nil")
	}
	if handlers.Pull == nil {
		t.Error("Handlers().Pull is nil")
	}
	if handlers.Snapshot == nil {
		t.Error("Handlers().Snapshot is nil")
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
	body := sync.PullRequest{Scope: "", Cursor: 0}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp sync.PullResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 1 {
		t.Errorf("Got %d changes, want 1", len(resp.Changes))
	}
}
