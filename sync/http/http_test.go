package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/sync"
	synchttp "github.com/go-mizu/mizu/sync/http"
	"github.com/go-mizu/mizu/sync/memory"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

func testApplyFunc() sync.ApplyFunc {
	return func(ctx context.Context, mut sync.Mutation) ([]sync.Change, error) {
		switch mut.Name {
		case "create", "update":
			return []sync.Change{{Data: mut.Args}}, nil
		case "noop":
			return nil, nil
		case "error":
			return nil, sync.ErrNotFound
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

func newTestApp() (*mizu.App, *sync.Engine) {
	e := newTestEngine()
	t := synchttp.New(synchttp.Options{Engine: e})
	app := mizu.New()
	t.Mount(app)
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

func makeArgs(entity, id, data string) json.RawMessage {
	args, _ := json.Marshal(map[string]string{
		"entity": entity,
		"id":     id,
		"data":   data,
	})
	return args
}

// -----------------------------------------------------------------------------
// HTTP Push Tests
// -----------------------------------------------------------------------------

func TestHTTP_Push_Success(t *testing.T) {
	app, _ := newTestApp()

	body := synchttp.PushRequest{
		Mutations: []sync.Mutation{
			{
				ID:    "mut-1",
				Name:  "create",
				Scope: "test",
				Args:  makeArgs("users", "1", `{}`),
			},
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp synchttp.PushResponse
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

	body := synchttp.PushRequest{Mutations: []sync.Mutation{}}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTP_Push_MultipleMutations(t *testing.T) {
	app, _ := newTestApp()

	body := synchttp.PushRequest{
		Mutations: []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", "1", "{}")},
			{Name: "create", Scope: "test", Args: makeArgs("e", "2", "{}")},
			{Name: "error", Scope: "test"}, // This one should fail
		},
	}

	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp synchttp.PushResponse
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

func TestHTTP_Push_MaxBatch(t *testing.T) {
	e := newTestEngine()
	tr := synchttp.New(synchttp.Options{
		Engine:       e,
		MaxPushBatch: 3, // Limit to 3
	})

	app := mizu.New()
	tr.Mount(app)

	// Try to push more than limit
	mutations := make([]sync.Mutation, 5)
	for i := 0; i < 5; i++ {
		mutations[i] = sync.Mutation{
			Name:  "create",
			Scope: "test",
			Args:  makeArgs("e", string(rune('a'+i)), "{}"),
		}
	}

	body := synchttp.PushRequest{Mutations: mutations}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp synchttp.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "too many mutations in batch" {
		t.Errorf("error = %q, want %q", resp.Error, "too many mutations in batch")
	}
}

// -----------------------------------------------------------------------------
// HTTP Pull Tests
// -----------------------------------------------------------------------------

func TestHTTP_Pull_Success(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create some data
	e.Push(ctx, []sync.Mutation{
		{Name: "create", Scope: "test", Args: makeArgs("e", "1", "{}")},
		{Name: "create", Scope: "test", Args: makeArgs("e", "2", "{}")},
	})

	body := synchttp.PullRequest{Scope: "test", Cursor: 0, Limit: 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp synchttp.PullResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 2 {
		t.Errorf("Got %d changes, want 2", len(resp.Changes))
	}
	if resp.HasMore {
		t.Error("HasMore should be false")
	}
	if resp.NextCursor != 2 {
		t.Errorf("NextCursor = %d, want 2", resp.NextCursor)
	}
}

func TestHTTP_Pull_WithCursor(t *testing.T) {
	app, e := newTestApp()
	ctx := context.Background()

	// Create some data
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", string(rune('a'+i)), "{}")},
		})
	}

	body := synchttp.PullRequest{Scope: "test", Cursor: 2, Limit: 10}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp synchttp.PullResponse
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
			{Name: "create", Scope: "test", Args: makeArgs("e", string(rune('a'+i)), "{}")},
		})
	}

	body := synchttp.PullRequest{Scope: "test", Cursor: 0, Limit: 3}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp synchttp.PullResponse
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

func TestHTTP_Pull_CursorTooOld(t *testing.T) {
	log := memory.NewLog()
	e := sync.New(sync.Options{
		Log:    log,
		Dedupe: memory.NewDedupe(),
		Apply:  testApplyFunc(),
	})

	tr := synchttp.New(synchttp.Options{Engine: e})
	app := mizu.New()
	tr.Mount(app)

	ctx := context.Background()

	// Create some data and trim
	for i := 0; i < 5; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", string(rune('a'+i)), "{}")},
		})
	}
	log.Trim(ctx, "test", 3)

	// Pull with trimmed cursor
	body := synchttp.PullRequest{Scope: "test", Cursor: 1}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	if rec.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGone)
	}

	var resp synchttp.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Code != synchttp.CodeCursorTooOld {
		t.Errorf("code = %q, want %q", resp.Code, synchttp.CodeCursorTooOld)
	}
}

func TestHTTP_Pull_MaxLimit(t *testing.T) {
	e := newTestEngine()
	tr := synchttp.New(synchttp.Options{
		Engine:       e,
		MaxPullLimit: 5, // Limit to 5
	})

	app := mizu.New()
	tr.Mount(app)

	ctx := context.Background()

	// Create 10 items
	for i := 0; i < 10; i++ {
		e.Push(ctx, []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", string(rune('a'+i)), "{}")},
		})
	}

	// Try to pull with a large limit
	body := synchttp.PullRequest{Scope: "test", Cursor: 0, Limit: 1000}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp synchttp.PullResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 5 {
		t.Errorf("Got %d changes, want 5 (capped by MaxPullLimit)", len(resp.Changes))
	}
	if !resp.HasMore {
		t.Error("Expected hasMore=true")
	}
}

// -----------------------------------------------------------------------------
// HTTP Snapshot Tests
// -----------------------------------------------------------------------------

func TestHTTP_Snapshot_Success(t *testing.T) {
	e := sync.New(sync.Options{
		Log:   memory.NewLog(),
		Apply: testApplyFunc(),
		Snapshot: func(ctx context.Context, scope string) (json.RawMessage, uint64, error) {
			return json.RawMessage(`{"users":{"1":{"name":"Alice"}}}`), 5, nil
		},
	})

	tr := synchttp.New(synchttp.Options{Engine: e})
	app := mizu.New()
	tr.Mount(app)

	body := synchttp.SnapshotRequest{Scope: "test"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp synchttp.SnapshotResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if string(resp.Data) != `{"users":{"1":{"name":"Alice"}}}` {
		t.Errorf("Data = %q, want users data", string(resp.Data))
	}
	if resp.Cursor != 5 {
		t.Errorf("Cursor = %d, want 5", resp.Cursor)
	}
}

func TestHTTP_Snapshot_Empty(t *testing.T) {
	app, _ := newTestApp()

	body := synchttp.SnapshotRequest{Scope: "empty"}
	rec := doRequest(app, "POST", "/_sync/snapshot", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp synchttp.SnapshotResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if string(resp.Data) != "{}" {
		t.Errorf("Data = %q, want empty object", string(resp.Data))
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

// -----------------------------------------------------------------------------
// HTTP Mount Tests
// -----------------------------------------------------------------------------

func TestHTTP_MountAt(t *testing.T) {
	e := newTestEngine()
	tr := synchttp.New(synchttp.Options{Engine: e})

	app := mizu.New()
	tr.MountAt(app, "/api/v1/sync")

	body := synchttp.PullRequest{Scope: "test"}
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
		{Name: "create", Args: makeArgs("e", "1", "{}")},
	})

	// Pull with empty scope should work
	body := synchttp.PullRequest{Scope: "", Cursor: 0}
	rec := doRequest(app, "POST", "/_sync/pull", body)

	var resp synchttp.PullResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Changes) != 1 {
		t.Errorf("Got %d changes, want 1", len(resp.Changes))
	}
}

// -----------------------------------------------------------------------------
// ScopeFunc Tests
// -----------------------------------------------------------------------------

func TestHTTP_ScopeFunc(t *testing.T) {
	e := newTestEngine()

	tr := synchttp.New(synchttp.Options{
		Engine: e,
		ScopeFunc: func(ctx context.Context, claimed string) (string, error) {
			// Override scope to "authorized-scope"
			return "authorized-scope", nil
		},
	})

	app := mizu.New()
	tr.Mount(app)

	// Push with claimed scope
	body := synchttp.PushRequest{
		Mutations: []sync.Mutation{
			{Name: "create", Scope: "user-scope", Args: makeArgs("e", "1", "{}")},
		},
	}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify data is stored in the authorized scope
	ctx := context.Background()
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

func TestHTTP_ScopeFunc_Error(t *testing.T) {
	e := newTestEngine()

	tr := synchttp.New(synchttp.Options{
		Engine: e,
		ScopeFunc: func(ctx context.Context, claimed string) (string, error) {
			return "", errors.New("unauthorized")
		},
	})

	app := mizu.New()
	tr.Mount(app)

	body := synchttp.PushRequest{
		Mutations: []sync.Mutation{
			{Name: "create", Scope: "test", Args: makeArgs("e", "1", "{}")},
		},
	}
	rec := doRequest(app, "POST", "/_sync/push", body)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// -----------------------------------------------------------------------------
// MapError Tests
// -----------------------------------------------------------------------------

func TestMapError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{sync.ErrNotFound, http.StatusNotFound, synchttp.CodeNotFound},
		{sync.ErrInvalidMutation, http.StatusBadRequest, synchttp.CodeInvalid},
		{sync.ErrConflict, http.StatusConflict, synchttp.CodeConflict},
		{sync.ErrCursorTooOld, http.StatusGone, synchttp.CodeCursorTooOld},
		{errors.New("other"), http.StatusInternalServerError, synchttp.CodeInternal},
	}

	for _, tt := range tests {
		status, code := synchttp.MapError(tt.err)
		if status != tt.wantStatus {
			t.Errorf("MapError(%v) status = %d, want %d", tt.err, status, tt.wantStatus)
		}
		if code != tt.wantCode {
			t.Errorf("MapError(%v) code = %q, want %q", tt.err, code, tt.wantCode)
		}
	}
}

// -----------------------------------------------------------------------------
// Error Codes Tests
// -----------------------------------------------------------------------------

func TestErrorCodes(t *testing.T) {
	// Verify error codes are exported correctly
	codes := []string{
		synchttp.CodeNotFound,
		synchttp.CodeInvalid,
		synchttp.CodeCursorTooOld,
		synchttp.CodeConflict,
		synchttp.CodeInternal,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}
}
