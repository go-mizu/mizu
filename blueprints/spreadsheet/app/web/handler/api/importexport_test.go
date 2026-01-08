package api

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/export"
	"github.com/go-mizu/blueprints/spreadsheet/feature/importer"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// --- Mock implementations for import/export testing ---

type mockExporterAPI struct {
	exportWorkbookFunc func(ctx context.Context, workbookID string, format export.Format, opts *export.Options) (*export.Result, error)
	exportSheetFunc    func(ctx context.Context, sheetID string, format export.Format, opts *export.Options) (*export.Result, error)
}

func (m *mockExporterAPI) ExportWorkbook(ctx context.Context, workbookID string, format export.Format, opts *export.Options) (*export.Result, error) {
	if m.exportWorkbookFunc != nil {
		return m.exportWorkbookFunc(ctx, workbookID, format, opts)
	}
	return &export.Result{
		ContentType: "text/csv; charset=utf-8",
		Filename:    "test.csv",
		Data:        strings.NewReader("col1,col2\nval1,val2"),
		Size:        20,
	}, nil
}

func (m *mockExporterAPI) ExportSheet(ctx context.Context, sheetID string, format export.Format, opts *export.Options) (*export.Result, error) {
	if m.exportSheetFunc != nil {
		return m.exportSheetFunc(ctx, sheetID, format, opts)
	}
	return &export.Result{
		ContentType: "text/csv; charset=utf-8",
		Filename:    "sheet.csv",
		Data:        strings.NewReader("col1,col2\nval1,val2"),
		Size:        20,
	}, nil
}

func (m *mockExporterAPI) SupportedFormats() []export.Format {
	return []export.Format{export.FormatCSV, export.FormatTSV, export.FormatXLSX, export.FormatJSON, export.FormatPDF, export.FormatHTML}
}

type mockImporterAPI struct {
	importToWorkbookFunc func(ctx context.Context, workbookID string, reader io.Reader, filename string, format importer.Format, opts *importer.Options) (*importer.Result, error)
	importToSheetFunc    func(ctx context.Context, sheetID string, reader io.Reader, filename string, format importer.Format, opts *importer.Options) (*importer.Result, error)
}

func (m *mockImporterAPI) ImportToWorkbook(ctx context.Context, workbookID string, reader io.Reader, filename string, format importer.Format, opts *importer.Options) (*importer.Result, error) {
	if m.importToWorkbookFunc != nil {
		return m.importToWorkbookFunc(ctx, workbookID, reader, filename, format, opts)
	}
	return &importer.Result{
		SheetID:       "new-sheet-1",
		RowsImported:  10,
		ColsImported:  3,
		CellsImported: 30,
	}, nil
}

func (m *mockImporterAPI) ImportToSheet(ctx context.Context, sheetID string, reader io.Reader, filename string, format importer.Format, opts *importer.Options) (*importer.Result, error) {
	if m.importToSheetFunc != nil {
		return m.importToSheetFunc(ctx, sheetID, reader, filename, format, opts)
	}
	return &importer.Result{
		SheetID:       sheetID,
		RowsImported:  10,
		ColsImported:  3,
		CellsImported: 30,
	}, nil
}

func (m *mockImporterAPI) SupportedFormats() []importer.Format {
	return []importer.Format{importer.FormatCSV, importer.FormatTSV, importer.FormatXLSX, importer.FormatJSON}
}

func (m *mockImporterAPI) DetectFormat(filename string) importer.Format {
	if strings.HasSuffix(filename, ".csv") {
		return importer.FormatCSV
	}
	if strings.HasSuffix(filename, ".xlsx") {
		return importer.FormatXLSX
	}
	return importer.FormatCSV
}

func (m *mockImporterAPI) ValidateFile(ctx context.Context, reader io.Reader, format importer.Format) error {
	return nil
}

// --- Helper functions ---

func createMultipartFile(filename, content string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	return body, writer.FormDataContentType()
}

// --- Import/Export Tests ---

func TestExportWorkbook_Success(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-1
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Export: status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/csv; charset=utf-8", contentType)
	}

	// Check Content-Disposition
	disposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Errorf("Content-Disposition should contain 'attachment', got %s", disposition)
	}
}

func TestExportWorkbook_IDOR(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Handler with user-1's context (attacker)
	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-2/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestExportWorkbook_NotFound(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/nonexistent/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Not found: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestExportWorkbook_Unauthorized(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "" // No user
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestExportSheet_Success(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-1
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	// Create sheet in user-1's workbook
	mockSheets.sheets["sheet-1"] = &sheets.Sheet{
		ID:         "sheet-1",
		WorkbookID: "wb-1",
		Name:       "Sheet1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/sheets/{id}/export", handler.ExportSheet)

	req := httptest.NewRequest(http.MethodGet, "/sheets/sheet-1/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Export sheet: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestExportSheet_IDOR(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Create sheet in user-2's workbook
	mockSheets.sheets["sheet-2"] = &sheets.Sheet{
		ID:         "sheet-2",
		WorkbookID: "wb-2",
		Name:       "Secret Sheet",
	}

	// Handler with user-1's context (attacker)
	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/sheets/{id}/export", handler.ExportSheet)

	req := httptest.NewRequest(http.MethodGet, "/sheets/sheet-2/export?format=csv", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestImportToWorkbook_Success(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-1
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/workbooks/{id}/import", handler.ImportToWorkbook)

	// Create multipart form with file
	body, contentType := createMultipartFile("test.csv", "col1,col2\nval1,val2")

	req := httptest.NewRequest(http.MethodPost, "/workbooks/wb-1/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Import: status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Check response contains result
	if !strings.Contains(rec.Body.String(), "rowsImported") {
		t.Errorf("Response should contain import results, got: %s", rec.Body.String())
	}
}

func TestImportToWorkbook_IDOR(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Handler with user-1's context (attacker)
	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/workbooks/{id}/import", handler.ImportToWorkbook)

	body, contentType := createMultipartFile("malicious.csv", "col1,col2\nval1,val2")

	req := httptest.NewRequest(http.MethodPost, "/workbooks/wb-2/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestImportToWorkbook_NoFile(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-1
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/workbooks/{id}/import", handler.ImportToWorkbook)

	// Empty multipart form without file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/workbooks/wb-1/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("No file: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestImportToWorkbook_Unauthorized(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "" // No user
	})

	app := mizu.New()
	app.Post("/workbooks/{id}/import", handler.ImportToWorkbook)

	body, contentType := createMultipartFile("test.csv", "col1,col2\nval1,val2")

	req := httptest.NewRequest(http.MethodPost, "/workbooks/wb-1/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestImportToSheet_Success(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-1
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	// Create sheet in user-1's workbook
	mockSheets.sheets["sheet-1"] = &sheets.Sheet{
		ID:         "sheet-1",
		WorkbookID: "wb-1",
		Name:       "Sheet1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/sheets/{id}/import", handler.ImportToSheet)

	body, contentType := createMultipartFile("test.csv", "col1,col2\nval1,val2")

	req := httptest.NewRequest(http.MethodPost, "/sheets/sheet-1/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Import to sheet: status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestImportToSheet_IDOR(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	// Create workbook owned by user-2
	mockWorkbooks.workbooks["wb-2"] = &workbooks.Workbook{
		ID:      "wb-2",
		Name:    "User2 Workbook",
		OwnerID: "user-2",
	}

	// Create sheet in user-2's workbook
	mockSheets.sheets["sheet-2"] = &sheets.Sheet{
		ID:         "sheet-2",
		WorkbookID: "wb-2",
		Name:       "Secret Sheet",
	}

	// Handler with user-1's context (attacker)
	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/sheets/{id}/import", handler.ImportToSheet)

	body, contentType := createMultipartFile("malicious.csv", "col1,col2\nval1,val2")

	req := httptest.NewRequest(http.MethodPost, "/sheets/sheet-2/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("IDOR protection: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestSupportedFormats(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/formats", handler.SupportedFormats)

	req := httptest.NewRequest(http.MethodGet, "/formats", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Formats: status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "import") || !strings.Contains(body, "export") {
		t.Errorf("Response should contain import and export formats, got: %s", body)
	}
}

func TestExportWorkbook_DefaultFormat(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{
		exportWorkbookFunc: func(ctx context.Context, workbookID string, format export.Format, opts *export.Options) (*export.Result, error) {
			// Verify default format is XLSX
			if format != export.FormatXLSX {
				t.Errorf("Default format should be xlsx, got %s", format)
			}
			return &export.Result{
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				Filename:    "test.xlsx",
				Data:        strings.NewReader("xlsx content"),
				Size:        12,
			}, nil
		},
	}
	mockImporter := &mockImporterAPI{}

	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	// Request without format parameter
	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1/export", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Export with default format: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestExportWorkbook_WithOptions(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{
		exportWorkbookFunc: func(ctx context.Context, workbookID string, format export.Format, opts *export.Options) (*export.Result, error) {
			// Verify options are passed through
			if !opts.ExportFormatting {
				t.Error("ExportFormatting option should be true")
			}
			if !opts.ExportFormulas {
				t.Error("ExportFormulas option should be true")
			}
			if !opts.IncludeHeaders {
				t.Error("IncludeHeaders option should be true")
			}
			return &export.Result{
				ContentType: "text/csv; charset=utf-8",
				Filename:    "test.csv",
				Data:        strings.NewReader("col1,col2\nval1,val2"),
				Size:        20,
			}, nil
		},
	}
	mockImporter := &mockImporterAPI{}

	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Get("/workbooks/{id}/export", handler.ExportWorkbook)

	req := httptest.NewRequest(http.MethodGet, "/workbooks/wb-1/export?format=csv&formatting=true&formulas=true&headers=true", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Export with options: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestImportToWorkbook_WithOptions(t *testing.T) {
	mockWorkbooks := newMockWorkbooksAPI()
	mockSheets := newMockSheetsAPI()
	mockExporter := &mockExporterAPI{}
	mockImporter := &mockImporterAPI{
		importToWorkbookFunc: func(ctx context.Context, workbookID string, reader io.Reader, filename string, format importer.Format, opts *importer.Options) (*importer.Result, error) {
			// Verify options are passed through
			if !opts.HasHeaders {
				t.Error("HasHeaders option should be true")
			}
			if !opts.SkipEmptyRows {
				t.Error("SkipEmptyRows option should be true")
			}
			if !opts.AutoDetectTypes {
				t.Error("AutoDetectTypes option should be true by default")
			}
			return &importer.Result{
				SheetID:       "new-sheet-1",
				RowsImported:  10,
				ColsImported:  3,
				CellsImported: 30,
			}, nil
		},
	}

	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}

	handler := NewImportExport(mockExporter, mockImporter, mockWorkbooks, mockSheets, func(c *mizu.Ctx) string {
		return "user-1"
	})

	app := mizu.New()
	app.Post("/workbooks/{id}/import", handler.ImportToWorkbook)

	// Create multipart form with file and options
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "test.csv")
	part.Write([]byte("col1,col2\nval1,val2"))

	writer.WriteField("hasHeaders", "true")
	writer.WriteField("skipEmptyRows", "true")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/workbooks/wb-1/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Import with options: status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
