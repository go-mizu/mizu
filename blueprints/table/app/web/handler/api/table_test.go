package api_test

import (
	"net/http"
	"testing"
)

func TestTableWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "table@example.com")

	ws := createWorkspace(t, ts, token, "Workspace", "table-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	table := createTable(t, ts, token, requireString(t, base, "id"), "Table")
	tableID := requireString(t, table, "id")

	status, data := ts.doJSON(http.MethodGet, "/tables/"+tableID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	tableGet := requireMap(t, data, "table")
	if requireString(t, tableGet, "id") != tableID {
		t.Fatalf("table id mismatch")
	}
	if len(requireSlice(t, data, "fields")) != 0 {
		t.Fatalf("expected no fields in new table")
	}
	if len(requireSlice(t, data, "views")) != 0 {
		t.Fatalf("expected no views in new table")
	}

	status, data = ts.doJSON(http.MethodPatch, "/tables/"+tableID, map[string]any{
		"name":        "Table Updated",
		"description": "Updated description",
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "table"), "name") != "Table Updated" {
		t.Fatalf("expected updated table name")
	}

	field := createField(t, ts, token, tableID, "Name", "text")
	view := createView(t, ts, token, tableID, "Grid", "grid")

	status, data = ts.doJSON(http.MethodGet, "/tables/"+tableID+"/fields", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "fields")) != 1 {
		t.Fatalf("expected 1 field")
	}
	if requireString(t, requireSlice(t, data, "fields")[0].(map[string]any), "id") != requireString(t, field, "id") {
		t.Fatalf("expected field id to match")
	}

	status, data = ts.doJSON(http.MethodGet, "/tables/"+tableID+"/views", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "views")) != 1 {
		t.Fatalf("expected 1 view")
	}
	if requireString(t, requireSlice(t, data, "views")[0].(map[string]any), "id") != requireString(t, view, "id") {
		t.Fatalf("expected view id to match")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/tables/"+tableID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/tables/"+tableID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}
}
