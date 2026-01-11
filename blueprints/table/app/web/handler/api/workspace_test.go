package api_test

import (
	"net/http"
	"testing"
)

func TestWorkspaceWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "workspace@example.com")

	status, data := ts.doJSON(http.MethodGet, "/workspaces", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "workspaces")) != 0 {
		t.Fatalf("expected empty workspace list")
	}

	ws := createWorkspace(t, ts, token, "Acme", "acme")
	wsID := requireString(t, ws, "id")

	status, data = ts.doJSON(http.MethodGet, "/workspaces", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "workspaces")) != 1 {
		t.Fatalf("expected 1 workspace")
	}

	status, data = ts.doJSON(http.MethodGet, "/workspaces/"+wsID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	wsGet := requireMap(t, data, "workspace")
	if requireString(t, wsGet, "id") != wsID {
		t.Fatalf("workspace id mismatch")
	}

	status, data = ts.doJSON(http.MethodPatch, "/workspaces/"+wsID, map[string]any{
		"name": "Acme Updated",
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	wsUpdate := requireMap(t, data, "workspace")
	if requireString(t, wsUpdate, "name") != "Acme Updated" {
		t.Fatalf("expected updated name")
	}

	ws2 := createWorkspace(t, ts, token, "Beta", "beta")
	ws2Slug := requireString(t, ws2, "slug")

	status, _ = ts.doJSON(http.MethodPatch, "/workspaces/"+wsID, map[string]any{
		"slug": ws2Slug,
	}, token)
	if status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", status)
	}

	status, data = ts.doJSON(http.MethodGet, "/workspaces/"+wsID+"/bases", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "bases")) != 0 {
		t.Fatalf("expected empty base list")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/workspaces/"+wsID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/workspaces/"+wsID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}
}
