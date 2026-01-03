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

// Tests for all property types to ensure data persistence

func TestRowHandler_PropertyTypes_Text(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with text property
	body := `{"properties":{"name":"John Doe","description":"A long text description"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", created.Properties["name"])
	}

	// Update text property
	body = `{"properties":{"name":"Jane Doe"}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}

	var updated rows.Row
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated.Properties["name"] != "Jane Doe" {
		t.Errorf("expected updated name 'Jane Doe', got %v", updated.Properties["name"])
	}
}

func TestRowHandler_PropertyTypes_Number(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with number properties
	body := `{"properties":{"age":25,"price":99.99,"count":100}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["age"] != float64(25) {
		t.Errorf("expected age 25, got %v", created.Properties["age"])
	}

	if created.Properties["price"] != float64(99.99) {
		t.Errorf("expected price 99.99, got %v", created.Properties["price"])
	}

	// Update number property
	body = `{"properties":{"age":30}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}
}

func TestRowHandler_PropertyTypes_Checkbox(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with checkbox property
	body := `{"properties":{"completed":true,"archived":false}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["completed"] != true {
		t.Errorf("expected completed true, got %v", created.Properties["completed"])
	}

	if created.Properties["archived"] != false {
		t.Errorf("expected archived false, got %v", created.Properties["archived"])
	}

	// Toggle checkbox
	body = `{"properties":{"completed":false}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}

	var updated rows.Row
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated.Properties["completed"] != false {
		t.Errorf("expected completed toggled to false, got %v", updated.Properties["completed"])
	}
}

func TestRowHandler_PropertyTypes_Date(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with date property
	body := `{"properties":{"due_date":"2025-12-31T23:59:59Z","created_at":"2025-01-01T00:00:00Z"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["due_date"] != "2025-12-31T23:59:59Z" {
		t.Errorf("expected due_date '2025-12-31T23:59:59Z', got %v", created.Properties["due_date"])
	}

	// Update date
	body = `{"properties":{"due_date":"2026-01-15T12:00:00Z"}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}
}

func TestRowHandler_PropertyTypes_Select(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with select property (option ID)
	body := `{"properties":{"status":"opt_active","priority":"opt_high"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["status"] != "opt_active" {
		t.Errorf("expected status 'opt_active', got %v", created.Properties["status"])
	}

	// Change select value
	body = `{"properties":{"status":"opt_done"}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}

	var updated rows.Row
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated.Properties["status"] != "opt_done" {
		t.Errorf("expected status 'opt_done', got %v", updated.Properties["status"])
	}
}

func TestRowHandler_PropertyTypes_MultiSelect(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with multi-select property (array of option IDs)
	body := `{"properties":{"tags":["tag_work","tag_urgent"]}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	tags, ok := created.Properties["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags to be array, got %T", created.Properties["tags"])
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Update multi-select (add/remove tags)
	body = `{"properties":{"tags":["tag_work","tag_personal","tag_important"]}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}

	var updated rows.Row
	json.Unmarshal(w.Body.Bytes(), &updated)

	updatedTags, _ := updated.Properties["tags"].([]interface{})
	if len(updatedTags) != 3 {
		t.Errorf("expected 3 tags after update, got %d", len(updatedTags))
	}
}

func TestRowHandler_PropertyTypes_URL(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Patch("/rows/{id}", handler.Update)

	// Create with URL property
	body := `{"properties":{"website":"https://example.com","docs":"https://docs.example.com/api"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["website"] != "https://example.com" {
		t.Errorf("expected website 'https://example.com', got %v", created.Properties["website"])
	}

	// Update URL
	body = `{"properties":{"website":"https://new-example.com"}}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}
}

func TestRowHandler_PropertyTypes_Email(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)

	// Create with email property
	body := `{"properties":{"email":"user@example.com","contact":"support@company.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["email"] != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %v", created.Properties["email"])
	}
}

func TestRowHandler_PropertyTypes_Phone(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)

	// Create with phone property
	body := `{"properties":{"phone":"+1-555-123-4567","fax":"+1-555-987-6543"}}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	if created.Properties["phone"] != "+1-555-123-4567" {
		t.Errorf("expected phone '+1-555-123-4567', got %v", created.Properties["phone"])
	}
}

func TestRowHandler_PropertyTypes_AllTypesIntegration(t *testing.T) {
	store := newMockRowsStore()
	svc := rows.NewService(store)
	handler := api.NewRow(svc, func(c *mizu.Ctx) string { return "user123" })

	app := mizu.New()
	app.Post("/databases/{id}/rows", handler.Create)
	app.Get("/rows/{id}", handler.Get)
	app.Patch("/rows/{id}", handler.Update)

	// Create row with all property types
	body := `{
		"properties": {
			"title": "Test Item",
			"count": 42,
			"completed": true,
			"due_date": "2025-12-31T00:00:00Z",
			"status": "opt_active",
			"tags": ["tag_a", "tag_b"],
			"website": "https://example.com",
			"email": "test@example.com",
			"phone": "+1-555-000-0000"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/databases/db123/rows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", w.Body.String())
	}

	var created rows.Row
	json.Unmarshal(w.Body.Bytes(), &created)

	// Verify GET returns same data
	req = httptest.NewRequest(http.MethodGet, "/rows/"+created.ID, nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get failed: %s", w.Body.String())
	}

	var fetched rows.Row
	json.Unmarshal(w.Body.Bytes(), &fetched)

	// Verify all properties match
	if fetched.Properties["title"] != "Test Item" {
		t.Errorf("title mismatch after fetch")
	}
	if fetched.Properties["count"] != float64(42) {
		t.Errorf("count mismatch after fetch")
	}
	if fetched.Properties["completed"] != true {
		t.Errorf("completed mismatch after fetch")
	}
	if fetched.Properties["status"] != "opt_active" {
		t.Errorf("status mismatch after fetch")
	}

	// Update multiple properties at once
	body = `{
		"properties": {
			"title": "Updated Item",
			"count": 100,
			"completed": false
		}
	}`
	req = httptest.NewRequest(http.MethodPatch, "/rows/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update failed: %s", w.Body.String())
	}

	var updated rows.Row
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated.Properties["title"] != "Updated Item" {
		t.Errorf("title not updated")
	}
	if updated.Properties["count"] != float64(100) {
		t.Errorf("count not updated")
	}
	if updated.Properties["completed"] != false {
		t.Errorf("completed not updated")
	}
}
