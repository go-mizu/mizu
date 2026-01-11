package api_test

import (
	"net/http"
	"testing"
)

func TestBaseWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "base@example.com")

	ws := createWorkspace(t, ts, token, "Workspace", "workspace")
	wsID := requireString(t, ws, "id")

	base := createBase(t, ts, token, wsID, "Base One")
	baseID := requireString(t, base, "id")
	if requireString(t, base, "color") == "" {
		t.Fatalf("expected base color to be set")
	}

	status, data := ts.doJSON(http.MethodGet, "/bases/"+baseID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	baseGet := requireMap(t, data, "base")
	if requireString(t, baseGet, "id") != baseID {
		t.Fatalf("base id mismatch")
	}
	if len(requireSlice(t, data, "tables")) != 0 {
		t.Fatalf("expected no tables in base")
	}

	status, data = ts.doJSON(http.MethodPatch, "/bases/"+baseID, map[string]any{
		"name": "Base Updated",
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "base"), "name") != "Base Updated" {
		t.Fatalf("expected updated base name")
	}

	status, data = ts.doJSON(http.MethodGet, "/bases/"+baseID+"/tables", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "tables")) != 0 {
		t.Fatalf("expected empty tables list")
	}

	table := createTable(t, ts, token, baseID, "Table One")
	tableID := requireString(t, table, "id")

	status, data = ts.doJSON(http.MethodGet, "/bases/"+baseID+"/tables", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "tables")) != 1 {
		t.Fatalf("expected tables list to include created table")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/bases/"+baseID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/bases/"+baseID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/tables/"+tableID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404 for deleted base tables, got %d", status)
	}
}
