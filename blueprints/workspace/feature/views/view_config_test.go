package views

import (
	"encoding/json"
	"testing"
)

func TestViewConfigSerialization(t *testing.T) {
	t.Run("table view config", func(t *testing.T) {
		config := ViewConfig{
			FrozenColumns:  2,
			RowHeight:      "medium",
			WrapCells:      true,
			WrapColumns:    map[string]bool{"col1": true, "col2": false},
			PropertyWidths: map[string]int{"col1": 200, "col2": 150},
			PropertyOrder:  []string{"col1", "col2", "col3"},
			Calculations: []Calculation{
				{PropertyID: "col1", Type: "count_all"},
				{PropertyID: "col2", Type: "sum"},
			},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.FrozenColumns != 2 {
			t.Errorf("expected FrozenColumns=2, got %d", decoded.FrozenColumns)
		}
		if decoded.RowHeight != "medium" {
			t.Errorf("expected RowHeight=medium, got %s", decoded.RowHeight)
		}
		if !decoded.WrapCells {
			t.Error("expected WrapCells=true")
		}
		if len(decoded.Calculations) != 2 {
			t.Errorf("expected 2 calculations, got %d", len(decoded.Calculations))
		}
		if decoded.Calculations[0].Type != "count_all" {
			t.Errorf("expected first calc type=count_all, got %s", decoded.Calculations[0].Type)
		}
	})

	t.Run("board view config", func(t *testing.T) {
		config := ViewConfig{
			CardSize:        "large",
			CardPreview:     "page_cover",
			CardPreviewProp: "cover_image",
			FitCardImage:    true,
			ColorColumns:    true,
			HideEmptyGroups: true,
			CardProperties:  []string{"status", "priority", "assignee"},
			ColumnOrder:     []string{"todo", "in_progress", "done"},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.CardSize != "large" {
			t.Errorf("expected CardSize=large, got %s", decoded.CardSize)
		}
		if decoded.CardPreview != "page_cover" {
			t.Errorf("expected CardPreview=page_cover, got %s", decoded.CardPreview)
		}
		if !decoded.FitCardImage {
			t.Error("expected FitCardImage=true")
		}
		if !decoded.ColorColumns {
			t.Error("expected ColorColumns=true")
		}
		if len(decoded.CardProperties) != 3 {
			t.Errorf("expected 3 card properties, got %d", len(decoded.CardProperties))
		}
	})

	t.Run("timeline view config", func(t *testing.T) {
		config := ViewConfig{
			TimeScale:         "weeks",
			StartDateProperty: "start_date",
			EndDateProperty:   "end_date",
			ShowTablePanel:    true,
			TablePanelWidth:   300,
			TablePanelProps:   []string{"title", "status"},
			Dependencies: []Dependency{
				{FromRowID: "row1", ToRowID: "row2", Type: "finish_to_start"},
				{FromRowID: "row2", ToRowID: "row3", Type: "start_to_start"},
			},
			ShowDependencies: true,
			ShowMilestones:   true,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.TimeScale != "weeks" {
			t.Errorf("expected TimeScale=weeks, got %s", decoded.TimeScale)
		}
		if len(decoded.Dependencies) != 2 {
			t.Errorf("expected 2 dependencies, got %d", len(decoded.Dependencies))
		}
		if decoded.Dependencies[0].Type != "finish_to_start" {
			t.Errorf("expected first dep type=finish_to_start, got %s", decoded.Dependencies[0].Type)
		}
		if !decoded.ShowDependencies {
			t.Error("expected ShowDependencies=true")
		}
	})

	t.Run("calendar view config", func(t *testing.T) {
		config := ViewConfig{
			CalendarMode:      "week",
			StartWeekOnMonday: true,
			EventColorProp:    "priority",
			ShowWeekends:      false,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.CalendarMode != "week" {
			t.Errorf("expected CalendarMode=week, got %s", decoded.CalendarMode)
		}
		if !decoded.StartWeekOnMonday {
			t.Error("expected StartWeekOnMonday=true")
		}
	})

	t.Run("gallery view config", func(t *testing.T) {
		config := ViewConfig{
			GalleryCardSize: "medium",
			PreviewSource:   "files",
			FilesPropertyID: "images",
			FitImage:        true,
			ShowTitle:       true,
			HideCardNames:   false,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.GalleryCardSize != "medium" {
			t.Errorf("expected GalleryCardSize=medium, got %s", decoded.GalleryCardSize)
		}
		if decoded.PreviewSource != "files" {
			t.Errorf("expected PreviewSource=files, got %s", decoded.PreviewSource)
		}
		if !decoded.FitImage {
			t.Error("expected FitImage=true")
		}
	})

	t.Run("chart view config", func(t *testing.T) {
		config := ViewConfig{
			ChartType:        "vertical_bar",
			ChartGroupBy:     "status",
			ChartAggregation: "sum",
			ChartXAxis: &AxisConfig{
				PropertyID:     "category",
				Sort:           "ascending",
				VisibleGroups:  []string{"group1", "group2"},
				OmitZeroValues: true,
			},
			ChartYAxis: &AxisConfig{
				PropertyID: "value",
				Cumulative: true,
			},
			ChartStyle: &StyleConfig{
				Height:          "large",
				GridLines:       true,
				XAxisLabels:     true,
				YAxisLabels:     true,
				DataLabels:      true,
				SmoothLine:      false,
				GradientArea:    true,
				ShowCenterValue: false,
				ShowLegend:      true,
				ColorScheme:     "colorful",
				ColorByValue:    true,
				Stacked:         false,
			},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.ChartType != "vertical_bar" {
			t.Errorf("expected ChartType=vertical_bar, got %s", decoded.ChartType)
		}
		if decoded.ChartXAxis == nil {
			t.Fatal("expected ChartXAxis to be set")
		}
		if decoded.ChartXAxis.Sort != "ascending" {
			t.Errorf("expected XAxis.Sort=ascending, got %s", decoded.ChartXAxis.Sort)
		}
		if decoded.ChartStyle == nil {
			t.Fatal("expected ChartStyle to be set")
		}
		if decoded.ChartStyle.Height != "large" {
			t.Errorf("expected ChartStyle.Height=large, got %s", decoded.ChartStyle.Height)
		}
	})

	t.Run("list view config", func(t *testing.T) {
		config := ViewConfig{
			ListShowProperties: []string{"title", "status", "priority"},
			ListCompact:        true,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ViewConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(decoded.ListShowProperties) != 3 {
			t.Errorf("expected 3 list properties, got %d", len(decoded.ListShowProperties))
		}
		if !decoded.ListCompact {
			t.Error("expected ListCompact=true")
		}
	})
}

func TestDependency(t *testing.T) {
	t.Run("dependency types", func(t *testing.T) {
		deps := []Dependency{
			{FromRowID: "a", ToRowID: "b", Type: "finish_to_start"},
			{FromRowID: "b", ToRowID: "c", Type: "start_to_start"},
			{FromRowID: "c", ToRowID: "d", Type: "finish_to_finish"},
			{FromRowID: "d", ToRowID: "e", Type: "start_to_finish"},
		}

		data, err := json.Marshal(deps)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded []Dependency
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(decoded) != 4 {
			t.Errorf("expected 4 dependencies, got %d", len(decoded))
		}

		expectedTypes := []string{"finish_to_start", "start_to_start", "finish_to_finish", "start_to_finish"}
		for i, dep := range decoded {
			if dep.Type != expectedTypes[i] {
				t.Errorf("dependency %d: expected type=%s, got %s", i, expectedTypes[i], dep.Type)
			}
		}
	})
}

func TestCalculationTypes(t *testing.T) {
	calculationTypes := []string{
		"count_all",
		"count_values",
		"count_unique",
		"count_empty",
		"count_not_empty",
		"percent_empty",
		"percent_not_empty",
		"sum",
		"average",
		"median",
		"min",
		"max",
		"range",
		"earliest_date",
		"latest_date",
		"date_range",
	}

	for _, calcType := range calculationTypes {
		t.Run(calcType, func(t *testing.T) {
			calc := Calculation{
				PropertyID: "test_prop",
				Type:       calcType,
			}

			data, err := json.Marshal(calc)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", calcType, err)
			}

			var decoded Calculation
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", calcType, err)
			}

			if decoded.Type != calcType {
				t.Errorf("expected Type=%s, got %s", calcType, decoded.Type)
			}
		})
	}
}

func TestViewTypes(t *testing.T) {
	viewTypes := []ViewType{
		ViewTable,
		ViewBoard,
		ViewList,
		ViewCalendar,
		ViewGallery,
		ViewTimeline,
		ViewChart,
	}

	for _, vt := range viewTypes {
		t.Run(string(vt), func(t *testing.T) {
			view := View{
				ID:         "test-view",
				DatabaseID: "test-db",
				Name:       "Test View",
				Type:       vt,
			}

			data, err := json.Marshal(view)
			if err != nil {
				t.Fatalf("failed to marshal view with type %s: %v", vt, err)
			}

			var decoded View
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal view with type %s: %v", vt, err)
			}

			if decoded.Type != vt {
				t.Errorf("expected Type=%s, got %s", vt, decoded.Type)
			}
		})
	}
}

func TestAxisConfig(t *testing.T) {
	t.Run("full axis config", func(t *testing.T) {
		axis := AxisConfig{
			PropertyID:     "category",
			Sort:           "descending",
			VisibleGroups:  []string{"a", "b", "c"},
			OmitZeroValues: true,
			Cumulative:     true,
		}

		data, err := json.Marshal(axis)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded AxisConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.PropertyID != "category" {
			t.Errorf("expected PropertyID=category, got %s", decoded.PropertyID)
		}
		if decoded.Sort != "descending" {
			t.Errorf("expected Sort=descending, got %s", decoded.Sort)
		}
		if len(decoded.VisibleGroups) != 3 {
			t.Errorf("expected 3 visible groups, got %d", len(decoded.VisibleGroups))
		}
		if !decoded.OmitZeroValues {
			t.Error("expected OmitZeroValues=true")
		}
		if !decoded.Cumulative {
			t.Error("expected Cumulative=true")
		}
	})
}

func TestStyleConfig(t *testing.T) {
	t.Run("full style config", func(t *testing.T) {
		style := StyleConfig{
			Height:          "extra_large",
			GridLines:       true,
			XAxisLabels:     true,
			YAxisLabels:     true,
			DataLabels:      true,
			SmoothLine:      true,
			GradientArea:    true,
			ShowCenterValue: true,
			ShowLegend:      true,
			ColorScheme:     "#2383e2",
			ColorByValue:    true,
			Stacked:         true,
		}

		data, err := json.Marshal(style)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded StyleConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.Height != "extra_large" {
			t.Errorf("expected Height=extra_large, got %s", decoded.Height)
		}
		if !decoded.GridLines {
			t.Error("expected GridLines=true")
		}
		if !decoded.SmoothLine {
			t.Error("expected SmoothLine=true")
		}
		if !decoded.Stacked {
			t.Error("expected Stacked=true")
		}
		if decoded.ColorScheme != "#2383e2" {
			t.Errorf("expected ColorScheme=#2383e2, got %s", decoded.ColorScheme)
		}
	})
}

func TestOmitEmptyFields(t *testing.T) {
	t.Run("empty config produces minimal JSON", func(t *testing.T) {
		config := ViewConfig{}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Should just be empty object
		if string(data) != "{}" {
			t.Errorf("expected empty JSON object, got %s", string(data))
		}
	})

	t.Run("partial config omits empty fields", func(t *testing.T) {
		config := ViewConfig{
			RowHeight: "tall",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		// Should only have row_height
		if len(decoded) != 1 {
			t.Errorf("expected 1 field, got %d fields: %v", len(decoded), decoded)
		}
		if decoded["row_height"] != "tall" {
			t.Errorf("expected row_height=tall, got %v", decoded["row_height"])
		}
	})
}
