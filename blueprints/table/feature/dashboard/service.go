package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/google/uuid"
)

// Service implements the dashboard API.
type Service struct {
	views   views.API
	records records.API
	fields  fields.API
}

// NewService creates a new dashboard service.
func NewService(views views.API, records records.API, fields fields.API) *Service {
	return &Service{
		views:   views,
		records: records,
		fields:  fields,
	}
}

// GetDashboardData retrieves computed data for dashboard widgets.
func (s *Service) GetDashboardData(ctx context.Context, in *GetDataIn) (*GetDataOut, error) {
	// Get the view
	view, err := s.views.GetByID(ctx, in.ViewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	// Parse dashboard config
	var config DashboardConfig
	if view.Config != nil {
		if err := json.Unmarshal(view.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse dashboard config: %w", err)
		}
	}

	// Get all records for the table
	recordList, err := s.records.List(ctx, view.TableID, records.ListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}
	tableRecords := recordList.Records

	// Get all fields for the table
	tableFields, err := s.fields.ListByTable(ctx, view.TableID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fields: %w", err)
	}

	// Build field map
	fieldMap := make(map[string]*fields.Field)
	for _, f := range tableFields {
		fieldMap[f.ID] = f
	}

	// Compute data for each widget
	var widgetData []WidgetData
	for _, widget := range config.Widgets {
		// Filter by widget IDs if specified
		if len(in.WidgetIDs) > 0 {
			found := false
			for _, id := range in.WidgetIDs {
				if id == widget.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		data, err := s.computeWidgetData(ctx, &widget, tableRecords, fieldMap)
		if err != nil {
			// Log error but continue with other widgets
			continue
		}

		widgetData = append(widgetData, WidgetData{
			WidgetID: widget.ID,
			Type:     string(widget.Type),
			Data:     data,
		})
	}

	return &GetDataOut{
		Widgets:   widgetData,
		UpdatedAt: time.Now(),
	}, nil
}

// computeWidgetData computes data for a single widget.
func (s *Service) computeWidgetData(
	ctx context.Context,
	widget *Widget,
	allRecords []*records.Record,
	fieldMap map[string]*fields.Field,
) (any, error) {
	// Apply widget filters
	filteredRecords := s.applyFilters(allRecords, widget.Config.Filters, fieldMap)

	switch widget.Type {
	case WidgetTypeNumber:
		return s.computeNumberData(filteredRecords, widget, fieldMap)
	case WidgetTypeChart:
		return s.computeChartData(filteredRecords, widget, fieldMap)
	case WidgetTypeList:
		return s.computeListData(filteredRecords, widget, fieldMap)
	default:
		return nil, fmt.Errorf("unsupported widget type: %s", widget.Type)
	}
}

// applyFilters filters records based on widget filters.
func (s *Service) applyFilters(
	allRecords []*records.Record,
	filters []Filter,
	fieldMap map[string]*fields.Field,
) []*records.Record {
	if len(filters) == 0 {
		return allRecords
	}

	var filtered []*records.Record
	for _, record := range allRecords {
		if s.matchesFilters(record, filters, fieldMap) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// matchesFilters checks if a record matches all filters.
func (s *Service) matchesFilters(
	record *records.Record,
	filters []Filter,
	fieldMap map[string]*fields.Field,
) bool {
	for _, filter := range filters {
		if !s.matchesFilter(record, filter, fieldMap) {
			return false
		}
	}
	return true
}

// matchesFilter checks if a record matches a single filter.
func (s *Service) matchesFilter(
	record *records.Record,
	filter Filter,
	fieldMap map[string]*fields.Field,
) bool {
	value, ok := record.Cells[filter.FieldID]
	if !ok {
		return filter.Operator == "is_empty"
	}

	switch filter.Operator {
	case "equals":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", filter.Value)
	case "not_equals":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", filter.Value)
	case "is_empty":
		return value == nil || value == ""
	case "is_not_empty":
		return value != nil && value != ""
	case "contains":
		strValue := fmt.Sprintf("%v", value)
		filterValue := fmt.Sprintf("%v", filter.Value)
		return len(strValue) > 0 && len(filterValue) > 0 &&
			(strValue == filterValue || containsString(strValue, filterValue))
	case "not_contains":
		strValue := fmt.Sprintf("%v", value)
		filterValue := fmt.Sprintf("%v", filter.Value)
		return !containsString(strValue, filterValue)
	case "starts_with":
		strValue := fmt.Sprintf("%v", value)
		filterValue := fmt.Sprintf("%v", filter.Value)
		return len(strValue) >= len(filterValue) && strValue[:len(filterValue)] == filterValue
	case "ends_with":
		strValue := fmt.Sprintf("%v", value)
		filterValue := fmt.Sprintf("%v", filter.Value)
		return len(strValue) >= len(filterValue) && strValue[len(strValue)-len(filterValue):] == filterValue
	case "in":
		// Handle array of values
		if arr, ok := filter.Value.([]any); ok {
			strValue := fmt.Sprintf("%v", value)
			for _, v := range arr {
				if fmt.Sprintf("%v", v) == strValue {
					return true
				}
			}
		}
		return false
	case "not_in":
		if arr, ok := filter.Value.([]any); ok {
			strValue := fmt.Sprintf("%v", value)
			for _, v := range arr {
				if fmt.Sprintf("%v", v) == strValue {
					return false
				}
			}
		}
		return true
	case "greater_than":
		return compareNumeric(value, filter.Value) > 0
	case "greater_than_or_equal":
		return compareNumeric(value, filter.Value) >= 0
	case "less_than":
		return compareNumeric(value, filter.Value) < 0
	case "less_than_or_equal":
		return compareNumeric(value, filter.Value) <= 0
	case "between":
		// Expect filter.Value to be [min, max]
		if arr, ok := filter.Value.([]any); ok && len(arr) == 2 {
			cmpMin := compareNumeric(value, arr[0])
			cmpMax := compareNumeric(value, arr[1])
			return cmpMin >= 0 && cmpMax <= 0
		}
		return false
	case "is_before":
		return compareDates(value, filter.Value) < 0
	case "is_after":
		return compareDates(value, filter.Value) > 0
	default:
		return true
	}
}

// compareNumeric compares two values as numbers. Returns -1, 0, or 1.
func compareNumeric(a, b any) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	if aFloat < bFloat {
		return -1
	}
	if aFloat > bFloat {
		return 1
	}
	return 0
}

// toFloat64 converts a value to float64.
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return f
	}
}

// compareDates compares two date values. Returns -1, 0, or 1.
func compareDates(a, b any) int {
	aTime := toTime(a)
	bTime := toTime(b)
	if aTime.Before(bTime) {
		return -1
	}
	if aTime.After(bTime) {
		return 1
	}
	return 0
}

// toTime converts a value to time.Time.
func toTime(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		// Try common date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"2006/01/02",
			"01/02/2006",
			"Jan 2, 2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// computeNumberData computes number widget data.
func (s *Service) computeNumberData(
	filteredRecords []*records.Record,
	widget *Widget,
	fieldMap map[string]*fields.Field,
) (*NumberData, error) {
	var value float64

	switch widget.Config.Aggregation {
	case AggCount:
		value = float64(len(filteredRecords))
	case AggCountEmpty:
		count := 0
		for _, record := range filteredRecords {
			if v, ok := record.Cells[widget.Config.FieldID]; !ok || v == nil || v == "" {
				count++
			}
		}
		value = float64(count)
	case AggCountFilled:
		count := 0
		for _, record := range filteredRecords {
			if v, ok := record.Cells[widget.Config.FieldID]; ok && v != nil && v != "" {
				count++
			}
		}
		value = float64(count)
	case AggPercentEmpty:
		if len(filteredRecords) == 0 {
			value = 0
		} else {
			count := 0
			for _, record := range filteredRecords {
				if v, ok := record.Cells[widget.Config.FieldID]; !ok || v == nil || v == "" {
					count++
				}
			}
			value = float64(count) / float64(len(filteredRecords)) * 100
		}
	case AggPercentFilled:
		if len(filteredRecords) == 0 {
			value = 0
		} else {
			count := 0
			for _, record := range filteredRecords {
				if v, ok := record.Cells[widget.Config.FieldID]; ok && v != nil && v != "" {
					count++
				}
			}
			value = float64(count) / float64(len(filteredRecords)) * 100
		}
	case AggSum, AggAvg, AggMin, AggMax:
		values := s.extractNumericValues(filteredRecords, widget.Config.FieldID)
		if len(values) == 0 {
			value = 0
		} else {
			switch widget.Config.Aggregation {
			case AggSum:
				for _, v := range values {
					value += v
				}
			case AggAvg:
				for _, v := range values {
					value += v
				}
				value /= float64(len(values))
			case AggMin:
				value = values[0]
				for _, v := range values[1:] {
					if v < value {
						value = v
					}
				}
			case AggMax:
				value = values[0]
				for _, v := range values[1:] {
					if v > value {
						value = v
					}
				}
			}
		}
	default:
		value = float64(len(filteredRecords))
	}

	// Format the value
	formatted := s.formatNumber(value, widget.Config)

	return &NumberData{
		Value:     value,
		Formatted: formatted,
	}, nil
}

// extractNumericValues extracts numeric values from records for a field.
func (s *Service) extractNumericValues(recs []*records.Record, fieldID string) []float64 {
	var values []float64
	for _, record := range recs {
		if v, ok := record.Cells[fieldID]; ok && v != nil {
			switch val := v.(type) {
			case float64:
				values = append(values, val)
			case int:
				values = append(values, float64(val))
			case int64:
				values = append(values, float64(val))
			case json.Number:
				if f, err := val.Float64(); err == nil {
					values = append(values, f)
				}
			}
		}
	}
	return values
}

// formatNumber formats a number with prefix and suffix.
func (s *Service) formatNumber(value float64, config WidgetConfig) string {
	formatted := fmt.Sprintf("%.*f", config.Precision, value)
	if config.Prefix != "" {
		formatted = config.Prefix + formatted
	}
	if config.Suffix != "" {
		formatted = formatted + config.Suffix
	}
	return formatted
}

// computeChartData computes chart widget data.
func (s *Service) computeChartData(
	filteredRecords []*records.Record,
	widget *Widget,
	fieldMap map[string]*fields.Field,
) (*ChartData, error) {
	// Group records by the group_by_field
	groupField := fieldMap[widget.Config.GroupByField]
	if groupField == nil {
		return nil, fmt.Errorf("group by field not found: %s", widget.Config.GroupByField)
	}

	// Check if stacking is enabled
	if widget.Config.Stacking != "" && widget.Config.Stacking != StackNone && widget.Config.SecondaryGroup != "" {
		return s.computeStackedChartData(filteredRecords, widget, fieldMap)
	}

	// Count or aggregate by group
	groups := make(map[string]float64)
	groupColors := make(map[string]string)

	for _, record := range filteredRecords {
		groupValue := "Unknown"
		if v, ok := record.Cells[widget.Config.GroupByField]; ok && v != nil {
			groupValue = fmt.Sprintf("%v", v)
		}

		switch widget.Config.Aggregation {
		case AggSum, AggAvg:
			if widget.Config.ValueField != "" {
				if v, ok := record.Cells[widget.Config.ValueField]; ok && v != nil {
					if numVal, ok := v.(float64); ok {
						groups[groupValue] += numVal
					}
				}
			} else {
				groups[groupValue]++
			}
		default: // Default to count
			groups[groupValue]++
		}
	}

	// Note: For select fields, colors could be parsed from Options JSON
	// For now, we use default colors based on label order
	_ = groupColors // Will be populated by default color palette

	// Convert to arrays
	var labels []string
	var values []float64
	var colors []string

	// Sort labels for consistent ordering
	for label := range groups {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	for _, label := range labels {
		values = append(values, groups[label])
		if color, ok := groupColors[label]; ok {
			colors = append(colors, color)
		} else {
			colors = append(colors, s.getDefaultColor(len(colors)))
		}
	}

	return &ChartData{
		Labels: labels,
		Values: values,
		Colors: colors,
	}, nil
}

// computeStackedChartData computes stacked chart data with multiple series.
func (s *Service) computeStackedChartData(
	filteredRecords []*records.Record,
	widget *Widget,
	fieldMap map[string]*fields.Field,
) (*ChartData, error) {
	// Create a 2D grouping: primary group -> secondary group -> count/value
	// Example: Status -> Priority -> count
	stackedData := make(map[string]map[string]float64)
	allSecondaryGroups := make(map[string]bool)

	for _, record := range filteredRecords {
		primaryValue := "Unknown"
		if v, ok := record.Cells[widget.Config.GroupByField]; ok && v != nil {
			primaryValue = fmt.Sprintf("%v", v)
		}

		secondaryValue := "Unknown"
		if v, ok := record.Cells[widget.Config.SecondaryGroup]; ok && v != nil {
			secondaryValue = fmt.Sprintf("%v", v)
		}

		if stackedData[primaryValue] == nil {
			stackedData[primaryValue] = make(map[string]float64)
		}
		allSecondaryGroups[secondaryValue] = true

		switch widget.Config.Aggregation {
		case AggSum, AggAvg:
			if widget.Config.ValueField != "" {
				if v, ok := record.Cells[widget.Config.ValueField]; ok && v != nil {
					if numVal, ok := v.(float64); ok {
						stackedData[primaryValue][secondaryValue] += numVal
					}
				}
			} else {
				stackedData[primaryValue][secondaryValue]++
			}
		default: // Default to count
			stackedData[primaryValue][secondaryValue]++
		}
	}

	// Get sorted primary labels (x-axis)
	var labels []string
	for label := range stackedData {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	// Get sorted secondary group names (series names)
	var seriesNames []string
	for name := range allSecondaryGroups {
		seriesNames = append(seriesNames, name)
	}
	sort.Strings(seriesNames)

	// Build series data
	var series []ChartSeries
	for i, seriesName := range seriesNames {
		var values []float64
		for _, label := range labels {
			value := 0.0
			if secondaryMap, ok := stackedData[label]; ok {
				if v, ok := secondaryMap[seriesName]; ok {
					value = v
				}
			}
			values = append(values, value)
		}

		series = append(series, ChartSeries{
			Name:   seriesName,
			Values: values,
			Color:  s.getDefaultColor(i),
		})
	}

	// Also compute simple values (totals) for backwards compatibility
	var values []float64
	for _, label := range labels {
		total := 0.0
		for _, val := range stackedData[label] {
			total += val
		}
		values = append(values, total)
	}

	return &ChartData{
		Labels: labels,
		Values: values,
		Series: series,
	}, nil
}

// getDefaultColor returns a default color from a palette.
func (s *Service) getDefaultColor(index int) string {
	palette := []string{
		"#3B82F6", // Blue
		"#10B981", // Green
		"#F59E0B", // Amber
		"#EF4444", // Red
		"#8B5CF6", // Purple
		"#EC4899", // Pink
		"#14B8A6", // Teal
		"#F97316", // Orange
	}
	return palette[index%len(palette)]
}

// computeListData computes list widget data.
func (s *Service) computeListData(
	filteredRecords []*records.Record,
	widget *Widget,
	fieldMap map[string]*fields.Field,
) (*ListData, error) {
	total := len(filteredRecords)

	// Sort if specified
	if widget.Config.SortField != "" {
		sort.Slice(filteredRecords, func(i, j int) bool {
			vi := filteredRecords[i].Cells[widget.Config.SortField]
			vj := filteredRecords[j].Cells[widget.Config.SortField]

			// Handle nil values
			if vi == nil && vj == nil {
				return false
			}
			if vi == nil {
				return widget.Config.SortDirection == "desc"
			}
			if vj == nil {
				return widget.Config.SortDirection == "asc"
			}

			strI := fmt.Sprintf("%v", vi)
			strJ := fmt.Sprintf("%v", vj)

			if widget.Config.SortDirection == "desc" {
				return strI > strJ
			}
			return strI < strJ
		})
	}

	// Limit
	limit := widget.Config.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > len(filteredRecords) {
		limit = len(filteredRecords)
	}
	filteredRecords = filteredRecords[:limit]

	// Convert to maps
	var recordMaps []map[string]any
	for _, record := range filteredRecords {
		recordMap := make(map[string]any)
		recordMap["id"] = record.ID

		// Include all visible fields or all fields if not specified
		if len(widget.Config.VisibleFields) > 0 {
			for _, fieldID := range widget.Config.VisibleFields {
				if v, ok := record.Cells[fieldID]; ok {
					if field, ok := fieldMap[fieldID]; ok {
						recordMap[field.Name] = v
					}
				}
			}
		} else {
			for fieldID, v := range record.Cells {
				if field, ok := fieldMap[fieldID]; ok {
					recordMap[field.Name] = v
				}
			}
		}
		recordMaps = append(recordMaps, recordMap)
	}

	return &ListData{
		Records: recordMaps,
		Total:   total,
	}, nil
}

// AddWidget adds a new widget to the dashboard.
func (s *Service) AddWidget(ctx context.Context, in *AddWidgetIn) (*Widget, error) {
	// Get the view
	view, err := s.views.GetByID(ctx, in.ViewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	// Parse dashboard config
	var config DashboardConfig
	if view.Config != nil {
		if err := json.Unmarshal(view.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse dashboard config: %w", err)
		}
	}
	if config.GridCols == 0 {
		config.GridCols = 12
	}

	// Generate ID if not provided
	widget := in.Widget
	if widget.ID == "" {
		widget.ID = "widget-" + uuid.New().String()[:8]
	}

	// Add the widget
	config.Widgets = append(config.Widgets, widget)

	// Save the updated config using SetConfig
	configMap := configToMap(config)
	err = s.views.SetConfig(ctx, in.ViewID, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to update view config: %w", err)
	}

	return &widget, nil
}

// configToMap converts DashboardConfig to map for SetConfig
func configToMap(config DashboardConfig) map[string]interface{} {
	// Convert through JSON to get a proper map
	data, _ := json.Marshal(config)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// UpdateWidget updates an existing widget.
func (s *Service) UpdateWidget(ctx context.Context, in *UpdateWidgetIn) (*Widget, error) {
	// Get the view
	view, err := s.views.GetByID(ctx, in.ViewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	// Parse dashboard config
	var config DashboardConfig
	if view.Config != nil {
		if err := json.Unmarshal(view.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse dashboard config: %w", err)
		}
	}

	// Find and update the widget
	found := false
	widget := in.Widget
	widget.ID = in.WidgetID // Ensure ID matches
	for i, w := range config.Widgets {
		if w.ID == in.WidgetID {
			config.Widgets[i] = widget
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("widget not found: %s", in.WidgetID)
	}

	// Save the updated config using SetConfig
	configMap := configToMap(config)
	err = s.views.SetConfig(ctx, in.ViewID, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to update view config: %w", err)
	}

	return &widget, nil
}

// DeleteWidget removes a widget from the dashboard.
func (s *Service) DeleteWidget(ctx context.Context, in *DeleteWidgetIn) error {
	// Get the view
	view, err := s.views.GetByID(ctx, in.ViewID)
	if err != nil {
		return fmt.Errorf("failed to get view: %w", err)
	}

	// Parse dashboard config
	var config DashboardConfig
	if view.Config != nil {
		if err := json.Unmarshal(view.Config, &config); err != nil {
			return fmt.Errorf("failed to parse dashboard config: %w", err)
		}
	}

	// Find and remove the widget
	found := false
	for i, w := range config.Widgets {
		if w.ID == in.WidgetID {
			config.Widgets = append(config.Widgets[:i], config.Widgets[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("widget not found: %s", in.WidgetID)
	}

	// Save the updated config using SetConfig
	configMap := configToMap(config)
	err = s.views.SetConfig(ctx, in.ViewID, configMap)
	if err != nil {
		return fmt.Errorf("failed to update view config: %w", err)
	}

	return nil
}
