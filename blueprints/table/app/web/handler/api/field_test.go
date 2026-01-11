package api_test

import (
	"net/http"
	"testing"
)

func TestFieldWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "field@example.com")

	ws := createWorkspace(t, ts, token, "Workspace", "field-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	table := createTable(t, ts, token, requireString(t, base, "id"), "Table")
	tableID := requireString(t, table, "id")

	field := createField(t, ts, token, tableID, "Name", "text")
	fieldID := requireString(t, field, "id")

	status, _ := ts.doJSON(http.MethodPost, "/fields", map[string]any{
		"table_id": tableID,
		"name":     "Bad",
		"type":     "nope",
	}, token)
	if status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", status)
	}

	status, data := ts.doJSON(http.MethodGet, "/fields/"+fieldID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "field"), "id") != fieldID {
		t.Fatalf("expected field id to match")
	}

	status, data = ts.doJSON(http.MethodPatch, "/fields/"+fieldID, map[string]any{
		"name":      "Name Updated",
		"width":     320,
		"is_hidden": true,
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	updated := requireMap(t, data, "field")
	if requireString(t, updated, "name") != "Name Updated" {
		t.Fatalf("expected updated name")
	}
	if requireBool(t, updated, "is_hidden") != true {
		t.Fatalf("expected field to be hidden")
	}

	field2 := createField(t, ts, token, tableID, "Status", "single_select")
	field2ID := requireString(t, field2, "id")

	status, data = ts.doJSON(http.MethodPost, "/fields/"+tableID+"/reorder", map[string]any{
		"field_ids": []string{field2ID, fieldID},
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !requireBool(t, data, "success") {
		t.Fatalf("expected success true")
	}

	status, data = ts.doJSON(http.MethodGet, "/tables/"+tableID+"/fields", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	fields := requireSlice(t, data, "fields")
	firstField := fields[0].(map[string]any)
	if requireString(t, firstField, "id") != field2ID {
		t.Fatalf("expected reorder to place field2 first")
	}

	status, data = ts.doJSON(http.MethodPost, "/fields/"+field2ID+"/options", map[string]any{
		"name":  "Open",
		"color": "#FF0000",
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	option := requireMap(t, data, "option")
	optionID := requireString(t, option, "id")

	status, data = ts.doJSON(http.MethodGet, "/fields/"+field2ID+"/options", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "options")) != 1 {
		t.Fatalf("expected one option")
	}

	status, data = ts.doJSON(http.MethodPatch, "/fields/"+field2ID+"/options/"+optionID, map[string]any{
		"name":  "Closed",
		"color": "#00FF00",
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !requireBool(t, data, "success") {
		t.Fatalf("expected success true")
	}

	status, data = ts.doJSON(http.MethodGet, "/fields/"+field2ID+"/options", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	options := requireSlice(t, data, "options")
	if requireString(t, options[0].(map[string]any), "name") != "Closed" {
		t.Fatalf("expected updated option name")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/fields/"+field2ID+"/options/"+optionID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, data = ts.doJSON(http.MethodGet, "/fields/"+field2ID+"/options", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "options")) != 0 {
		t.Fatalf("expected options list to be empty")
	}
}
