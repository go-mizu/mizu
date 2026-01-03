package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/app/web/handler/api"
	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

func TestRowHandler_Create(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)

	body := `{"properties":{"title":"Test Row","status":"active"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result rows.Row
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.DatabaseID != "db123" {
		t.Errorf("expected database_id db123, got %s", result.DatabaseID)
	}

	if result.Properties["title"] != "Test Row" {
		t.Errorf("expected title 'Test Row', got %v", result.Properties["title"])
	}
}

func TestRowHandler_Get(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	// Create a row first
	ctx := context.Background()
	_, _ = svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "Test"},
		CreatedBy:  "user123",
	})

	app := mizu.New()
	app.Get("/rows/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/rows/"+store.createdRow.ID, nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRowHandler_Update(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	// Create a row first
	ctx := context.Background()
	row, _ := svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "Original"},
		CreatedBy:  "user123",
	})

	app := mizu.New()
	app.Patch("/rows/{id}", handler.Update)

	body := `{"properties":{"title":"Updated","status":"done"}}`
	req := httptest.NewRequest(http.MethodPatch, "/rows/"+row.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result rows.Row
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Properties["title"] != "Updated" {
		t.Errorf("expected title 'Updated', got %v", result.Properties["title"])
	}
}

func TestRowHandler_Delete(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	// Create a row first
	ctx := context.Background()
	row, _ := svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "To Delete"},
		CreatedBy:  "user123",
	})

	app := mizu.New()
	app.Delete("/rows/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/rows/"+row.ID, nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRowHandler_List(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	ctx := context.Background()
	// Create some rows
	for i := 0; i < 5; i++ {
		_, _ = svc.Create(ctx, &rows.CreateIn{
			DatabaseID: "db123",
			Properties: map[string]interface{}{"title": "Row"},
			CreatedBy:  "user123",
		})
	}

	app := mizu.New()
	app.Get("/databases/{id}/rows", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/databases/db123/rows", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result rows.ListResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(result.Rows))
	}
}

func TestRowHandler_ListWithFilters(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	ctx := context.Background()
	// Create rows with different statuses
	_, _ = svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "Row 1", "status": "active"},
		CreatedBy:  "user123",
	})
	_, _ = svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "Row 2", "status": "done"},
		CreatedBy:  "user123",
	})

	app := mizu.New()
	app.Get("/databases/{id}/rows", handler.List)

	// Test with filter
	filters := `[{"property":"status","operator":"is","value":"active"}]`
	req := httptest.NewRequest(http.MethodGet, "/databases/db123/rows?filters="+filters, nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestRowHandler_Duplicate(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	ctx := context.Background()
	// Create a row first
	row, _ := svc.Create(ctx, &rows.CreateIn{
		DatabaseID: "db123",
		Properties: map[string]interface{}{"title": "Original"},
		CreatedBy:  "user123",
	})

	app := mizu.New()
	app.Post("/rows/{id}/duplicate", handler.Duplicate)

	req := httptest.NewRequest(http.MethodPost, "/rows/"+row.ID+"/duplicate", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result rows.Row
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID == row.ID {
		t.Error("duplicated row should have different ID")
	}
}

// mockRowsStore implements rows.Store for testing
type mockRowsStore struct {
	rows       map[string]*rows.Row
	createdRow *rows.Row
}

func newMockRowsStore() *mockRowsStore {
	return &mockRowsStore{
		rows: make(map[string]*rows.Row),
	}
}

func (s *mockRowsStore) Create(_ context.Context, row *rows.Row) error {
	s.rows[row.ID] = row
	s.createdRow = row
	return nil
}

func (s *mockRowsStore) GetByID(_ context.Context, id string) (*rows.Row, error) {
	if row, ok := s.rows[id]; ok {
		return row, nil
	}
	return nil, rows.ErrNotFound
}

func (s *mockRowsStore) Update(_ context.Context, id string, in *rows.UpdateIn) error {
	if row, ok := s.rows[id]; ok {
		row.Properties = in.Properties
		row.UpdatedBy = in.UpdatedBy
		return nil
	}
	return rows.ErrNotFound
}

func (s *mockRowsStore) Delete(_ context.Context, id string) error {
	delete(s.rows, id)
	return nil
}

func (s *mockRowsStore) List(_ context.Context, in *rows.ListIn) ([]*rows.Row, error) {
	var result []*rows.Row
	for _, row := range s.rows {
		if row.DatabaseID == in.DatabaseID {
			result = append(result, row)
		}
	}
	return result, nil
}

func (s *mockRowsStore) Count(_ context.Context, databaseID string, _ []rows.Filter) (int, error) {
	count := 0
	for _, row := range s.rows {
		if row.DatabaseID == databaseID {
			count++
		}
	}
	return count, nil
}

func (s *mockRowsStore) DeleteByDatabase(_ context.Context, databaseID string) error {
	for id, row := range s.rows {
		if row.DatabaseID == databaseID {
			delete(s.rows, id)
		}
	}
	return nil
}
