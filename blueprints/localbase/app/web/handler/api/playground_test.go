package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestRequestHistory(t *testing.T) {
	t.Run("NewRequestHistory creates empty history", func(t *testing.T) {
		h := NewRequestHistory(10)
		if h == nil {
			t.Fatal("expected non-nil history")
		}
		if len(h.entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(h.entries))
		}
		if h.maxSize != 10 {
			t.Errorf("expected maxSize 10, got %d", h.maxSize)
		}
	})

	t.Run("Add inserts at front", func(t *testing.T) {
		h := NewRequestHistory(10)
		h.Add(RequestHistoryEntry{ID: "1", Path: "/first"})
		h.Add(RequestHistoryEntry{ID: "2", Path: "/second"})

		entries := h.List(10, 0)
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
		if entries[0].Path != "/second" {
			t.Errorf("expected first entry to be /second, got %s", entries[0].Path)
		}
		if entries[1].Path != "/first" {
			t.Errorf("expected second entry to be /first, got %s", entries[1].Path)
		}
	})

	t.Run("Add trims to maxSize", func(t *testing.T) {
		h := NewRequestHistory(3)
		h.Add(RequestHistoryEntry{ID: "1"})
		h.Add(RequestHistoryEntry{ID: "2"})
		h.Add(RequestHistoryEntry{ID: "3"})
		h.Add(RequestHistoryEntry{ID: "4"})

		entries := h.List(10, 0)
		if len(entries) != 3 {
			t.Errorf("expected 3 entries (maxSize), got %d", len(entries))
		}
		// Oldest entry should be removed
		if entries[2].ID != "2" {
			t.Errorf("expected oldest entry to be 2, got %s", entries[2].ID)
		}
	})

	t.Run("List with offset", func(t *testing.T) {
		h := NewRequestHistory(10)
		h.Add(RequestHistoryEntry{ID: "1"})
		h.Add(RequestHistoryEntry{ID: "2"})
		h.Add(RequestHistoryEntry{ID: "3"})

		entries := h.List(10, 1)
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("List with limit", func(t *testing.T) {
		h := NewRequestHistory(10)
		h.Add(RequestHistoryEntry{ID: "1"})
		h.Add(RequestHistoryEntry{ID: "2"})
		h.Add(RequestHistoryEntry{ID: "3"})

		entries := h.List(2, 0)
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("List with offset beyond length returns empty", func(t *testing.T) {
		h := NewRequestHistory(10)
		h.Add(RequestHistoryEntry{ID: "1"})

		entries := h.List(10, 5)
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("Clear removes all entries", func(t *testing.T) {
		h := NewRequestHistory(10)
		h.Add(RequestHistoryEntry{ID: "1"})
		h.Add(RequestHistoryEntry{ID: "2"})
		h.Clear()

		entries := h.List(10, 0)
		if len(entries) != 0 {
			t.Errorf("expected 0 entries after clear, got %d", len(entries))
		}
	})
}

func TestPlaygroundHandler_GetEndpoints(t *testing.T) {
	handler := NewPlaygroundHandler(nil) // nil store is OK for this test

	app := mizu.New()
	app.Get("/api/playground/endpoints", handler.GetEndpoints)

	req := httptest.NewRequest("GET", "/api/playground/endpoints", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Categories []EndpointCategory `json:"categories"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Categories) == 0 {
		t.Error("expected at least one category")
	}

	// Check that expected categories exist
	categoryNames := make(map[string]bool)
	for _, cat := range response.Categories {
		categoryNames[cat.Name] = true
	}

	expectedCategories := []string{"Authentication", "Database", "Storage", "Edge Functions", "Realtime", "Dashboard"}
	for _, name := range expectedCategories {
		if !categoryNames[name] {
			t.Errorf("expected category %q not found", name)
		}
	}
}

func TestPlaygroundHandler_GetHistory(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	// Add some history entries
	handler.history.Add(RequestHistoryEntry{ID: "1", Path: "/test1", Status: 200})
	handler.history.Add(RequestHistoryEntry{ID: "2", Path: "/test2", Status: 201})

	app := mizu.New()
	app.Get("/api/playground/history", handler.GetHistory)

	req := httptest.NewRequest("GET", "/api/playground/history", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response struct {
		History []RequestHistoryEntry `json:"history"`
		Total   int                   `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(response.History))
	}
}

func TestPlaygroundHandler_GetHistory_WithPagination(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	// Add some history entries
	for i := 0; i < 10; i++ {
		handler.history.Add(RequestHistoryEntry{ID: string(rune('0' + i)), Path: "/test"})
	}

	app := mizu.New()
	app.Get("/api/playground/history", handler.GetHistory)

	req := httptest.NewRequest("GET", "/api/playground/history?limit=5&offset=2", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response struct {
		History []RequestHistoryEntry `json:"history"`
		Total   int                   `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.History) != 5 {
		t.Errorf("expected 5 history entries with limit, got %d", len(response.History))
	}
}

func TestPlaygroundHandler_ClearHistory(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	// Add some history entries
	handler.history.Add(RequestHistoryEntry{ID: "1", Path: "/test1"})
	handler.history.Add(RequestHistoryEntry{ID: "2", Path: "/test2"})

	app := mizu.New()
	app.Delete("/api/playground/history", handler.ClearHistory)

	req := httptest.NewRequest("DELETE", "/api/playground/history", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify history is empty
	entries := handler.history.List(10, 0)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestPlaygroundHandler_SaveHistory(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	app := mizu.New()
	app.Post("/api/playground/history", handler.SaveHistory)

	entry := RequestHistoryEntry{
		Method:     "GET",
		Path:       "/test",
		Status:     200,
		DurationMs: 50,
	}
	body, _ := json.Marshal(entry)

	req := httptest.NewRequest("POST", "/api/playground/history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify entry was saved
	entries := handler.history.List(10, 0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Path != "/test" {
		t.Errorf("expected path /test, got %s", entries[0].Path)
	}
	if entries[0].ID == "" {
		t.Error("expected ID to be generated")
	}
	if entries[0].Timestamp == "" {
		t.Error("expected Timestamp to be generated")
	}
}

func TestPlaygroundHandler_Execute_InvalidMethod(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	app := mizu.New()
	app.Post("/api/playground/execute", handler.Execute)

	reqBody := ExecuteRequest{
		Method: "INVALID",
		Path:   "/test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/playground/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "invalid HTTP method" {
		t.Errorf("expected error 'invalid HTTP method', got %q", response["error"])
	}
}

func TestPlaygroundHandler_Execute_InvalidBody(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	app := mizu.New()
	app.Post("/api/playground/execute", handler.Execute)

	req := httptest.NewRequest("POST", "/api/playground/execute", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPlaygroundHandler_Execute_SecurityCheck(t *testing.T) {
	handler := NewPlaygroundHandler(nil)

	app := mizu.New()
	app.Post("/api/playground/execute", handler.Execute)

	reqBody := ExecuteRequest{
		Method: "GET",
		Path:   "https://external.com/api", // Should be blocked
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/playground/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "only local endpoints are allowed" {
		t.Errorf("expected security error, got %q", response["error"])
	}
}

func TestEndpointCategory_JSON(t *testing.T) {
	cat := EndpointCategory{
		Name:        "Test",
		Icon:        "test",
		Description: "Test category",
		Endpoints: []Endpoint{
			{
				Method:      "GET",
				Path:        "/test",
				Description: "Test endpoint",
			},
		},
	}

	data, err := json.Marshal(cat)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded EndpointCategory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != cat.Name {
		t.Errorf("expected name %q, got %q", cat.Name, decoded.Name)
	}
}

func TestExecuteRequest_JSON(t *testing.T) {
	req := ExecuteRequest{
		Method:  "POST",
		Path:    "/test",
		Headers: map[string]string{"Content-Type": "application/json"},
		Query:   map[string]string{"limit": "10"},
		Body:    json.RawMessage(`{"key": "value"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ExecuteRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Method != req.Method {
		t.Errorf("expected method %q, got %q", req.Method, decoded.Method)
	}
	if decoded.Path != req.Path {
		t.Errorf("expected path %q, got %q", req.Path, decoded.Path)
	}
}

func TestExecuteResponse_JSON(t *testing.T) {
	resp := ExecuteResponse{
		Status:     200,
		StatusText: "OK",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       json.RawMessage(`{"result": "success"}`),
		DurationMs: 50,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ExecuteResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("expected status %d, got %d", resp.Status, decoded.Status)
	}
	if decoded.DurationMs != resp.DurationMs {
		t.Errorf("expected duration %d, got %d", resp.DurationMs, decoded.DurationMs)
	}
}

func TestRequestHistoryConcurrency(t *testing.T) {
	h := NewRequestHistory(100)

	// Run concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				h.Add(RequestHistoryEntry{ID: string(rune('0' + id*10 + j))})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have at most maxSize entries
	entries := h.List(200, 0)
	if len(entries) > 100 {
		t.Errorf("expected at most 100 entries, got %d", len(entries))
	}
}

// Mock context for testing
type mockContext struct {
	context.Context
}
