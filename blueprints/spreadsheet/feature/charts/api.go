// Package charts provides chart management functionality.
package charts

import (
	"context"
	"time"
)

// ChartType represents the type of chart.
type ChartType string

const (
	ChartTypeLine          ChartType = "line"
	ChartTypeBar           ChartType = "bar"
	ChartTypeColumn        ChartType = "column"
	ChartTypePie           ChartType = "pie"
	ChartTypeDoughnut      ChartType = "doughnut"
	ChartTypeArea          ChartType = "area"
	ChartTypeScatter       ChartType = "scatter"
	ChartTypeCombo         ChartType = "combo"
	ChartTypeStackedBar    ChartType = "stacked_bar"
	ChartTypeStackedColumn ChartType = "stacked_column"
	ChartTypeStackedArea   ChartType = "stacked_area"
	ChartTypeRadar         ChartType = "radar"
	ChartTypeBubble        ChartType = "bubble"
	ChartTypeWaterfall     ChartType = "waterfall"
	ChartTypeHistogram     ChartType = "histogram"
	ChartTypeTreemap       ChartType = "treemap"
	ChartTypeGauge         ChartType = "gauge"
	ChartTypeCandlestick   ChartType = "candlestick"
)

// Chart represents a chart embedded in a sheet.
type Chart struct {
	ID         string         `json:"id"`
	SheetID    string         `json:"sheetId"`
	Name       string         `json:"name"`
	ChartType  ChartType      `json:"chartType"`
	Position   Position       `json:"position"`
	Size       Size           `json:"size"`
	DataRanges []DataRange    `json:"dataRanges"`
	Title      *ChartTitle    `json:"title,omitempty"`
	Subtitle   *ChartTitle    `json:"subtitle,omitempty"`
	Legend     *LegendConfig  `json:"legend,omitempty"`
	Axes       *AxesConfig    `json:"axes,omitempty"`
	Series     []SeriesConfig `json:"series,omitempty"`
	Options    *ChartOptions  `json:"options,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// Position represents the chart's position on the grid.
type Position struct {
	Row     int `json:"row"`     // Top-left row (0-indexed)
	Col     int `json:"col"`     // Top-left column (0-indexed)
	OffsetX int `json:"offsetX"` // Pixel offset within cell
	OffsetY int `json:"offsetY"` // Pixel offset within cell
}

// Size represents the chart's dimensions.
type Size struct {
	Width  int `json:"width"`  // Width in pixels
	Height int `json:"height"` // Height in pixels
}

// DataRange represents a range of cells for chart data.
type DataRange struct {
	SheetID   string `json:"sheetId,omitempty"` // Optional cross-sheet reference
	StartRow  int    `json:"startRow"`
	StartCol  int    `json:"startCol"`
	EndRow    int    `json:"endRow"`
	EndCol    int    `json:"endCol"`
	HasHeader bool   `json:"hasHeader"` // First row is header
}

// ChartTitle represents a title configuration.
type ChartTitle struct {
	Text       string `json:"text"`
	FontFamily string `json:"fontFamily,omitempty"`
	FontSize   int    `json:"fontSize,omitempty"`
	FontColor  string `json:"fontColor,omitempty"`
	Bold       bool   `json:"bold,omitempty"`
	Italic     bool   `json:"italic,omitempty"`
	Position   string `json:"position,omitempty"` // "top", "bottom", "left", "right", "none"
}

// LegendConfig represents legend configuration.
type LegendConfig struct {
	Enabled    bool   `json:"enabled"`
	Position   string `json:"position"`  // "top", "bottom", "left", "right", "none"
	Alignment  string `json:"alignment"` // "start", "center", "end"
	FontFamily string `json:"fontFamily,omitempty"`
	FontSize   int    `json:"fontSize,omitempty"`
	FontColor  string `json:"fontColor,omitempty"`
}

// AxesConfig represents axes configuration.
type AxesConfig struct {
	XAxis  *AxisConfig `json:"xAxis,omitempty"`
	YAxis  *AxisConfig `json:"yAxis,omitempty"`
	Y2Axis *AxisConfig `json:"y2Axis,omitempty"` // Secondary Y-axis for combo charts
}

// AxisConfig represents a single axis configuration.
type AxisConfig struct {
	Title       *ChartTitle `json:"title,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	StepSize    *float64    `json:"stepSize,omitempty"`
	GridLines   bool        `json:"gridLines"`
	GridColor   string      `json:"gridColor,omitempty"`
	TickColor   string      `json:"tickColor,omitempty"`
	LabelFormat string      `json:"labelFormat,omitempty"` // Number format
	Logarithmic bool        `json:"logarithmic,omitempty"`
	Reversed    bool        `json:"reversed,omitempty"`
}

// SeriesConfig represents a data series configuration.
type SeriesConfig struct {
	Name            string      `json:"name"`
	DataRange       *DataRange  `json:"dataRange,omitempty"`
	ChartType       ChartType   `json:"chartType,omitempty"` // For combo charts
	Color           string      `json:"color,omitempty"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	BorderWidth     int         `json:"borderWidth,omitempty"`
	PointStyle      string      `json:"pointStyle,omitempty"` // "circle", "triangle", "rect", "star", "cross"
	PointRadius     int         `json:"pointRadius,omitempty"`
	Fill            bool        `json:"fill,omitempty"`
	Tension         float64     `json:"tension,omitempty"` // Line smoothing (0-1)
	YAxisID         string      `json:"yAxisId,omitempty"` // "y" or "y2"
	Stack           string      `json:"stack,omitempty"`   // Stack group name
	DataLabels      *DataLabels `json:"dataLabels,omitempty"`
	Trendline       *Trendline  `json:"trendline,omitempty"`
}

// DataLabels represents data label configuration.
type DataLabels struct {
	Enabled   bool   `json:"enabled"`
	Position  string `json:"position"` // "top", "center", "bottom", "outside"
	Format    string `json:"format,omitempty"`
	FontSize  int    `json:"fontSize,omitempty"`
	FontColor string `json:"fontColor,omitempty"`
}

// Trendline represents trendline configuration.
type Trendline struct {
	Type         string `json:"type"` // "linear", "exponential", "polynomial", "moving_average"
	Degree       int    `json:"degree,omitempty"`
	Period       int    `json:"period,omitempty"`
	Color        string `json:"color,omitempty"`
	Width        int    `json:"width,omitempty"`
	ShowEquation bool   `json:"showEquation,omitempty"`
	ShowR2       bool   `json:"showR2,omitempty"`
}

// ChartOptions represents general chart options.
type ChartOptions struct {
	// General
	BackgroundColor string `json:"backgroundColor,omitempty"`
	BorderColor     string `json:"borderColor,omitempty"`
	BorderWidth     int    `json:"borderWidth,omitempty"`
	BorderRadius    int    `json:"borderRadius,omitempty"`

	// Animation
	Animated          bool `json:"animated"`
	AnimationDuration int  `json:"animationDuration,omitempty"`

	// Interaction
	Interactive    bool   `json:"interactive"`
	HoverMode      string `json:"hoverMode,omitempty"` // "nearest", "point", "index", "dataset"
	TooltipEnabled bool   `json:"tooltipEnabled"`

	// Pie/Doughnut specific
	CutoutPercentage int `json:"cutoutPercentage,omitempty"` // Doughnut hole size
	StartAngle       int `json:"startAngle,omitempty"`

	// 3D effect (visual only)
	Is3D bool `json:"is3D,omitempty"`

	// Sparkline mode (small inline chart)
	Sparkline bool `json:"sparkline,omitempty"`
}

// CreateIn contains chart creation input.
type CreateIn struct {
	SheetID    string         `json:"sheetId"`
	Name       string         `json:"name"`
	ChartType  ChartType      `json:"chartType"`
	Position   Position       `json:"position"`
	Size       Size           `json:"size"`
	DataRanges []DataRange    `json:"dataRanges"`
	Title      *ChartTitle    `json:"title,omitempty"`
	Subtitle   *ChartTitle    `json:"subtitle,omitempty"`
	Legend     *LegendConfig  `json:"legend,omitempty"`
	Axes       *AxesConfig    `json:"axes,omitempty"`
	Series     []SeriesConfig `json:"series,omitempty"`
	Options    *ChartOptions  `json:"options,omitempty"`
}

// UpdateIn contains chart update input.
type UpdateIn struct {
	Name       string         `json:"name,omitempty"`
	ChartType  ChartType      `json:"chartType,omitempty"`
	Position   *Position      `json:"position,omitempty"`
	Size       *Size          `json:"size,omitempty"`
	DataRanges []DataRange    `json:"dataRanges,omitempty"`
	Title      *ChartTitle    `json:"title,omitempty"`
	Subtitle   *ChartTitle    `json:"subtitle,omitempty"`
	Legend     *LegendConfig  `json:"legend,omitempty"`
	Axes       *AxesConfig    `json:"axes,omitempty"`
	Series     []SeriesConfig `json:"series,omitempty"`
	Options    *ChartOptions  `json:"options,omitempty"`
}

// ChartData represents resolved chart data from cell values.
type ChartData struct {
	Labels   []string    `json:"labels"`
	Datasets []Dataset   `json:"datasets"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Dataset represents a single dataset in chart data.
type Dataset struct {
	Label           string      `json:"label"`
	Data            []float64   `json:"data"`
	BackgroundColor interface{} `json:"backgroundColor,omitempty"` // string or []string
	BorderColor     interface{} `json:"borderColor,omitempty"`
	BorderWidth     int         `json:"borderWidth,omitempty"`
	Fill            bool        `json:"fill,omitempty"`
	Tension         float64     `json:"tension,omitempty"`
	PointRadius     int         `json:"pointRadius,omitempty"`
	PointStyle      string      `json:"pointStyle,omitempty"`
}

// API defines the charts service interface.
type API interface {
	// Create creates a new chart.
	Create(ctx context.Context, in *CreateIn) (*Chart, error)

	// GetByID retrieves a chart by ID.
	GetByID(ctx context.Context, id string) (*Chart, error)

	// ListBySheet lists charts in a sheet.
	ListBySheet(ctx context.Context, sheetID string) ([]*Chart, error)

	// Update updates a chart.
	Update(ctx context.Context, id string, in *UpdateIn) (*Chart, error)

	// Delete deletes a chart.
	Delete(ctx context.Context, id string) error

	// Duplicate duplicates a chart.
	Duplicate(ctx context.Context, id string) (*Chart, error)

	// GetData retrieves resolved chart data from cell values.
	GetData(ctx context.Context, id string) (*ChartData, error)
}

// Store defines the charts data access interface.
type Store interface {
	Create(ctx context.Context, chart *Chart) error
	GetByID(ctx context.Context, id string) (*Chart, error)
	ListBySheet(ctx context.Context, sheetID string) ([]*Chart, error)
	Update(ctx context.Context, chart *Chart) error
	Delete(ctx context.Context, id string) error
}

// CellDataProvider provides cell data for chart rendering.
type CellDataProvider interface {
	// GetCellValues retrieves cell values in a range.
	GetCellValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error)
}
