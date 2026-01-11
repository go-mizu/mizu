// Package dashboard provides dashboard widget management functionality.
package dashboard

import (
	"context"
	"time"
)

// WidgetType represents the type of dashboard widget.
type WidgetType string

const (
	WidgetTypeChart   WidgetType = "chart"
	WidgetTypeNumber  WidgetType = "number"
	WidgetTypeList    WidgetType = "list"
	WidgetTypePivot   WidgetType = "pivot"
	WidgetTypeTable   WidgetType = "table"
)

// ChartType represents the type of chart.
type ChartType string

const (
	ChartTypeBar     ChartType = "bar"
	ChartTypeLine    ChartType = "line"
	ChartTypePie     ChartType = "pie"
	ChartTypeDonut   ChartType = "donut"
	ChartTypeScatter ChartType = "scatter"
	ChartTypeArea    ChartType = "area"
)

// AggregationType represents the type of aggregation.
type AggregationType string

const (
	AggSum           AggregationType = "sum"
	AggAvg           AggregationType = "avg"
	AggCount         AggregationType = "count"
	AggCountEmpty    AggregationType = "count_empty"
	AggCountFilled   AggregationType = "count_filled"
	AggPercentEmpty  AggregationType = "percent_empty"
	AggPercentFilled AggregationType = "percent_filled"
	AggMin           AggregationType = "min"
	AggMax           AggregationType = "max"
)

// Widget represents a dashboard widget.
type Widget struct {
	ID       string         `json:"id"`
	Type     WidgetType     `json:"type"`
	Title    string         `json:"title"`
	Position WidgetPosition `json:"position"`
	Size     WidgetSize     `json:"size"`
	Config   WidgetConfig   `json:"config"`
}

// WidgetPosition represents the widget's position on the grid.
type WidgetPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// WidgetSize represents the widget's dimensions.
type WidgetSize struct {
	Width  int `json:"width"`  // Grid units (1-12)
	Height int `json:"height"` // Grid units
}

// StackingType represents how bars should be stacked.
type StackingType string

const (
	StackNone     StackingType = "none"
	StackStandard StackingType = "standard"
	StackPercent  StackingType = "percent"
)

// WidgetConfig represents widget-specific configuration.
type WidgetConfig struct {
	// Data source
	TableID string   `json:"table_id,omitempty"`
	FieldID string   `json:"field_id,omitempty"`
	Filters []Filter `json:"filters,omitempty"`

	// Chart specific
	ChartType      ChartType       `json:"chart_type,omitempty"`
	GroupByField   string          `json:"group_by_field,omitempty"`
	ValueField     string          `json:"value_field,omitempty"`
	Aggregation    AggregationType `json:"aggregation,omitempty"`
	ShowLegend     bool            `json:"show_legend,omitempty"`
	ColorScheme    string          `json:"color_scheme,omitempty"`
	Stacking       StackingType    `json:"stacking,omitempty"`
	SecondaryGroup string          `json:"secondary_group,omitempty"` // For stacked charts

	// Number specific
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Precision int    `json:"precision,omitempty"`

	// List specific
	Limit         int      `json:"limit,omitempty"`
	SortField     string   `json:"sort_field,omitempty"`
	SortDirection string   `json:"sort_direction,omitempty"`
	VisibleFields []string `json:"visible_fields,omitempty"`
}

// Filter defines a filter condition.
type Filter struct {
	FieldID  string `json:"field_id"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// DashboardConfig represents the dashboard view configuration.
type DashboardConfig struct {
	Widgets  []Widget `json:"widgets"`
	GridCols int      `json:"grid_cols"` // Usually 12
}

// WidgetData represents the computed data for a widget.
type WidgetData struct {
	WidgetID string `json:"widget_id"`
	Type     string `json:"type"`
	Data     any    `json:"data"`
}

// ChartSeries represents a data series for stacked charts.
type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color"`
}

// ChartData represents chart data.
type ChartData struct {
	Labels []string       `json:"labels"`
	Values []float64      `json:"values"`
	Colors []string       `json:"colors,omitempty"`
	Series []ChartSeries  `json:"series,omitempty"` // For stacked charts
}

// NumberData represents number widget data.
type NumberData struct {
	Value     float64 `json:"value"`
	Formatted string  `json:"formatted"`
}

// ListData represents list widget data.
type ListData struct {
	Records []map[string]any `json:"records"`
	Total   int              `json:"total"`
}

// GetDataIn contains input for getting dashboard data.
type GetDataIn struct {
	ViewID    string   `json:"view_id"`
	WidgetIDs []string `json:"widget_ids,omitempty"` // If empty, get all
}

// GetDataOut contains output for getting dashboard data.
type GetDataOut struct {
	Widgets   []WidgetData `json:"widgets"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// AddWidgetIn contains input for adding a widget.
type AddWidgetIn struct {
	ViewID string `json:"view_id"`
	Widget Widget `json:"widget"`
}

// UpdateWidgetIn contains input for updating a widget.
type UpdateWidgetIn struct {
	ViewID   string `json:"view_id"`
	WidgetID string `json:"widget_id"`
	Widget   Widget `json:"widget"`
}

// DeleteWidgetIn contains input for deleting a widget.
type DeleteWidgetIn struct {
	ViewID   string `json:"view_id"`
	WidgetID string `json:"widget_id"`
}

// API defines the dashboard service interface.
type API interface {
	// GetDashboardData retrieves computed data for dashboard widgets.
	GetDashboardData(ctx context.Context, in *GetDataIn) (*GetDataOut, error)
	// AddWidget adds a new widget to the dashboard.
	AddWidget(ctx context.Context, in *AddWidgetIn) (*Widget, error)
	// UpdateWidget updates an existing widget.
	UpdateWidget(ctx context.Context, in *UpdateWidgetIn) (*Widget, error)
	// DeleteWidget removes a widget from the dashboard.
	DeleteWidget(ctx context.Context, in *DeleteWidgetIn) error
}
