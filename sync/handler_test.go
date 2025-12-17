package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func setupTestApp() (*mizu.App, *Engine) {
	app := mizu.New()
	engine := createTestEngine()
	engine.Mount(app)
	return app, engine
}

func TestPushHandler_Valid(t *testing.T) {
	app, _ := setupTestApp()

	req := PushRequest{
		Mutations: []Mutation{{
			Name:  "todo/create",
			Scope: "user:123",
			Args:  map[string]any{"id": "1", "title": "Test"},
		}},
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/push", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp PushResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if !resp.Results[0].OK {
		t.Errorf("expected OK, got error: %s", resp.Results[0].Error)
	}
	if resp.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", resp.Cursor)
	}
}

func TestPushHandler_InvalidJSON(t *testing.T) {
	app, _ := setupTestApp()

	r := httptest.NewRequest("POST", "/_sync/push", bytes.NewReader([]byte("invalid json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPushHandler_EmptyMutations(t *testing.T) {
	app, _ := setupTestApp()

	req := PushRequest{Mutations: []Mutation{}}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/push", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPushHandler_MutationErrors(t *testing.T) {
	app, _ := setupTestApp()

	req := PushRequest{
		Mutations: []Mutation{
			{Name: "todo/create", Scope: "test", Args: map[string]any{"id": "1", "title": "Valid"}},
			{Name: "unknown", Scope: "test"}, // Will fail
		},
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/push", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp PushResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Results[0].OK != true {
		t.Error("first mutation should succeed")
	}
	if resp.Results[1].OK != false {
		t.Error("second mutation should fail")
	}
}

func TestPullHandler_Valid(t *testing.T) {
	app, engine := setupTestApp()
	ctx := context.Background()

	// Push some data first
	engine.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
	})

	req := PullRequest{
		Scope:  "user:123",
		Cursor: 0,
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/pull", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp PullResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(resp.Changes))
	}
	if resp.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", resp.Cursor)
	}
	if resp.HasMore {
		t.Error("expected hasMore to be false")
	}
}

func TestPullHandler_WithCursor(t *testing.T) {
	app, engine := setupTestApp()
	ctx := context.Background()

	engine.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "3", "title": "Third"}},
	})

	req := PullRequest{
		Scope:  "user:123",
		Cursor: 1, // Start after first change
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/pull", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	var resp PullResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Changes) != 2 {
		t.Errorf("expected 2 changes (from cursor 1), got %d", len(resp.Changes))
	}
}

func TestPullHandler_WithLimit(t *testing.T) {
	app, engine := setupTestApp()
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		engine.Push(ctx, []Mutation{{
			Name:  "todo/create",
			Scope: "user:123",
			Args:  map[string]any{"id": string(rune('0' + i)), "title": "Todo"},
		}})
	}

	req := PullRequest{
		Scope:  "user:123",
		Cursor: 0,
		Limit:  3,
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/pull", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	var resp PullResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Changes) != 3 {
		t.Errorf("expected 3 changes (limited), got %d", len(resp.Changes))
	}
	if !resp.HasMore {
		t.Error("expected hasMore to be true")
	}
}

func TestPullHandler_InvalidJSON(t *testing.T) {
	app, _ := setupTestApp()

	r := httptest.NewRequest("POST", "/_sync/pull", bytes.NewReader([]byte("invalid")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSnapshotHandler_Valid(t *testing.T) {
	app, engine := setupTestApp()
	ctx := context.Background()

	engine.Push(ctx, []Mutation{
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "1", "title": "First"}},
		{Name: "todo/create", Scope: "user:123", Args: map[string]any{"id": "2", "title": "Second"}},
	})

	req := SnapshotRequest{Scope: "user:123"}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/snapshot", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp SnapshotResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", resp.Cursor)
	}
	if len(resp.Data["todo"]) != 2 {
		t.Errorf("expected 2 todos, got %d", len(resp.Data["todo"]))
	}
}

func TestSnapshotHandler_MissingScope(t *testing.T) {
	app, _ := setupTestApp()

	req := SnapshotRequest{Scope: ""}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/_sync/snapshot", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMountAt_CustomPrefix(t *testing.T) {
	app := mizu.New()
	engine := createTestEngine()
	engine.MountAt(app, "/api/sync")

	req := PullRequest{Scope: "test", Cursor: 0}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/sync/pull", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetHandlers(t *testing.T) {
	engine := createTestEngine()
	handlers := engine.GetHandlers()

	if handlers.Push == nil {
		t.Error("expected non-nil Push handler")
	}
	if handlers.Pull == nil {
		t.Error("expected non-nil Pull handler")
	}
	if handlers.Snapshot == nil {
		t.Error("expected non-nil Snapshot handler")
	}
}
