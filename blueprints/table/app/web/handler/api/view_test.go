package api_test

import (
	"net/http"
	"testing"
)

func TestViewWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "view@example.com")

	ws := createWorkspace(t, ts, token, "Workspace", "view-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	table := createTable(t, ts, token, requireString(t, base, "id"), "Table")
	tableID := requireString(t, table, "id")

	viewA := createView(t, ts, token, tableID, "Grid", "grid")
	viewB := createView(t, ts, token, tableID, "Kanban", "kanban")
	viewAID := requireString(t, viewA, "id")
	viewBID := requireString(t, viewB, "id")

	status, data := ts.doJSON(http.MethodGet, "/views/"+viewAID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "view"), "id") != viewAID {
		t.Fatalf("view id mismatch")
	}

	status, data = ts.doJSON(http.MethodPatch, "/views/"+viewAID, map[string]any{
		"name": "Grid Updated",
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "view"), "name") != "Grid Updated" {
		t.Fatalf("expected updated view name")
	}

	status, data = ts.doJSON(http.MethodPost, "/views/"+viewAID+"/duplicate", nil, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	duplicate := requireMap(t, data, "view")

	status, data = ts.doJSON(http.MethodPost, "/views/"+tableID+"/reorder", map[string]any{
		"view_ids": []string{viewBID, viewAID, requireString(t, duplicate, "id")},
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !requireBool(t, data, "success") {
		t.Fatalf("expected success true")
	}

	status, data = ts.doJSON(http.MethodGet, "/tables/"+tableID+"/views", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "views")) != 3 {
		t.Fatalf("expected 3 views")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/views/"+viewAID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/views/"+viewAID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}
}
