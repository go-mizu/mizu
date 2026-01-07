package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
)

func TestChartsStore_Create(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "Test Chart",
		ChartType: charts.ChartTypeColumn,
		Position:  charts.Position{Row: 5, Col: 5, OffsetX: 10, OffsetY: 20},
		Size:      charts.Size{Width: 600, Height: 400},
		DataRanges: []charts.DataRange{{
			StartRow:  0,
			StartCol:  0,
			EndRow:    10,
			EndCol:    3,
			HasHeader: true,
		}},
		Title:     &charts.ChartTitle{Text: "Monthly Sales", FontSize: 16, Bold: true},
		Legend:    &charts.LegendConfig{Enabled: true, Position: "bottom"},
		Options:   &charts.ChartOptions{Animated: true, TooltipEnabled: true},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.Create(ctx, chart)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != chart.ID {
		t.Errorf("ID = %v, want %v", got.ID, chart.ID)
	}
	if got.Name != "Test Chart" {
		t.Errorf("Name = %v, want Test Chart", got.Name)
	}
	if got.ChartType != charts.ChartTypeColumn {
		t.Errorf("ChartType = %v, want %v", got.ChartType, charts.ChartTypeColumn)
	}
	if got.Position.Row != 5 {
		t.Errorf("Position.Row = %v, want 5", got.Position.Row)
	}
	if got.Position.Col != 5 {
		t.Errorf("Position.Col = %v, want 5", got.Position.Col)
	}
	if got.Size.Width != 600 {
		t.Errorf("Size.Width = %v, want 600", got.Size.Width)
	}
	if got.Size.Height != 400 {
		t.Errorf("Size.Height = %v, want 400", got.Size.Height)
	}
}

func TestChartsStore_Create_AllChartTypes(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	chartTypes := []charts.ChartType{
		charts.ChartTypeLine,
		charts.ChartTypeBar,
		charts.ChartTypeColumn,
		charts.ChartTypePie,
		charts.ChartTypeDoughnut,
		charts.ChartTypeArea,
		charts.ChartTypeScatter,
		charts.ChartTypeCombo,
		charts.ChartTypeStackedBar,
		charts.ChartTypeStackedColumn,
		charts.ChartTypeStackedArea,
		charts.ChartTypeRadar,
	}

	for _, chartType := range chartTypes {
		t.Run(string(chartType), func(t *testing.T) {
			now := FixedTime()
			chart := &charts.Chart{
				ID:        NewTestID(),
				SheetID:   f.Sheet.ID,
				Name:      string(chartType) + " Chart",
				ChartType: chartType,
				Position:  charts.Position{Row: 0, Col: 0},
				Size:      charts.Size{Width: 500, Height: 300},
				DataRanges: []charts.DataRange{{
					StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
				}},
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := store.Create(ctx, chart); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			got, err := store.GetByID(ctx, chart.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if got.ChartType != chartType {
				t.Errorf("ChartType = %v, want %v", got.ChartType, chartType)
			}
		})
	}
}

func TestChartsStore_Create_WithFullOptions(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	min := 0.0
	max := 100.0

	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "Full Options Chart",
		ChartType: charts.ChartTypeLine,
		Position:  charts.Position{Row: 10, Col: 10, OffsetX: 5, OffsetY: 5},
		Size:      charts.Size{Width: 800, Height: 500},
		DataRanges: []charts.DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 20, EndCol: 5, HasHeader: true,
		}},
		Title:    &charts.ChartTitle{Text: "Sales Report", FontSize: 18, Bold: true, FontColor: "#333"},
		Subtitle: &charts.ChartTitle{Text: "Q1 2024", FontSize: 14, Italic: true},
		Legend:   &charts.LegendConfig{Enabled: true, Position: "right", Alignment: "center"},
		Axes: &charts.AxesConfig{
			XAxis: &charts.AxisConfig{
				Title:     &charts.ChartTitle{Text: "Month"},
				GridLines: false,
			},
			YAxis: &charts.AxisConfig{
				Title:     &charts.ChartTitle{Text: "Sales ($)"},
				GridLines: true,
				Min:       &min,
				Max:       &max,
			},
		},
		Series: []charts.SeriesConfig{
			{
				Name:            "Revenue",
				Color:           "#4CAF50",
				BackgroundColor: "#4CAF5080",
				BorderWidth:     2,
				PointRadius:     4,
				Tension:         0.3,
			},
			{
				Name:  "Expenses",
				Color: "#F44336",
			},
		},
		Options: &charts.ChartOptions{
			BackgroundColor:   "#ffffff",
			BorderRadius:      8,
			Animated:          true,
			AnimationDuration: 1000,
			Interactive:       true,
			TooltipEnabled:    true,
			HoverMode:         "index",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, chart); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Verify title
	if got.Title == nil {
		t.Fatal("Title is nil")
	}
	if got.Title.Text != "Sales Report" {
		t.Errorf("Title.Text = %v, want Sales Report", got.Title.Text)
	}

	// Verify subtitle
	if got.Subtitle == nil {
		t.Fatal("Subtitle is nil")
	}
	if got.Subtitle.Text != "Q1 2024" {
		t.Errorf("Subtitle.Text = %v, want Q1 2024", got.Subtitle.Text)
	}

	// Verify axes
	if got.Axes == nil {
		t.Fatal("Axes is nil")
	}
	if got.Axes.YAxis == nil {
		t.Fatal("Axes.YAxis is nil")
	}
	if *got.Axes.YAxis.Min != 0 {
		t.Errorf("Axes.YAxis.Min = %v, want 0", *got.Axes.YAxis.Min)
	}

	// Verify series
	if len(got.Series) != 2 {
		t.Errorf("Series length = %d, want 2", len(got.Series))
	}
	if got.Series[0].Name != "Revenue" {
		t.Errorf("Series[0].Name = %v, want Revenue", got.Series[0].Name)
	}

	// Verify options
	if got.Options == nil {
		t.Fatal("Options is nil")
	}
	if got.Options.BorderRadius != 8 {
		t.Errorf("Options.BorderRadius = %v, want 8", got.Options.BorderRadius)
	}
}

func TestChartsStore_GetByID(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	// Create a chart first
	now := FixedTime()
	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "Get Test Chart",
		ChartType: charts.ChartTypePie,
		Position:  charts.Position{Row: 0, Col: 0},
		Size:      charts.Size{Width: 400, Height: 400},
		DataRanges: []charts.DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 1, HasHeader: true,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, chart); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != chart.ID {
		t.Errorf("ID = %v, want %v", got.ID, chart.ID)
	}
	if got.SheetID != f.Sheet.ID {
		t.Errorf("SheetID = %v, want %v", got.SheetID, f.Sheet.ID)
	}
}

func TestChartsStore_GetByID_NotFound(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent-id")
	if err != charts.ErrNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, charts.ErrNotFound)
	}
}

func TestChartsStore_ListBySheet(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()

	// Create multiple charts
	for i := 0; i < 3; i++ {
		chart := &charts.Chart{
			ID:        NewTestID(),
			SheetID:   f.Sheet.ID,
			Name:      "Chart " + string(rune('A'+i)),
			ChartType: charts.ChartTypeColumn,
			Position:  charts.Position{Row: i * 10, Col: 0},
			Size:      charts.Size{Width: 500, Height: 300},
			DataRanges: []charts.DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
			}},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := store.Create(ctx, chart); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	list, err := store.ListBySheet(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("ListBySheet() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListBySheet() returned %d charts, want 3", len(list))
	}
}

func TestChartsStore_ListBySheet_Empty(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	list, err := store.ListBySheet(ctx, f.Sheet.ID)
	if err != nil {
		t.Fatalf("ListBySheet() error = %v", err)
	}

	if list == nil {
		t.Error("ListBySheet() returned nil, want empty slice")
	}
	if len(list) != 0 {
		t.Errorf("ListBySheet() returned %d charts, want 0", len(list))
	}
}

func TestChartsStore_Update(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	// Create a chart
	now := FixedTime()
	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "Original Name",
		ChartType: charts.ChartTypeLine,
		Position:  charts.Position{Row: 0, Col: 0},
		Size:      charts.Size{Width: 500, Height: 300},
		DataRanges: []charts.DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, chart); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the chart
	chart.Name = "Updated Name"
	chart.ChartType = charts.ChartTypeBar
	chart.Position = charts.Position{Row: 10, Col: 10}
	chart.Size = charts.Size{Width: 800, Height: 500}
	chart.Title = &charts.ChartTitle{Text: "New Title"}
	chart.Legend = &charts.LegendConfig{Enabled: true, Position: "top"}

	if err := store.Update(ctx, chart); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", got.Name)
	}
	if got.ChartType != charts.ChartTypeBar {
		t.Errorf("ChartType = %v, want bar", got.ChartType)
	}
	if got.Position.Row != 10 {
		t.Errorf("Position.Row = %v, want 10", got.Position.Row)
	}
	if got.Size.Width != 800 {
		t.Errorf("Size.Width = %v, want 800", got.Size.Width)
	}
	if got.Title == nil || got.Title.Text != "New Title" {
		t.Errorf("Title.Text = %v, want New Title", got.Title)
	}
	if got.Legend == nil || got.Legend.Position != "top" {
		t.Errorf("Legend.Position = %v, want top", got.Legend)
	}
}

func TestChartsStore_Delete(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	// Create a chart
	now := FixedTime()
	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "To Be Deleted",
		ChartType: charts.ChartTypePie,
		Position:  charts.Position{Row: 0, Col: 0},
		Size:      charts.Size{Width: 400, Height: 400},
		DataRanges: []charts.DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 1, HasHeader: true,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, chart); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify it exists
	_, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("Chart should exist before delete: %v", err)
	}

	// Delete
	if err := store.Delete(ctx, chart.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = store.GetByID(ctx, chart.ID)
	if err != charts.ErrNotFound {
		t.Errorf("GetByID() after Delete() error = %v, want %v", err, charts.ErrNotFound)
	}
}

func TestChartsStore_Create_NullableFields(t *testing.T) {
	f := SetupTestFixture(t)
	store := NewChartsStore(f.DB)
	ctx := context.Background()

	now := FixedTime()
	chart := &charts.Chart{
		ID:        NewTestID(),
		SheetID:   f.Sheet.ID,
		Name:      "", // Empty name
		ChartType: charts.ChartTypeLine,
		Position:  charts.Position{Row: 0, Col: 0},
		Size:      charts.Size{Width: 500, Height: 300},
		DataRanges: []charts.DataRange{{
			StartRow: 0, StartCol: 0, EndRow: 5, EndCol: 2, HasHeader: true,
		}},
		// All nullable fields left as nil
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, chart); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.GetByID(ctx, chart.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Title != nil {
		t.Errorf("Title should be nil, got %v", got.Title)
	}
	if got.Subtitle != nil {
		t.Errorf("Subtitle should be nil, got %v", got.Subtitle)
	}
	if got.Axes != nil {
		t.Errorf("Axes should be nil, got %v", got.Axes)
	}
}
