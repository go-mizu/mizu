package api_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestRecordWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, _ := registerUser(t, ts, "record@example.com")

	ws := createWorkspace(t, ts, token, "Workspace", "record-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	table := createTable(t, ts, token, requireString(t, base, "id"), "Table")
	tableID := requireString(t, table, "id")

	status, _ := ts.doJSON(http.MethodGet, "/records", nil, token)
	if status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", status)
	}

	record := createRecord(t, ts, token, tableID, map[string]any{
		"Name": "Alpha",
	})
	recordID := requireString(t, record, "id")

	status, data := ts.doJSON(http.MethodGet, "/records/"+recordID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if requireString(t, requireMap(t, data, "record"), "id") != recordID {
		t.Fatalf("record id mismatch")
	}

	status, data = ts.doJSON(http.MethodPatch, "/records/"+recordID, map[string]any{
		"fields": map[string]any{
			"Name": "Alpha Updated",
		},
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	updated := requireMap(t, data, "record")
	values := updated["values"].(map[string]any)
	if values["Name"] != "Alpha Updated" {
		t.Fatalf("expected record value to update")
	}

	createRecord(t, ts, token, tableID, map[string]any{
		"Name": "Beta",
	})
	status, data = ts.doJSON(http.MethodGet, fmt.Sprintf("/records?table_id=%s&limit=1", tableID), nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "records")) != 1 {
		t.Fatalf("expected 1 record from limited list")
	}
	if !requireBool(t, data, "has_more") {
		t.Fatalf("expected has_more true")
	}
	nextCursor, ok := data["next_cursor"].(string)
	if !ok || nextCursor == "" {
		t.Fatalf("expected next_cursor to be set")
	}

	status, data = ts.doJSON(http.MethodGet, fmt.Sprintf("/records?table_id=%s&cursor=%s&limit=1", tableID, nextCursor), nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "records")) != 1 {
		t.Fatalf("expected 1 record from cursor list")
	}

	status, data = ts.doJSON(http.MethodPost, "/records/batch", map[string]any{
		"table_id": tableID,
		"records": []map[string]any{
			{"fields": map[string]any{"Name": "Gamma"}},
			{"fields": map[string]any{"Name": "Delta"}},
		},
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	batchRecords := requireSlice(t, data, "records")
	if len(batchRecords) != 2 {
		t.Fatalf("expected 2 batch records")
	}
	firstBatch := batchRecords[0].(map[string]any)
	secondBatch := batchRecords[1].(map[string]any)

	status, data = ts.doJSON(http.MethodPatch, "/records/batch", map[string]any{
		"records": []map[string]any{
			{"id": requireString(t, firstBatch, "id"), "fields": map[string]any{"Name": "Gamma Updated"}},
			{"id": requireString(t, secondBatch, "id"), "fields": map[string]any{"Name": "Delta Updated"}},
		},
	}, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	updatedBatch := requireSlice(t, data, "records")
	if len(updatedBatch) != 2 {
		t.Fatalf("expected 2 updated records")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/records/batch", map[string]any{
		"ids": []string{requireString(t, firstBatch, "id"), requireString(t, secondBatch, "id")},
	}, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodDelete, "/records/"+recordID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/records/"+recordID, nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}
}
