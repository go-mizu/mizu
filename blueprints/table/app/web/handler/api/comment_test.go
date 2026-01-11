package api_test

import (
	"net/http"
	"testing"
)

func TestCommentWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, user := registerUser(t, ts, "comment@example.com")
	userID := requireString(t, user, "id")

	ws := createWorkspace(t, ts, token, "Workspace", "comment-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	table := createTable(t, ts, token, requireString(t, base, "id"), "Table")
	record := createRecord(t, ts, token, requireString(t, table, "id"), map[string]any{
		"Name": "Alpha",
	})
	recordID := requireString(t, record, "id")

	comment := createComment(t, ts, token, recordID, "Initial comment")
	commentID := requireString(t, comment, "id")

	status, data := ts.doJSON(http.MethodGet, "/comments/record/"+recordID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "comments")) != 1 {
		t.Fatalf("expected 1 comment")
	}

	status, data = ts.doJSON(http.MethodGet, "/comments/"+commentID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	commentGet := requireMap(t, data, "comment")
	if requireString(t, commentGet, "userId") != userID {
		t.Fatalf("expected user id in comment")
	}

	status, data = ts.doJSON(http.MethodPatch, "/comments/"+commentID, map[string]any{
		"content":     "Updated comment",
		"is_resolved": true,
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	updated := requireMap(t, data, "comment")
	if requireString(t, updated, "content") != "Updated comment" {
		t.Fatalf("expected updated content")
	}
	if !requireBool(t, updated, "isResolved") {
		t.Fatalf("expected comment to be resolved")
	}

	status, data = ts.doJSON(http.MethodPost, "/comments/"+commentID+"/resolve", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !requireBool(t, requireMap(t, data, "comment"), "isResolved") {
		t.Fatalf("expected comment to be resolved")
	}

	status, data = ts.doJSON(http.MethodPost, "/comments/"+commentID+"/unresolve", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireBool(t, requireMap(t, data, "comment"), "isResolved") {
		t.Fatalf("expected comment to be unresolved")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/comments/"+commentID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/comments/"+commentID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}
}
