package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// --- Mock implementations for security testing ---

type mockWorkbooksAPI struct {
	workbooks map[string]*workbooks.Workbook
}

func newMockWorkbooksAPI() *mockWorkbooksAPI {
	return &mockWorkbooksAPI{workbooks: make(map[string]*workbooks.Workbook)}
}

func (m *mockWorkbooksAPI) Create(ctx context.Context, in *workbooks.CreateIn) (*workbooks.Workbook, error) {
	now := time.Now()
	wb := &workbooks.Workbook{
		ID:        "wb-" + in.Name,
		Name:      in.Name,
		OwnerID:   in.OwnerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.workbooks[wb.ID] = wb
	return wb, nil
}

func (m *mockWorkbooksAPI) GetByID(ctx context.Context, id string) (*workbooks.Workbook, error) {
	if wb, ok := m.workbooks[id]; ok {
		return wb, nil
	}
	return nil, workbooks.ErrNotFound
}

func (m *mockWorkbooksAPI) List(ctx context.Context, ownerID string) ([]*workbooks.Workbook, error) {
	var result []*workbooks.Workbook
	for _, wb := range m.workbooks {
		if wb.OwnerID == ownerID {
			result = append(result, wb)
		}
	}
	return result, nil
}

func (m *mockWorkbooksAPI) Update(ctx context.Context, id string, in *workbooks.UpdateIn) (*workbooks.Workbook, error) {
	if wb, ok := m.workbooks[id]; ok {
		if in.Name != "" {
			wb.Name = in.Name
		}
		wb.UpdatedAt = time.Now()
		return wb, nil
	}
	return nil, workbooks.ErrNotFound
}

func (m *mockWorkbooksAPI) Delete(ctx context.Context, id string) error {
	if _, ok := m.workbooks[id]; !ok {
		return workbooks.ErrNotFound
	}
	delete(m.workbooks, id)
	return nil
}

func (m *mockWorkbooksAPI) Copy(ctx context.Context, id string, newName string, userID string) (*workbooks.Workbook, error) {
	if _, ok := m.workbooks[id]; !ok {
		return nil, workbooks.ErrNotFound
	}
	return m.Create(ctx, &workbooks.CreateIn{Name: newName, OwnerID: userID})
}

type mockSheetsAPI struct {
	sheets map[string]*sheets.Sheet
}

func newMockSheetsAPI() *mockSheetsAPI {
	return &mockSheetsAPI{sheets: make(map[string]*sheets.Sheet)}
}

func (m *mockSheetsAPI) Create(ctx context.Context, in *sheets.CreateIn) (*sheets.Sheet, error) {
	now := time.Now()
	sheet := &sheets.Sheet{
		ID:         "sheet-" + in.Name,
		WorkbookID: in.WorkbookID,
		Name:       in.Name,
		Index:      in.Index,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.sheets[sheet.ID] = sheet
	return sheet, nil
}

func (m *mockSheetsAPI) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	if sheet, ok := m.sheets[id]; ok {
		return sheet, nil
	}
	return nil, sheets.ErrNotFound
}

func (m *mockSheetsAPI) List(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	var result []*sheets.Sheet
	for _, sheet := range m.sheets {
		if sheet.WorkbookID == workbookID {
			result = append(result, sheet)
		}
	}
	return result, nil
}

func (m *mockSheetsAPI) Update(ctx context.Context, id string, in *sheets.UpdateIn) (*sheets.Sheet, error) {
	if sheet, ok := m.sheets[id]; ok {
		if in.Name != "" {
			sheet.Name = in.Name
		}
		sheet.UpdatedAt = time.Now()
		return sheet, nil
	}
	return nil, sheets.ErrNotFound
}

func (m *mockSheetsAPI) Delete(ctx context.Context, id string) error {
	if _, ok := m.sheets[id]; !ok {
		return sheets.ErrNotFound
	}
	delete(m.sheets, id)
	return nil
}

func (m *mockSheetsAPI) Copy(ctx context.Context, id string, newName string) (*sheets.Sheet, error) {
	if s, ok := m.sheets[id]; ok {
		return m.Create(ctx, &sheets.CreateIn{WorkbookID: s.WorkbookID, Name: newName})
	}
	return nil, sheets.ErrNotFound
}

func (m *mockSheetsAPI) SetRowHeight(ctx context.Context, sheetID string, row int, height int) error {
	return nil
}

func (m *mockSheetsAPI) SetColWidth(ctx context.Context, sheetID string, col int, width int) error {
	return nil
}

func (m *mockSheetsAPI) HideRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *mockSheetsAPI) HideCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

func (m *mockSheetsAPI) ShowRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *mockSheetsAPI) ShowCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

// --- Security Tests ---

// TestWorkbook_IDOR_Get verifies users cannot access other users' workbooks.
func TestWorkbook_IDOR_Get(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbooks for different users
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "User1 Workbook",
		OwnerID: "user-1",
	}
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Create handler with user-1's context
	handler := NewWorkbook(mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}", handler.Get)

	// Test: user-1 can access their own workbook
	t.Run("owner_can_access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Owner access: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	// Test: user-1 cannot access user-2's workbook (IDOR protection)
	t.Run("cannot_access_others_workbook", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-2", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})
}

// TestWorkbook_IDOR_Update verifies users cannot update other users' workbooks.
func TestWorkbook_IDOR_Update(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Handler with user-1's context (attacker)
	handler := NewWorkbook(mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Patch("/workbooks/{id}", handler.Update)

	// Test: user-1 cannot update user-2's workbook
	body := `{"name": "Hacked Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/workbooks/wb-2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	// Verify workbook name unchanged
	if mockWorkbooks.workbooks["wb-2"].Name != "User2 Workbook" {
		t.Error("Workbook name should not have changed")
	}
}

// TestWorkbook_IDOR_Delete verifies users cannot delete other users' workbooks.
func TestWorkbook_IDOR_Delete(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Handler with user-1's context (attacker)
	handler := NewWorkbook(mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Delete("/workbooks/{id}", handler.Delete)

	// Test: user-1 cannot delete user-2's workbook
	req := httptest.NewRequest(http.MethodDelete, "/workbooks/wb-2", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	// Verify workbook still exists
	if _, exists := mockWorkbooks.workbooks["wb-2"]; !exists {
		t.Error("Workbook should not have been deleted")
	}
}

// TestSheet_IDOR_Get verifies users cannot access sheets in other users' workbooks.
func TestSheet_IDOR_Get(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Create sheet in user-2's workbook
	mockSheets.sheets["sheet-1"] = &sheets.Sheet{
		ID:         "sheet-1",
		WorkbookID: "wb-2",
		Name:       "Secret Data",
	}

	// Handler with user-1's context (attacker)
	handler := NewSheet(mockSheets, mockWorkbooks, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/sheets/{id}", handler.Get)

	// Test: user-1 cannot access sheet in user-2's workbook
	req := httptest.NewRequest(http.MethodGet, "/sheets/sheet-1", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestSheet_IDOR_Create verifies users cannot create sheets in other users' workbooks.
func TestSheet_IDOR_Create(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Handler with user-1's context (attacker)
	handler := NewSheet(mockSheets, mockWorkbooks, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/sheets", handler.Create)

	// Test: user-1 cannot create sheet in user-2's workbook
	body := `{"workbookId": "wb-2", "name": "Injected Sheet"}`
	req := httptest.NewRequest(http.MethodPost, "/sheets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	// Verify no sheet was created
	if len(mockSheets.sheets) != 0 {
		t.Error("Sheet should not have been created")
	}
}

// TestUnauthorized_NoUser verifies endpoints reject requests without authentication.
func TestUnauthorized_NoUser(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	// Create workbook
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	// Handler with no user (simulating missing auth)
	handler := NewWorkbook(mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "" // Empty user ID = no authentication
	})

	app := mizu.New()
	app.Get("/workbooks/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized check: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestFilename_Sanitization verifies filenames are sanitized to prevent header injection.
func TestFilename_Sanitization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "workbook.xlsx", "workbook.xlsx"},
		{"with_quotes", "work\"book.xlsx", "work_book.xlsx"},
		{"with_backslash", "work\\book.xlsx", "work_book.xlsx"},
		{"with_newline", "work\nbook.xlsx", "work_book.xlsx"},
		{"with_carriage_return", "work\rbook.xlsx", "work_book.xlsx"},
		{"with_null", "work\x00book.xlsx", "work_book.xlsx"},
		{"header_injection_attempt", "a.xlsx\r\nX-Injected: true", "a.xlsx__X-Injected: true"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestErrorMessages_NoLeak verifies error messages don't leak sensitive information.
func TestErrorMessages_NoLeak(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()

	handler := NewWorkbook(mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}", handler.Get)

	// Request non-existent workbook
	req := httptest.NewRequest(http.MethodGet, "/workbooks/nonexistent", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	// Parse response
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Error message should be generic, not leak internal details
	errorMsg := response["error"]
	forbiddenPatterns := []string{
		"sql", "SQL",
		"database", "Database",
		"stack", "trace",
		"panic",
		"/Users/", "/home/",
		".go:",
	}

	for _, pattern := range forbiddenPatterns {
		if bytes.Contains([]byte(errorMsg), []byte(pattern)) {
			t.Errorf("Error message contains sensitive pattern %q: %s", pattern, errorMsg)
		}
	}
}
