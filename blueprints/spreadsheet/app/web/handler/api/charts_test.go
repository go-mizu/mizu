package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// mockChartsAPI implements charts.API for testing.
type mockChartsAPI struct {
	charts map[string]*charts.Chart
}

func newMockChartsAPI() *mockChartsAPI {
	return &mockChartsAPI{charts: make(map[string]*charts.Chart)}
}

func (m *mockChartsAPI) Create(ctx context.Context, in *charts.CreateIn) (*charts.Chart, error) {
	if !isValidChartType(in.ChartType) {
		return nil, charts.ErrInvalidType
	}
	if len(in.DataRanges) == 0 {
		return nil, charts.ErrEmptyDataRange
	}

	now := time.Now()
	chart := &charts.Chart{
		ID:         "chart-" + in.Name,
		SheetID:    in.SheetID,
		Name:       in.Name,
		ChartType:  in.ChartType,
		Position:   in.Position,
		Size:       in.Size,
		DataRanges: in.DataRanges,
		Title:      in.Title,
		Legend:     in.Legend,
		Options:    in.Options,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.charts[chart.ID] = chart
	return chart, nil
}

func (m *mockChartsAPI) GetByID(ctx context.Context, id string) (*charts.Chart, error) {
	if chart, ok := m.charts[id]; ok {
		return chart, nil
	}
	return nil, charts.ErrNotFound
}

func (m *mockChartsAPI) ListBySheet(ctx context.Context, sheetID string) ([]*charts.Chart, error) {
	result := make([]*charts.Chart, 0)
	for _, chart := range m.charts {
		if chart.SheetID == sheetID {
			result = append(result, chart)
		}
	}
	return result, nil
}

func (m *mockChartsAPI) Update(ctx context.Context, id string, in *charts.UpdateIn) (*charts.Chart, error) {
	chart, ok := m.charts[id]
	if !ok {
		return nil, charts.ErrNotFound
	}
	if in.ChartType != "" && !isValidChartType(in.ChartType) {
		return nil, charts.ErrInvalidType
	}
	if in.Name != "" {
		chart.Name = in.Name
	}
	if in.ChartType != "" {
		chart.ChartType = in.ChartType
	}
	if in.Position != nil {
		chart.Position = *in.Position
	}
	if in.Size != nil {
		chart.Size = *in.Size
	}
	chart.UpdatedAt = time.Now()
	return chart, nil
}

func (m *mockChartsAPI) Delete(ctx context.Context, id string) error {
	if _, ok := m.charts[id]; !ok {
		return charts.ErrNotFound
	}
	delete(m.charts, id)
	return nil
}

func (m *mockChartsAPI) Duplicate(ctx context.Context, id string) (*charts.Chart, error) {
	original, ok := m.charts[id]
	if !ok {
		return nil, charts.ErrNotFound
	}
	now := time.Now()
	dup := &charts.Chart{
		ID:         original.ID + "-copy",
		SheetID:    original.SheetID,
		Name:       original.Name + " (Copy)",
		ChartType:  original.ChartType,
		Position:   charts.Position{Row: original.Position.Row + 2, Col: original.Position.Col + 2},
		Size:       original.Size,
		DataRanges: original.DataRanges,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.charts[dup.ID] = dup
	return dup, nil
}

func (m *mockChartsAPI) GetData(ctx context.Context, id string) (*charts.ChartData, error) {
	if _, ok := m.charts[id]; !ok {
		return nil, charts.ErrNotFound
	}
	return &charts.ChartData{
		Labels: []string{"A", "B", "C"},
		Datasets: []charts.Dataset{
			{Label: "Series 1", Data: []float64{10, 20, 30}},
		},
	}, nil
}

func isValidChartType(t charts.ChartType) bool {
	switch t {
	case charts.ChartTypeLine, charts.ChartTypeBar, charts.ChartTypeColumn, charts.ChartTypePie,
		charts.ChartTypeDoughnut, charts.ChartTypeArea, charts.ChartTypeScatter, charts.ChartTypeCombo,
		charts.ChartTypeStackedBar, charts.ChartTypeStackedColumn, charts.ChartTypeStackedArea,
		charts.ChartTypeRadar, charts.ChartTypeBubble, charts.ChartTypeWaterfall,
		charts.ChartTypeHistogram, charts.ChartTypeTreemap, charts.ChartTypeGauge,
		charts.ChartTypeCandlestick:
		return true
	}
	return false
}

// chartTestSheetsAPI implements sheets.API for chart testing.
type chartTestSheetsAPI struct {
	sheets map[string]*sheets.Sheet
}

func newChartTestSheetsAPI() *chartTestSheetsAPI {
	return &chartTestSheetsAPI{sheets: make(map[string]*sheets.Sheet)}
}

func (m *chartTestSheetsAPI) Create(ctx context.Context, in *sheets.CreateIn) (*sheets.Sheet, error) {
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

func (m *chartTestSheetsAPI) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	if sheet, ok := m.sheets[id]; ok {
		return sheet, nil
	}
	return nil, sheets.ErrNotFound
}

func (m *chartTestSheetsAPI) List(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	var result []*sheets.Sheet
	for _, sheet := range m.sheets {
		if sheet.WorkbookID == workbookID {
			result = append(result, sheet)
		}
	}
	return result, nil
}

func (m *chartTestSheetsAPI) Update(ctx context.Context, id string, in *sheets.UpdateIn) (*sheets.Sheet, error) {
	return nil, nil
}

func (m *chartTestSheetsAPI) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *chartTestSheetsAPI) Copy(ctx context.Context, id string, newName string) (*sheets.Sheet, error) {
	return nil, nil
}

func (m *chartTestSheetsAPI) SetRowHeight(ctx context.Context, sheetID string, row int, height int) error {
	return nil
}

func (m *chartTestSheetsAPI) SetColWidth(ctx context.Context, sheetID string, col int, width int) error {
	return nil
}

func (m *chartTestSheetsAPI) HideRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *chartTestSheetsAPI) HideCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

func (m *chartTestSheetsAPI) ShowRow(ctx context.Context, sheetID string, row int) error {
	return nil
}

func (m *chartTestSheetsAPI) ShowCol(ctx context.Context, sheetID string, col int) error {
	return nil
}

// chartTestWorkbooksAPI implements workbooks.API for chart testing.
type chartTestWorkbooksAPI struct {
	workbooks map[string]*workbooks.Workbook
}

func newChartTestWorkbooksAPI() *chartTestWorkbooksAPI {
	return &chartTestWorkbooksAPI{workbooks: make(map[string]*workbooks.Workbook)}
}

func (m *chartTestWorkbooksAPI) Create(ctx context.Context, in *workbooks.CreateIn) (*workbooks.Workbook, error) {
	return nil, nil
}

func (m *chartTestWorkbooksAPI) GetByID(ctx context.Context, id string) (*workbooks.Workbook, error) {
	if wb, ok := m.workbooks[id]; ok {
		return wb, nil
	}
	return nil, workbooks.ErrNotFound
}

func (m *chartTestWorkbooksAPI) List(ctx context.Context, ownerID string) ([]*workbooks.Workbook, error) {
	return nil, nil
}

func (m *chartTestWorkbooksAPI) Update(ctx context.Context, id string, in *workbooks.UpdateIn) (*workbooks.Workbook, error) {
	return nil, nil
}

func (m *chartTestWorkbooksAPI) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *chartTestWorkbooksAPI) Copy(ctx context.Context, id string, newName string, userID string) (*workbooks.Workbook, error) {
	return nil, nil
}

func mockGetUserID(c *mizu.Ctx) string {
	return "user-1"
}

func setupChartsHandler() (*Charts, *mockChartsAPI, *chartTestSheetsAPI, *chartTestWorkbooksAPI) {
	mockCharts := newMockChartsAPI()
	mockSheets := newChartTestSheetsAPI()
	mockWorkbooks := newChartTestWorkbooksAPI()

	// Set up default sheet and workbook that charts can reference
	mockWorkbooks.workbooks["wb-1"] = &workbooks.Workbook{
		ID:      "wb-1",
		Name:    "Test Workbook",
		OwnerID: "user-1",
	}
	mockSheets.sheets["sheet-1"] = &sheets.Sheet{
		ID:         "sheet-1",
		WorkbookID: "wb-1",
		Name:       "Sheet 1",
	}

	handler := NewCharts(mockCharts, mockSheets, mockWorkbooks, mockGetUserID)
	return handler, mockCharts, mockSheets, mockWorkbooks
}

func TestCharts_Create(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts", handler.Create)

	body := `{
		"sheetId": "sheet-1",
		"name": "Test Chart",
		"chartType": "column",
		"position": {"row": 5, "col": 5},
		"size": {"width": 600, "height": 400},
		"dataRanges": [{"startRow": 0, "startCol": 0, "endRow": 10, "endCol": 3, "hasHeader": true}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/charts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var chart charts.Chart
	if err := json.NewDecoder(rec.Body).Decode(&chart); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if chart.Name != "Test Chart" {
		t.Errorf("Name = %v, want Test Chart", chart.Name)
	}
	if chart.ChartType != charts.ChartTypeColumn {
		t.Errorf("ChartType = %v, want column", chart.ChartType)
	}
}

func TestCharts_Create_InvalidType(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts", handler.Create)

	body := `{
		"sheetId": "sheet-1",
		"name": "Invalid",
		"chartType": "invalid_type",
		"position": {"row": 0, "col": 0},
		"size": {"width": 500, "height": 300},
		"dataRanges": [{"startRow": 0, "startCol": 0, "endRow": 5, "endCol": 2, "hasHeader": true}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/charts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCharts_Create_EmptyDataRange(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts", handler.Create)

	body := `{
		"sheetId": "sheet-1",
		"name": "No Data",
		"chartType": "line",
		"position": {"row": 0, "col": 0},
		"size": {"width": 500, "height": 300},
		"dataRanges": []
	}`

	req := httptest.NewRequest(http.MethodPost, "/charts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCharts_Create_InvalidJSON(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts", handler.Create)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/charts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCharts_Get(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Get("/charts/{id}", handler.Get)

	// Create a chart first
	mock.charts["chart-1"] = &charts.Chart{
		ID:        "chart-1",
		SheetID:   "sheet-1",
		Name:      "Test Chart",
		ChartType: charts.ChartTypePie,
	}

	req := httptest.NewRequest(http.MethodGet, "/charts/chart-1", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var chart charts.Chart
	if err := json.NewDecoder(rec.Body).Decode(&chart); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if chart.ID != "chart-1" {
		t.Errorf("ID = %v, want chart-1", chart.ID)
	}
}

func TestCharts_Get_NotFound(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Get("/charts/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/charts/nonexistent", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCharts_Update(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Patch("/charts/{id}", handler.Update)

	// Create a chart first
	mock.charts["chart-1"] = &charts.Chart{
		ID:        "chart-1",
		SheetID:   "sheet-1",
		Name:      "Original",
		ChartType: charts.ChartTypeLine,
		Position:  charts.Position{Row: 0, Col: 0},
		Size:      charts.Size{Width: 500, Height: 300},
	}

	body := `{
		"name": "Updated",
		"chartType": "bar"
	}`

	req := httptest.NewRequest(http.MethodPatch, "/charts/chart-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		bodyBytes, _ := io.ReadAll(rec.Body)
		t.Errorf("status = %d, want %d. Body: %s", rec.Code, http.StatusOK, string(bodyBytes))
	}

	var chart charts.Chart
	if err := json.NewDecoder(rec.Body).Decode(&chart); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if chart.Name != "Updated" {
		t.Errorf("Name = %v, want Updated", chart.Name)
	}
	if chart.ChartType != charts.ChartTypeBar {
		t.Errorf("ChartType = %v, want bar", chart.ChartType)
	}
}

func TestCharts_Update_NotFound(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Patch("/charts/{id}", handler.Update)

	body := `{"name": "New Name"}`

	req := httptest.NewRequest(http.MethodPatch, "/charts/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCharts_Update_InvalidType(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Patch("/charts/{id}", handler.Update)

	mock.charts["chart-1"] = &charts.Chart{
		ID:        "chart-1",
		SheetID:   "sheet-1", // Required for authorization check
		ChartType: charts.ChartTypeLine,
	}

	body := `{"chartType": "invalid"}`

	req := httptest.NewRequest(http.MethodPatch, "/charts/chart-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCharts_Delete(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Delete("/charts/{id}", handler.Delete)

	mock.charts["chart-1"] = &charts.Chart{ID: "chart-1", SheetID: "sheet-1"}

	req := httptest.NewRequest(http.MethodDelete, "/charts/chart-1", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if _, exists := mock.charts["chart-1"]; exists {
		t.Error("Chart should be deleted")
	}
}

func TestCharts_Delete_NotFound(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Delete("/charts/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/charts/nonexistent", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCharts_ListBySheet(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Get("/sheets/{sheetId}/charts", handler.ListBySheet)

	// Add charts
	mock.charts["chart-1"] = &charts.Chart{ID: "chart-1", SheetID: "sheet-1"}
	mock.charts["chart-2"] = &charts.Chart{ID: "chart-2", SheetID: "sheet-1"}
	mock.charts["chart-3"] = &charts.Chart{ID: "chart-3", SheetID: "sheet-2"}

	req := httptest.NewRequest(http.MethodGet, "/sheets/sheet-1/charts", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var chartList []*charts.Chart
	if err := json.NewDecoder(rec.Body).Decode(&chartList); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(chartList) != 2 {
		t.Errorf("length = %d, want 2", len(chartList))
	}
}

func TestCharts_Duplicate(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts/{id}/duplicate", handler.Duplicate)

	mock.charts["chart-1"] = &charts.Chart{
		ID:        "chart-1",
		SheetID:   "sheet-1",
		Name:      "Original",
		ChartType: charts.ChartTypeColumn,
		Position:  charts.Position{Row: 5, Col: 5},
		Size:      charts.Size{Width: 600, Height: 400},
	}

	req := httptest.NewRequest(http.MethodPost, "/charts/chart-1/duplicate", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var dup charts.Chart
	if err := json.NewDecoder(rec.Body).Decode(&dup); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dup.ID == "chart-1" {
		t.Error("Duplicate should have different ID")
	}
	if dup.Name != "Original (Copy)" {
		t.Errorf("Name = %v, want Original (Copy)", dup.Name)
	}
}

func TestCharts_Duplicate_NotFound(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Post("/charts/{id}/duplicate", handler.Duplicate)

	req := httptest.NewRequest(http.MethodPost, "/charts/nonexistent/duplicate", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCharts_GetData(t *testing.T) {
	handler, mock, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Get("/charts/{id}/data", handler.GetData)

	mock.charts["chart-1"] = &charts.Chart{
		ID:        "chart-1",
		SheetID:   "sheet-1",
		ChartType: charts.ChartTypeLine,
	}

	req := httptest.NewRequest(http.MethodGet, "/charts/chart-1/data", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data charts.ChartData
	if err := json.NewDecoder(rec.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(data.Labels) != 3 {
		t.Errorf("Labels length = %d, want 3", len(data.Labels))
	}
	if len(data.Datasets) != 1 {
		t.Errorf("Datasets length = %d, want 1", len(data.Datasets))
	}
}

func TestCharts_GetData_NotFound(t *testing.T) {
	handler, _, _, _ := setupChartsHandler()
	app := mizu.New()
	app.Get("/charts/{id}/data", handler.GetData)

	req := httptest.NewRequest(http.MethodGet, "/charts/nonexistent/data", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
