package charts

import (
	"context"
	"testing"
	"time"
)

// mockStore is a mock implementation of Store for testing.
type mockStore struct {
	charts map[string]*Chart
}

func newMockStore() *mockStore {
	return &mockStore{charts: make(map[string]*Chart)}
}

func (m *mockStore) Create(ctx context.Context, chart *Chart) error {
	m.charts[chart.ID] = chart
	return nil
}

func (m *mockStore) GetByID(ctx context.Context, id string) (*Chart, error) {
	if chart, ok := m.charts[id]; ok {
		return chart, nil
	}
	return nil, ErrNotFound
}

func (m *mockStore) ListBySheet(ctx context.Context, sheetID string) ([]*Chart, error) {
	result := make([]*Chart, 0)
	for _, chart := range m.charts {
		if chart.SheetID == sheetID {
			result = append(result, chart)
		}
	}
	return result, nil
}

func (m *mockStore) Update(ctx context.Context, chart *Chart) error {
	if _, ok := m.charts[chart.ID]; !ok {
		return ErrNotFound
	}
	m.charts[chart.ID] = chart
	return nil
}

func (m *mockStore) Delete(ctx context.Context, id string) error {
	delete(m.charts, id)
	return nil
}

// mockCellProvider is a mock implementation of CellDataProvider.
type mockCellProvider struct {
	data map[string][][]interface{}
}

func newMockCellProvider() *mockCellProvider {
	return &mockCellProvider{data: make(map[string][][]interface{})}
}

func (m *mockCellProvider) SetData(sheetID string, data [][]interface{}) {
	m.data[sheetID] = data
}

func (m *mockCellProvider) GetCellValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error) {
	if data, ok := m.data[sheetID]; ok {
		// Extract the requested range
		numRows := endRow - startRow + 1
		numCols := endCol - startCol + 1
		result := make([][]interface{}, numRows)
		for i := 0; i < numRows; i++ {
			result[i] = make([]interface{}, numCols)
			srcRow := startRow + i
			if srcRow < len(data) {
				for j := 0; j < numCols; j++ {
					srcCol := startCol + j
					if srcCol < len(data[srcRow]) {
						result[i][j] = data[srcRow][srcCol]
					}
				}
			}
		}
		return result, nil
	}
	return nil, nil
}

func TestService_Create(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Test Chart",
		ChartType: ChartTypeColumn,
		Position:  Position{Row: 5, Col: 5},
		Size:      Size{Width: 600, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 10, EndCol: 3, HasHeader: true,
		}},
		Title:  &ChartTitle{Text: "Sales Data"},
		Legend: &LegendConfig{Enabled: true, Position: "bottom"},
	}

	chart, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if chart.ID == "" {
		t.Error("Chart ID should be generated")
	}
	if chart.Name != "Test Chart" {
		t.Errorf("Name = %v, want Test Chart", chart.Name)
	}
	if chart.ChartType != ChartTypeColumn {
		t.Errorf("ChartType = %v, want column", chart.ChartType)
	}
	if chart.Size.Width != 600 {
		t.Errorf("Size.Width = %v, want 600", chart.Size.Width)
	}
}

func TestService_Create_InvalidType(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Invalid Chart",
		ChartType: "invalid_type",
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
	}

	_, err := svc.Create(ctx, in)
	if err != ErrInvalidType {
		t.Errorf("Create() error = %v, want %v", err, ErrInvalidType)
	}
}

func TestService_Create_EmptyDataRange(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	in := &CreateIn{
		SheetID:    "sheet-1",
		Name:       "No Data Chart",
		ChartType:  ChartTypeLine,
		Position:   Position{Row: 0, Col: 0},
		Size:       Size{Width: 500, Height: 300},
		DataRanges: []DataRange{}, // Empty
	}

	_, err := svc.Create(ctx, in)
	if err != ErrEmptyDataRange {
		t.Errorf("Create() error = %v, want %v", err, ErrEmptyDataRange)
	}
}

func TestService_Create_AppliesDefaults(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Defaults Chart",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		// Size, Legend, and Options not specified
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
	}

	chart, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Default size
	if chart.Size.Width != 600 {
		t.Errorf("Size.Width = %v, want 600 (default)", chart.Size.Width)
	}
	if chart.Size.Height != 400 {
		t.Errorf("Size.Height = %v, want 400 (default)", chart.Size.Height)
	}

	// Default legend
	if chart.Legend == nil {
		t.Fatal("Legend should be set to default")
	}
	if !chart.Legend.Enabled {
		t.Error("Legend.Enabled should be true by default")
	}

	// Default options
	if chart.Options == nil {
		t.Fatal("Options should be set to default")
	}
	if !chart.Options.Animated {
		t.Error("Options.Animated should be true by default")
	}
	if !chart.Options.TooltipEnabled {
		t.Error("Options.TooltipEnabled should be true by default")
	}
}

func TestService_GetByID(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create a chart first
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Get Test Chart",
		ChartType: ChartTypePie,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 400, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 1, HasHeader: true,
		}},
	}

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %v, want %v", got.ID, created.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestService_ListBySheet(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create multiple charts
	sheetID := "sheet-1"
	for i := 0; i < 3; i++ {
		in := &CreateIn{
			SheetID:   sheetID,
			Name:      "Chart",
			ChartType: ChartTypeColumn,
			Position:  Position{Row: i * 10, Col: 0},
			Size:      Size{Width: 500, Height: 300},
			DataRanges: []DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
			}},
		}
		if _, err := svc.Create(ctx, in); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	list, err := svc.ListBySheet(ctx, sheetID)
	if err != nil {
		t.Fatalf("ListBySheet() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListBySheet() returned %d charts, want 3", len(list))
	}
}

func TestService_Update(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create a chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Original",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
	}

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	updateIn := &UpdateIn{
		Name:      "Updated",
		ChartType: ChartTypeBar,
		Position:  &Position{Row: 10, Col: 10},
		Size:      &Size{Width: 800, Height: 500},
		Title:     &ChartTitle{Text: "New Title"},
	}

	updated, err := svc.Update(ctx, created.ID, updateIn)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("Name = %v, want Updated", updated.Name)
	}
	if updated.ChartType != ChartTypeBar {
		t.Errorf("ChartType = %v, want bar", updated.ChartType)
	}
	if updated.Position.Row != 10 {
		t.Errorf("Position.Row = %v, want 10", updated.Position.Row)
	}
	if updated.Size.Width != 800 {
		t.Errorf("Size.Width = %v, want 800", updated.Size.Width)
	}
	if updated.Title == nil || updated.Title.Text != "New Title" {
		t.Errorf("Title.Text = %v, want New Title", updated.Title)
	}
}

func TestService_Update_InvalidType(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create a chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Test",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
	}

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Try to update with invalid type
	updateIn := &UpdateIn{
		ChartType: "invalid_type",
	}

	_, err = svc.Update(ctx, created.ID, updateIn)
	if err != ErrInvalidType {
		t.Errorf("Update() error = %v, want %v", err, ErrInvalidType)
	}
}

func TestService_Delete(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create a chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "To Delete",
		ChartType: ChartTypePie,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 400, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 1, HasHeader: true,
		}},
	}

	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = svc.GetByID(ctx, created.ID)
	if err != ErrNotFound {
		t.Errorf("GetByID() after Delete() error = %v, want %v", err, ErrNotFound)
	}
}

func TestService_Duplicate(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Create a chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Original Chart",
		ChartType: ChartTypeColumn,
		Position:  Position{Row: 5, Col: 5},
		Size:      Size{Width: 600, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 10, EndCol: 3, HasHeader: true,
		}},
		Title: &ChartTitle{Text: "Original Title"},
	}

	original, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Duplicate
	duplicate, err := svc.Duplicate(ctx, original.ID)
	if err != nil {
		t.Fatalf("Duplicate() error = %v", err)
	}

	if duplicate.ID == original.ID {
		t.Error("Duplicate should have different ID")
	}
	if duplicate.Name != "Original Chart (Copy)" {
		t.Errorf("Name = %v, want Original Chart (Copy)", duplicate.Name)
	}
	if duplicate.Position.Row != original.Position.Row+2 {
		t.Errorf("Position.Row = %v, want %v", duplicate.Position.Row, original.Position.Row+2)
	}
	if duplicate.Title.Text != "Original Title" {
		t.Errorf("Title.Text = %v, want Original Title", duplicate.Title.Text)
	}
}

func TestService_GetData(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Set up mock data
	cellProvider.SetData("sheet-1", [][]interface{}{
		{"Month", "Sales", "Expenses"},
		{"Jan", 12000, 8000},
		{"Feb", 15000, 9000},
		{"Mar", 18000, 10000},
	})

	// Create a chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Sales Chart",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 600, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 3, EndCol: 2, HasHeader: true,
		}},
	}

	chart, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get data
	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	if len(data.Labels) != 3 {
		t.Errorf("Labels length = %d, want 3", len(data.Labels))
	}
	if data.Labels[0] != "Jan" {
		t.Errorf("Labels[0] = %v, want Jan", data.Labels[0])
	}

	if len(data.Datasets) != 2 {
		t.Errorf("Datasets length = %d, want 2", len(data.Datasets))
	}
	if data.Datasets[0].Label != "Sales" {
		t.Errorf("Datasets[0].Label = %v, want Sales", data.Datasets[0].Label)
	}
	if len(data.Datasets[0].Data) != 3 {
		t.Errorf("Datasets[0].Data length = %d, want 3", len(data.Datasets[0].Data))
	}
	if data.Datasets[0].Data[0] != 12000 {
		t.Errorf("Datasets[0].Data[0] = %v, want 12000", data.Datasets[0].Data[0])
	}
}

func TestIsValidChartType(t *testing.T) {
	validTypes := []ChartType{
		ChartTypeLine,
		ChartTypeBar,
		ChartTypeColumn,
		ChartTypePie,
		ChartTypeDoughnut,
		ChartTypeArea,
		ChartTypeScatter,
		ChartTypeCombo,
		ChartTypeStackedBar,
		ChartTypeStackedColumn,
		ChartTypeStackedArea,
		ChartTypeRadar,
		ChartTypeBubble,
		ChartTypeWaterfall,
		ChartTypeHistogram,
		ChartTypeTreemap,
		ChartTypeGauge,
		ChartTypeCandlestick,
	}

	for _, ct := range validTypes {
		if !isValidChartType(ct) {
			t.Errorf("isValidChartType(%v) = false, want true", ct)
		}
	}

	invalidTypes := []ChartType{"invalid", "unknown", ""}
	for _, ct := range invalidTypes {
		if isValidChartType(ct) {
			t.Errorf("isValidChartType(%v) = true, want false", ct)
		}
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", "hello"},
		{42.5, "42.5"},
		{100, "100"},
		{int64(200), "200"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
		{struct{}{}, ""},
	}

	for _, tt := range tests {
		result := toString(tt.input)
		if result != tt.expected {
			t.Errorf("toString(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
	}{
		{42.5, 42.5},
		{100, 100.0},
		{int64(200), 200.0},
		{"123.45", 123.45},
		{"invalid", 0},
		{nil, 0},
		{struct{}{}, 0},
	}

	for _, tt := range tests {
		result := toFloat64(tt.input)
		if result != tt.expected {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseChartData_PieChart(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Set up mock data for pie chart
	cellProvider.SetData("sheet-1", [][]interface{}{
		{"Category", "Value"},
		{"A", 30},
		{"B", 50},
		{"C", 20},
	})

	// Create a pie chart
	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Pie Chart",
		ChartType: ChartTypePie,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 400, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 3, EndCol: 1, HasHeader: true,
		}},
	}

	chart, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	if len(data.Labels) != 3 {
		t.Errorf("Labels length = %d, want 3", len(data.Labels))
	}

	// Pie chart should have colors as an array
	if len(data.Datasets) != 1 {
		t.Errorf("Datasets length = %d, want 1", len(data.Datasets))
	}

	colors, ok := data.Datasets[0].BackgroundColor.([]string)
	if !ok {
		t.Error("BackgroundColor should be []string for pie chart")
	}
	if len(colors) != 3 {
		t.Errorf("BackgroundColor length = %d, want 3", len(colors))
	}
}

func TestService_Timestamps(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	before := time.Now()

	in := &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Timestamp Test",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
	}

	chart, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	after := time.Now()

	if chart.CreatedAt.Before(before) || chart.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after")
	}
	if chart.UpdatedAt.Before(before) || chart.UpdatedAt.After(after) {
		t.Error("UpdatedAt should be between before and after")
	}
}

func TestService_Create_AllChartTypes(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	chartTypes := []ChartType{
		ChartTypeLine, ChartTypeBar, ChartTypeColumn, ChartTypePie,
		ChartTypeDoughnut, ChartTypeArea, ChartTypeScatter, ChartTypeCombo,
		ChartTypeStackedBar, ChartTypeStackedColumn, ChartTypeStackedArea,
		ChartTypeRadar, ChartTypeBubble, ChartTypeWaterfall,
		ChartTypeHistogram, ChartTypeTreemap, ChartTypeGauge, ChartTypeCandlestick,
	}

	for _, ct := range chartTypes {
		t.Run(string(ct), func(t *testing.T) {
			in := &CreateIn{
				SheetID:   "sheet-1",
				Name:      string(ct) + " Chart",
				ChartType: ct,
				Position:  Position{Row: 0, Col: 0},
				Size:      Size{Width: 500, Height: 300},
				DataRanges: []DataRange{{
					StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2,
					HasHeader: true,
				}},
			}

			chart, err := svc.Create(ctx, in)
			if err != nil {
				t.Fatalf("Create() error = %v for type %s", err, ct)
			}
			if chart.ChartType != ct {
				t.Errorf("ChartType = %v, want %v", chart.ChartType, ct)
			}
		})
	}
}

func TestService_GetData_AreaChart(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	cellProvider.SetData("sheet-1", [][]interface{}{
		{"Month", "Sales"},
		{"Jan", 100},
		{"Feb", 150},
	})

	chart, err := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Area Chart",
		ChartType: ChartTypeArea,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 2, EndCol: 1,
			HasHeader: true,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	// Area charts should have fill enabled
	if !data.Datasets[0].Fill {
		t.Error("Area chart datasets should have Fill = true")
	}
}

func TestService_GetData_WithSeriesConfig(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	cellProvider.SetData("sheet-1", [][]interface{}{
		{"Month", "Sales", "Expenses"},
		{"Jan", 100, 80},
		{"Feb", 150, 90},
	})

	chart, err := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Series Test",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 2, EndCol: 2,
			HasHeader: true,
		}},
		Series: []SeriesConfig{
			{Name: "Revenue", Color: "#FF0000", BorderWidth: 3, Tension: 0.4},
			{Name: "Costs", Color: "#00FF00"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	if data.Datasets[0].Label != "Revenue" {
		t.Errorf("Datasets[0].Label = %v, want Revenue", data.Datasets[0].Label)
	}
	if data.Datasets[0].BorderColor != "#FF0000" {
		t.Errorf("Datasets[0].BorderColor = %v, want #FF0000", data.Datasets[0].BorderColor)
	}
	if data.Datasets[0].BorderWidth != 3 {
		t.Errorf("Datasets[0].BorderWidth = %v, want 3", data.Datasets[0].BorderWidth)
	}
	if data.Datasets[0].Tension != 0.4 {
		t.Errorf("Datasets[0].Tension = %v, want 0.4", data.Datasets[0].Tension)
	}
}

func TestService_GetData_NoHeader(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// Data without header row
	cellProvider.SetData("sheet-1", [][]interface{}{
		{"Jan", 12000, 8000},
		{"Feb", 15000, 9000},
		{"Mar", 18000, 10000},
	})

	chart, err := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "No Header",
		ChartType: ChartTypeColumn,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 2, EndCol: 2,
			HasHeader: false, // No header
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	if len(data.Labels) != 3 {
		t.Errorf("Labels count = %d, want 3", len(data.Labels))
	}
	// First column becomes labels
	if data.Labels[0] != "Jan" {
		t.Errorf("Labels[0] = %v, want Jan", data.Labels[0])
	}
	// Generated series names
	if data.Datasets[0].Label != "Series 1" {
		t.Errorf("Datasets[0].Label = %v, want Series 1", data.Datasets[0].Label)
	}
}

func TestService_GetData_EmptyData(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	// No data set for sheet-1
	chart, err := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Empty Data",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 500, Height: 300},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2,
			HasHeader: true,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := svc.GetData(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetData() error = %v", err)
	}

	if len(data.Labels) != 0 {
		t.Errorf("Labels should be empty for empty data, got %d", len(data.Labels))
	}
	if len(data.Datasets) != 0 {
		t.Errorf("Datasets should be empty for empty data, got %d", len(data.Datasets))
	}
}

func TestService_GetData_NotFound(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	_, err := svc.GetData(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("GetData() error = %v, want %v", err, ErrNotFound)
	}
}

func TestService_Duplicate_NotFound(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	_, err := svc.Duplicate(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Duplicate() error = %v, want %v", err, ErrNotFound)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	_, err := svc.Update(ctx, "nonexistent", &UpdateIn{Name: "New Name"})
	if err != ErrNotFound {
		t.Errorf("Update() error = %v, want %v", err, ErrNotFound)
	}
}

func TestService_Update_PartialUpdate(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	chart, _ := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Original",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 5, Col: 5},
		Size:      Size{Width: 600, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2,
			HasHeader: true,
		}},
		Title: &ChartTitle{Text: "Original Title"},
	})

	// Update only the name
	updated, err := svc.Update(ctx, chart.ID, &UpdateIn{
		Name: "Updated Name",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", updated.Name)
	}
	// Original values should remain unchanged
	if updated.ChartType != ChartTypeLine {
		t.Errorf("ChartType = %v, want line (unchanged)", updated.ChartType)
	}
	if updated.Position.Row != 5 {
		t.Errorf("Position.Row = %v, want 5 (unchanged)", updated.Position.Row)
	}
	if updated.Title.Text != "Original Title" {
		t.Errorf("Title.Text = %v, want Original Title (unchanged)", updated.Title.Text)
	}
}

func TestService_Duplicate_DeepCopy(t *testing.T) {
	store := newMockStore()
	cellProvider := newMockCellProvider()
	svc := NewService(store, cellProvider)
	ctx := context.Background()

	original, _ := svc.Create(ctx, &CreateIn{
		SheetID:   "sheet-1",
		Name:      "Original",
		ChartType: ChartTypeLine,
		Position:  Position{Row: 0, Col: 0},
		Size:      Size{Width: 600, Height: 400},
		DataRanges: []DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 10, EndCol: 3,
			HasHeader: true,
		}},
		Title:    &ChartTitle{Text: "My Title", Bold: true},
		Subtitle: &ChartTitle{Text: "My Subtitle"},
		Legend:   &LegendConfig{Enabled: true, Position: "right"},
		Axes: &AxesConfig{
			XAxis: &AxisConfig{GridLines: false},
			YAxis: &AxisConfig{GridLines: true},
		},
		Series: []SeriesConfig{
			{Name: "Series A", Color: "#FF0000"},
		},
		Options: &ChartOptions{Animated: true, TooltipEnabled: true},
	})

	dup, err := svc.Duplicate(ctx, original.ID)
	if err != nil {
		t.Fatalf("Duplicate() error = %v", err)
	}

	// Verify deep copies - modifications to duplicate shouldn't affect original
	if dup.Title == nil || dup.Title.Text != "My Title" {
		t.Error("Title not copied")
	}
	if dup.Subtitle == nil || dup.Subtitle.Text != "My Subtitle" {
		t.Error("Subtitle not copied")
	}
	if dup.Legend == nil || dup.Legend.Position != "right" {
		t.Error("Legend not copied")
	}
	if dup.Axes == nil || dup.Axes.YAxis == nil || !dup.Axes.YAxis.GridLines {
		t.Error("Axes not copied")
	}
	if len(dup.Series) != 1 || dup.Series[0].Color != "#FF0000" {
		t.Error("Series not copied")
	}
	if dup.Options == nil || !dup.Options.TooltipEnabled {
		t.Error("Options not copied")
	}
}
